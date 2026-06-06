// Package cmd wires the cobra CLI to the TUI application.
package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/config"
	mongoClient "github.com/saheersk/lazymongo/internal/mongo"
	"github.com/saheersk/lazymongo/internal/tui"
	"github.com/spf13/cobra"
)

var (
	flagURI     string
	flagHost    string
	flagPort    int
	flagProfile string
	flagSave    string
	flagTheme   string

	buildVersion = "dev"
	buildCommit  = "none"
	buildDate    = "unknown"
)

// SetVersion is called from main with values injected by GoReleaser.
func SetVersion(version, commit, date string) {
	buildVersion = version
	buildCommit = commit
	buildDate = date
}

var rootCmd = &cobra.Command{
	Use:   "lazymongo [profile]",
	Short: "A terminal UI for MongoDB — like lazygit, but for Mongo",
	Long: `lazymongo is a fast, keyboard-driven TUI for browsing and editing MongoDB.

Connect with a URI:
  lazymongo --uri mongodb://localhost:27017

Save a named profile:
  lazymongo --uri mongodb://localhost:27017 --save local

Load a saved profile:
  lazymongo --profile local
  lazymongo local`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTUI,
}

// Execute is the entry point called from main.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVar(&flagURI, "uri", "", "MongoDB connection URI (overrides config)")
	rootCmd.Flags().StringVar(&flagHost, "host", "", "MongoDB host (default: localhost)")
	rootCmd.Flags().IntVar(&flagPort, "port", 0, "MongoDB port (default: 27017)")
	rootCmd.Flags().StringVar(&flagProfile, "profile", "", "Named connection profile from config")
	rootCmd.Flags().StringVar(&flagSave, "save", "", "Save connection as named profile")
	rootCmd.Flags().StringVar(&flagTheme, "theme", "", "Color theme (catppuccin, high-contrast, tokyo-night, nord, dracula)")
	rootCmd.Version = fmt.Sprintf("%s (commit %s, built %s)", buildVersion, buildCommit, buildDate)
}

func runTUI(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// Positional arg is a profile name shorthand.
	if len(args) == 1 && flagProfile == "" {
		flagProfile = args[0]
	}

	uri, themeName := resolveURIAndTheme(cfg)

	// --save: persist the profile before connecting, then continue.
	if flagSave != "" {
		if err := config.SaveProfile(flagSave, uri, themeName); err != nil {
			return fmt.Errorf("saving profile %q: %w", flagSave, err)
		}
		fmt.Fprintf(os.Stderr, "profile %q saved\n", flagSave)
	}

	client, err := mongoClient.NewClient(uri)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer client.Disconnect()

	app := tui.New(client, themeName)
	p := tea.NewProgram(app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err = p.Run()
	return err
}

// resolveURIAndTheme picks URI and theme in priority order:
// flag > named profile > config default > localhost fallback.
func resolveURIAndTheme(cfg *config.Config) (uri, theme string) {
	// Explicit --theme flag always wins for the theme.
	theme = flagTheme

	// Named profile resolution.
	if flagProfile != "" {
		if conn := cfg.FindConnection(flagProfile); conn != nil {
			if flagURI == "" {
				uri = conn.URI
			}
			if theme == "" {
				theme = conn.Theme
			}
		}
	}

	// --uri flag overrides profile URI.
	if flagURI != "" {
		uri = flagURI
	}

	// --host / --port flags.
	if uri == "" && (flagHost != "" || flagPort != 0) {
		host := flagHost
		if host == "" {
			host = "localhost"
		}
		port := flagPort
		if port == 0 {
			port = 27017
		}
		uri = fmt.Sprintf("mongodb://%s:%d", host, port)
	}

	// Fall back to config default connection.
	if uri == "" && cfg.HasDefaultConnection() {
		def := cfg.DefaultConnection()
		uri = def.URI
		if theme == "" {
			theme = def.Theme
		}
	}

	// Final fallback.
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	return uri, theme
}
