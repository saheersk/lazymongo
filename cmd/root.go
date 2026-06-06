// Package cmd wires the cobra CLI to the TUI application.
package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/config"
	mongoClient "github.com/saheersk/lazymongo/internal/mongo"
	"github.com/saheersk/lazymongo/internal/tui"
	"github.com/spf13/cobra"
)

var (
	flagURI  string
	flagHost string
	flagPort int
)

var rootCmd = &cobra.Command{
	Use:   "lazymongo",
	Short: "A terminal UI for MongoDB — like lazygit, but for Mongo",
	Long: `lazymongo is a fast, keyboard-driven TUI for browsing and editing MongoDB.

Connect with a URI:
  lazymongo --uri mongodb://localhost:27017

Or use a saved profile from ~/.config/lazymongo/config.yaml:
  lazymongo`,
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
}

func runTUI(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	uri := resolveURI(cfg)

	client, err := mongoClient.NewClient(uri)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer client.Disconnect()

	app := tui.New(client)
	p := tea.NewProgram(app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err = p.Run()
	return err
}

// resolveURI picks the connection URI in priority order:
// flag > config default > localhost fallback.
func resolveURI(cfg *config.Config) string {
	if flagURI != "" {
		return flagURI
	}
	if flagHost != "" || flagPort != 0 {
		host := flagHost
		if host == "" {
			host = "localhost"
		}
		port := flagPort
		if port == 0 {
			port = 27017
		}
		return fmt.Sprintf("mongodb://%s:%d", host, port)
	}
	if cfg.HasDefaultConnection() {
		return cfg.DefaultConnection().URI
	}
	return "mongodb://localhost:27017"
}
