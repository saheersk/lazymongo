// Package detail implements the right-hand single-document JSON viewer.
package detail

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/tui/keymap"
	"github.com/saheersk/lazymongo/internal/tui/style"
	"github.com/saheersk/lazymongo/internal/util"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Model is the bubbletea model for the document detail panel.
type Model struct {
	doc     bson.M
	docID   string // extracted _id string for copy
	rawJSON string // uncoloured JSON, used for clipboard copy

	filterExpr string // active filter from the documents panel (for display)

	viewport viewport.Model
	ready    bool
	focused  bool

	width, height int

	th *style.Theme
	km *keymap.Map
}

// New returns an empty detail panel. It becomes visible once a DocumentSelected
// message arrives.
func New(th *style.Theme, km *keymap.Map) Model {
	return Model{th: th, km: km}
}

// Init is a no-op; content is populated by DocumentSelected messages.
func (m Model) Init() tea.Cmd { return nil }

// SetSize updates panel dimensions and reconfigures the internal viewport.
func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h

	innerW := w - 4 // 2 border + 2 padding
	innerH := h - 4 // 2 border + 1 title + 1 blank

	// Reserve a 2-column gutter on the right for the scrollbar track.
	vpW := innerW - 2
	if vpW < 1 {
		vpW = 1
	}

	if !m.ready {
		m.viewport = viewport.New(vpW, innerH)
		m.viewport.YPosition = 0
		m.ready = true
	} else {
		m.viewport.Width = vpW
		m.viewport.Height = innerH
	}

	// re-render content at new width if we already have a doc
	if m.doc != nil {
		m.viewport.SetContent(m.renderContent())
	}
	return m
}

// SetFocused controls focused-border rendering.
func (m Model) SetFocused(f bool) Model {
	m.focused = f
	return m
}

// SetFilterExpr stores the active filter expression for display in the header.
func (m Model) SetFilterExpr(expr string) Model {
	m.filterExpr = expr
	return m
}

// HasDoc reports whether a document is currently loaded.
func (m Model) HasDoc() bool { return m.doc != nil }

// DocID returns the _id of the loaded document as a string.
func (m Model) DocID() string { return m.docID }

// RawJSON returns the uncoloured JSON of the loaded document.
func (m Model) RawJSON() string { return m.rawJSON }

// load stores a new document and refreshes the viewport content.
func (m Model) load(doc bson.M) Model {
	m.doc = doc

	raw, err := util.BSONToJSON(doc)
	if err != nil {
		raw = "error rendering document: " + err.Error()
	}
	m.rawJSON = raw
	m.docID = util.FormatValue(doc["_id"])

	if m.ready {
		m.viewport.SetContent(m.renderContent())
		m.viewport.GotoTop()
	}
	return m
}

// renderContent returns syntax-highlighted JSON for the viewport.
func (m Model) renderContent() string {
	if m.doc == nil || m.rawJSON == "" {
		return ""
	}
	return util.SyntaxHighlight(m.rawJSON, m.th)
}

// ScrollPercent returns the viewport scroll position as 0–100.
func (m Model) ScrollPercent() int {
	return int(m.viewport.ScrollPercent() * 100)
}
