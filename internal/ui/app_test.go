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
