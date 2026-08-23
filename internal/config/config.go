package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Theme ThemeConfig `toml:"theme"`
	Keys  KeyConfig   `toml:"keys"`
	AI    AIConfig    `toml:"ai"`
}

type ThemeConfig struct {
	HeaderColor  string `toml:"header_color"`
	CommentColor string `toml:"comment_color"`
}

type KeyConfig struct {
	Quit          string `toml:"quit"`
	ScrollDown    string `toml:"scroll_down"`
	ScrollUp      string `toml:"scroll_up"`
	TransposeUp   string `toml:"transpose_up"`
	TransposeDown string `toml:"transpose_down"`
}

type AIConfig struct {
	BaseURL string `toml:"base_url"`
	APIKey  string `toml:"api_key"`
	Model   string `toml:"model"`
}

func Default() Config {
	return Config{
		Theme: ThemeConfig{HeaderColor: "cyan", CommentColor: "yellow"},
		Keys: KeyConfig{
			Quit:          "q",
			ScrollDown:    "j",
			ScrollUp:      "k",
			TransposeUp:   "+",
			TransposeDown: "-",
		},
		AI: AIConfig{
			BaseURL: "https://api.openai.com/v1",
			Model:   "gpt-4o-mini",
		},
	}
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine config directory: %w", err)
	}
	return filepath.Join(dir, "chart-tty", "config.toml"), nil
}

func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing %s: %w", path, err)
	}
	return cfg, nil
}