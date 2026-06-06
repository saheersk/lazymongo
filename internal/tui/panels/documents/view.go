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
	// Show aggregate mode badge.
	if m.aggMode {
		title += "  " + m.th.ErrText.Render("[AGG]")
	}
	// Show active filter/sort badges in the title.
	if m.filterExpr != "" {
		short := m.filterExpr
		if len(short) > 22 {
			short = short[:21] + "…"
		}
		title += "  " + m.th.StatusFilter.Render("[f: "+short+"]")
	}
	if m.sortExpr != "" {
		short := m.sortExpr
		if len(short) > 16 {
			short = short[:15] + "…"
		}
		title += " " + m.th.StatusPager.Render("[s: "+short+"]")
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
		noResult := "  no documents found"
		if m.filterExpr != "" {
			noResult += "  (r to clear filter)"
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			header, "",
			m.th.DimText.Render(noResult),
		)
	}

	innerW := m.width - 4
	colWidths := distributeWidths(m.columns, innerW)

	colHeader := m.renderHeaderRow(colWidths)

	// visibleRows: subtract 1 extra when input bar is visible (replaces pager).
	visibleRows := m.height - 6 // border(2) + title(1) + blank(1) + colheader(1) + bottom(1)
	if visibleRows < 1 {
		visibleRows = 1
	}

	start, end := m.viewportWindow(visibleRows)
	var rows []string
	for i := start; i < end; i++ {
		rows = append(rows, m.renderDocRow(i, colWidths))
	}

	bottom := m.renderBottom()

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		colHeader,
		strings.Join(rows, "\n"),
		bottom,
	)
}

// renderBottom renders the bottom bar: delete confirm, an input bar, or the
// standard pager + key hints.
func (m Model) renderBottom() string {
	innerW := m.width - 4

	// ── delete confirmation ────────────────────────────────────────────────
	if m.deleteConfirm {
		doc := m.ActiveDoc()
		idStr := ""
		if doc != nil {
			idStr = util.FormatValue(doc["_id"])
			if len(idStr) > 20 {
				idStr = idStr[:19] + "…"
			}
		}
		bar := m.th.ErrText.Render("  Delete ") +
			m.th.DimText.Render(idStr) +
			m.th.ErrText.Render("?  ") +
			m.th.HelpKey.Render("y") + m.th.HelpDesc.Render(" yes  ") +
			m.th.HelpKey.Render("any") + m.th.HelpDesc.Render(" cancel")
		return lipgloss.NewStyle().Width(innerW).Render(bar)
	}

	// ── filter bar ────────────────────────────────────────────────────────
	if m.mode == modeFilter {
		prompt := m.th.StatusFilter.Render("  filter › ")
		bar := prompt + m.input.View()
		if m.inputErr != "" {
			bar += "  " + m.th.ErrText.Render(m.inputErr)
		}
		var hint string
		if m.inputErr != "" {
			hint = m.th.DimText.Render("  fix query or esc to cancel")
		} else {
			hint = m.th.DimText.Render("  enter apply  esc cancel  ctrl+u clear  empty = no filter")
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Width(innerW).Render(bar),
			hint,
		)
	}

	// ── sort bar ──────────────────────────────────────────────────────────
	if m.mode == modeSort {
		prompt := m.th.StatusPager.Render("  sort › ")
		bar := prompt + m.input.View()
		if m.inputErr != "" {
			bar += "  " + m.th.ErrText.Render(m.inputErr)
		}
		var hint string
		if m.inputErr != "" {
			hint = m.th.DimText.Render("  fix sort or esc to cancel")
		} else {
			hint = m.th.DimText.Render("  field / -field / {\"f\":1}  enter apply  esc cancel")
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Width(innerW).Render(bar),
			hint,
		)
	}

	// ── normal: pager + compact hints ─────────────────────────────────────
	pager := m.renderPager()
	var hints []string
	if m.aggMode {
		hints = append(hints, m.th.HelpKey.Render("a")+" "+m.th.HelpDesc.Render("re-run"))
		hints = append(hints, m.th.HelpKey.Render("esc")+" "+m.th.HelpDesc.Render("exit agg"))
	} else {
		hints = append(hints, m.th.HelpKey.Render("n")+" "+m.th.HelpDesc.Render("new"))
		hints = append(hints, m.th.HelpKey.Render("e")+" "+m.th.HelpDesc.Render("edit"))
		hints = append(hints, m.th.HelpKey.Render("d")+" "+m.th.HelpDesc.Render("delete"))
		hints = append(hints, m.th.HelpKey.Render("/")+" "+m.th.HelpDesc.Render("filter"))
		hints = append(hints, m.th.HelpKey.Render("s")+" "+m.th.HelpDesc.Render("sort"))
		hints = append(hints, m.th.HelpKey.Render("a")+" "+m.th.HelpDesc.Render("aggregate"))
		hints = append(hints, m.th.HelpKey.Render("I")+" "+m.th.HelpDesc.Render("indexes"))
		if m.filterExpr != "" || m.sortExpr != "" {
			hints = append(hints, m.th.HelpKey.Render("r")+" "+m.th.HelpDesc.Render("reset"))
		}
	}
	return pager + "  " + m.th.DimText.Render(strings.Join(hints, "  "))
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

// distributeWidths gives _id a fixed 24-char slot and splits remaining
// space evenly among other columns.
func distributeWidths(cols []string, totalW int) []int {
	if len(cols) == 0 {
		return nil
	}
	widths := make([]int, len(cols))

	const idW = 24
	widths[0] = idW
	remaining := totalW - idW - len(cols) // account for 1-space separators

	if len(cols) > 1 && remaining > 0 {
		each := remaining / (len(cols) - 1)
		for i := 1; i < len(cols); i++ {
			widths[i] = each
		}
	}
	return widths
}
