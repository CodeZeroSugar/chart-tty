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

func TestSetAPIKeyReplacesExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := "[ai]\nbase_url = \"http://x\"\napi_key = \"old-key\"\nmodel = \"m\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := SetAPIKey(path, "new-secret"); err != nil {
		t.Fatalf("SetAPIKey() unexpected error: %v", err)
	}
	out, _ := os.ReadFile(path)
	text := string(out)
	if strings.Contains(text, "old-key") {
		t.Errorf("file still contains old key:\n%s", text)
	}
	if !strings.Contains(text, `api_key = "new-secret"`) {
		t.Errorf("file missing new key:\n%s", text)
	}
	for _, want := range []string{"base_url = \"http://x\"", "model = \"m\"", "[ai]"} {
		if !strings.Contains(text, want) {
			t.Errorf("file lost %q:\n%s", want, text)
		}
	}
}

func TestSetAPIKeyInsertsIntoExistingSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := "# my config\n[theme]\nheader_color = \"red\"\n\n[ai]\n# comment inside ai\nmodel = \"m\""
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := SetAPIKey(path, "key-123"); err != nil {
		t.Fatalf("SetAPIKey() unexpected error: %v", err)
	}
	out, _ := os.ReadFile(path)
	text := string(out)

	aiIdx := strings.Index(text, "[ai]")
	keyIdx := strings.Index(text, `api_key = "key-123"`)
	if aiIdx < 0 || keyIdx < 0 || keyIdx < aiIdx {
		t.Fatalf("api_key not inside [ai] section:\n%s", text)
	}
	for _, want := range []string{"# my config", "header_color = \"red\"", "# comment inside ai", "model = \"m\""} {
		if !strings.Contains(text, want) {
			t.Errorf("file lost %q:\n%s", want, text)
		}
	}
}

func TestSetAPIKeyAppendsMissingSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := "[theme]\nheader_color = \"red\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := SetAPIKey(path, "key-456"); err != nil {
		t.Fatalf("SetAPIKey() unexpected error: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() after set: %v", err)
	}
	if cfg.AI.APIKey != "key-456" {
		t.Errorf("AI.APIKey = %q, want key-456", cfg.AI.APIKey)
	}
	if cfg.Theme.HeaderColor != "red" {
		t.Errorf("HeaderColor = %q, want red (existing setting preserved)", cfg.Theme.HeaderColor)
	}
}

func TestSetAPIKeyCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.toml")
	if err := SetAPIKey(path, "fresh-key"); err != nil {
		t.Fatalf("SetAPIKey() unexpected error: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() after set: %v", err)
	}
	if cfg.AI.APIKey != "fresh-key" {
		t.Errorf("AI.APIKey = %q, want fresh-key", cfg.AI.APIKey)
	}
	if cfg.Keys.Quit != "q" {
		t.Errorf("Keys.Quit = %q, want template default q", cfg.Keys.Quit)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("file perms = %v, want 0600", info.Mode().Perm())
	}
}

func TestSetAPIKeyTrimsWhitespaceAndRejectsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := SetAPIKey(path, "   "); err == nil {
		t.Fatal("SetAPIKey(whitespace) expected error")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should not be created for empty key")
	}

	if err := SetAPIKey(path, "  padded-key  "); err != nil {
		t.Fatalf("SetAPIKey(padded) unexpected error: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if cfg.AI.APIKey != "padded-key" {
		t.Errorf("AI.APIKey = %q, want trimmed padded-key", cfg.AI.APIKey)
	}
}