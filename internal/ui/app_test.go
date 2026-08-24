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
	m := update(t, NewModel(testLines(10), RenderConfig{}).SetShowHelp(false), tea.WindowSizeMsg{Width: 20, Height: 4})

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
	m := update(t, NewModel(testLines(5), RenderConfig{}).SetShowHelp(false), tea.WindowSizeMsg{Width: 20, Height: 2})
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
	m := update(t, NewModel(testLines(10), RenderConfig{}).SetShowHelp(false), tea.WindowSizeMsg{Width: 20, Height: 8})
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

func TestModelPageUpWithB(t *testing.T) {
	m := update(t, NewModel(testLines(12), RenderConfig{}).SetShowHelp(false), tea.WindowSizeMsg{Width: 20, Height: 4})

	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if got := m.Offset(); got != 8 {
		t.Fatalf("after G offset = %d, want 8", got)
	}

	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	if got := m.Offset(); got != 4 {
		t.Errorf("after b offset = %d, want 4", got)
	}

	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	if got := m.Offset(); got != 0 {
		t.Errorf("after second b offset = %d, want 0", got)
	}

	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	if got := m.Offset(); got != 0 {
		t.Errorf("after third b offset = %d, want clamp at 0", got)
	}
}

func TestModelBodyHeightReservesChromeRows(t *testing.T) {
	m := update(t, NewModel(testLines(10), RenderConfig{}).SetTitle("Song"), tea.WindowSizeMsg{Width: 40, Height: 6})
	view := m.View()
	for _, want := range []string{"Song", "line0", "line2"} {
		if !strings.Contains(view, want) {
			t.Errorf("view %q missing %q", view, want)
		}
	}
	for _, unwanted := range []string{"line4", "line5"} {
		if strings.Contains(view, unwanted) {
			t.Errorf("view %q must not contain %q (chrome reserves rows)", view, unwanted)
		}
	}
}

func TestModelTwoColumnWideView(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e", "f"}
	m := update(t, NewModel(lines, RenderConfig{}).SetShowHelp(false), tea.WindowSizeMsg{Width: 84, Height: 3})

	view := m.View()
	if !strings.Contains(view, "║") {
		t.Errorf("wide view %q missing divider", view)
	}
	firstRow := strings.Split(view, "\n")[0]
	parts := strings.Split(firstRow, " ║ ")
	if len(parts) != 2 {
		t.Fatalf("first row %q does not split on divider", firstRow)
	}
	if !strings.Contains(parts[0], "a") || !strings.Contains(parts[1], "d") {
		t.Errorf("first row = %q, want a | d flow", firstRow)
	}
	lastRow := strings.Split(view, "\n")[len(strings.Split(view, "\n"))-1]
	if !strings.Contains(lastRow, "c") || !strings.Contains(lastRow, "f") {
		t.Errorf("last row = %q, want c | f flow", lastRow)
	}
}

func TestModelNarrowViewNoDivider(t *testing.T) {
	m := update(t, NewModel(testLines(6), RenderConfig{}).SetShowHelp(false), tea.WindowSizeMsg{Width: 40, Height: 3})
	if strings.Contains(m.View(), "║") {
		t.Errorf("narrow view %q must not contain divider", m.View())
	}
}

func TestModelWideButShortStaysSingleColumn(t *testing.T) {
	m := update(t, NewModel(testLines(4), RenderConfig{}).SetShowHelp(false), tea.WindowSizeMsg{Width: 100, Height: 6})
	if m.useColumns() {
		t.Error("content fitting one column must not activate two-column layout")
	}
	if strings.Contains(m.View(), "║") {
		t.Errorf("view %q must not contain divider for short content", m.View())
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

	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m2, ok := nm.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", nm)
	}
	if cmd == nil {
		t.Fatal("pressing c with a converter must return a cmd")
	}
	if !m2.converting {
		t.Error("model not marked converting during conversion")
	}
	if !strings.Contains(m2.View(), "converting") {
		t.Errorf("view %q does not show converting state", m2.View())
	}

	done := cmd()
	cdm, ok := done.(convertDoneMsg)
	if !ok {
		t.Fatalf("cmd produced %T, want convertDoneMsg", done)
	}

	m3 := update(t, m2, cdm)
	if m3.converting {
		t.Error("still converting after convertDoneMsg")
	}
	view := m3.View()
	if !strings.Contains(view, "Converted Song") {
		t.Errorf("view %q does not show converted title", view)
	}
	if !strings.Contains(view, "converted") && !strings.Contains(view, "converting") {
		t.Errorf("view %q lost conversion message", view)
	}
	if strings.Contains(view, "converting…") {
		t.Errorf("view %q still shows converting after completion", view)
	}
}

func TestModelConvertNoDoubleStart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(srv.Close)
	client := &aichart.Client{BaseURL: srv.URL, Model: "m", HTTP: srv.Client()}

	doc := &parser.Document{Title: "T", Metadata: map[string][]string{}}
	m := NewDocModel(doc, RenderConfig{}).SetConverter("raw", client)

	m2, cmd := m.startConversion()
	if cmd == nil || !m2.converting {
		t.Fatal("first startConversion should begin conversion")
	}
	m3, cmd2 := m2.startConversion()
	if cmd2 != nil {
		t.Error("second startConversion while converting must be a no-op")
	}
	if !m3.converting {
		t.Error("converting state lost on double press")
	}
}

func TestModelConvertWithoutConverterNoop(t *testing.T) {
	doc := &parser.Document{Title: "T", Metadata: map[string][]string{}}
	nm, cmd := NewDocModel(doc, RenderConfig{}).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m, ok := nm.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", nm)
	}
	if cmd != nil {
		t.Error("c without converter must not return a cmd")
	}
	if strings.Contains(m.View(), "converting") {
		t.Errorf("view %q should not mention converting without a converter", m.View())
	}
}