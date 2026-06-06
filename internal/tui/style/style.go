// Package style defines the single Theme object used across all panels.
package style

import "github.com/charmbracelet/lipgloss"

// Theme centralises every lipgloss style used in the application.
// Panels receive a *Theme at construction time and must not create ad-hoc styles.
type Theme struct {
	// borders
	ActiveBorder   lipgloss.Style
	InactiveBorder lipgloss.Style

	// sidebar items
	DatabaseItem   lipgloss.Style
	CollectionItem lipgloss.Style
	SelectedItem   lipgloss.Style
	CursorItem     lipgloss.Style

	// document table
	TableHeader   lipgloss.Style
	TableRow      lipgloss.Style
	TableRowAlt   lipgloss.Style
	TableSelected lipgloss.Style
	TableDivider  lipgloss.Style

	// status bar
	StatusBar    lipgloss.Style
	StatusConn   lipgloss.Style
	StatusPath   lipgloss.Style
	StatusFilter lipgloss.Style
	StatusPager  lipgloss.Style

	// misc
	Spinner  lipgloss.Style
	ErrText  lipgloss.Style
	DimText  lipgloss.Style
	HelpKey  lipgloss.Style
	HelpDesc lipgloss.Style

	// json syntax highlight
	JSONKey     lipgloss.Style
	JSONString  lipgloss.Style
	JSONNumber  lipgloss.Style
	JSONBool    lipgloss.Style
	JSONNull    lipgloss.Style
	JSONBracket lipgloss.Style
	JSONOID     lipgloss.Style
}

// Default returns a high-contrast dark theme.
//
// Palette:
//
//	bg:      #0e0e0e  (near-black)
//	accent:  #00D4FF  (electric cyan)
//	green:   #00FF9C  (neon green)
//	yellow:  #FFD700  (gold)
//	red:     #FF4C4C  (bright red)
//	purple:  #C678DD  (soft purple)
//	text:    #F0F0F0  (near-white)
//	dim:     #666666
func Default() *Theme {
	bg := lipgloss.Color("#0e0e0e")
	bgAlt := lipgloss.Color("#161616")
	bgSel := lipgloss.Color("#1a2a3a")
	bgStatus := lipgloss.Color("#111111")
	borderActive := lipgloss.Color("#00D4FF")
	borderIdle := lipgloss.Color("#333333")
	accent := lipgloss.Color("#00D4FF")
	green := lipgloss.Color("#00FF9C")
	yellow := lipgloss.Color("#FFD700")
	red := lipgloss.Color("#FF4C4C")
	purple := lipgloss.Color("#C678DD")
	text := lipgloss.Color("#F0F0F0")
	dim := lipgloss.Color("#666666")
	divider := lipgloss.Color("#444444")

	_ = bg // used in TableRowAlt

	base := lipgloss.NewStyle()

	return &Theme{
		// ── borders ─────────────────────────────────────────────────────────
		ActiveBorder: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderActive),
		InactiveBorder: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderIdle),

		// ── sidebar ──────────────────────────────────────────────────────────
		DatabaseItem: base.
			Bold(true).
			Foreground(text),
		CollectionItem: base.
			Foreground(lipgloss.Color("#AAAAAA")).
			PaddingLeft(2),
		SelectedItem: base.
			Background(bgSel).
			Foreground(text),
		CursorItem: base.
			Foreground(accent).
			Bold(true),

		// ── document table ────────────────────────────────────────────────────
		TableHeader: base.
			Bold(true).
			Foreground(accent).
			Underline(true),
		TableRow: base.
			Foreground(text),
		TableRowAlt: base.
			Foreground(lipgloss.Color("#C8C8C8")).
			Background(bgAlt),
		TableSelected: base.
			Background(bgSel).
			Foreground(green).
			Bold(true),
		TableDivider: base.
			Foreground(divider),

		// ── status bar ────────────────────────────────────────────────────────
		StatusBar:    base.Foreground(text).Background(bgStatus),
		StatusConn:   base.Foreground(green).Background(bgStatus).Bold(true),
		StatusPath:   base.Foreground(accent).Background(bgStatus).Bold(true),
		StatusFilter: base.Foreground(yellow).Background(bgStatus),
		StatusPager:  base.Foreground(lipgloss.Color("#AAAAAA")).Background(bgStatus),

		// ── misc ──────────────────────────────────────────────────────────────
		Spinner:  base.Foreground(accent),
		ErrText:  base.Foreground(red).Bold(true),
		DimText:  base.Foreground(dim),
		HelpKey:  base.Foreground(yellow).Bold(true),
		HelpDesc: base.Foreground(dim),

		// ── json syntax ───────────────────────────────────────────────────────
		JSONKey:     base.Foreground(accent).Bold(true),
		JSONString:  base.Foreground(green),
		JSONNumber:  base.Foreground(yellow),
		JSONBool:    base.Foreground(red),
		JSONNull:    base.Foreground(dim),
		JSONBracket: base.Foreground(text),
		JSONOID:     base.Foreground(purple),
	}
}
