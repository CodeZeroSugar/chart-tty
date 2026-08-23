package parser

import (
	"reflect"
	"strings"
	"testing"
)

func TestIsChordLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"single chord", "C", true},
		{"multiple chords", "C G Am7 D7", true},
		{"indented chords", "  C  G  ", true},
		{"slash chord", "C/G", true},
		{"slash-number chord", "G6/9", true},
		{"escaped annotation", "*N.C.", true},
		{"all single letters", "A B C D", true},
		{"empty string", "", false},
		{"whitespace only", "   ", false},
		{"section header", "Verse 1:", false},
		{"plain lyric", "Swing low", false},
		{"no-chord marker", "N.C.", false},
		{"tab-like line", "E|--0--|", false},
		{"mixed garbage", "C foo G", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isChordLine(tt.line); got != tt.want {
				t.Errorf("isChordLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestExtractBasicChords(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []ChordToken
	}{
		{"single chord", "C", []ChordToken{{Name: "C", Position: 0}}},
		{"multiple chords", "C  G", []ChordToken{{Name: "C", Position: 0}, {Name: "G", Position: 3}}},
		{"indented", "   Am7", []ChordToken{{Name: "Am7", Position: 3}}},
		{"irregular spacing", "C G   D7", []ChordToken{{Name: "C", Position: 0}, {Name: "G", Position: 2}, {Name: "D7", Position: 6}}},
		{"empty", "", nil},
		{"whitespace only", "   ", nil},
		{"invalid tokens skipped", "foo C", []ChordToken{{Name: "C", Position: 4}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBasicChords(tt.line)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractBasicChords(%q) = %#v, want %#v", tt.line, got, tt.want)
			}
		})
	}
}

func TestParseBasicPairs(t *testing.T) {
	tests := []struct {
		name  string
		chart string
		want  []Section
	}{
		{
			name:  "single chord lyric pair",
			chart: "C         G\nSwing low, sweet chariot",
			want: []Section{
				{
					Name: "",
					Lines: []ParsedLine{
						{
							Type:   LineTypeChordAndLyric,
							Raw:    "Swing low, sweet chariot",
							Lyrics: "Swing low, sweet chariot",
							Chords: []ChordToken{{Name: "C", Position: 0}, {Name: "G", Position: 10}},
						},
					},
				},
			},
		},
		{
			name:  "consecutive pairs",
			chart: "C  G\nSwing low\nAm7 D7\nComin' home",
			want: []Section{
				{
					Name: "",
					Lines: []ParsedLine{
						{
							Type:   LineTypeChordAndLyric,
							Raw:    "Swing low",
							Lyrics: "Swing low",
							Chords: []ChordToken{{Name: "C", Position: 0}, {Name: "G", Position: 3}},
						},
						{
							Type:   LineTypeChordAndLyric,
							Raw:    "Comin' home",
							Lyrics: "Comin' home",
							Chords: []ChordToken{{Name: "Am7", Position: 0}, {Name: "D7", Position: 4}},
						},
					},
				},
			},
		},
		{
			name:  "indented pair preserves alignment",
			chart: "     C      G\n     Swing low",
			want: []Section{
				{
					Name: "",
					Lines: []ParsedLine{
						{
							Type:   LineTypeChordAndLyric,
							Raw:    "     Swing low",
							Lyrics: "     Swing low",
							Chords: []ChordToken{{Name: "C", Position: 5}, {Name: "G", Position: 12}},
						},
					},
				},
			},
		},
		{
			name:  "stacked chord lines",
			chart: "C G\nAm7 D7\nlyric here",
			want: []Section{
				{
					Name: "",
					Lines: []ParsedLine{
						{
							Type:   LineTypeChord,
							Raw:    "C G",
							Lyrics: "",
							Chords: []ChordToken{{Name: "C", Position: 0}, {Name: "G", Position: 2}},
						},
						{
							Type:   LineTypeChordAndLyric,
							Raw:    "lyric here",
							Lyrics: "lyric here",
							Chords: []ChordToken{{Name: "Am7", Position: 0}, {Name: "D7", Position: 4}},
						},
					},
				},
			},
		},
		{
			name:  "dangling chord line at EOF",
			chart: "C G",
			want: []Section{
				{
					Name: "",
					Lines: []ParsedLine{
						{
							Type:   LineTypeChord,
							Raw:    "C G",
							Lyrics: "",
							Chords: []ChordToken{{Name: "C", Position: 0}, {Name: "G", Position: 2}},
						},
					},
				},
			},
		},
		{
			name:  "chord beyond lyric width keeps absolute column",
			chart: "C            G\nhome",
			want: []Section{
				{
					Name: "",
					Lines: []ParsedLine{
						{
							Type:   LineTypeChordAndLyric,
							Raw:    "home",
							Lyrics: "home",
							Chords: []ChordToken{{Name: "C", Position: 0}, {Name: "G", Position: 13}},
						},
					},
				},
			},
		},
		{
			name:  "lyric without chord line",
			chart: "Just a lyric line",
			want: []Section{
				{
					Name: "",
					Lines: []ParsedLine{
						{Type: LineTypeLyric, Raw: "Just a lyric line", Lyrics: "Just a lyric line"},
					},
				},
			},
		},
		{
			name:  "chord line followed by blank is chord-only",
			chart: "C G\n\nnext stanza",
			want: []Section{
				{
					Name: "",
					Lines: []ParsedLine{
						{
							Type:   LineTypeChord,
							Raw:    "C G",
							Lyrics: "",
							Chords: []ChordToken{{Name: "C", Position: 0}, {Name: "G", Position: 2}},
						},
					},
				},
				{
					Name: "",
					Lines: []ParsedLine{
						{Type: LineTypeLyric, Raw: "next stanza", Lyrics: "next stanza"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := NewParser(ModeBasic).Parse(tt.chart)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.chart, err)
			}
			if !reflect.DeepEqual(doc.Sections, tt.want) {
				t.Errorf("Sections = %#v, want %#v", doc.Sections, tt.want)
			}
		})
	}
}

func TestParseBasicSections(t *testing.T) {
	tests := []struct {
		name  string
		chart string
		want  int
	}{
		{"stanzas split by blank lines", "C G\nStanza one\n\nAm7\nStanza two", 2},
		{"single stanza", "C\nSwing low", 1},
		{"trailing stanza flushed", "C\nSwing low\n\nD7\nCarry me home", 2},
		{"blank-only chart", "   \n\n", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := NewParser(ModeBasic).Parse(tt.chart)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.chart, err)
			}
			if len(doc.Sections) != tt.want {
				t.Errorf("Parse(%q) Sections = %#v, want %d sections", tt.chart, doc.Sections, tt.want)
			}
		})
	}
}

func TestParseBasicCommentsAndTabs(t *testing.T) {
	chart := "# intro\nC G\nSwing low\n# outro\n\nE|--0--|--2--|\nB|--1--|--3--|"
	doc, err := NewParser(ModeBasic).Parse(chart)
	if err != nil {
		t.Fatalf("Parse(%q) unexpected error: %v", chart, err)
	}

	if len(doc.Sections) != 2 {
		t.Fatalf("Sections = %#v, want 2 sections", doc.Sections)
	}
	first := doc.Sections[0]
	if len(first.Lines) != 1 {
		t.Fatalf("first section Lines = %#v, want 1 line (comments skipped)", first.Lines)
	}
	if first.Lines[0].Type != LineTypeChordAndLyric {
		t.Errorf("first section line Type = %v, want ChordAndLyric", first.Lines[0].Type)
	}

	second := doc.Sections[1]
	if len(second.Lines) != 2 {
		t.Fatalf("second section Lines = %#v, want 2 tab lines", second.Lines)
	}
	for _, ln := range second.Lines {
		if ln.Type != LineTypeTab {
			t.Errorf("line Type = %v, want LineTypeTab", ln.Type)
		}
	}
}

func TestParseBasicNonChordTraps(t *testing.T) {
	tests := []struct {
		name  string
		chart string
	}{
		{"section header is lyric", "Verse 1:"},
		{"no-chord marker is lyric", "N.C."},
		{"garbage line is lyric", "foo bar baz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := NewParser(ModeBasic).Parse(tt.chart)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.chart, err)
			}
			if len(doc.Sections) != 1 || len(doc.Sections[0].Lines) != 1 {
				t.Fatalf("Sections = %#v, want 1 section with 1 line", doc.Sections)
			}
			got := doc.Sections[0].Lines[0]
			if got.Type != LineTypeLyric {
				t.Errorf("line Type = %v, want LineTypeLyric", got.Type)
			}
			if got.Lyrics != tt.chart {
				t.Errorf("Lyrics = %q, want %q", got.Lyrics, tt.chart)
			}
		})
	}
}

func TestParseBasicEmptyChart(t *testing.T) {
	doc, err := NewParser(ModeBasic).Parse("")
	if doc != nil {
		t.Errorf("Parse(\"\") doc = %#v, want nil", doc)
	}
	if err == nil {
		t.Fatal("Parse(\"\") expected error, got nil")
	}
	if !strings.Contains(err.Error(), "chart is empty") {
		t.Errorf("Parse(\"\") error %q does not contain %q", err.Error(), "chart is empty")
	}
}

func TestParseBasicCRLF(t *testing.T) {
	lf, err := NewParser(ModeBasic).Parse("C G\nSwing low")
	if err != nil {
		t.Fatalf("LF parse error: %v", err)
	}
	crlf, err := NewParser(ModeBasic).Parse("C G\r\nSwing low\r\n")
	if err != nil {
		t.Fatalf("CRLF parse error: %v", err)
	}
	if !reflect.DeepEqual(crlf.Sections, lf.Sections) {
		t.Errorf("CRLF Sections = %#v, want %#v", crlf.Sections, lf.Sections)
	}
}