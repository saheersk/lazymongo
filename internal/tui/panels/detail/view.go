package detail

import (
	"fmt"
	"strings"
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

	if m.doc == nil {
		header := m.th.PanelTitle.Width(innerW).Render("  DOCUMENT")
		return header + "\n\n" + m.th.DimText.Render("  press enter on a document")
	}

	// title: "DOCUMENT  •  <id>  <scroll%>"
	idStr := m.docID
	if len(idStr) > 28 {
		idStr = idStr[:25] + "…"
	}
	scrollInfo := fmt.Sprintf("%d%%", m.ScrollPercent())
	title := "DOCUMENT"
	padding := innerW - len(title) - len(idStr) - len(scrollInfo) - 4
	if padding < 1 {
		padding = 1
	}
	header := m.th.PanelTitle.Width(innerW).Render(
		"  " + title +
			strings.Repeat(" ", padding) +
			m.th.DimText.Render(idStr) +
			"  " +
			m.th.StatusPager.Render(scrollInfo),
	)

	// hint bar at the bottom
	hints := m.th.DimText.Render("  j/k scroll  ctrl+d/u half-page  g/G top/bot  y copy-id  Y copy-doc  esc back")

	// viewport takes remaining height
	content := ""
	if m.ready {
		content = m.viewport.View()
	}

	return header + "\n" + content + "\n" + hints
}
