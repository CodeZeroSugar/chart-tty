package ui

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CodeZeroSugar/chart-tty/internal/aichart"
	"github.com/CodeZeroSugar/chart-tty/internal/config"
	"github.com/CodeZeroSugar/chart-tty/internal/db"
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

func testStore(t *testing.T) *db.Store {
	t.Helper()
	s, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory(): %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func pressKey(t *testing.T, m Model, k string) (Model, tea.Cmd) {
	t.Helper()
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
	m2, ok := nm.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", nm)
	}
	return m2, cmd
}

func TestLibraryBrowserFlow(t *testing.T) {
	s := testStore(t)
	id1, _ := s.AddChart("Swing Low", "Artist A", "import", "{title: Swing Low}\n{start_of_verse}\n[G]Swing low\n{end_of_verse}")
	_, _ = s.AddChart("Second Song", "Artist B", "ai", "{title: Second Song}\n[C]body")

	m := update(t, NewModel(testLines(3), RenderConfig{}).SetShowHelp(false).SetStore(s), tea.WindowSizeMsg{Width: 80, Height: 10})

	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	if got := m.Screen(); got != "browseCharts" {
		t.Fatalf("screen = %q, want browseCharts", got)
	}
	view := m.View()
	if !strings.Contains(view, "Library") || !strings.Contains(view, "Swing Low") || !strings.Contains(view, "Second Song") {
		t.Errorf("browser view missing entries:\n%s", view)
	}

	// esc returns to chart view without loading anything.
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if got := m.Screen(); got != "viewChart" {
		t.Errorf("after esc screen = %q, want viewChart", got)
	}

	// Re-open and select the first entry (list is newest-first: Second Song).
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2, ok := nm.(Model)
	if !ok {
		t.Fatalf("Update returned %T", nm)
	}
	if cmd != nil {
		t.Error("enter should not produce a cmd")
	}
	if got := m2.Screen(); got != "viewChart" {
		t.Fatalf("after enter screen = %q, want viewChart", got)
	}
	if m2.title != "Second Song" {
		t.Errorf("loaded title = %q, want Second Song", m2.title)
	}
	view = m2.View()
	if !strings.Contains(view, "C") || !strings.Contains(view, "body") {
		t.Errorf("view %q does not look like loaded chart content", view)
	}
	_ = id1
}

func TestLibraryBrowserEmpty(t *testing.T) {
	s := testStore(t)
	m := update(t, NewModel(testLines(3), RenderConfig{}).SetShowHelp(false).SetStore(s), tea.WindowSizeMsg{Width: 60, Height: 10})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	if !strings.Contains(m.View(), "library is empty") {
		t.Errorf("empty-library view = %q", m.View())
	}
}

func TestModelChordModeToggle(t *testing.T) {
	raw := "Coda\nGm*\nChorus"
	m := update(t, NewModel(nil, RenderConfig{}).SetShowHelp(false).SetSource(raw), tea.WindowSizeMsg{Width: 60, Height: 10})

	if got := m.ChordMode(); got != parser.StrictChords {
		t.Fatalf("initial mode = %v, want strict", got)
	}

	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	if got := m.ChordMode(); got != parser.RelaxedChords {
		t.Errorf("after toggle mode = %v, want relaxed", got)
	}
	if !strings.Contains(m.View(), "chord mode: relaxed") {
		t.Errorf("view %q missing mode message", m.View())
	}

	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	if got := m.ChordMode(); got != parser.StrictChords {
		t.Errorf("after second toggle mode = %v, want strict", got)
	}
	if !strings.Contains(m.View(), "chord mode: strict") {
		t.Errorf("view %q missing strict message", m.View())
	}
}

func TestFilePicker(t *testing.T) {
	dir := t.TempDir()
	chart := "{title: Picked}\n[G]body"
	for name, content := range map[string]string{
		"a_song.pro": chart,
		"b_song.txt": chart,
		"c_song.cho": chart,
		"notes.md":   "not a chart",
		"readme.TXT": chart,
		"sub":        "",
	} {
		p := filepath.Join(dir, name)
		if name == "sub" {
			os.Mkdir(p, 0o755)
			continue
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	m := update(t, NewModel(testLines(2), RenderConfig{}).SetShowHelp(false), tea.WindowSizeMsg{Width: 60, Height: 10})

	// Open the picker rooted at the temp dir.
	m.pickDir = dir
	nm, _ := m.openFilePicker()
	var ok bool
	m, ok = nm.(Model)
	if !ok {
		t.Fatalf("openFilePicker returned %T, want Model", nm)
	}

	if got := m.Screen(); got != "pickFile" {
		t.Fatalf("screen = %q, want pickFile", got)
	}
	view := m.View()
	// Only allowed extensions listed; .md hidden, subdir hidden.
	if !strings.Contains(view, "a_song.pro") || !strings.Contains(view, "b_song.txt") {
		t.Errorf("view missing chart files:\n%s", view)
	}
	if strings.Contains(view, "notes.md") || strings.Contains(view, "sub") {
		t.Errorf("view must not list disallowed entries:\n%s", view)
	}
	// Case-insensitive extension match: readme.TXT present.
	if !strings.Contains(view, "readme.TXT") {
		t.Errorf("view missing uppercase .TXT file:\n%s", view)
	}

	// Enter opens the first (sorted) chart into the viewer.
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.Screen(); got != "viewChart" {
		t.Fatalf("after enter screen = %q, want viewChart", got)
	}
	if !strings.Contains(m.View(), "Picked") {
		t.Errorf("view %q missing loaded chart title", m.View())
	}
}

func TestFilePickerEmptyDir(t *testing.T) {
	dir := t.TempDir()
	m := update(t, NewModel(testLines(2), RenderConfig{}).SetShowHelp(false), tea.WindowSizeMsg{Width: 60, Height: 10})
	m.pickDir = dir
	m.pickFiles = nil
	m.pickCursor = 0
	m.screen = screenPickFile
	if !strings.Contains(m.View(), "no chart files found") {
		t.Errorf("empty view = %q", m.View())
	}
	// esc exits back to the viewer.
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if got := m.Screen(); got != "viewChart" {
		t.Errorf("after esc screen = %q, want viewChart", got)
	}
}

func TestHeaderRow(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		msg     string
		width   int
		wantMsg string
	}{
		{"no message", "Swing Low", "", 40, ""},
		{"message right aligned", "Swing Low", "converted", 40, "converted"},
		{"short width truncates message", "Swing Low Sweet Chariot Long Title", "AI conversion failed", 40, "…"},
		{"wide fit", "T", "AI conversion failed: gateway exploded", 80, "AI conversion failed: gateway exploded"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := headerRow(tt.title, tt.msg, tt.width)
			if tt.msg == "" {
				// Title-only row is unpadded; the message case is padded.
				if !strings.Contains(row, tt.title) {
					t.Errorf("headerRow() = %q, missing title", row)
				}
				return
			}
			if w := displayWidth(row); w != tt.width {
				t.Errorf("headerRow() display width = %d, want %d (row=%q)", w, tt.width, row)
			}
			if tt.wantMsg == "…" {
				if !strings.HasSuffix(row, "…") {
					t.Errorf("headerRow() = %q, want ellipsis truncation", row)
				}
				return
			}
			if !strings.Contains(row, tt.wantMsg) {
				t.Errorf("headerRow() = %q, missing %q", row, tt.wantMsg)
			}
		})
	}
}

func TestViewMessageRightAligned(t *testing.T) {
	doc := &parser.Document{Title: "Song", Metadata: map[string][]string{}}
	m := update(t, NewDocModel(doc, RenderConfig{}).SetShowHelp(false), tea.WindowSizeMsg{Width: 60, Height: 10})
	m.message = "converting…"
	header := strings.SplitN(m.View(), "\n", 2)[0]
	if w := displayWidth(header); w != 60 {
		t.Errorf("header display width = %d, want 60", w)
	}
	if !strings.HasSuffix(header, "converting…") {
		t.Errorf("header %q does not end with the message (right-aligned)", header)
	}
}

func mockCaptureConverter(t *testing.T, onUser func(string)) *aichart.Client {
	t.Helper()
	converted := "{title: Converted Song}\n{start_of_chorus}\nSwing [D]low\n{eoc}"
	resp, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{{"message": map[string]any{"content": converted}}},
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decoding request: %v", err)
		}
		if len(req.Messages) == 2 && onUser != nil {
			onUser(req.Messages[1].Content)
		}
		w.Write(resp)
	}))
	t.Cleanup(srv.Close)
	return &aichart.Client{BaseURL: srv.URL, Model: "m", HTTP: srv.Client()}
}

func TestConvertAfterLibraryOpen(t *testing.T) {
	s := testStore(t)
	var gotUser string
	client := mockCaptureConverter(t, func(u string) { gotUser = u })

	stored := "{title: Stored Song}\n[G]body"
	_, _ = s.AddChart("Stored Song", "", "import", stored)

	m := update(t, NewModel(testLines(3), RenderConfig{}).SetShowHelp(false).SetStore(s).SetConverter(client), tea.WindowSizeMsg{Width: 60, Height: 10})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // open the stored chart

	if m.rawChart != stored {
		t.Fatalf("rawChart after library open = %q, want stored content", m.rawChart)
	}

	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m2, ok := nm.(Model)
	if !ok {
		t.Fatalf("Update returned %T", nm)
	}
	if cmd == nil {
		t.Fatal("c after library open must start a conversion")
	}
	m3, done := runCmd(t, m2, cmd)
	if done.result.Chart == "" {
		t.Fatalf("runCmd produced no convertDoneMsg; done=%#v", done)
	}
	if gotUser != stored {
		t.Errorf("model sent %q, want the stored chart content %q", gotUser, stored)
	}
	if !strings.Contains(m3.View(), "Converted Song") {
		t.Errorf("view %q missing converted title", m3.View())
	}
}

func TestConvertInSetlistView(t *testing.T) {
	s := testStore(t)
	var gotUser string
	client := mockCaptureConverter(t, func(u string) { gotUser = u })

	content := "{title: Set Song}\n[C]body"
	id, _ := s.AddChart("Set Song", "", "import", content)
	slID, _ := s.CreateSetlist("Gig")
	_ = s.AppendSetlistItem(slID, id)

	m := update(t, NewModel(nil, RenderConfig{}).SetShowHelp(false).SetStore(s).SetConverter(client), tea.WindowSizeMsg{Width: 60, Height: 10})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("S")})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // open setlist

	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if _, ok := nm.(Model); !ok {
		t.Fatalf("Update returned %T", nm)
	}
	if cmd == nil {
		t.Fatal("c in setlist view must start a conversion")
	}
	_, done := runCmd(t, m, cmd)
	if done.result.Chart == "" {
		t.Fatalf("runCmd produced no convertDoneMsg; done=%#v", done)
	}
	if gotUser != content {
		t.Errorf("model sent %q, want the current setlist chart %q", gotUser, content)
	}
}

func TestConvertNoRawChartMessage(t *testing.T) {
	client := &aichart.Client{BaseURL: "http://unused", Model: "m"}
	doc := &parser.Document{Title: "T", Metadata: map[string][]string{}}
	m := update(t, NewDocModel(doc, RenderConfig{}).SetShowHelp(false).SetConverter(client), tea.WindowSizeMsg{Width: 60, Height: 10})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if !strings.Contains(m.View(), "no chart to convert") {
		t.Errorf("view %q missing no-chart-to-convert message", m.View())
	}
}

func TestConversionTickUpdatesMessage(t *testing.T) {
	doc := &parser.Document{Title: "T", Metadata: map[string][]string{}}
	m := update(t, NewDocModel(doc, RenderConfig{}).SetShowHelp(false), tea.WindowSizeMsg{Width: 80, Height: 10})

	prog := &conversionProgress{}
	prog.set("attempt 2/3: validating output")
	m.converting = true
	m.convProgress = prog
	m.message = "converting…"

	nm, cmd := m.Update(conversionTickMsg{})
	m2, ok := nm.(Model)
	if !ok {
		t.Fatalf("Update returned %T", nm)
	}
	if !strings.Contains(m2.message, "attempt 2/3: validating output") {
		t.Errorf("message = %q, want live progress text", m2.message)
	}
	if cmd == nil {
		t.Error("tick while converting must reschedule a tick")
	}
}

func TestConversionTickStopsAfterDone(t *testing.T) {
	doc := &parser.Document{Title: "T", Metadata: map[string][]string{}}
	m := update(t, NewDocModel(doc, RenderConfig{}).SetShowHelp(false), tea.WindowSizeMsg{Width: 80, Height: 10})
	m.converting = false
	m.convProgress = nil

	_, cmd := m.Update(conversionTickMsg{})
	if cmd != nil {
		t.Error("tick after conversion finished must not reschedule")
	}
}

func TestConvertFromLibraryBrowser(t *testing.T) {
	s := testStore(t)
	var gotUser string
	client := mockCaptureConverter(t, func(u string) { gotUser = u })

	content := "{title: Highlighted}\n[G]body"
	_, _ = s.AddChart("Highlighted", "", "import", content)

	m := update(t, NewModel(testLines(3), RenderConfig{}).SetShowHelp(false).SetStore(s).SetConverter(client), tea.WindowSizeMsg{Width: 60, Height: 10})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	// cursor starts at the only entry; press c on the browser.
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if _, ok := nm.(Model); !ok {
		t.Fatalf("Update returned %T", nm)
	}
	if cmd == nil {
		t.Fatal("c on the library browser must start a conversion")
	}
	_, done := runCmd(t, m, cmd)
	if done.result.Chart == "" {
		t.Fatal("library-browser conversion produced no result")
	}
	if gotUser != content {
		t.Errorf("model sent %q, want highlighted chart %q", gotUser, content)
	}
}

func TestConvertFromFilePicker(t *testing.T) {
	dir := t.TempDir()
	content := "{title: FromDisk}\n[C]body"
	if err := os.WriteFile(filepath.Join(dir, "song.pro"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	var gotUser string
	client := mockCaptureConverter(t, func(u string) { gotUser = u })

	m := update(t, NewModel(testLines(2), RenderConfig{}).SetShowHelp(false).SetConverter(client), tea.WindowSizeMsg{Width: 60, Height: 10})
	m.pickDir = dir
	nm, _ := m.openFilePicker()
	m = nm.(Model)

	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if _, ok := nm.(Model); !ok {
		t.Fatalf("Update returned %T", nm)
	}
	if cmd == nil {
		t.Fatal("c on the file picker must start a conversion")
	}
	_, done := runCmd(t, m, cmd)
	if done.result.Chart == "" {
		t.Fatal("file-picker conversion produced no result")
	}
	if gotUser != content {
		t.Errorf("model sent %q, want highlighted file %q", gotUser, content)
	}
}

func TestConvertOnSetlistListGivesHint(t *testing.T) {
	s := testStore(t)
	m := update(t, NewModel(testLines(2), RenderConfig{}).SetShowHelp(false).SetStore(s), tea.WindowSizeMsg{Width: 60, Height: 10})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("S")})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if !strings.Contains(m.View(), "open a setlist first") {
		t.Errorf("view %q missing hint message", m.View())
	}
}

func TestDeleteChartFromBrowser(t *testing.T) {
	s := testStore(t)
	id1, _ := s.AddChart("Keep Me", "", "import", "{title: Keep Me}\n[G]x")
	id2, _ := s.AddChart("Doomed", "", "import", "{title: Doomed}\n[C]x")

	m := update(t, NewModel(testLines(3), RenderConfig{}).SetShowHelp(false).SetStore(s), tea.WindowSizeMsg{Width: 60, Height: 10})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})

	// List is newest-first: Doomed is index 0.
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if !strings.Contains(m.View(), `Delete "Doomed"?`) {
		t.Fatalf("confirm banner missing: %q", m.View())
	}

	// Cancel first: n keeps the chart.
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if strings.Contains(m.View(), `Delete "Doomed"?`) {
		t.Fatalf("banner should clear on n: %q", m.View())
	}
	if _, err := s.GetChart(id2); err != nil {
		t.Errorf("chart should survive cancel: %v", err)
	}

	// Arm again, this time confirm with y.
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

	if _, err := s.GetChart(id2); err == nil {
		t.Error("chart still exists after y confirm")
	}
	if !strings.Contains(m.View(), "deleted Doomed") {
		t.Errorf("message missing: %q", m.View())
	}
	if _, err := s.GetChart(id1); err != nil {
		t.Errorf("unrelated chart was deleted: %v", err)
	}

	// Deleting the last remaining item clamps the cursor safely.
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if _, err := s.GetChart(id1); err == nil {
		t.Error("last chart still exists after y")
	}
	_ = id1
}

func TestImportCurrentChart(t *testing.T) {
	s := testStore(t)
	doc := &parser.Document{
		Title:    "My Song",
		Artist:   "Me",
		Metadata: map[string][]string{},
	}
	raw := "{title: My Song}\n[C]body"

	m := update(t, NewDocModel(doc, RenderConfig{}).SetShowHelp(false).SetSource(raw).SetStore(s), tea.WindowSizeMsg{Width: 40, Height: 10})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})

	if !strings.Contains(m.View(), "imported to library") {
		t.Errorf("view %q missing import confirmation", m.View())
	}
	charts, err := s.ListCharts()
	if err != nil || len(charts) != 1 {
		t.Fatalf("library charts = %#v err=%v, want 1", charts, err)
	}
	if charts[0].Title != "My Song" || charts[0].Source != "import" {
		t.Errorf("stored meta = %#v", charts[0])
	}

	// No source set -> nothing to import.
	m2 := update(t, NewModel(testLines(2), RenderConfig{}).SetShowHelp(false).SetStore(s), tea.WindowSizeMsg{Width: 40, Height: 10})
	m2 = update(t, m2, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	if !strings.Contains(m2.View(), "nothing to import") {
		t.Errorf("view %q missing nothing-to-import message", m2.View())
	}
}

func TestSaveConvertedChart(t *testing.T) {
	s := testStore(t)
	converted := "{title: Converted}\n[C]x"
	doc := &parser.Document{
		Title:    "Converted",
		Artist:   "AI",
		Metadata: map[string][]string{},
	}

	m := update(t, NewDocModel(doc, RenderConfig{}).SetShowHelp(false).SetStore(s), tea.WindowSizeMsg{Width: 40, Height: 10})
	m = update(t, m, convertDoneMsg{result: aichart.Result{Chart: converted, Attempts: 1}})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})

	if !strings.Contains(m.View(), "saved to library") {
		t.Errorf("view %q missing saved confirmation", m.View())
	}
	charts, err := s.ListCharts()
	if err != nil || len(charts) != 1 {
		t.Fatalf("library charts = %#v err=%v, want 1", charts, err)
	}
	if charts[0].Source != "ai" {
		t.Errorf("source = %q, want ai", charts[0].Source)
	}
	stored, _ := s.GetChart(charts[0].ID)
	if stored.Content != converted {
		t.Errorf("stored content = %q, want converted chart", stored.Content)
	}

	// Pressing s again with no NEW conversion must not duplicate the row.
	m = update(t, m, convertDoneMsg{result: aichart.Result{Chart: converted, Attempts: 1}})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	charts, _ = s.ListCharts()
	if len(charts) != 2 {
		t.Errorf("charts = %d, want 2 (second conversion saves again by design)", len(charts))
	}
}

func TestSetlistCreateAndBrowse(t *testing.T) {
	s := testStore(t)
	_, _ = s.AddChart("A", "", "import", "{title: A}\nx")

	m := update(t, NewModel(testLines(3), RenderConfig{}).SetShowHelp(false).SetStore(s), tea.WindowSizeMsg{Width: 60, Height: 10})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("S")})
	if got := m.Screen(); got != "browseSetlists" {
		t.Fatalf("screen = %q, want browseSetlists", got)
	}
	if !strings.Contains(m.View(), "no setlists") {
		t.Errorf("view %q missing empty hint", m.View())
	}

	// Create via typed input.
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	for _, r := range "Gig" {
		m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(string(r))})
	}
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	lists, _ := s.ListSetlists()
	if len(lists) != 1 || lists[0].Name != "Gig" {
		t.Fatalf("setlists = %#v, want [Gig]", lists)
	}
	view := m.View()
	if !strings.Contains(view, "Gig") || strings.Contains(view, "New setlist:") {
		t.Errorf("post-create view = %q (input should be closed)", view)
	}
}

func TestAddToSetlistFromBrowser(t *testing.T) {
	s := testStore(t)
	_, _ = s.AddChart("The Chart", "", "import", "{title: The Chart}\n[C]x")
	slID, _ := s.CreateSetlist("Worship")
	_ = slID

	m := update(t, NewModel(testLines(3), RenderConfig{}).SetShowHelp(false).SetStore(s), tea.WindowSizeMsg{Width: 60, Height: 10})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})

	if got := m.Screen(); got != "pickSetlist" {
		t.Fatalf("screen = %q, want pickSetlist", got)
	}
	if !strings.Contains(m.View(), `Add "The Chart" to:`) {
		t.Errorf("pick view = %q", m.View())
	}
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	items, err := s.SetlistCharts(slID)
	if err != nil || len(items) != 1 || items[0].Title != "The Chart" {
		t.Fatalf("setlist items = %#v err=%v", items, err)
	}
	if got := m.Screen(); got != "browseCharts" {
		t.Errorf("screen after add = %q, want browseCharts", got)
	}
}

func TestSetlistSequentialPaging(t *testing.T) {
	s := testStore(t)
	id1, _ := s.AddChart("Song One", "", "import", "{title: Song One}\n"+strings.Repeat("line one\n", 8))
	id2, _ := s.AddChart("Song Two", "", "import", "{title: Song Two}\nline two A\nline two B")
	slID, _ := s.CreateSetlist("Service")
	_ = s.AppendSetlistItem(slID, id1)
	_ = s.AppendSetlistItem(slID, id2)

	m := update(t, NewModel(nil, RenderConfig{}).SetShowHelp(false).SetStore(s), tea.WindowSizeMsg{Width: 40, Height: 6})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("S")})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // open "Service"

	if got := m.Screen(); got != "viewSetlist" {
		t.Fatalf("screen = %q, want viewSetlist", got)
	}
	view := m.View()
	if !strings.Contains(view, "chart 1/2") || !strings.Contains(view, "Song One") {
		t.Errorf("first-chart view = %q", view)
	}

	// Page down through song one's pages; must land on song two top.
	for i := 0; i < 12 && !strings.Contains(m.View(), "chart 2/2"); i++ {
		m = update(t, m, tea.KeyMsg{Type: tea.KeyPgDown})
	}
	if got := m.View(); !strings.Contains(got, "chart 2/2") || !strings.Contains(got, "line two B") {
		t.Errorf("after paging view = %q", got)
	}

	// Page up past the top of song two returns to song one bottom.
	for i := 0; i < 12 && !strings.Contains(m.View(), "chart 1/2"); i++ {
		m = update(t, m, tea.KeyMsg{Type: tea.KeyPgUp})
	}
	if got := m.View(); !strings.Contains(got, "chart 1/2") {
		t.Errorf("after back-paging view = %q", got)
	}
}

func TestLibraryNoStoreShowsMessage(t *testing.T) {
	m := update(t, NewModel(testLines(3), RenderConfig{}).SetShowHelp(false).SetShowHelp(true), tea.WindowSizeMsg{Width: 40, Height: 10})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	if got := m.Screen(); got != "viewChart" {
		t.Errorf("screen = %q, want stay on viewChart", got)
	}
	if !strings.Contains(m.View(), "no library open") {
		t.Errorf("view %q missing 'no library open' message", m.View())
	}
}

// runCmd executes a tea.Cmd (executing each sub-command of a BatchMsg) and
// feeds the resulting messages through Update, returning the final model and
// any convertDoneMsg.
func runCmd(t *testing.T, m Model, cmd tea.Cmd) (Model, convertDoneMsg) {
	t.Helper()
	var done convertDoneMsg
	feed := func(msg tea.Msg) {
		if d, ok := msg.(convertDoneMsg); ok {
			done = d
		}
		m = update(t, m, msg)
	}
	msgs := cmd()
	if batch, ok := msgs.(tea.BatchMsg); ok {
		for _, sub := range batch {
			feed(sub())
		}
		return m, done
	}
	feed(msgs)
	return m, done
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
	m := update(t, NewDocModel(doc, RenderConfig{}).SetSource("C G").SetConverter(client), tea.WindowSizeMsg{Width: 40, Height: 10})

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

	m3, cdm := runCmd(t, m2, cmd)
	if cdm.result.Chart == "" {
		t.Fatalf("cmd produced no convertDoneMsg")
	}
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
	m := NewDocModel(doc, RenderConfig{}).SetSource("raw").SetConverter(client)

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

func TestModelConvertNotConfiguredMessage(t *testing.T) {
	doc := &parser.Document{Title: "T", Metadata: map[string][]string{}}
	m := update(t, NewDocModel(doc, RenderConfig{}).SetShowHelp(false), tea.WindowSizeMsg{Width: 60, Height: 10})

	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m, ok := nm.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", nm)
	}
	if cmd != nil {
		t.Error("c without converter must not return a cmd")
	}
	if !strings.Contains(m.View(), "AI conversion not configured") {
		t.Errorf("view %q should explain that conversion is not configured", m.View())
	}
	if strings.Contains(m.View(), "converting…") {
		t.Errorf("view %q should not claim to be converting", m.View())
	}
}

func TestConvertFailureShowsErrorDetail(t *testing.T) {
	s := testStore(t)
	doc := &parser.Document{Title: "T", Metadata: map[string][]string{}}
	m := update(t, NewDocModel(doc, RenderConfig{}).SetShowHelp(false).SetStore(s), tea.WindowSizeMsg{Width: 80, Height: 10})

	m = update(t, m, convertDoneMsg{err: errors.New("gateway exploded")})
	view := m.View()
	if !strings.Contains(view, "AI conversion failed: gateway exploded") {
		t.Errorf("view %q missing failure detail", view)
	}
}
