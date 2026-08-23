package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Theme.HeaderColor == "" || cfg.Theme.CommentColor == "" {
		t.Errorf("theme defaults missing: %#v", cfg.Theme)
	}
	if cfg.Keys.Quit != "q" || cfg.Keys.ScrollDown != "j" || cfg.Keys.ScrollUp != "k" ||
		cfg.Keys.TransposeUp != "+" || cfg.Keys.TransposeDown != "-" {
		t.Errorf("key defaults wrong: %#v", cfg.Keys)
	}
	if cfg.AI.BaseURL == "" || cfg.AI.Model == "" {
		t.Errorf("AI defaults missing: %#v", cfg.AI)
	}
}

func TestLoadOverridesWithDefaultsPreserved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `[theme]
header_color = "magenta"

[keys]
transpose_up = "="

[ai]
model = "local-model"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.Theme.HeaderColor != "magenta" {
		t.Errorf("HeaderColor = %q, want magenta", cfg.Theme.HeaderColor)
	}
	if cfg.Theme.CommentColor != "yellow" {
		t.Errorf("CommentColor = %q, want default yellow", cfg.Theme.CommentColor)
	}
	if cfg.Keys.TransposeUp != "=" {
		t.Errorf("TransposeUp = %q, want =", cfg.Keys.TransposeUp)
	}
	if cfg.Keys.Quit != "q" {
		t.Errorf("Quit = %q, want default q", cfg.Keys.Quit)
	}
	if cfg.AI.Model != "local-model" {
		t.Errorf("AI Model = %q, want local-model", cfg.AI.Model)
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.Keys.Quit != "q" {
		t.Errorf("Quit = %q, want default", cfg.Keys.Quit)
	}
}

func TestLoadMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(path, []byte("not = [valid"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load() expected error for malformed TOML")
	}
}

func TestDefaultPath(t *testing.T) {
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() unexpected error: %v", err)
	}
	if !strings.HasSuffix(path, string(filepath.Separator)+"chart-tty"+string(filepath.Separator)+"config.toml") {
		t.Errorf("DefaultPath() = %q, want suffix chart-tty/config.toml", path)
	}
}