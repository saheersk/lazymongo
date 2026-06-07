package sidebar

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	arrowRight = "▸"
	arrowDown  = "▾"
	bullet     = "●"
	indent     = "  "
)

// View renders the sidebar panel.
func (m Model) View() string {
	inner := m.renderInner()

	border := m.th.InactiveBorder
	if m.focused {
		border = m.th.ActiveBorder
	}

	return border.
		Width(m.width - 2).
		Height(m.height - 2).
		Render(inner)
}

func (m Model) renderInner() string {
	innerW := m.width - 4
	header := m.th.PanelTitle.Width(innerW).Render("  DATABASES")

	bottomBar := m.renderBottomBar()

	// innerH = rows inside the border (excl. border lines themselves)
	innerH := m.height - 2
	if innerH < 3 {
		innerH = 3
	}

	if m.loading {
		rows := []string{header, "", "  " + m.spinner.View() + " loading…"}
		rows = m.padToBottom(rows, innerH, bottomBar)
		return strings.Join(rows, "\n")
	}

	if m.err != nil {
		rows := []string{header, "", m.th.ErrText.Render("  " + m.err.Error())}
		rows = m.padToBottom(rows, innerH, bottomBar)
		return strings.Join(rows, "\n")
	}

	visible := m.visibleItems()

	if len(visible) == 0 {
		empty := "  no databases found"
		if m.searchMode && m.searchInput.Value() != "" {
			empty = "  no results"
		}
		rows := []string{header, "", m.th.DimText.Render(empty)}
		rows = m.padToBottom(rows, innerH, bottomBar)
		return strings.Join(rows, "\n")
	}

	// Rows: header(1) + spacer(1) + items... + bottomBar(1)
	visibleRows := innerH - 3
	if visibleRows < 1 {
		visibleRows = 1
	}

	start, end := viewportWindow(m.cursor, len(visible), visibleRows)

	var rows []string
	rows = append(rows, header, "")
	for i := start; i < end; i++ {
		rows = append(rows, m.renderVisibleItem(i, visible))
	}

	rows = m.padToBottom(rows, innerH, bottomBar)
	return strings.Join(rows, "\n")
}

// renderBottomBar returns a search input line when searching, or a hint line otherwise.
// Hints are progressively shortened so the bar never exceeds the panel width.
// Context-sensitive hints are shown based on whether cursor is on a collection or database.
func (m Model) renderBottomBar() string {
	if m.searchMode {
		prompt := m.th.StatusFilter.Render("/")
		return " " + prompt + " " + m.searchInput.View()
	}
	k := func(s string) string { return m.th.HelpKey.Render(s) }
	d := func(s string) string { return m.th.HelpDesc.Render(s) }
	avail := m.width - 4

	if m.CursorIsCollection() {
		// Collection-level hints
		full := "  " + k("n") + " " + d("new") + "  " + k("r") + " " + d("rename") + "  " + k("D") + " " + d("drop") + "  " + k("s") + " " + d("stats") + "  " + k("/") + " " + d("search") + "  " + k("?") + " " + d("help") + "  " + k("T") + " " + d("theme")
		if lipgloss.Width(full) <= avail {
			return full
		}
		med := "  " + k("n") + " " + d("new") + "  " + k("r") + " " + d("rename") + "  " + k("D") + " " + d("drop") + "  " + k("s") + " " + d("stats") + "  " + k("/") + " " + d("search") + "  " + k("?") + " " + d("help")
		if lipgloss.Width(med) <= avail {
			return med
		}
		return "  " + k("n") + "  " + k("r") + "  " + k("D") + "  " + k("s") + "  " + k("/") + "  " + k("?")
	}

	// Database-level hints
	full := "  " + k("n") + " " + d("new col") + "  " + k("D") + " " + d("drop db") + "  " + k("/") + " " + d("search") + "  " + k("?") + " " + d("help") + "  " + k("T") + " " + d("theme")
	if lipgloss.Width(full) <= avail {
		return full
	}
	med := "  " + k("n") + " " + d("new col") + "  " + k("D") + " " + d("drop db") + "  " + k("/") + " " + d("search") + "  " + k("?") + " " + d("help")
	if lipgloss.Width(med) <= avail {
		return med
	}
	return "  " + k("n") + "  " + k("D") + "  " + k("/") + "  " + k("?") + "  " + k("T")
}

// padToBottom pads rows with blank lines then appends bottomBar so it is
// pinned to the absolute bottom of the inner panel area (lazygit style).
func (m Model) padToBottom(rows []string, innerH int, bottomBar string) []string {
	filler := innerH - len(rows) - 1
	for i := 0; i < filler; i++ {
		rows = append(rows, "")
	}
	return append(rows, bottomBar)
}

func (m Model) renderVisibleItem(idx int, visible []treeItem) string {
	it := visible[idx]
	isCursor := idx == m.cursor
	innerW := m.width - 4

	switch it.kind {
	case kindDatabase:
		arrow := arrowRight
		if it.expanded {
			arrow = arrowDown
		}
		label := fmt.Sprintf("  %s %s", arrow, it.name)
		label = truncate(label, innerW)
		if isCursor {
			return m.th.CursorItem.Width(innerW).Render(label)
		}
		return m.th.DatabaseItem.Render(label)

	case kindCollection:
		label := "    " + bullet + " " + it.name
		label = truncate(label, innerW)
		if isCursor {
			return m.th.CursorItem.Width(innerW).Render(label)
		}
		return m.th.CollectionItem.Render(label)
	}

	return ""
}

func viewportWindow(cursor, total, rows int) (int, int) {
	start := 0
	end := total
	if end > rows {
		end = rows
	}
	if cursor >= end {
		start = cursor - rows + 1
		end = cursor + 1
	}
	if end > total {
		end = total
	}
	return start, end
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
