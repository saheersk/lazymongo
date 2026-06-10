package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

type Connection struct {
	Name    string `mapstructure:"name"       yaml:"name"`
	URI     string `mapstructure:"uri"        yaml:"uri"`
	Default bool   `mapstructure:"default"    yaml:"default,omitempty"`
	TLS     bool   `mapstructure:"tls"        yaml:"tls,omitempty"`
	TLSCert string `mapstructure:"tlsCertFile" yaml:"tlsCertFile,omitempty"`
	Theme   string `mapstructure:"theme"      yaml:"theme,omitempty"`
}

type UIConfig struct {
	Theme       string            `mapstructure:"theme"        yaml:"theme,omitempty"`
	Mouse       bool              `mapstructure:"mouse"        yaml:"mouse,omitempty"`
	Editor      string            `mapstructure:"editor"       yaml:"editor,omitempty"`
	PageSize    int               `mapstructure:"pageSize"     yaml:"pageSize,omitempty"`
	NerdFonts   bool              `mapstructure:"nerdFonts"    yaml:"nerdFonts,omitempty"`
	Keybindings map[string]string `mapstructure:"keybindings"  yaml:"keybindings,omitempty"`
}

type Config struct {
	Connections []Connection `mapstructure:"connections" yaml:"connections,omitempty"`
	UI          UIConfig     `mapstructure:"ui"          yaml:"ui,omitempty"`
}

func Load() (*Config, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(dir)
	viper.SetEnvPrefix("LAZYMONGO")
	viper.AutomaticEnv()

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := writeDefaultConfig(dir); err != nil {
				return nil, fmt.Errorf("creating default config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) HasDefaultConnection() bool {
	for _, conn := range c.Connections {
		if conn.Default {
			return true
		}
	}
	return false
}

func (c *Config) DefaultConnection() *Connection {
	for i, conn := range c.Connections {
		if conn.Default {
			return &c.Connections[i]
		}
	}
	return nil
}

func (c *Config) FindConnection(name string) *Connection {
	for i, conn := range c.Connections {
		if conn.Name == name {
			return &c.Connections[i]
		}
	}
	return nil
}

// EditorCmd returns the editor to use, in priority order:
// config file > $EDITOR > $VISUAL > vim.
func (c *Config) EditorCmd() string {
	if c.UI.Editor != "" {
		return c.UI.Editor
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	return "vim"
}

// SaveProfile upserts a named connection profile in the config file.
// If a connection with the given name already exists, its URI and Theme are
// updated in-place; otherwise a new entry is appended.
func SaveProfile(name, uri, theme string) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "config.yaml")

	// Load existing config (or start empty).
	cfg, err := loadRaw(path)
	if err != nil {
		return fmt.Errorf("loading config for save: %w", err)
	}

	// Upsert.
	found := false
	for i, conn := range cfg.Connections {
		if conn.Name == name {
			cfg.Connections[i].URI = uri
			cfg.Connections[i].Theme = theme
			found = true
			break
		}
	}
	if !found {
		cfg.Connections = append(cfg.Connections, Connection{
			Name:  name,
			URI:   uri,
			Theme: theme,
		})
	}

	return writeConfig(cfg)
}

// loadRaw reads and unmarshals the YAML config file directly (without viper)
// so we can round-trip it faithfully.
func loadRaw(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// writeConfig marshals cfg to YAML and persists it to the config file.
func writeConfig(cfg *Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "config.yaml")

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

func setDefaults() {
	viper.SetDefault("ui.theme", "dark")
	viper.SetDefault("ui.mouse", true)
	viper.SetDefault("ui.pageSize", 50)
	viper.SetDefault("ui.editor", "vim")
	viper.SetDefault("ui.nerdFonts", true)
}

func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "lazymongo")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func writeDefaultConfig(dir string) error {
	path := filepath.Join(dir, "config.yaml")
	content := `connections:
  - name: local
    uri: mongodb://localhost:27017
    default: true

ui:
  theme: dark
  mouse: true
  pageSize: 50
  nerdFonts: true   # set false if your terminal font has no Nerd Font glyphs
  editor: vim   # vim | nvim | nano | emacs | "code --wait"
`
	return os.WriteFile(path, []byte(content), 0o600)
}
