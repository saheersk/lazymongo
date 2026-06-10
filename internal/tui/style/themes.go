package style

import "github.com/charmbracelet/lipgloss"

// HighContrast returns a high-contrast accessibility theme inspired by
// Claude's colour palette (green-on-black).
func HighContrast() *Theme {
	bgSel := lipgloss.Color("#14532d")
	bgMantle := lipgloss.Color("#052e16")
	bgStatus := lipgloss.Color("#000000")

	borderActive := lipgloss.Color("#22c55e")
	borderInactive := lipgloss.Color("#374151")

	text := lipgloss.Color("#f9fafb")
	textAlt := lipgloss.Color("#d1fae5")
	dim := lipgloss.Color("#6b7280")

	cursorFg := lipgloss.Color("#86efac")
	colHdr := lipgloss.Color("#22c55e")
	connFg := lipgloss.Color("#4ade80")
	pathFg := lipgloss.Color("#86efac")
	filterFg := lipgloss.Color("#fbbf24")
	helpKeyFg := lipgloss.Color("#fbbf24")
	errFg := lipgloss.Color("#f87171")
	boolFg := lipgloss.Color("#f87171")
	numFg := lipgloss.Color("#fcd34d")
	oidFg := lipgloss.Color("#c084fc")

	base := lipgloss.NewStyle()

	return &Theme{
		ActiveBorder: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderActive),
		InactiveBorder: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderInactive),

		DatabaseItem:   base.Bold(true).Foreground(text),
		CollectionItem: base.Foreground(textAlt),
		SelectedItem:   base.Background(bgSel).Foreground(text),
		CursorItem:     base.Background(bgSel).Foreground(cursorFg).Bold(true),

		PanelTitle:  base.Bold(true).Foreground(cursorFg).Background(bgMantle),
		TableHeader: base.Bold(true).Foreground(cursorFg).Background(bgMantle),
		ColHeader:   base.Bold(true).Foreground(colHdr),

		TableRow:      base.Foreground(text),
		TableRowAlt:   base.Foreground(textAlt),
		TableSelected: base.Background(bgSel).Foreground(cursorFg).Bold(true),
		TableDivider:  base.Foreground(borderInactive),

		StatusBar:    base.Foreground(textAlt).Background(bgStatus),
		StatusConn:   base.Foreground(connFg).Background(bgStatus).Bold(true),
		StatusPath:   base.Foreground(pathFg).Background(bgStatus).Bold(true),
		StatusFilter: base.Foreground(filterFg).Background(bgStatus),
		StatusPager:  base.Foreground(dim).Background(bgStatus),

		Spinner:     base.Foreground(borderActive),
		ErrText:     base.Foreground(errFg).Bold(true),
		DimText:     base.Foreground(dim),
		HelpKey:     base.Foreground(helpKeyFg).Bold(true),
		HelpDesc:    base.Foreground(dim),
		HelpTitle:   base.Bold(true).Foreground(cursorFg).Background(bgMantle),
		HelpSection: base.Bold(true).Foreground(borderActive),

		JSONKey:     base.Foreground(borderActive).Bold(true),
		JSONString:  base.Foreground(cursorFg),
		JSONNumber:  base.Foreground(numFg),
		JSONBool:    base.Foreground(boolFg),
		JSONNull:    base.Foreground(dim),
		JSONBracket: base.Foreground(text),
		JSONOID:     base.Foreground(oidFg),
	}
}

// TokyoNight returns the popular Tokyo Night dark theme.
func TokyoNight() *Theme {
	bgSel := lipgloss.Color("#283457")
	bgMantle := lipgloss.Color("#1a1b26")
	bgStatus := lipgloss.Color("#16161e")

	borderActive := lipgloss.Color("#7aa2f7")
	borderInactive := lipgloss.Color("#3b4261")

	text := lipgloss.Color("#c0caf5")
	textAlt := lipgloss.Color("#a9b1d6")
	dim := lipgloss.Color("#565f89")

	cursorFg := lipgloss.Color("#7dcfff")
	colHdr := lipgloss.Color("#7aa2f7")
	connFg := lipgloss.Color("#9ece6a")
	filterFg := lipgloss.Color("#e0af68")
	helpKeyFg := lipgloss.Color("#e0af68")
	errFg := lipgloss.Color("#f7768e")
	boolFg := lipgloss.Color("#f7768e")
	numFg := lipgloss.Color("#ff9e64")
	strFg := lipgloss.Color("#9ece6a")
	oidFg := lipgloss.Color("#bb9af7")

	base := lipgloss.NewStyle()

	return &Theme{
		ActiveBorder: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderActive),
		InactiveBorder: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderInactive),

		DatabaseItem:   base.Bold(true).Foreground(text),
		CollectionItem: base.Foreground(textAlt),
		SelectedItem:   base.Background(bgSel).Foreground(text),
		CursorItem:     base.Background(bgSel).Foreground(cursorFg).Bold(true),

		PanelTitle:  base.Bold(true).Foreground(text).Background(bgMantle),
		TableHeader: base.Bold(true).Foreground(text).Background(bgMantle),
		ColHeader:   base.Bold(true).Foreground(colHdr),

		TableRow:      base.Foreground(text),
		TableRowAlt:   base.Foreground(textAlt),
		TableSelected: base.Background(bgSel).Foreground(cursorFg).Bold(true),
		TableDivider:  base.Foreground(borderInactive),

		StatusBar:    base.Foreground(textAlt).Background(bgStatus),
		StatusConn:   base.Foreground(connFg).Background(bgStatus).Bold(true),
		StatusPath:   base.Foreground(cursorFg).Background(bgStatus).Bold(true),
		StatusFilter: base.Foreground(filterFg).Background(bgStatus),
		StatusPager:  base.Foreground(dim).Background(bgStatus),

		Spinner:     base.Foreground(borderActive),
		ErrText:     base.Foreground(errFg).Bold(true),
		DimText:     base.Foreground(dim),
		HelpKey:     base.Foreground(helpKeyFg).Bold(true),
		HelpDesc:    base.Foreground(dim),
		HelpTitle:   base.Bold(true).Foreground(text).Background(bgMantle),
		HelpSection: base.Bold(true).Foreground(filterFg),

		JSONKey:     base.Foreground(borderActive).Bold(true),
		JSONString:  base.Foreground(strFg),
		JSONNumber:  base.Foreground(numFg),
		JSONBool:    base.Foreground(boolFg),
		JSONNull:    base.Foreground(dim),
		JSONBracket: base.Foreground(text),
		JSONOID:     base.Foreground(oidFg),
	}
}

// Nord returns the Nord arctic theme.
func Nord() *Theme {
	bgSel := lipgloss.Color("#3b4252")
	bgMantle := lipgloss.Color("#2e3440")
	bgStatus := lipgloss.Color("#191c24")

	borderActive := lipgloss.Color("#88c0d0")
	borderInactive := lipgloss.Color("#434c5e")

	text := lipgloss.Color("#eceff4")
	textAlt := lipgloss.Color("#d8dee9")
	dim := lipgloss.Color("#4c566a")

	cursorFg := lipgloss.Color("#88c0d0")
	colHdr := lipgloss.Color("#81a1c1")
	connFg := lipgloss.Color("#a3be8c")
	filterFg := lipgloss.Color("#ebcb8b")
	helpKeyFg := lipgloss.Color("#ebcb8b")
	errFg := lipgloss.Color("#bf616a")
	boolFg := lipgloss.Color("#bf616a")
	numFg := lipgloss.Color("#d08770")
	strFg := lipgloss.Color("#a3be8c")
	oidFg := lipgloss.Color("#b48ead")

	base := lipgloss.NewStyle()

	return &Theme{
		ActiveBorder: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderActive),
		InactiveBorder: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderInactive),

		DatabaseItem:   base.Bold(true).Foreground(text),
		CollectionItem: base.Foreground(textAlt),
		SelectedItem:   base.Background(bgSel).Foreground(text),
		CursorItem:     base.Background(bgSel).Foreground(cursorFg).Bold(true),

		PanelTitle:  base.Bold(true).Foreground(text).Background(bgMantle),
		TableHeader: base.Bold(true).Foreground(text).Background(bgMantle),
		ColHeader:   base.Bold(true).Foreground(colHdr),

		TableRow:      base.Foreground(text),
		TableRowAlt:   base.Foreground(textAlt),
		TableSelected: base.Background(bgSel).Foreground(cursorFg).Bold(true),
		TableDivider:  base.Foreground(borderInactive),

		StatusBar:    base.Foreground(textAlt).Background(bgStatus),
		StatusConn:   base.Foreground(connFg).Background(bgStatus).Bold(true),
		StatusPath:   base.Foreground(cursorFg).Background(bgStatus).Bold(true),
		StatusFilter: base.Foreground(filterFg).Background(bgStatus),
		StatusPager:  base.Foreground(dim).Background(bgStatus),

		Spinner:     base.Foreground(borderActive),
		ErrText:     base.Foreground(errFg).Bold(true),
		DimText:     base.Foreground(dim),
		HelpKey:     base.Foreground(helpKeyFg).Bold(true),
		HelpDesc:    base.Foreground(dim),
		HelpTitle:   base.Bold(true).Foreground(text).Background(bgMantle),
		HelpSection: base.Bold(true).Foreground(colHdr),

		JSONKey:     base.Foreground(colHdr).Bold(true),
		JSONString:  base.Foreground(strFg),
		JSONNumber:  base.Foreground(numFg),
		JSONBool:    base.Foreground(boolFg),
		JSONNull:    base.Foreground(dim),
		JSONBracket: base.Foreground(text),
		JSONOID:     base.Foreground(oidFg),
	}
}

// Dracula returns the Dracula theme.
func Dracula() *Theme {
	bgSel := lipgloss.Color("#44475a")
	bgMantle := lipgloss.Color("#282a36")
	bgStatus := lipgloss.Color("#191a21")

	borderActive := lipgloss.Color("#bd93f9")
	borderInactive := lipgloss.Color("#44475a")

	text := lipgloss.Color("#f8f8f2")
	textAlt := lipgloss.Color("#d4d4d4")
	dim := lipgloss.Color("#6272a4")

	cursorFg := lipgloss.Color("#ff79c6")
	colHdr := lipgloss.Color("#8be9fd")
	connFg := lipgloss.Color("#50fa7b")
	filterFg := lipgloss.Color("#f1fa8c")
	helpKeyFg := lipgloss.Color("#f1fa8c")
	errFg := lipgloss.Color("#ff5555")
	boolFg := lipgloss.Color("#ff5555")
	numFg := lipgloss.Color("#ffb86c")
	strFg := lipgloss.Color("#50fa7b")
	oidFg := lipgloss.Color("#ff79c6")

	base := lipgloss.NewStyle()

	return &Theme{
		ActiveBorder: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderActive),
		InactiveBorder: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderInactive),

		DatabaseItem:   base.Bold(true).Foreground(text),
		CollectionItem: base.Foreground(textAlt),
		SelectedItem:   base.Background(bgSel).Foreground(text),
		CursorItem:     base.Background(bgSel).Foreground(cursorFg).Bold(true),

		PanelTitle:  base.Bold(true).Foreground(text).Background(bgMantle),
		TableHeader: base.Bold(true).Foreground(text).Background(bgMantle),
		ColHeader:   base.Bold(true).Foreground(colHdr),

		TableRow:      base.Foreground(text),
		TableRowAlt:   base.Foreground(textAlt),
		TableSelected: base.Background(bgSel).Foreground(cursorFg).Bold(true),
		TableDivider:  base.Foreground(borderInactive),

		StatusBar:    base.Foreground(textAlt).Background(bgStatus),
		StatusConn:   base.Foreground(connFg).Background(bgStatus).Bold(true),
		StatusPath:   base.Foreground(colHdr).Background(bgStatus).Bold(true),
		StatusFilter: base.Foreground(filterFg).Background(bgStatus),
		StatusPager:  base.Foreground(dim).Background(bgStatus),

		Spinner:     base.Foreground(borderActive),
		ErrText:     base.Foreground(errFg).Bold(true),
		DimText:     base.Foreground(dim),
		HelpKey:     base.Foreground(helpKeyFg).Bold(true),
		HelpDesc:    base.Foreground(dim),
		HelpTitle:   base.Bold(true).Foreground(text).Background(bgMantle),
		HelpSection: base.Bold(true).Foreground(borderActive),

		JSONKey:     base.Foreground(colHdr).Bold(true),
		JSONString:  base.Foreground(strFg),
		JSONNumber:  base.Foreground(numFg),
		JSONBool:    base.Foreground(boolFg),
		JSONNull:    base.Foreground(dim),
		JSONBracket: base.Foreground(text),
		JSONOID:     base.Foreground(oidFg),
	}
}

// Latte returns Catppuccin Latte — a light theme for bright terminals.
//
//	Reference: https://github.com/catppuccin/catppuccin (Latte palette)
func Latte() *Theme {
	bgSel := lipgloss.Color("#ccd0da")    // Surface0
	bgMantle := lipgloss.Color("#e6e9ef") // Mantle
	bgStatus := lipgloss.Color("#dce0e8") // Crust
	bgAlt := lipgloss.Color("#e6e9ef")    // subtle alt-row stripe

	borderActive := lipgloss.Color("#7287fd")   // Lavender
	borderInactive := lipgloss.Color("#9ca0b0") // Overlay0

	text := lipgloss.Color("#4c4f69")    // Text
	textAlt := lipgloss.Color("#5c5f77") // Subtext1
	dim := lipgloss.Color("#8c8fa1")     // Overlay1

	lavender := lipgloss.Color("#7287fd")
	blue := lipgloss.Color("#1e66f5")
	sapphire := lipgloss.Color("#209fb5")
	green := lipgloss.Color("#40a02b")
	yellow := lipgloss.Color("#df8e1d")
	peach := lipgloss.Color("#fe640b")
	maroon := lipgloss.Color("#e64553")
	mauve := lipgloss.Color("#8839ef")

	base := lipgloss.NewStyle()

	return &Theme{
		ActiveBorder: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderActive),
		InactiveBorder: base.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderInactive),

		DatabaseItem:   base.Bold(true).Foreground(text),
		CollectionItem: base.Foreground(textAlt),
		SelectedItem:   base.Background(bgSel).Foreground(text),
		CursorItem:     base.Background(bgSel).Foreground(sapphire).Bold(true),

		PanelTitle:  base.Bold(true).Foreground(lavender).Background(bgMantle),
		TableHeader: base.Bold(true).Foreground(lavender).Background(bgMantle),
		ColHeader:   base.Bold(true).Foreground(blue),

		TableRow:      base.Foreground(text),
		TableRowAlt:   base.Foreground(textAlt).Background(bgAlt),
		TableSelected: base.Background(bgSel).Foreground(sapphire).Bold(true),
		TableDivider:  base.Foreground(borderInactive),

		StatusBar:    base.Foreground(textAlt).Background(bgStatus),
		StatusConn:   base.Foreground(green).Background(bgStatus).Bold(true),
		StatusPath:   base.Foreground(sapphire).Background(bgStatus).Bold(true),
		StatusFilter: base.Foreground(yellow).Background(bgStatus),
		StatusPager:  base.Foreground(dim).Background(bgStatus),

		Spinner:     base.Foreground(blue),
		ErrText:     base.Foreground(maroon).Bold(true),
		DimText:     base.Foreground(dim),
		HelpKey:     base.Foreground(yellow).Bold(true),
		HelpDesc:    base.Foreground(dim),
		HelpTitle:   base.Bold(true).Foreground(lavender).Background(bgMantle),
		HelpSection: base.Bold(true).Foreground(yellow),

		JSONKey:     base.Foreground(blue).Bold(true),
		JSONString:  base.Foreground(green),
		JSONNumber:  base.Foreground(peach),
		JSONBool:    base.Foreground(maroon),
		JSONNull:    base.Foreground(dim),
		JSONBracket: base.Foreground(text),
		JSONOID:     base.Foreground(mauve),
	}
}
