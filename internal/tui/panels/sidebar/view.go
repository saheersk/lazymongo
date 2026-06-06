package sidebar

import (
	"fmt"
	"strings"
)

const (
	arrowRight = "▶"
	arrowDown  = "▼"
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
	header := m.th.TableHeader.
		Width(m.width - 4).
		Render("DATABASES")

	// Bottom bar: search input when active, key hints otherwise.
	bottomBar := m.renderBottomBar()

	// innerH = rows inside the border; used to pin bottomBar to the absolute bottom.
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

	// Rows available for items: innerH − header(1) − blank(1) − bottomBar(1)
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
func (m Model) renderBottomBar() string {
	if m.searchMode {
		prompt := m.th.StatusFilter.Render("/ ")
		return "  " + prompt + m.searchInput.View()
	}
	k := func(s string) string { return m.th.HelpKey.Render(s) }
	d := func(s string) string { return m.th.HelpDesc.Render(s) }
	return "  " + k("/") + " " + d("search") + "  " + k("?") + " " + d("help")
}

// padToBottom pads rows with blank lines then appends bottomBar so it is
// pinned to the absolute bottom of the inner panel area (lazygit style).
func (m Model) padToBottom(rows []string, innerH int, bottomBar string) []string {
	filler := innerH - len(rows) - 1 // -1 reserves the last row for bottomBar
	for i := 0; i < filler; i++ {
		rows = append(rows, "")
	}
	return append(rows, bottomBar)
}

func (m Model) renderVisibleItem(idx int, visible []treeItem) string {
	it := visible[idx]
	isCursor := idx == m.cursor
	maxW := m.width - 6

	var line string
	switch it.kind {
	case kindDatabase:
		arrow := arrowRight
		if it.expanded {
			arrow = arrowDown
		}
		label := fmt.Sprintf("%s %s", arrow, it.name)
		line = truncate(label, maxW)
		if isCursor {
			line = m.th.CursorItem.Render(line)
		} else {
			line = m.th.DatabaseItem.Render(line)
		}

	case kindCollection:
		label := indent + bullet + " " + it.name
		line = truncate(label, maxW)
		if isCursor {
			line = m.th.CursorItem.PaddingLeft(0).Render(line)
		} else {
			line = m.th.CollectionItem.PaddingLeft(0).Render(line)
		}
	}

	return "  " + line
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
