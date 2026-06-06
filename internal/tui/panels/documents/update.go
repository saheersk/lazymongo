package documents

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/tui/msg"
	"github.com/saheersk/lazymongo/internal/util"
)

// Update handles all messages for the documents panel.
func (m Model) Update(message tea.Msg) (Model, tea.Cmd) {
	switch message := message.(type) {

	case msg.CollectionSelected:
		m.db = message.DB
		m.collection = message.Collection
		m.page = 0
		m.cursor = 0
		m.docs = nil
		m.columns = nil
		m.filter = nil
		m.sort = nil
		m.err = nil
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadPage(0))

	case msg.DocumentsLoaded:
		m.loading = false
		if message.Err != nil {
			m.err = message.Err
			return m, nil
		}
		m.err = nil
		m.docs = message.Result.Docs
		m.total = message.Result.Total
		m.page = message.Result.Page
		m.columns = util.BuildColumns(m.docs, maxColumns(m.width))
		if m.pendingBottom {
			m.cursor = len(m.docs) - 1
			m.pendingBottom = false
		}
		return m.clamp(), nil

	case msg.FilterChanged:
		m.filter = message.Filter
		m.sort = message.Sort
		m.page = 0
		m.cursor = 0
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadPage(0))

	case tea.KeyMsg:
		return m.handleKey(message)

	default:
		var spCmd tea.Cmd
		m.spinner, spCmd = m.spinner.Update(message)
		return m, spCmd
	}
}

func (m Model) handleKey(key tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case isKey(key, m.km.Down):
		m.cursor++
		if m.cursor >= len(m.docs) && m.page < m.pageCount()-1 {
			m.cursor = 0
			m.page++
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadPage(m.page))
		}
		return m.clamp(), nil

	case isKey(key, m.km.Up):
		m.cursor--
		if m.cursor < 0 && m.page > 0 {
			m.page--
			m.loading = true
			m.cursor = m.pageSize - 1
			return m, tea.Batch(m.spinner.Tick, m.loadPage(m.page))
		}
		return m.clamp(), nil

	case isKey(key, m.km.Top):
		if m.page == 0 {
			m.cursor = 0
			return m, nil
		}
		m.page = 0
		m.cursor = 0
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadPage(0))

	case isKey(key, m.km.Bottom):
		lastPage := m.pageCount() - 1
		if m.page == lastPage {
			m.cursor = len(m.docs) - 1
			return m.clamp(), nil
		}
		m.page = lastPage
		m.cursor = 0
		m.pendingBottom = true
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadPage(lastPage))

	case isKey(key, m.km.PageDown):
		if m.page < m.pageCount()-1 {
			m.page++
			m.cursor = 0
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadPage(m.page))
		}
		return m, nil

	case isKey(key, m.km.PageUp):
		if m.page > 0 {
			m.page--
			m.cursor = 0
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadPage(m.page))
		}
		return m, nil

	case isKey(key, m.km.Refresh):
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadPage(m.page))

	case isKey(key, m.km.Select):
		doc := m.ActiveDoc()
		if doc == nil {
			return m, nil
		}
		return m, func() tea.Msg {
			return msg.DocumentSelected{Doc: doc}
		}

	case key.String() == "y":
		doc := m.ActiveDoc()
		if doc == nil {
			return m, nil
		}
		id := util.FormatValue(doc["_id"])
		_ = util.CopyToClipboard(id)
		return m, statusCmd("copied _id: " + id)

	case key.String() == "Y":
		doc := m.ActiveDoc()
		if doc == nil {
			return m, nil
		}
		raw, err := util.BSONToJSON(doc)
		if err == nil {
			_ = util.CopyToClipboard(raw)
		}
		return m, statusCmd("copied document JSON to clipboard")
	}

	return m, nil
}

func statusCmd(text string) tea.Cmd {
	return func() tea.Msg {
		return msg.StatusUpdate{Text: text}
	}
}

func isKey(km tea.KeyMsg, b interface{ Keys() []string }) bool {
	for _, k := range b.Keys() {
		if km.String() == k {
			return true
		}
	}
	return false
}

// maxColumns decides how many document fields to show based on panel width.
func maxColumns(width int) int {
	switch {
	case width < 80:
		return 2
	case width < 120:
		return 4
	default:
		return 5
	}
}
