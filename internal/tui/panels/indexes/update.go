package indexes

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/tui/msg"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Update handles all messages for the index panel.
func (m Model) Update(message tea.Msg) (Model, tea.Cmd) {
	switch message := message.(type) {

	case msg.CollectionSelected:
		return m.Load(message.DB, message.Collection)

	case msg.IndexesLoaded:
		m.loading = false
		if message.Err != nil {
			m.err = message.Err
			return m, nil
		}
		m.err = nil
		m.indexes = message.Indexes
		m.stats = message.Stats
		return m.clamp(), nil

	case msg.IndexEditorDone:
		if message.Err != nil {
			return m, statusCmd("index error: " + message.Err.Error())
		}
		return m, m.createIndex(m.db, m.collection, message.Keys, message.Unique, message.Sparse)

	case msg.IndexCreated:
		if message.Err != nil {
			return m, statusCmd("create index failed: " + message.Err.Error())
		}
		m.loading = true
		return m, tea.Batch(
			m.spinner.Tick,
			m.fetchIndexes(m.db, m.collection),
			statusCmd("index created: "+message.Name),
		)

	case msg.IndexDropped:
		if message.Err != nil {
			return m, statusCmd("drop index failed: " + message.Err.Error())
		}
		if m.cursor > 0 {
			m.cursor--
		}
		m.loading = true
		return m, tea.Batch(
			m.spinner.Tick,
			m.fetchIndexes(m.db, m.collection),
			statusCmd("index dropped"),
		)

	case tea.KeyMsg:
		return m.handleKey(message)

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(message)
		return m, cmd
	}
}

func (m Model) handleKey(key tea.KeyMsg) (Model, tea.Cmd) {
	if m.deleteConfirm {
		return m.handleDropConfirm(key)
	}

	switch key.String() {
	case "j", "down":
		m.cursor++
		return m.clamp(), nil
	case "k", "up":
		m.cursor--
		return m.clamp(), nil
	case "g":
		m.cursor = 0
		return m, nil
	case "G":
		m.cursor = len(m.indexes) - 1
		return m.clamp(), nil
	case "R":
		if m.db == "" {
			return m, nil
		}
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.fetchIndexes(m.db, m.collection))
	case "n":
		if m.db == "" {
			return m, nil
		}
		return m.openCreateEditor()
	case "d":
		idx := m.ActiveIndex()
		if idx == nil || idx.Name == "_id_" {
			return m, statusCmd("cannot drop the _id index")
		}
		m.deleteConfirm = true
		return m, nil
	}
	return m, nil
}

func (m Model) handleDropConfirm(key tea.KeyMsg) (Model, tea.Cmd) {
	m.deleteConfirm = false
	if key.String() == "y" || key.String() == "Y" {
		idx := m.ActiveIndex()
		if idx == nil {
			return m, nil
		}
		return m, m.dropIndex(m.db, m.collection, idx.Name)
	}
	return m, nil
}

// ── editor ────────────────────────────────────────────────────────────────────

const indexTemplate = `{
  "keys": { "fieldName": 1 },
  "unique": false,
  "sparse": false
}`

func (m Model) openCreateEditor() (Model, tea.Cmd) {
	ec, err := buildIndexEditorCmd(indexTemplate)
	if err != nil {
		return m, statusCmd("error: " + err.Error())
	}
	return m, tea.ExecProcess(ec.cmd, func(execErr error) tea.Msg {
		defer os.Remove(ec.path)
		if execErr != nil {
			return msg.IndexEditorDone{Err: execErr}
		}
		return readIndexFile(ec.path)
	})
}

type indexEditorCmd struct {
	cmd  *exec.Cmd
	path string
}

func buildIndexEditorCmd(content string) (indexEditorCmd, error) {
	f, err := os.CreateTemp("", "lazymongo-index-*.json")
	if err != nil {
		return indexEditorCmd{}, err
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return indexEditorCmd{}, err
	}
	f.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	parts := strings.Fields(editor)
	args := append(parts[1:], f.Name())
	cmd := exec.Command(parts[0], args...)
	return indexEditorCmd{cmd: cmd, path: f.Name()}, nil
}

func readIndexFile(path string) msg.IndexEditorDone {
	data, err := os.ReadFile(path)
	if err != nil {
		return msg.IndexEditorDone{Err: err}
	}
	data = []byte(strings.TrimSpace(string(data)))
	if len(data) == 0 {
		return msg.IndexEditorDone{Err: fmt.Errorf("empty file — no changes")}
	}

	// Parse {"keys":{...},"unique":bool,"sparse":bool}
	var raw struct {
		Keys   bson.M `bson:"keys"   json:"keys"`
		Unique bool   `bson:"unique" json:"unique"`
		Sparse bool   `bson:"sparse" json:"sparse"`
	}
	if err := bson.UnmarshalExtJSON(data, false, &raw); err != nil {
		return msg.IndexEditorDone{Err: fmt.Errorf("invalid JSON: %w", err)}
	}
	if len(raw.Keys) == 0 {
		return msg.IndexEditorDone{Err: fmt.Errorf(`"keys" must not be empty`)}
	}

	var keys bson.D
	for k, v := range raw.Keys {
		keys = append(keys, bson.E{Key: k, Value: v})
	}
	return msg.IndexEditorDone{Keys: keys, Unique: raw.Unique, Sparse: raw.Sparse}
}

func statusCmd(text string) tea.Cmd {
	return func() tea.Msg { return msg.StatusUpdate{Text: text} }
}
