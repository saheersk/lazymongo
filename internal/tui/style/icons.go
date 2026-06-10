package style

// IconSet holds the glyphs used across panels. The default set uses Nerd Font
// glyphs; SetASCIIIcons switches to plain characters for terminals without a
// patched font (config: ui.nerdFonts: false).
type IconSet struct {
	Database   string // sidebar database row
	Collection string // sidebar collection row
	Document   string // detail panel title
	Docs       string // documents panel title
}

// Icons is the active icon set, read by the panels at render time.
var Icons = IconSet{
	Database:   "",          // nf-fa-database
	Collection: "",          // nf-fa-table
	Document:   "\U000f0219", // nf-md-file_document
	Docs:       "",          // nf-fa-list_alt
}

// SetASCIIIcons switches the active icon set to plain ASCII-safe characters.
// Empty icons are dropped (no double spaces) by the panels at render time.
func SetASCIIIcons() {
	Icons = IconSet{
		Database:   "",
		Collection: "●",
		Document:   "",
		Docs:       "",
	}
}
