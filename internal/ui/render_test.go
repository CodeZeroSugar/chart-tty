package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/CodeZeroSugar/chart-tty/internal/config"
	"github.com/CodeZeroSugar/chart-tty/internal/parser"
)

func TestResolveColor(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"cyan", "6"},
		{"yellow", "3"},
		{"magenta", "5"},
		{"red", "1"},
		{"gray", "8"},
		{"grey", "8"},
		{"brightblue", "12"},
		{"CYAN", "6"}, // case-insensitive
		{"#ff8800", "#ff8800"},
		{"196", "196"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := resolveColor(tt.in); got != tt.want {
			t.Errorf("resolveColor(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRenderMixedSection(t *testing.T) {
	doc := &parser.Document{
		Title:    "Swing Low",
		Metadata: map[string][]string{},
		Sections: []parser.Section{
			{
				Name: "chorus",
				Lines: []parser.ParsedLine{
					{Type: parser.LineTypeChordAndLyric, Raw: "Swing [D]low", Lyrics: "Swing low", Chords: []parser.ChordToken{{Name: "D", Position: 6}}},
					{Type: parser.LineTypeLyric, Raw: "Sweet chariot", Lyrics: "Sweet chariot"},
					{Type: parser.LineTypeComment, Raw: "{comment: Solo}", Lyrics: "Solo"},
					{Type: parser.LineTypeTab, Raw: "E|--0--|", Lyrics: ""},
				},
			},
			{
				Name: "",
				Lines: []parser.ParsedLine{
					{Type: parser.LineTypeChord, Raw: "[A] [F#m] [E]", Chords: []parser.ChordToken{{Name: "A", Position: 0}, {Name: "F#m", Position: 1}, {Name: "E", Position: 2}}},
				},
			},
		},
	}

	want := []string{
		"Swing Low",
		"",
		"[chorus]",
		"      D  ",
		"Swing low",
		"Sweet chariot",
		"Solo",
		"E|--0--|",
		"",
		"A F#m E",
	}

	got := Render(doc, RenderConfig{})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Render() = %#v, want %#v", got, want)
	}
}

func TestRenderChordBeyondLyricWidth(t *testing.T) {
	doc := &parser.Document{
		Metadata: map[string][]string{},
		Sections: []parser.Section{
			{
				Name: "",
				Lines: []parser.ParsedLine{
					{Type: parser.LineTypeChordAndLyric, Raw: "home", Lyrics: "home", Chords: []parser.ChordToken{{Name: "C", Position: 0}, {Name: "G", Position: 13}}},
				},
			},
		},
	}

	want := []string{
		"C            G",
		"home",
	}

	got := Render(doc, RenderConfig{})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Render() = %#v, want %#v", got, want)
	}
}

func TestRenderNamedSectionsGetHeader(t *testing.T) {
	doc := &parser.Document{
		Metadata: map[string][]string{},
		Sections: []parser.Section{
			{Name: "Verse 1", Lines: []parser.ParsedLine{{Type: parser.LineTypeLyric, Raw: "line", Lyrics: "line"}}},
			{Name: "", Lines: []parser.ParsedLine{{Type: parser.LineTypeLyric, Raw: "bare", Lyrics: "bare"}}},
		},
	}

	want := []string{
		"[Verse 1]",
		"line",
		"",
		"bare",
	}

	got := Render(doc, RenderConfig{})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Render() = %#v, want %#v", got, want)
	}
}

func TestRenderStyles(t *testing.T) {
	doc := &parser.Document{
		Metadata: map[string][]string{},
		Sections: []parser.Section{
			{
				Name: "chorus",
				Lines: []parser.ParsedLine{
					{Type: parser.LineTypeComment, Raw: "{comment: Solo}", Lyrics: "Solo"},
				},
			},
		},
	}

	cfg := RenderConfig{
		HeaderStyle:  lipgloss.NewStyle().Bold(true),
		CommentStyle: lipgloss.NewStyle().Italic(true),
	}
	want := []string{
		lipgloss.NewStyle().Bold(true).Render("[chorus]"),
		lipgloss.NewStyle().Italic(true).Render("Solo"),
	}

	got := Render(doc, cfg)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Render() = %#v, want %#v", got, want)
	}
}

func TestRenderEmptyDocument(t *testing.T) {
	doc := &parser.Document{Metadata: map[string][]string{}}
	if got := Render(doc, RenderConfig{}); len(got) != 0 {
		t.Errorf("Render() = %#v, want empty", got)
	}
}

func TestRenderMetaBlock(t *testing.T) {
	doc := &parser.Document{
		Title:    "Long Road Home",
		Artist:   "Chart TTY",
		Key:      "G",
		Tempo:    "112",
		Capo:     "2",
		Metadata: map[string][]string{"time": {"4/4"}},
		Sections: []parser.Section{{
			Name:  "verse",
			Lines: []parser.ParsedLine{{Type: parser.LineTypeLyric, Raw: "Home", Lyrics: "Home"}},
		}},
	}

	got := Render(doc, RenderConfig{})
	want := []string{
		"Long Road Home",
		"Chart TTY",
		"Capo: 2 · Key: G · Tempo: 112 · Time: 4/4",
		"",
		"[verse]",
		"Home",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Render() = %#v, want %#v", got, want)
	}
}

func TestRenderMetaBlockOmittedWhenEmpty(t *testing.T) {
	doc := &parser.Document{
		Metadata: map[string][]string{},
		Sections: []parser.Section{{
			Name:  "verse",
			Lines: []parser.ParsedLine{{Type: parser.LineTypeLyric, Raw: "x", Lyrics: "x"}},
		}},
	}
	got := Render(doc, RenderConfig{})
	want := []string{"[verse]", "x"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Render() = %#v, want %#v", got, want)
	}
}

func TestMetaLinePresentOnly(t *testing.T) {
	tests := []struct {
		name string
		doc  *parser.Document
		want string
	}{
		{"all present", &parser.Document{Capo: "2", Key: "G", Tempo: "112", Metadata: map[string][]string{"time": {"4/4"}, "duration": {"3:00"}}},
			"Capo: 2 · Key: G · Tempo: 112 · Time: 4/4 · Duration: 3:00"},
		{"only capo", &parser.Document{Capo: "5", Metadata: map[string][]string{}}, "Capo: 5"},
		{"none", &parser.Document{Metadata: map[string][]string{}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := metaLine(tt.doc, false); got != tt.want {
				t.Errorf("metaLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMetaLineSkipKey(t *testing.T) {
	doc := &parser.Document{Capo: "2", Key: "G", Tempo: "112", Metadata: map[string][]string{"time": {"4/4"}}}
	if got := metaLine(doc, true); got != "Capo: 2 · Tempo: 112 · Time: 4/4" {
		t.Errorf("metaLine(skipKey) = %q, want %q", got, "Capo: 2 · Tempo: 112 · Time: 4/4")
	}
}

func TestRenderMetaBlockSuppressesTitleAndKey(t *testing.T) {
	doc := &parser.Document{
		Title:    "Long Road Home",
		Artist:   "Chart TTY",
		Key:      "G",
		Tempo:    "112",
		Capo:     "2",
		Metadata: map[string][]string{},
	}
	cfg := RenderConfig{SuppressMetaTitleKey: true}
	block := renderMetaBlock(doc, cfg)
	want := []string{"Chart TTY", "Capo: 2 · Tempo: 112"}
	if !reflect.DeepEqual(block, want) {
		t.Errorf("renderMetaBlock() = %#v, want %#v", block, want)
	}

	// The title and key remain in the full piped Render output.
	cfg = RenderConfig{}
	doc.Sections = []parser.Section{{
		Name:  "verse",
		Lines: []parser.ParsedLine{{Type: parser.LineTypeLyric, Raw: "Home", Lyrics: "Home"}},
	}}
	got := Render(doc, RenderConfig{})
	if got[0] != "Long Road Home" || !strings.Contains(got[2], "Key: G") {
		t.Errorf("piped Render lost title/key: %#v", got)
	}
}

func TestRenderKeepFlags(t *testing.T) {
	doc := &parser.Document{
		Metadata: map[string][]string{},
		Sections: []parser.Section{{
			Name: "verse",
			Lines: []parser.ParsedLine{
				{Type: parser.LineTypeChordAndLyric, Raw: "Swing [D]low", Lyrics: "Swing low", Chords: []parser.ChordToken{{Name: "D", Position: 6}}},
				{Type: parser.LineTypeChord, Raw: "[A] [F#m]", Chords: []parser.ChordToken{{Name: "A", Position: 0}, {Name: "F#m", Position: 2}}},
				{Type: parser.LineTypeLyric, Raw: "Sweet chariot", Lyrics: "Sweet chariot"},
				{Type: parser.LineTypeTab, Raw: "E|--0--|"},
				{Type: parser.LineTypeComment, Raw: "{c: hi}", Lyrics: "hi"},
			},
		}},
	}
	lines, keep, keepTab, keepPage := renderLines(doc, RenderConfig{})
	// Rows: [verse], chordRow, lyric, chordNames, lyric, tab, comment.
	wantKeep := []bool{true, true, false, false, false, false, false}
	wantKeepPage := []bool{true, true, true, true, true, true, false}
	if !reflect.DeepEqual(keep, wantKeep) {
		t.Errorf("keep = %#v, want %#v", keep, wantKeep)
	}
	if !reflect.DeepEqual(keepTab, make([]bool, len(keep))) {
		t.Errorf("keepTab = %#v, want all false", keepTab)
	}
	if !reflect.DeepEqual(keepPage, wantKeepPage) {
		t.Errorf("keepPage = %#v, want %#v", keepPage, wantKeepPage)
	}
	_ = lines
}

func TestRenderTabBlockChaining(t *testing.T) {
	doc := &parser.Document{
		Metadata: map[string][]string{},
		Sections: []parser.Section{{
			Name: "tab",
			Lines: []parser.ParsedLine{
				{Type: parser.LineTypeTab, Raw: "E|--0--|"},
				{Type: parser.LineTypeTab, Raw: "B|--1--|"},
				{Type: parser.LineTypeTab, Raw: "G|--2--|"},
				{Type: parser.LineTypeLyric, Raw: "song", Lyrics: "song"},
			},
		}},
	}
	_, keep, keepTab, keepPage := renderLines(doc, RenderConfig{})
	// Rows: [tab], E|, B|, G|, song. The three tab rows chain together in
	// keepTab; keepPage chains the whole named section.
	wantKeep := []bool{true, false, false, false, false}
	wantKeepTab := []bool{false, true, true, false, false}
	wantKeepPage := []bool{true, true, true, true, false}
	if !reflect.DeepEqual(keep, wantKeep) {
		t.Errorf("keep = %#v, want %#v", keep, wantKeep)
	}
	if !reflect.DeepEqual(keepTab, wantKeepTab) {
		t.Errorf("keepTab = %#v, want %#v", keepTab, wantKeepTab)
	}
	if !reflect.DeepEqual(keepPage, wantKeepPage) {
		t.Errorf("keepPage = %#v, want %#v", keepPage, wantKeepPage)
	}
}

func TestRenderConfigFromConfig(t *testing.T) {
	cfg := config.Default()
	cfg.Theme.HeaderColor = "magenta"
	cfg.Theme.CommentColor = "green"
	cfg.Theme.HighlightColor = "red"
	rcfg := RenderConfigFromConfig(cfg)

	doc := &parser.Document{
		Metadata: map[string][]string{},
		Sections: []parser.Section{
			{
				Name: "verse",
				Lines: []parser.ParsedLine{
					{Type: parser.LineTypeComment, Raw: "{c: hi}", Lyrics: "hi"},
				},
			},
		},
	}
	got := Render(doc, rcfg)
	want := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("magenta")).Render("[verse]"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("green")).Render("hi"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Render() = %#v, want %#v", got, want)
	}

	// The chrome styles derive from the theme too.
	wantBanner := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("magenta")).Render("CHORD")
	if got := rcfg.BannerStyle.Render("CHORD"); got != wantBanner {
		t.Errorf("BannerStyle.Render() = %q, want %q", got, wantBanner)
	}
	wantHighlight := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("red")).Render("> Library")
	if got := rcfg.HighlightStyle.Render("> Library"); got != wantHighlight {
		t.Errorf("HighlightStyle.Render() = %q, want %q", got, wantHighlight)
	}
}

func TestThemeColorsEmitANSI(t *testing.T) {
	// Named colors must resolve to real ANSI color codes, not nil. Force a
	// color-capable profile so lipgloss actually emits SGR color sequences.
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	rcfg := RenderConfigFromConfig(config.Default()) // header=cyan, comment=yellow, highlight=yellow
	wantHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("[chorus]")
	if got := rcfg.HeaderStyle.Render("[chorus]"); got != wantHeader {
		t.Errorf("HeaderStyle.Render() = %q, want %q", got, wantHeader)
	}
	if !strings.Contains(rcfg.HeaderStyle.Render("[chorus]"), "\x1b[") {
		t.Errorf("HeaderStyle emits no ANSI color: %q", rcfg.HeaderStyle.Render("[chorus]"))
	}
	if !strings.Contains(rcfg.CommentStyle.Render("hi"), "\x1b[") {
		t.Errorf("CommentStyle emits no ANSI color: %q", rcfg.CommentStyle.Render("hi"))
	}
	if !strings.Contains(rcfg.HighlightStyle.Render("> x"), "\x1b[") {
		t.Errorf("HighlightStyle emits no ANSI color: %q", rcfg.HighlightStyle.Render("> x"))
	}

	// End to end: a rendered section header carries a color code.
	doc := &parser.Document{
		Metadata: map[string][]string{},
		Sections: []parser.Section{{
			Name:  "chorus",
			Lines: []parser.ParsedLine{{Type: parser.LineTypeLyric, Raw: "x", Lyrics: "x"}},
		}},
	}
	rendered := Render(doc, rcfg)
	if len(rendered) == 0 || !strings.Contains(rendered[0], "\x1b[") {
		t.Errorf("Render() section header has no ANSI color: %#v", rendered)
	}
}

func TestRenderBasicModeEndToEnd(t *testing.T) {
	chart := "C  G\nSwing low\n\nAm7\nCarry me home"
	doc, err := parser.NewParser(parser.ModeBasic).Parse(chart)
	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	want := []string{
		"C  G     ",
		"Swing low",
		"",
		"Am7          ",
		"Carry me home",
	}

	got := Render(doc, RenderConfig{})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Render() = %#v, want %#v", got, want)
	}
}
