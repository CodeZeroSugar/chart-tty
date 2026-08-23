package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CodeZeroSugar/chart-tty/internal/aichart"
	"github.com/CodeZeroSugar/chart-tty/internal/config"
	"github.com/CodeZeroSugar/chart-tty/internal/parser"
)

func testLines(n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = "line" + string(rune('0'+i))
	}
	return lines
}

func update(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	nm, _ := m.Update(msg)
	m2, ok := nm.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", nm)
	}
	return m2
}

func TestModelScroll(t *testing.T) {
	m := update(t, NewModel(testLines(10), RenderConfig{}), tea.WindowSizeMsg{Width: 20, Height: 4})

	if got := m.View(); !strings.Contains(got, "line0") || !strings.Contains(got, "line3") {
		t.Errorf("initial view = %q, want lines 0-3", got)
	}
	if strings.Contains(m.View(), "line4") {
		t.Errorf("initial view = %q, must not contain line4", m.View())
	}

	m = update(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if got := m.Offset(); got != 1 {
		t.Errorf("after down offset = %d, want 1", got)
	}

	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if got := m.Offset(); got != 2 {
		t.Errorf("after j offset = %d, want 2", got)
	}

	m = update(t, m, tea.KeyMsg{Type: tea.KeyPgDown})
	if got := m.Offset(); got != 6 {
		t.Errorf("after pgdown offset = %d, want 6", got)
	}
	if got := m.View(); !strings.Contains(got, "line6") || !strings.Contains(got, "line9") {
		t.Errorf("bottom view = %q, want lines 6-9", got)
	}

	m = update(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if got := m.Offset(); got != 6 {
		t.Errorf("after extra down offset = %d, want clamp at 6", got)
	}

	m = update(t, m, tea.KeyMsg{Type: tea.KeyHome})
	if got := m.Offset(); got != 0 {
		t.Errorf("after home offset = %d, want 0", got)
	}
}

func TestModelScrollUpClamps(t *testing.T) {
	m := update(t, NewModel(testLines(5), RenderConfig{}), tea.WindowSizeMsg{Width: 20, Height: 2})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if got := m.Offset(); got != 0 {
		t.Errorf("offset = %d, want 0 (clamp at top)", got)
	}
}

func TestModelQuit(t *testing.T) {
	m := update(t, NewModel(testLines(3), RenderConfig{}), tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if !m.Quitting() {
		t.Error("q did not quit")
	}

	m = update(t, NewModel(testLines(3), RenderConfig{}), tea.KeyMsg{Type: tea.KeyCtrlC})
	if !m.Quitting() {
		t.Error("ctrl+c did not quit")
	}
}

func TestModelResize(t *testing.T) {
	m := update(t, NewModel(testLines(10), RenderConfig{}), tea.WindowSizeMsg{Width: 20, Height: 8})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyPgDown}) // offset 8, max 2
	if got := m.Offset(); got != 2 {
		t.Errorf("offset after resize+pgdown = %d, want 2", got)
	}
	m = update(t, m, tea.WindowSizeMsg{Width: 20, Height: 3}) // max becomes 7
	if got := m.Offset(); got != 2 {
		t.Errorf("offset after shrink = %d, want unchanged 2", got)
	}
	m = update(t, m, tea.WindowSizeMsg{Width: 20, Height: 1}) // max becomes 9
	if got := m.Offset(); got != 2 {
		t.Errorf("offset after further shrink = %d, want unchanged 2", got)
	}
}

func TestModelViewTitleAndHelp(t *testing.T) {
	m := update(t, NewModel(testLines(3), RenderConfig{}).SetTitle("Swing Low"), tea.WindowSizeMsg{Width: 40, Height: 10})

	view := m.View()
	if !strings.Contains(view, "Swing Low") {
		t.Errorf("view %q does not contain title", view)
	}
	if !strings.Contains(view, "j/k scroll") {
		t.Errorf("view %q does not contain help text", view)
	}
	if !strings.Contains(view, "line2") {
		t.Errorf("view %q does not contain body lines", view)
	}
}

func TestModelViewNoHelp(t *testing.T) {
	m := update(t, NewModel(testLines(2), RenderConfig{}).SetShowHelp(false), tea.WindowSizeMsg{Width: 40, Height: 10})
	if strings.Contains(m.View(), "j/k scroll") {
		t.Errorf("view %q should not contain help", m.View())
	}
}

func TestNewDocModel(t *testing.T) {
	doc := &parser.Document{
		Title:    "Swing Low",
		Metadata: map[string][]string{},
		Sections: []parser.Section{
			{
				Name: "chorus",
				Lines: []parser.ParsedLine{
					{Type: parser.LineTypeChordAndLyric, Raw: "Swing [D]low", Lyrics: "Swing low", Chords: []parser.ChordToken{{Name: "D", Position: 6}}},
				},
			},
		},
	}
	m := NewDocModel(doc, RenderConfig{})
	if m.title != "Swing Low" {
		t.Errorf("title = %q, want %q", m.title, "Swing Low")
	}
	wantLines := []string{"[chorus]", "      D  ", "Swing low"}
	if !reflect.DeepEqual(m.lines, wantLines) {
		t.Errorf("lines = %#v, want %#v", m.lines, wantLines)
	}
}

func TestModelTransposeKeys(t *testing.T) {
	doc := &parser.Document{
		Title:    "Swing Low",
		Key:      "G",
		Metadata: map[string][]string{},
		Sections: []parser.Section{
			{
				Name: "",
				Lines: []parser.ParsedLine{
					{Type: parser.LineTypeChordAndLyric, Raw: "[C] home", Lyrics: "home", Chords: []parser.ChordToken{{Name: "C", Position: 0}}},
				},
			},
		},
	}
	m := update(t, NewDocModel(doc, RenderConfig{}), tea.WindowSizeMsg{Width: 40, Height: 10})

	view := m.View()
	if !strings.Contains(view, "Key: G") {
		t.Errorf("view %q does not show original key", view)
	}

	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	if m.Transpose() != 1 {
		t.Errorf("transpose = %d, want 1", m.Transpose())
	}
	view = m.View()
	if !strings.Contains(view, "C#") {
		t.Errorf("view %q does not contain transposed chord C#", view)
	}
	if !strings.Contains(view, "Key: G#") {
		t.Errorf("view %q does not show transposed key", view)
	}

	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("-")})
	if m.Transpose() != 0 {
		t.Errorf("transpose = %d, want 0", m.Transpose())
	}
	if got := m.View(); strings.Contains(got, "C#") {
		t.Errorf("view %q still contains C# after transpose back", got)
	}
}

func TestModelSetTranspose(t *testing.T) {
	doc := &parser.Document{
		Title:    "Song",
		Key:      "C",
		Metadata: map[string][]string{},
		Sections: []parser.Section{
			{
				Name: "",
				Lines: []parser.ParsedLine{
					{Type: parser.LineTypeChordAndLyric, Raw: "[F] here", Lyrics: "here", Chords: []parser.ChordToken{{Name: "F", Position: 0}}},
				},
			},
		},
	}
	m := update(t, NewDocModel(doc, RenderConfig{}).SetTranspose(2), tea.WindowSizeMsg{Width: 40, Height: 10})
	if m.Transpose() != 2 {
		t.Errorf("transpose = %d, want 2", m.Transpose())
	}
	if got := m.View(); !strings.Contains(got, "G") || !strings.Contains(got, "Key: D") {
		t.Errorf("view %q does not show transposed F->G and key D", got)
	}
}

func TestModelTransposeNoDocIsNoop(t *testing.T) {
	m := update(t, NewModel(testLines(3), RenderConfig{}), tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	if got := m.Offset(); got != 0 {
		t.Errorf("offset = %d, want unchanged 0", got)
	}
	if got := m.View(); strings.Contains(got, "line1") {
		t.Errorf("view %q unchanged lines expected", got)
	}
}

func TestModelCustomKeys(t *testing.T) {
	keys := config.Default().Keys
	keys.Quit = "x"
	keys.ScrollDown = "s"

	m := update(t, NewModel(testLines(10), RenderConfig{}).SetKeys(keys), tea.WindowSizeMsg{Width: 20, Height: 4})

	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if got := m.Offset(); got != 1 {
		t.Errorf("custom scroll-down offset = %d, want 1", got)
	}

	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if m.Quitting() {
		t.Error("default quit key q should not quit after remap")
	}

	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if !m.Quitting() {
		t.Error("custom quit key x did not quit")
	}
}

func TestModelConvertKey(t *testing.T) {
	converted := "{title: Converted Song}\n{start_of_chorus}\nSwing [D]low\n{eoc}"
	resp, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{{"message": map[string]any{"content": converted}}},
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(resp)
	}))
	t.Cleanup(srv.Close)
	client := &aichart.Client{BaseURL: srv.URL, Model: "m", HTTP: srv.Client()}

	doc := &parser.Document{
		Title:    "Original",
		Metadata: map[string][]string{},
		Sections: []parser.Section{
			{
				Name: "",
				Lines: []parser.ParsedLine{
					{Type: parser.LineTypeLyric, Raw: "C G", Lyrics: "C G"},
				},
			},
		},
	}
	m := update(t, NewDocModel(doc, RenderConfig{}).SetConverter("C G", client), tea.WindowSizeMsg{Width: 40, Height: 10})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})

	if !strings.Contains(m.View(), "Converted Song") {
		t.Errorf("view %q does not show converted title", m.View())
	}
	if !strings.Contains(m.View(), "converted") {
		t.Errorf("view %q does not show conversion message", m.View())
	}
}

func TestModelConvertWithoutConverterNoop(t *testing.T) {
	doc := &parser.Document{Title: "T", Metadata: map[string][]string{}}
	m := update(t, NewDocModel(doc, RenderConfig{}), tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if strings.Contains(m.View(), "converted") {
		t.Errorf("view %q should not mention conversion without a converter", m.View())
	}
}