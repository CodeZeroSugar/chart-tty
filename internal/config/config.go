package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Theme   ThemeConfig   `toml:"theme"`
	Keys    KeyConfig     `toml:"keys"`
	AI      AIConfig      `toml:"ai"`
	Library LibraryConfig `toml:"library"`
	Parser  ParserConfig  `toml:"parser"`
}

// ParserConfig holds parsing options. ChordMode is "strict" (default) or
// "relaxed"; the AI converter always uses strict regardless of this setting.
type ParserConfig struct {
	Chords string `toml:"chords"`
}

// LibraryConfig holds chart library settings. An empty Path means the
// default location ($XDG_DATA_HOME/chart-tty/library.db).
type LibraryConfig struct {
	Path string `toml:"path"`
}

type ThemeConfig struct {
	HeaderColor    string `toml:"header_color"`
	CommentColor   string `toml:"comment_color"`
	HighlightColor string `toml:"highlight_color"`
}

type KeyConfig struct {
	Quit          string `toml:"quit"`
	ScrollDown    string `toml:"scroll_down"`
	ScrollUp      string `toml:"scroll_up"`
	TransposeUp   string `toml:"transpose_up"`
	TransposeDown string `toml:"transpose_down"`
	Home          string `toml:"home"`
}

type AIConfig struct {
	BaseURL string `toml:"base_url"`
	APIKey  string `toml:"api_key"`
	Model   string `toml:"model"`
}

func Default() Config {
	return Config{
		Theme: ThemeConfig{HeaderColor: "cyan", CommentColor: "yellow", HighlightColor: "yellow"},
		Keys: KeyConfig{
			Quit:          "q",
			ScrollDown:    "j",
			ScrollUp:      "k",
			TransposeUp:   "+",
			TransposeDown: "-",
			Home:          "h",
		},
		AI: AIConfig{
			BaseURL: "https://api.openai.com/v1",
			Model:   "gpt-4o-mini",
		},
		Parser: ParserConfig{Chords: "strict"},
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

const defaultTemplate = `# chart-tty configuration
# All fields optional — unset fields fall back to these defaults.

[theme]
# lipgloss color names: red, green, yellow, blue, magenta, cyan, white, gray,
# or ANSI numbers 0-255 ("196"), or hex "#ff8800"
header_color = "cyan"    # section headers like [chorus], titles, main menu banner
comment_color = "yellow" # {comment:} directive text
highlight_color = "yellow" # selected row / cursor / delete banner

[keys]
# Single-key names as bubbletea reports them
quit = "q"
scroll_down = "j"
scroll_up = "k"
transpose_up = "+"
transpose_down = "-"
home = "h"

[parser]
# Chord grammar for user charts: "strict" (default) or "relaxed".
# The AI converter always uses strict.
chords = "strict"

[ai]
# OpenAI-compatible endpoint. Works with OpenAI, gateways, Ollama/LM Studio local models.
base_url = "https://api.openai.com/v1"
api_key = ""             # set via: chart-tty --set-api-key <key>
model = "gpt-4o-mini"
`

// initConfigTemplate is what -init-config writes: the default options with
// the AI section left blank so no provider-specific values are pre-filled.
const initConfigTemplate = `# chart-tty configuration
# All fields optional — unset fields fall back to these defaults.

[theme]
# lipgloss color names: red, green, yellow, blue, magenta, cyan, white, gray,
# or ANSI numbers 0-255 ("196"), or hex "#ff8800"
header_color = "cyan"    # section headers like [chorus], titles, main menu banner
comment_color = "yellow" # {comment:} directive text
highlight_color = "yellow" # selected row / cursor / delete banner

[keys]
# Single-key names as bubbletea reports them
quit = "q"
scroll_down = "j"
scroll_up = "k"
transpose_up = "+"
transpose_down = "-"
home = "h"

[parser]
# Chord grammar for user charts: "strict" (default) or "relaxed".
# The AI converter always uses strict.
chords = "strict"

[ai]
# OpenAI-compatible endpoint. Works with OpenAI, gateways, Ollama/LM Studio local models.
base_url = ""
api_key = ""             # set via: chart-tty --set-api-key <key>
model = ""
`

// WriteDefault creates a config file at path with the default options and a
// blank [ai] section. It never overwrites an existing file.
func WriteDefault(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config already exists at %s", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(initConfigTemplate), 0o600); err != nil {
		return err
	}
	return verifyParses(path)
}

// SetAPIKey stores the API key in the config file without disturbing the rest
// of the file (comments and unrelated settings are preserved). Missing files
// are created from the default template. The file is written with 0600
// permissions since it holds a secret.
func SetAPIKey(path, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("api key must not be empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		out := strings.Replace(defaultTemplate, `api_key = ""`, fmt.Sprintf("api_key = %q", key), 1)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(out), 0o600); err != nil {
			return err
		}
		return verifyParses(path)
	}

	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines)+3)
	inAI := false
	replaced := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			if inAI && !replaced {
				out = append(out, fmt.Sprintf("api_key = %q", key))
				replaced = true
			}
			inAI = trimmed == "[ai]"
		} else if inAI && strings.HasPrefix(trimmed, "api_key") {
			line = fmt.Sprintf("api_key = %q", key)
			replaced = true
		}
		out = append(out, line)
	}
	if inAI && !replaced {
		out = append(out, fmt.Sprintf("api_key = %q", key))
		replaced = true
	}
	if !replaced {
		out = append(out, "", "[ai]", fmt.Sprintf("api_key = %q", key))
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o600); err != nil {
		return err
	}
	return verifyParses(path)
}

func verifyParses(path string) error {
	if _, err := Load(path); err != nil {
		return fmt.Errorf("config no longer parses after update: %w", err)
	}
	return nil
}
