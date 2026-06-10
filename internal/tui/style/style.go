// Package style defines the single Theme object used across all panels.
package style

import "github.com/charmbracelet/lipgloss"

// Theme centralises every lipgloss style used in the application.
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
	PanelTitle    lipgloss.Style
	ColHeader     lipgloss.Style
	TableHeader   lipgloss.Style // alias of PanelTitle for backwards compat
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
	Spinner     lipgloss.Style
	ErrText     lipgloss.Style
	DimText     lipgloss.Style
	HelpKey     lipgloss.Style
	HelpDesc    lipgloss.Style
	HelpTitle   lipgloss.Style // top title bar of the ? help overlay
	HelpSection lipgloss.Style // section headers inside help overlay

	// json syntax highlight
	JSONKey     lipgloss.Style
	JSONString  lipgloss.Style
	JSONNumber  lipgloss.Style
	JSONBool    lipgloss.Style
	JSONNull    lipgloss.Style
	JSONBracket lipgloss.Style
	JSONOID     lipgloss.Style
}

// ByName dispatches by theme name and returns the matching Theme.
// Unknown names fall back to Catppuccin.
func ByName(name string) *Theme {
	switch name {
	case "high-contrast":
		return HighContrast()
	case "tokyo-night":
		return TokyoNight()
	case "nord":
		return Nord()
	case "dracula":
		return Dracula()
	case "catppuccin-latte":
		return Latte()
	default:
		return Catppuccin()
	}
}

// Default returns the Catppuccin Mocha theme. Kept as an alias for
// backwards-compatibility with existing call-sites.
func Default() *Theme { return Catppuccin() }

// Catppuccin returns a Catppuccin Mocha theme — the palette used by most
// lazygit / lazydocker setups.
//
//	Reference: https://github.com/catppuccin/catppuccin
//
//	Crust:     #11111b  (deepest bg — status bar)
//	Mantle:    #181825  (panel title bar bg)
//	Base:      #1e1e2e  (main bg)
//	Surface0:  #313244  (selection / alt-row bg)
//	Overlay1:  #7f849c  (dim text)
//	Subtext0:  #a6adc8  (secondary text)
//	Text:      #cdd6f4  (primary text)
//	Lavender:  #b4befe  (active border, panel titles)
//	Blue:      #89b4fa  (col headers, JSON keys)
//	Sapphire:  #74c7ec  (cursor fg, selected row fg)
//	Green:     #a6e3a1  (connection indicator, strings)
//	Yellow:    #f9e2af  (key bindings, warnings)
//	Peach:     #fab387  (numbers)
//	Maroon:    #eba0ac  (errors)
//	Mauve:     #cba6f7  (OIDs)
func Catppuccin() *Theme {
	// backgrounds
	bgCrust := lipgloss.Color("#11111b")  // status bar
	bgMantle := lipgloss.Color("#181825") // panel title bar
	bgSurf0 := lipgloss.Color("#313244")  // selected / cursor row
	bgSurf1 := lipgloss.Color("#24273a")  // alt table row (very subtle)

	// borders
	borderActive := lipgloss.Color("#b4befe")   // Lavender — bright, clear
	borderInactive := lipgloss.Color("#45475a") // Surface1 — visible but quiet

	// text
	text := lipgloss.Color("#cdd6f4")    // Text
	textAlt := lipgloss.Color("#a6adc8") // Subtext0
	dim := lipgloss.Color("#6c7086")     // Overlay1

	// accent colours
	lavender := lipgloss.Color("#b4befe")
	blue := lipgloss.Color("#89b4fa")
	sapphire := lipgloss.Color("#74c7ec")
	green := lipgloss.Color("#a6e3a1")
	yellow := lipgloss.Color("#f9e2af")
	peach := lipgloss.Color("#fab387")
	maroon := lipgloss.Color("#eba0ac")
	mauve := lipgloss.Color("#cba6f7")

	base := lipgloss.NewStyle()

	return &Theme{
		// ── borders ──────────────────────────────────────────────────────────
		ActiveBorder: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderActive),
		InactiveBorder: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderInactive),

		// ── sidebar items ─────────────────────────────────────────────────────
		DatabaseItem: base.
			Bold(true).
			Foreground(text),
		CollectionItem: base.
			Foreground(textAlt),
		SelectedItem: base.
			Background(bgSurf0).
			Foreground(text),
		// Full-width row highlight — background fills the row.
		CursorItem: base.
			Background(bgSurf0).
			Foreground(sapphire).
			Bold(true),

		// ── table ─────────────────────────────────────────────────────────────
		// PanelTitle: title bar shown at the top of every panel.
		// Uses a distinct background so the heading is always readable.
		PanelTitle: base.
			Bold(true).
			Foreground(lavender).
			Background(bgMantle),
		// TableHeader kept as alias so older call-sites compile.
		TableHeader: base.
			Bold(true).
			Foreground(lavender).
			Background(bgMantle),
		// ColHeader: column-name row inside the table.
		ColHeader: base.
			Bold(true).
			Foreground(blue),

		TableRow: base.
			Foreground(text),
		TableRowAlt: base.
			Foreground(textAlt).
			Background(bgSurf1),
		TableSelected: base.
			Background(bgSurf0).
			Foreground(sapphire).
			Bold(true),
		TableDivider: base.
			Foreground(borderInactive),

		// ── status bar ────────────────────────────────────────────────────────
		StatusBar:    base.Foreground(textAlt).Background(bgCrust),
		StatusConn:   base.Foreground(green).Background(bgCrust).Bold(true),
		StatusPath:   base.Foreground(sapphire).Background(bgCrust).Bold(true),
		StatusFilter: base.Foreground(yellow).Background(bgCrust),
		StatusPager:  base.Foreground(dim).Background(bgCrust),

		// ── misc ──────────────────────────────────────────────────────────────
		Spinner:     base.Foreground(blue),
		ErrText:     base.Foreground(maroon).Bold(true),
		DimText:     base.Foreground(dim),
		HelpKey:     base.Foreground(yellow).Bold(true),
		HelpDesc:    base.Foreground(dim),
		HelpTitle:   base.Bold(true).Foreground(lavender).Background(bgMantle),
		HelpSection: base.Bold(true).Foreground(yellow),

		// ── json syntax ───────────────────────────────────────────────────────
		JSONKey:     base.Foreground(blue).Bold(true),
		JSONString:  base.Foreground(green),
		JSONNumber:  base.Foreground(peach),
		JSONBool:    base.Foreground(maroon),
		JSONNull:    base.Foreground(dim),
		JSONBracket: base.Foreground(text),
		JSONOID:     base.Foreground(mauve),
	}
}
