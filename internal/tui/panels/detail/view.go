package detail

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/saheersk/lazymongo/internal/tui/style"
)

// View renders the detail panel.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

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

	docTitle := "DOCUMENT"
	iconW := 0
	if ic := style.Icons.Document; ic != "" {
		docTitle = ic + " " + docTitle
		iconW = 2 // glyph + space — both single-cell
	}

	if m.doc == nil {
		header := m.th.PanelTitle.Width(innerW).Render("  " + docTitle)
		return header + "\n\n" + m.th.DimText.Render("  press enter on a document")
	}

	// title: "DOCUMENT  [f: ...]  •  <id>  <scroll%>"
	idStr := m.docID
	if len([]rune(idStr)) > 20 {
		idStr = string([]rune(idStr)[:19]) + "…"
	}
	scrollInfo := fmt.Sprintf("%d%%", m.ScrollPercent())
	title := "DOCUMENT"

	filterBadge := ""
	filterBadgeRunes := 0
	if m.filterExpr != "" {
		short := m.filterExpr
		if len([]rune(short)) > 16 {
			short = string([]rune(short)[:15]) + "…"
		}
		filterBadge = "[f: " + short + "]"
		filterBadgeRunes = len([]rune(filterBadge)) + 2 // 2 for "  " prefix
	}

	padding := innerW - len(title) - iconW - filterBadgeRunes - len([]rune(idStr)) - len(scrollInfo) - 4
	if padding < 1 {
		padding = 1
	}
	titlePart := "  " + docTitle
	if filterBadge != "" {
		titlePart += "  " + m.th.StatusFilter.Render(filterBadge)
	}
	header := m.th.PanelTitle.Width(innerW).Render(
		titlePart +
			strings.Repeat(" ", padding) +
			m.th.DimText.Render(idStr) +
			"  " +
			m.th.StatusPager.Render(scrollInfo),
	)

	// hint bar at the bottom
	hints := m.th.DimText.Render("  j/k scroll  ctrl+d/u half-page  g/G top/bot  ") +
		m.th.HelpKey.Render("e") + m.th.DimText.Render(" edit  / filter  y copy-id  Y copy-doc  esc back")

	// viewport takes remaining height
	content := ""
	if m.ready {
		content = m.renderViewportWithScrollbar()
	}

	return header + "\n" + content + "\n" + hints
}

// renderViewportWithScrollbar appends a thin scrollbar track to the right of
// every viewport line. When the document fits without scrolling, the gutter
// stays blank.
func (m Model) renderViewportWithScrollbar() string {
	content := m.viewport.View()
	total := m.viewport.TotalLineCount()
	vh := m.viewport.Height
	if total <= vh || vh < 1 {
		return content
	}

	// Thumb size proportional to the visible fraction, position from YOffset.
	thumbH := vh * vh / total
	if thumbH < 1 {
		thumbH = 1
	}
	maxOff := total - vh
	thumbStart := 0
	if maxOff > 0 {
		thumbStart = (vh - thumbH) * m.viewport.YOffset / maxOff
	}

	track := m.th.DimText.Render("│")
	thumb := lipgloss.NewStyle().
		Foreground(m.th.ActiveBorder.GetBorderTopForeground()).
		Render("┃")

	vpW := m.viewport.Width
	lines := strings.Split(content, "\n")
	out := make([]string, vh)
	for i := 0; i < vh; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		if w := lipgloss.Width(line); w < vpW {
			line += strings.Repeat(" ", vpW-w)
		}
		ch := track
		if i >= thumbStart && i < thumbStart+thumbH {
			ch = thumb
		}
		out[i] = line + " " + ch
	}
	return strings.Join(out, "\n")
}
