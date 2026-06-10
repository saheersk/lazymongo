package indexes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/saheersk/lazymongo/internal/tui/msg"
	"github.com/saheersk/lazymongo/internal/util"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// View renders the index panel.
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
	title := "INDEXES"
	if m.collection != "" {
		title = m.db + " > " + m.collection + "  INDEXES"
	}
	if m.stats.IndexCount > 0 {
		title += fmt.Sprintf("  (%d docs, %d idx)", m.stats.DocCount, m.stats.IndexCount)
	}

	header := m.th.PanelTitle.
		Width(m.width - 4).
		Render("  " + title)

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

	if len(m.indexes) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			header, "",
			m.th.DimText.Render("  no indexes — ")+
				m.th.HelpKey.Render("n")+
				m.th.DimText.Render(" create one"),
		)
	}

	var rows []string
	for i, idx := range m.indexes {
		rows = append(rows, m.renderRow(i, idx))
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header, "",
		strings.Join(rows, "\n"),
		"",
		m.renderBottom(),
	)
}

func (m Model) renderRow(i int, idx msg.IndexInfo) string {
	var flags []string
	if idx.Unique {
		flags = append(flags, "UNIQUE")
	}
	if idx.Sparse {
		flags = append(flags, "SPARSE")
	}
	if idx.TTLSeconds >= 0 {
		flags = append(flags, fmt.Sprintf("TTL(%ds)", idx.TTLSeconds))
	}

	flagStr := ""
	if len(flags) > 0 {
		flagStr = "  " + strings.Join(flags, " ")
	}

	name := util.PadRight(idx.Name, 26)
	keys := formatKeys(idx.Keys)
	line := fmt.Sprintf("  %s%s%s", name, keys, flagStr)

	if i == m.cursor {
		return m.th.TableSelected.Width(m.width - 4).Render(line)
	}
	if i%2 == 0 {
		return m.th.TableRow.Render(line)
	}
	return m.th.TableRowAlt.Render(line)
}

func (m Model) renderBottom() string {
	innerW := m.width - 4

	if m.deleteConfirm {
		idx := m.ActiveIndex()
		name := ""
		if idx != nil {
			name = idx.Name
		}
		bar := m.th.ErrText.Render("  Drop index ") +
			m.th.DimText.Render(name) +
			m.th.ErrText.Render("?  ") +
			m.th.HelpKey.Render("y") + m.th.HelpDesc.Render(" yes  ") +
			m.th.HelpKey.Render("any") + m.th.HelpDesc.Render(" cancel")
		return lipgloss.NewStyle().Width(innerW).Render(bar)
	}

	hints := []string{
		m.th.HelpKey.Render("n") + " " + m.th.HelpDesc.Render("create"),
		m.th.HelpKey.Render("d") + " " + m.th.HelpDesc.Render("drop"),
		m.th.HelpKey.Render("R") + " " + m.th.HelpDesc.Render("refresh"),
		m.th.HelpKey.Render("esc") + " " + m.th.HelpDesc.Render("close"),
	}
	return m.th.DimText.Render("  " + strings.Join(hints, "  "))
}

// formatKeys renders a bson.D like {field: 1, other: -1}.
func formatKeys(keys bson.D) string {
	if len(keys) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(keys))
	for _, e := range keys {
		parts = append(parts, fmt.Sprintf("%s:%v", e.Key, e.Value))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
