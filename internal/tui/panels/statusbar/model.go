// Package statusbar implements the single-line status bar at the bottom.
package statusbar

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/saheersk/lazymongo/internal/tui/msg"
	"github.com/saheersk/lazymongo/internal/tui/style"
)

// Model renders the bottom status bar.
type Model struct {
	width int

	connURI    string
	db         string
	collection string
	filter     string
	total      int64
	page       int
	pageCount  int

	flash    string // transient message (errors, confirmations)
	flashErr bool

	th *style.Theme
}

// New constructs a status bar.
func New(th *style.Theme, connURI string) Model {
	return Model{th: th, connURI: connURI}
}

// Init is a no-op.
func (m Model) Init() tea.Cmd { return nil }

// SetWidth updates the bar width.
func (m Model) SetWidth(w int) Model {
	m.width = w
	return m
}

// Update handles status-relevant messages.
func (m Model) Update(message tea.Msg) (Model, tea.Cmd) {
	switch message := message.(type) {
	case msg.CollectionSelected:
		m.db = message.DB
		m.collection = message.Collection
		m.filter = ""
		m.total = 0
		m.page = 0
		m.pageCount = 0
		m.flash = ""

	case msg.DocumentsLoaded:
		if message.Err == nil {
			m.total = message.Result.Total
			m.page = message.Result.Page
			ps := message.Result.PageSize
			if ps > 0 {
				m.pageCount = int(m.total)/ps + 1
			}
		}

	case msg.FilterChanged:
		if message.Filter != nil {
			short := message.Expr
			if len(short) > 24 {
				short = short[:23] + "…"
			}
			m.filter = short
		} else {
			m.filter = ""
		}

	case msg.StatusUpdate:
		m.flash = message.Text
		m.flashErr = message.IsErr
	}
	return m, nil
}

// View renders the full-width status bar.
func (m Model) View() string {
	if m.flash != "" {
		text := m.flash
		if m.flashErr {
			text = m.th.ErrText.Render(text)
		}
		return m.th.StatusBar.Width(m.width).Render(" " + text)
	}

	left := m.th.StatusConn.Render(" ● " + truncURI(m.connURI))

	var mid string
	if m.db != "" {
		path := m.db
		if m.collection != "" {
			path += " › " + m.collection
		}
		mid = m.th.StatusPath.Render("  " + path + "  ")
	}

	if m.filter != "" {
		mid += m.th.StatusFilter.Render(" ["+m.filter+"]  ")
	}

	var right string
	if m.total > 0 {
		right = m.th.StatusPager.Render(
			fmt.Sprintf("%d docs  pg %d/%d  ", m.total, m.page+1, m.pageCount),
		)
	}

	// fill gap between left+mid and right
	leftMid := left + mid
	gap := m.width - lipgloss.Width(leftMid) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	fill := m.th.StatusBar.Render(strings.Repeat(" ", gap))

	return leftMid + fill + right
}

func truncURI(uri string) string {
	const max = 40
	if len(uri) <= max {
		return uri
	}
	return uri[:max-1] + "…"
}
