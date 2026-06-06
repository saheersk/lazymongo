package sidebar

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

	if m.loading {
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			"",
			"  "+m.spinner.View()+" loading…",
		)
	}

	if m.err != nil {
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			"",
			m.th.ErrText.Render("  "+m.err.Error()),
		)
	}

	if len(m.items) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			"",
			m.th.DimText.Render("  no databases found"),
		)
	}

	// how many rows fit inside the border
	visibleRows := m.height - 4 // 2 border + 1 header + 1 blank
	if visibleRows < 1 {
		visibleRows = 1
	}

	// viewport: keep cursor visible
	start, end := m.viewportWindow(visibleRows)

	var rows []string
	rows = append(rows, header, "")

	for i := start; i < end; i++ {
		rows = append(rows, m.renderItem(i))
	}

	// scroll indicator
	if len(m.items) > visibleRows {
		pct := int(float64(m.cursor) / float64(len(m.items)-1) * 100)
		rows = append(rows, m.th.DimText.Render(fmt.Sprintf("  %d%%", pct)))
	}

	return strings.Join(rows, "\n")
}

func (m Model) renderItem(idx int) string {
	it := m.items[idx]
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

// viewportWindow returns start/end indices to keep cursor visible.
func (m Model) viewportWindow(rows int) (int, int) {
	start := 0
	end := len(m.items)
	if end > rows {
		end = rows
	}

	if m.cursor >= end {
		start = m.cursor - rows + 1
		end = m.cursor + 1
	}
	if end > len(m.items) {
		end = len(m.items)
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
