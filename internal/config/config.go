package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Connection struct {
	Name     string `mapstructure:"name"`
	URI      string `mapstructure:"uri"`
	Default  bool   `mapstructure:"default"`
	TLS      bool   `mapstructure:"tls"`
	TLSCert  string `mapstructure:"tlsCertFile"`
}

type UIConfig struct {
	Theme    string `mapstructure:"theme"`
	Mouse    bool   `mapstructure:"mouse"`
	Editor   string `mapstructure:"editor"`
	PageSize int    `mapstructure:"pageSize"`
}

type Config struct {
	Connections []Connection `mapstructure:"connections"`
	UI          UIConfig     `mapstructure:"ui"`
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

func (c *Config) EditorCmd() string {
	if c.UI.Editor != "" {
		return c.UI.Editor
	}
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	return "nano"
}

func setDefaults() {
	viper.SetDefault("ui.theme", "dark")
	viper.SetDefault("ui.mouse", true)
	viper.SetDefault("ui.pageSize", 50)
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
`
	return os.WriteFile(path, []byte(content), 0o600)
}
