package documents

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/saheersk/lazymongo/internal/tui/style"
	"github.com/saheersk/lazymongo/internal/util"
	"go.mongodb.org/mongo-driver/v2/bson"
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
	if ic := style.Icons.Docs; ic != "" {
		title = ic + " " + title
	}
	// Show aggregate mode badge with truncated pipeline preview.
	if m.aggMode {
		badge := "AGG"
		if m.aggPipeline != "" {
			p := strings.Join(strings.Fields(m.aggPipeline), " ")
			runes := []rune(p)
			if len(runes) > 22 {
				p = string(runes[:21]) + "…"
			}
			badge = "AGG: " + p
		}
		title += "  " + m.th.ErrText.Render("["+badge+"]")
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

	header := m.th.PanelTitle.
		Width(m.width - 4).
		Render("  " + title)

	innerH := m.height - 2
	if innerH < 3 {
		innerH = 3
	}

	pinBottom := func(topRows ...string) string {
		bottom := m.renderBottom()
		// renderBottom always returns 2 visual lines joined by \n.
		filler := innerH - len(topRows) - 2
		parts := append(topRows, make([]string, max(0, filler))...)
		parts = append(parts, bottom)
		return strings.Join(parts, "\n")
	}

	if m.db == "" {
		return pinBottom(header, "", m.th.DimText.Render("  ← select a collection"))
	}

	if m.loading {
		rows := []string{header, "", "  " + m.spinner.View() + " loading…", ""}
		rows = append(rows, m.renderSkeleton()...)
		return pinBottom(rows...)
	}

	if m.err != nil {
		return pinBottom(header, "", m.th.ErrText.Render("  "+m.err.Error()))
	}

	if len(m.docs) == 0 {
		return pinBottom(m.renderEmptyState(header, innerH)...)
	}

	innerW := m.width - 4
	colWidths := distributeWidths(m.columns, innerW)

	colHeader := m.renderHeaderRow(colWidths)
	divider := m.th.TableDivider.Render(strings.Repeat("─", m.width-4))

	// visibleRows: border(2) + title(1) + blank(1) + colheader(1) + divider(1) + bottom(2)
	visibleRows := m.height - 8
	if visibleRows < 1 {
		visibleRows = 1
	}

	start, end := m.viewportWindow(visibleRows)
	var rows []string
	for i := start; i < end; i++ {
		rows = append(rows, m.renderDocRow(i, colWidths))
	}

	// Pad to visibleRows so the bottom bar is always pinned at the absolute bottom.
	for len(rows) < visibleRows {
		rows = append(rows, "")
	}

	// Overlay a floating dropdown on the last N rows: filter completions or
	// the recent-pipeline picker.
	var dd []string
	switch {
	case m.mode == modeFilter && len(m.filterCompletions) > 0:
		dd = m.renderCompletionDropdown(innerW)
	case m.aggPick:
		dd = m.renderAggPickDropdown(innerW)
	}
	if len(dd) > 0 && len(dd) <= len(rows) {
		off := len(rows) - len(dd)
		for i, dl := range dd {
			rows[off+i] = dl
		}
	}

	bottom := m.renderBottom()

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		colHeader,
		divider,
		strings.Join(rows, "\n"),
		bottom,
	)
}

// renderBottom renders the bottom bar. It always returns exactly 2 visual lines
// (joined by \n) so the layout stays stable regardless of mode.
func (m Model) renderBottom() string {
	innerW := m.width - 4
	lineStyle := lipgloss.NewStyle().Width(innerW)

	// ── bulk delete confirmation ───────────────────────────────────────────
	if m.bulkDeleteConfirm {
		n := len(m.selectedIDs)
		bar := m.th.ErrText.Render(fmt.Sprintf("  Delete %d selected documents?  ", n)) +
			m.th.HelpKey.Render("y") + m.th.HelpDesc.Render(" yes  ") +
			m.th.HelpKey.Render("any") + m.th.HelpDesc.Render(" cancel")
		return lineStyle.Render(bar) + "\n" + lineStyle.Render("")
	}

	// ── delete confirmation ────────────────────────────────────────────────
	if m.deleteConfirm {
		doc := m.ActiveDoc()
		label := ""
		if doc != nil {
			for _, field := range []string{"name", "email", "title", "username", "slug", "label"} {
				if v, ok := doc[field]; ok {
					label = util.FormatValue(v)
					break
				}
			}
			if label == "" {
				label = util.FormatValue(doc["_id"])
			}
			runes := []rune(label)
			if len(runes) > 28 {
				label = string(runes[:27]) + "…"
			}
		}
		bar := m.th.ErrText.Render("  Delete ") +
			m.th.DimText.Render(label) +
			m.th.ErrText.Render("?  ") +
			m.th.HelpKey.Render("y") + m.th.HelpDesc.Render(" yes  ") +
			m.th.HelpKey.Render("any") + m.th.HelpDesc.Render(" cancel")
		return lineStyle.Render(bar) + "\n" + lineStyle.Render("")
	}

	// ── recent-pipeline picker ────────────────────────────────────────────
	if m.aggPick {
		bar := m.th.StatusFilter.Render("  aggregate › ") +
			m.th.DimText.Render("pick a pipeline")
		hint := m.th.DimText.Render("  ↑/↓ select  enter edit  esc cancel")
		return lineStyle.Render(bar) + "\n" + hint
	}

	// ── filter bar ────────────────────────────────────────────────────────
	if m.mode == modeFilter {
		prompt := m.th.StatusFilter.Render("  filter › ")
		bar := prompt + m.input.View()
		if m.inputErr != "" {
			bar += "  " + m.th.ErrText.Render(m.inputErr)
		}
		var secondLine string
		if len(m.filterCompletions) > 0 {
			// Completions are shown as a floating dropdown above — hint the count.
			n := len(m.filterCompletions)
			secondLine = m.th.DimText.Render(fmt.Sprintf("  ↹ %d match(es) shown above  esc cancel", n))
		} else if m.inputErr != "" {
			secondLine = m.th.DimText.Render("  fix query or esc to cancel")
		} else {
			histHint := ""
			if len(m.filterHistory) > 0 {
				histHint = "  ↑/↓ history"
			}
			secondLine = m.th.DimText.Render("  enter apply  esc cancel  ctrl+u clear" + histHint + "  tab field/$op")
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			lineStyle.Render(bar),
			secondLine,
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
			lineStyle.Render(bar),
			hint,
		)
	}

	// ── normal: pager line + hints line ───────────────────────────────────
	// Line 1: pager info.
	pagerLine := lineStyle.Render(m.renderPager())

	// Line 2: key hints — filled progressively so they never overflow innerW.
	var allHints []string
	if m.aggMode {
		allHints = []string{
			m.th.HelpKey.Render("a") + " " + m.th.HelpDesc.Render("re-run"),
			m.th.HelpKey.Render("esc") + " " + m.th.HelpDesc.Render("exit agg"),
		}
	} else if len(m.selectedIDs) > 0 {
		sel := fmt.Sprintf("%d selected", len(m.selectedIDs))
		allHints = []string{
			m.th.StatusFilter.Render("▪ " + sel),
			m.th.HelpKey.Render("space") + " " + m.th.HelpDesc.Render("toggle"),
			m.th.HelpKey.Render("D") + " " + m.th.HelpDesc.Render("delete all"),
			m.th.HelpKey.Render("esc") + " " + m.th.HelpDesc.Render("clear"),
		}
	} else {
		// "/ filter" → "/ edit filter" when a filter is already active so the
		// user knows pressing / will let them edit the existing expression.
		filterLabel := "filter"
		if m.filterExpr != "" {
			filterLabel = "edit filter"
		}
		allHints = []string{
			m.th.HelpKey.Render("n") + " " + m.th.HelpDesc.Render("new"),
			m.th.HelpKey.Render("e") + " " + m.th.HelpDesc.Render("edit"),
			m.th.HelpKey.Render("d") + " " + m.th.HelpDesc.Render("del"),
			m.th.HelpKey.Render("/") + " " + m.th.HelpDesc.Render(filterLabel),
			m.th.HelpKey.Render("s") + " " + m.th.HelpDesc.Render("sort"),
			m.th.HelpKey.Render("x") + " " + m.th.HelpDesc.Render("export"),
		}
		if m.filterExpr != "" || m.sortExpr != "" {
			allHints = append(allHints, m.th.HelpKey.Render("r")+" "+m.th.HelpDesc.Render("reset"))
		}
	}

	// Add hints one by one until the next one would overflow innerW.
	var shown []string
	usedW := 2 // leading "  "
	const hintSep = 2
	for _, h := range allHints {
		w := lipgloss.Width(h)
		if len(shown) > 0 {
			w += hintSep
		}
		if usedW+w > innerW {
			break
		}
		usedW += w
		shown = append(shown, h)
	}
	hintsLine := m.th.DimText.Render("  " + strings.Join(shown, "  "))

	return pagerLine + "\n" + hintsLine
}

// renderEmptyState builds a centred empty-collection message with key hints.
func (m Model) renderEmptyState(header string, innerH int) []string {
	innerW := m.width - 4
	center := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Center)

	title := "no documents in this collection"
	var hint string
	if m.filterExpr != "" {
		title = "no documents match this filter"
		hint = m.th.HelpKey.Render("r") + m.th.DimText.Render(" clear filter") +
			m.th.DimText.Render("  ·  ") +
			m.th.HelpKey.Render("/") + m.th.DimText.Render(" edit filter")
	} else {
		hint = m.th.HelpKey.Render("n") + m.th.DimText.Render(" new document") +
			m.th.DimText.Render("  ·  ") +
			m.th.HelpKey.Render("i") + m.th.DimText.Render(" import file")
	}

	// Place the block roughly a third of the way down the panel.
	topPad := (innerH - 5) / 3
	if topPad < 1 {
		topPad = 1
	}

	rows := []string{header}
	for i := 0; i < topPad; i++ {
		rows = append(rows, "")
	}
	rows = append(rows,
		center.Render(m.th.DimText.Render("∅")),
		"",
		center.Render(m.th.DimText.Render(title)),
		center.Render(hint),
	)
	return rows
}

// renderSkeleton returns dim placeholder bars shown while a page loads, sized
// to the current column layout when one is known.
func (m Model) renderSkeleton() []string {
	innerW := m.width - 4
	widths := distributeWidths(m.columns, innerW)
	if len(widths) == 0 || widths[0] == 0 {
		// No columns yet (first load) — use a generic three-column layout.
		widths = []int{10, innerW/2 - 6, innerW/2 - 8}
	}

	n := m.height - 12 // leave room for header, spinner, bottom bar
	if n > 8 {
		n = 8
	}
	if n < 1 {
		return nil
	}

	rows := make([]string, n)
	for r := 0; r < n; r++ {
		var parts []string
		for _, w := range widths {
			if w <= 0 {
				break
			}
			// Vary bar length per row so the skeleton doesn't look like a grid.
			barW := w - 2 - (r % 3)
			if barW < 1 {
				barW = 1
			}
			parts = append(parts, util.PadRight(strings.Repeat("▁", barW), w))
		}
		rows[r] = m.th.DimText.Render("  " + strings.Join(parts, " "))
	}
	return rows
}

func (m Model) renderHeaderRow(widths []int) string {
	var parts []string
	for i, col := range m.columns {
		if i >= len(widths) || widths[i] == 0 {
			break
		}
		parts = append(parts, util.PadRight(col, widths[i]))
	}
	return m.th.ColHeader.Render("  " + strings.Join(parts, " "))
}

func (m Model) renderDocRow(idx int, widths []int) string {
	doc := m.docs[idx]
	isCursor := idx == m.cursor
	isSel := m.isSelected(doc)

	marker := "  "
	if isSel {
		marker = "▪ "
	}

	// Cursor / selected rows keep a uniform full-row style so the highlight
	// background stays unbroken.
	if isCursor || isSel {
		var parts []string
		for i, col := range m.columns {
			if i >= len(widths) || widths[i] == 0 {
				break
			}
			val := util.FormatValue(doc[col])
			parts = append(parts, util.PadRight(val, widths[i]))
		}
		line := marker + strings.Join(parts, " ")
		switch {
		case isCursor && isSel:
			return m.th.StatusFilter.Width(m.width - 4).Render(line)
		case isCursor:
			return m.th.TableSelected.Width(m.width - 4).Render(line)
		default:
			return m.th.StatusFilter.Render(line)
		}
	}

	// Normal rows: colour each cell by its BSON type, preserving the row's
	// stripe background under every cell and separator.
	rowStyle := m.th.TableRow
	if idx%2 == 1 {
		rowStyle = m.th.TableRowAlt
	}
	rowBG := rowStyle.GetBackground()
	bgFill := lipgloss.NewStyle().Background(rowBG)

	var parts []string
	for i, col := range m.columns {
		if i >= len(widths) || widths[i] == 0 {
			break
		}
		val := util.FormatValue(doc[col])
		cell := util.PadRight(val, widths[i])
		st := m.cellStyle(doc[col], rowStyle).Background(rowBG)
		parts = append(parts, st.Render(cell))
	}

	line := bgFill.Render(marker) + strings.Join(parts, bgFill.Render(" "))
	if w := lipgloss.Width(line); w < m.width-4 {
		line += bgFill.Render(strings.Repeat(" ", m.width-4-w))
	}
	return line
}

// cellStyle picks a foreground style for a table cell based on the raw BSON
// value type, falling back to the row's own foreground for strings.
func (m Model) cellStyle(v interface{}, rowStyle lipgloss.Style) lipgloss.Style {
	switch v.(type) {
	case nil:
		return lipgloss.NewStyle().Foreground(m.th.JSONNull.GetForeground())
	case bool:
		return lipgloss.NewStyle().Foreground(m.th.JSONBool.GetForeground())
	case int32, int64, float64:
		return lipgloss.NewStyle().Foreground(m.th.JSONNumber.GetForeground())
	case bson.ObjectID:
		return lipgloss.NewStyle().Foreground(m.th.JSONOID.GetForeground())
	case bson.DateTime:
		return lipgloss.NewStyle().Foreground(m.th.JSONString.GetForeground())
	case bson.A, bson.M, bson.D:
		return lipgloss.NewStyle().Foreground(m.th.DimText.GetForeground())
	default:
		return lipgloss.NewStyle().Foreground(rowStyle.GetForeground())
	}
}

func (m Model) renderPager() string {
	from := m.page*m.pageSize + 1
	to := from + len(m.docs) - 1
	info := fmt.Sprintf("  %d-%d of %d  •  pg %d/%d  •  %d/pg",
		from, to, m.total, m.page+1, m.pageCount(), m.pageSize)
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

// distributeWidths allocates column widths. _id gets a compact 10-char slot so
// other fields remain visible even in narrow layouts. Extra columns beyond what
// fits at the minimum width (10 chars each) get a width of 0 and are skipped.
func distributeWidths(cols []string, totalW int) []int {
	if len(cols) == 0 {
		return nil
	}
	widths := make([]int, len(cols))

	if len(cols) == 1 {
		widths[0] = totalW
		return widths
	}

	const (
		idColW  = 10 // compact _id: shows ~8 hex chars + "…"
		minColW = 10 // minimum usable width per extra column
		sep     = 1  // space separator between columns
	)

	// How many extra (non-_id) columns can fit?
	maxExtra := len(cols) - 1
	nExtra := maxExtra
	for nExtra > 0 {
		// budget = space left for extra columns after reserving _id and separators
		budget := totalW - idColW - nExtra*sep
		if budget >= nExtra*minColW {
			break
		}
		nExtra--
	}

	if nExtra == 0 {
		// Extremely narrow panel — give everything to _id.
		widths[0] = totalW
		return widths
	}

	widths[0] = idColW
	each := (totalW - idColW - nExtra*sep) / nExtra
	for i := 1; i <= nExtra; i++ {
		widths[i] = each
	}
	// widths[nExtra+1:] stay 0 — those columns are hidden.
	return widths
}

// renderCompletionDropdown builds the floating dropdown for filter
// completions, highlighting the item at m.filterCompletionIdx.
func (m Model) renderCompletionDropdown(innerW int) []string {
	return m.renderDropdownBox(m.filterCompletions, m.filterCompletionIdx, innerW)
}

// renderAggPickDropdown builds the floating dropdown for the recent-pipeline
// picker: "new pipeline" followed by history entries (whitespace-condensed).
func (m Model) renderAggPickDropdown(innerW int) []string {
	items := make([]string, 0, len(m.aggHistory)+1)
	items = append(items, "✚ new pipeline")
	for _, p := range m.aggHistory {
		items = append(items, strings.Join(strings.Fields(p), " "))
	}
	return m.renderDropdownBox(items, m.aggPickIdx, innerW)
}

// renderDropdownBox builds a floating bordered dropdown overlaying the last N
// rows of the document table. The item at selIdx is highlighted with the
// cursor background.
func (m Model) renderDropdownBox(items []string, selIdx, innerW int) []string {
	completions := items
	const maxItems = 7
	extra := 0
	if len(completions) > maxItems {
		extra = len(completions) - maxItems
		completions = completions[:maxItems]
	}

	// Measure content width needed.
	contentW := 8
	for _, c := range completions {
		if l := len([]rune(c)); l > contentW {
			contentW = l
		}
	}
	if extra > 0 {
		if l := len(fmt.Sprintf("+%d more", extra)); l > contentW {
			contentW = l
		}
	}

	// boxW = 2 borders + 2 inner padding + content.
	boxW := contentW + 4
	const (
		minBoxW = 14
		maxBoxW = 44 // wide enough for condensed pipeline previews
	)
	if boxW < minBoxW {
		boxW = minBoxW
	}
	if boxW > maxBoxW {
		boxW = maxBoxW
	}
	if boxW > innerW-2 {
		boxW = innerW - 2
	}
	contentW = boxW - 4
	if contentW < 1 {
		contentW = 1
	}

	// Colors from theme.
	bgColor := m.th.PanelTitle.GetBackground()
	bgSel := m.th.TableSelected.GetBackground() // cursor row background
	borderFg := m.th.ActiveBorder.GetBorderTopForeground()
	fgSel := m.th.TableSelected.GetForeground() // cursor row foreground
	fgNormal := m.th.TableRow.GetForeground()
	fgDim := m.th.DimText.GetForeground()

	itemBase := lipgloss.NewStyle().Background(bgColor).Width(contentW)
	selItem := lipgloss.NewStyle().Background(bgSel).Foreground(fgSel).Bold(true).Width(contentW)
	dimItem := lipgloss.NewStyle().Background(bgColor).Foreground(fgDim).Width(contentW)

	// Build one styled line per item.
	var itemLines []string
	for i, c := range completions {
		runes := []rune(c)
		if len(runes) > contentW {
			c = string(runes[:contentW-1]) + "…"
		}
		var rendered string
		if i == selIdx {
			rendered = selItem.Render(c)
		} else {
			rendered = itemBase.Copy().Foreground(fgNormal).Render(c)
		}
		itemLines = append(itemLines, " "+rendered+" ")
	}
	if extra > 0 {
		label := fmt.Sprintf("+%d more", extra)
		itemLines = append(itemLines, " "+dimItem.Render(label)+" ")
	}

	// Render as a bordered box.
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderFg)
	rendered := box.Render(strings.Join(itemLines, "\n"))
	boxLines := strings.Split(rendered, "\n")

	// Pad each line to innerW with a left indent of 2.
	const indent = 2
	result := make([]string, len(boxLines))
	for i, bl := range boxLines {
		line := strings.Repeat(" ", indent) + bl
		if w := lipgloss.Width(line); w < innerW {
			line += strings.Repeat(" ", innerW-w)
		}
		result[i] = line
	}
	return result
}
