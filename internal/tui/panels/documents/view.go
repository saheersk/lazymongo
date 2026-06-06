package documents

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/saheersk/lazymongo/internal/util"
)

// View renders the document list panel.
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
	title := m.Collection()
	if title == "" {
		title = "DOCUMENTS"
	}

	header := m.th.TableHeader.
		Width(m.width - 4).
		Render(title)

	if m.db == "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			header, "",
			m.th.DimText.Render("  ← select a collection"),
		)
	}

	if m.loading {
		return lipgloss.JoinVertical(lipgloss.Left,
			header, "",
			"  "+m.spinner.View()+" loading…",
		)
	}

	if m.err != nil {
		return lipgloss.JoinVertical(lipgloss.Left,
			header, "",
			m.th.ErrText.Render("  "+m.err.Error()),
		)
	}

	if len(m.docs) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			header, "",
			m.th.DimText.Render("  no documents found"),
		)
	}

	innerW := m.width - 4 // account for border + padding
	colWidths := distributeWidths(m.columns, innerW)

	// column header row
	colHeader := m.renderHeaderRow(colWidths)

	// visible rows
	visibleRows := m.height - 6 // border(2) + title(1) + blank(1) + colheader(1) + pager(1)
	if visibleRows < 1 {
		visibleRows = 1
	}

	start, end := m.viewportWindow(visibleRows)
	var rows []string
	for i := start; i < end; i++ {
		rows = append(rows, m.renderDocRow(i, colWidths))
	}

	pager := m.renderPager()

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		colHeader,
		strings.Join(rows, "\n"),
		pager,
	)
}

func (m Model) renderHeaderRow(widths []int) string {
	var parts []string
	for i, col := range m.columns {
		if i < len(widths) {
			parts = append(parts, util.PadRight(col, widths[i]))
		}
	}
	return m.th.TableHeader.Render("  " + strings.Join(parts, " "))
}

func (m Model) renderDocRow(idx int, widths []int) string {
	doc := m.docs[idx]
	isCursor := idx == m.cursor

	var parts []string
	for i, col := range m.columns {
		if i >= len(widths) {
			break
		}
		val := util.FormatValue(doc[col])
		parts = append(parts, util.PadRight(val, widths[i]))
	}

	line := "  " + strings.Join(parts, " ")

	if isCursor {
		return m.th.TableSelected.Width(m.width - 4).Render(line)
	}
	if idx%2 == 0 {
		return m.th.TableRow.Render(line)
	}
	return m.th.TableRowAlt.Render(line)
}

func (m Model) renderPager() string {
	from := m.page*m.pageSize + 1
	to := from + len(m.docs) - 1
	info := fmt.Sprintf("  %d-%d of %d  •  page %d/%d",
		from, to, m.total, m.page+1, m.pageCount())
	return m.th.StatusPager.Render(info)
}

// viewportWindow returns the slice indices to keep the cursor on screen.
func (m Model) viewportWindow(rows int) (int, int) {
	start := 0
	end := len(m.docs)
	if end > rows {
		end = rows
	}
	if m.cursor >= end {
		start = m.cursor - rows + 1
		end = m.cursor + 1
	}
	if end > len(m.docs) {
		end = len(m.docs)
	}
	return start, end
}

// distributeWidths assigns pixel-widths to each column proportionally,
// giving _id a fixed width and splitting the rest evenly.
func distributeWidths(cols []string, totalW int) []int {
	if len(cols) == 0 {
		return nil
	}
	widths := make([]int, len(cols))

	// _id always gets 24 chars (ObjectID hex length)
	const idW = 24
	widths[0] = idW
	remaining := totalW - idW - len(cols) // account for spaces

	if len(cols) > 1 && remaining > 0 {
		each := remaining / (len(cols) - 1)
		for i := 1; i < len(cols); i++ {
			widths[i] = each
		}
	}
	return widths
}
