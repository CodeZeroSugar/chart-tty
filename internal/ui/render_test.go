package ui

import (
	"reflect"
	"testing"

	"github.com/CodeZeroSugar/chart-tty/internal/parser"
)

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

	cfg := RenderConfig{HeaderStyle: "\x1b[1m", CommentStyle: "\x1b[3m"}
	want := []string{"\x1b[1m[chorus]\x1b[0m", "\x1b[3mSolo\x1b[0m"}

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