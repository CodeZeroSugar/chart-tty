package parser

import (
	"reflect"
	"strings"
	"testing"
)

func parseChart(t *testing.T, chart string) *Document {
	t.Helper()
	doc, err := NewParser(ModeChordPro).Parse(chart)
	if err != nil {
		t.Fatalf("Parse(%q) unexpected error: %v", chart, err)
	}
	if doc == nil {
		t.Fatalf("Parse(%q) returned nil document", chart)
	}
	return doc
}

func TestNewParser(t *testing.T) {
	if got := NewParser(ModeChordPro).Mode; got != ModeChordPro {
		t.Errorf("NewParser(ModeChordPro).Mode = %v, want %v", got, ModeChordPro)
	}
	if got := NewParser(ModeBasic).Mode; got != ModeBasic {
		t.Errorf("NewParser(ModeBasic).Mode = %v, want %v", got, ModeBasic)
	}
}

func TestParseMetaDirectives(t *testing.T) {
	chart := `{title: Swing Low}
{t:Take Me Home}
{artist: Green Day}
{key: C}
{tempo: 120}
{capo: 2}
{composer: John}
{composer: Jane}
{album: Greatest Hits}
{year:}`
	doc := parseChart(t, chart)

	if doc.Title != "Take Me Home" {
		t.Errorf("Title = %q, want %q (t overwrites title)", doc.Title, "Take Me Home")
	}
	if doc.Artist != "Green Day" {
		t.Errorf("Artist = %q, want %q", doc.Artist, "Green Day")
	}
	if doc.Key != "C" {
		t.Errorf("Key = %q, want %q", doc.Key, "C")
	}
	if doc.Tempo != "120" {
		t.Errorf("Tempo = %q, want %q", doc.Tempo, "120")
	}
	if doc.Capo != "2" {
		t.Errorf("Capo = %q, want %q", doc.Capo, "2")
	}

	if got := doc.Metadata["composer"]; !reflect.DeepEqual(got, []string{"John", "Jane"}) {
		t.Errorf("Metadata[composer] = %#v, want [John Jane] (repeated directives append)", got)
	}
	if got := doc.Metadata["album"]; !reflect.DeepEqual(got, []string{"Greatest Hits"}) {
		t.Errorf("Metadata[album] = %#v, want [Greatest Hits]", got)
	}
	if _, exists := doc.Metadata["year"]; exists {
		t.Error("Metadata[year] present, want absent for empty value")
	}
	if len(doc.Sections) != 0 {
		t.Errorf("Sections = %#v, want empty", doc.Sections)
	}
}

func TestParseSubtitleMapsToArtist(t *testing.T) {
	chart := `{subtitle: John Denver}`
	doc := parseChart(t, chart)
	if doc.Artist != "John Denver" {
		t.Errorf("Artist = %q, want %q (subtitle maps to artist)", doc.Artist, "John Denver")
	}

	chart = `{st: The Beatles}`
	doc = parseChart(t, chart)
	if doc.Artist != "The Beatles" {
		t.Errorf("Artist = %q, want %q (st maps to artist)", doc.Artist, "The Beatles")
	}
}

func TestParseMetaCaseInsensitive(t *testing.T) {
	chart := `{Title: A Song}
{KEY: D}`
	doc := parseChart(t, chart)

	if doc.Title != "A Song" {
		t.Errorf("Title = %q, want %q (meta parsing is case-insensitive)", doc.Title, "A Song")
	}
	if doc.Key != "D" {
		t.Errorf("Key = %q, want %q", doc.Key, "D")
	}
	if len(doc.Metadata) != 0 {
		t.Errorf("Metadata = %#v, want empty", doc.Metadata)
	}
}

func TestParseUppercaseEnvironmentDirectives(t *testing.T) {
	chart := `{SOC}
Swing [D]low
{EOC}`
	doc := parseChart(t, chart)

	want := []Section{
		{
			Name: "chorus",
			Lines: []ParsedLine{
				{Type: LineTypeChordAndLyric, Raw: "Swing [D]low", Lyrics: "Swing low", Chords: []ChordToken{{Name: "D", Position: 6}}},
			},
		},
	}
	if !reflect.DeepEqual(doc.Sections, want) {
		t.Errorf("Sections = %#v, want %#v", doc.Sections, want)
	}
}

func TestParseSections(t *testing.T) {
	tests := []struct {
		name     string
		chart    string
		wantSecs []Section
	}{
		{
			name:  "chorus environment",
			chart: "{soc}\nSwing [D]low\n{eoc}",
			wantSecs: []Section{
				{
					Name: "chorus",
					Lines: []ParsedLine{
						{Type: LineTypeChordAndLyric, Raw: "Swing [D]low", Lyrics: "Swing low", Chords: []ChordToken{{Name: "D", Position: 6}}},
					},
				},
			},
		},
		{
			name:  "start_of_verse full name",
			chart: "{start_of_verse}\nA lyric line\n{end_of_verse}",
			wantSecs: []Section{
				{
					Name: "verse",
					Lines: []ParsedLine{
						{Type: LineTypeLyric, Raw: "A lyric line", Lyrics: "A lyric line"},
					},
				},
			},
		},
		{
			name:  "soc with label",
			chart: "{soc: Chorus 1}\nSwing low\n{eoc}",
			wantSecs: []Section{
				{
					Name: "Chorus 1",
					Lines: []ParsedLine{
						{Type: LineTypeLyric, Raw: "Swing low", Lyrics: "Swing low"},
					},
				},
			},
		},
		{
			name:  "tab block becomes environment section",
			chart: "{sot}\nE|--0--|--2--|\nB|--2--|--2--|\nG|--2--|--1--|\nD|--2--|--3--|\nA|--0--|--3--|\nE|--X--|--X--|\n{eot}",
			wantSecs: []Section{
				{
					Name: "tab",
					Lines: []ParsedLine{
						{Type: LineTypeTab, Raw: "E|--0--|--2--|"},
						{Type: LineTypeTab, Raw: "B|--2--|--2--|"},
						{Type: LineTypeTab, Raw: "G|--2--|--1--|"},
						{Type: LineTypeTab, Raw: "D|--2--|--3--|"},
						{Type: LineTypeTab, Raw: "A|--0--|--3--|"},
						{Type: LineTypeTab, Raw: "E|--X--|--X--|"},
					},
				},
			},
		},
		{
			name:     "decorative sot block dropped",
			chart:    "{sot}\n------\n{eot}",
			wantSecs: []Section{},
		},
		{
			name:     "empty environment produces no section",
			chart:    "{soc}\n{eoc}",
			wantSecs: []Section{},
		},
		{
			name:  "eoc flushes active chorus despite mismatched end",
			chart: "{soc}\nSwing low\n{eot}",
			wantSecs: []Section{
				{
					Name: "chorus",
					Lines: []ParsedLine{
						{Type: LineTypeLyric, Raw: "Swing low", Lyrics: "Swing low"},
					},
				},
			},
		},
		{
			name:  "chorus shorthand opens environment",
			chart: "{chorus}\nSwing low\n{eoc}",
			wantSecs: []Section{
				{
					Name: "chorus",
					Lines: []ParsedLine{
						{Type: LineTypeLyric, Raw: "Swing low", Lyrics: "Swing low"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseChart(t, tt.chart)
			if !reflect.DeepEqual(doc.Sections, tt.wantSecs) {
				t.Errorf("Sections = %#v, want %#v", doc.Sections, tt.wantSecs)
			}
		})
	}
}

func TestParseTabBlockGate(t *testing.T) {
	sixStrings := "E|--0--|\nB|--2--|\nG|--2--|\nD|--2--|\nA|--0--|\nE|--X--|"

	tests := []struct {
		name     string
		chart    string
		wantSecs []Section
	}{
		{
			name:  "four string lines is the boundary",
			chart: "{sot}\nE|--0--|\nB|--2--|\nG|--2--|\nD|--2--|\n{eot}",
			wantSecs: []Section{
				{Name: "tab", Lines: []ParsedLine{
					{Type: LineTypeTab, Raw: "E|--0--|"},
					{Type: LineTypeTab, Raw: "B|--2--|"},
					{Type: LineTypeTab, Raw: "G|--2--|"},
					{Type: LineTypeTab, Raw: "D|--2--|"},
				}},
			},
		},
		{
			name:     "three string lines dropped",
			chart:    "{sot}\nE|--0--|\nB|--2--|\nG|--2--|\nsome label\n{eot}",
			wantSecs: []Section{},
		},
		{
			name:     "pipe mandatory",
			chart:    "{sot}\nE --0--\nB --2--\nG --2--\nD --2--\nA --0--\n{eot}",
			wantSecs: []Section{},
		},
		{
			name:  "lowercase letters count",
			chart: "{sot}\ne|--0--|\nb|--2--|\ng|--2--|\nd|--2--|\n{eot}",
			wantSecs: []Section{
				{Name: "tab", Lines: []ParsedLine{
					{Type: LineTypeTab, Raw: "e|--0--|"},
					{Type: LineTypeTab, Raw: "b|--2--|"},
					{Type: LineTypeTab, Raw: "g|--2--|"},
					{Type: LineTypeTab, Raw: "d|--2--|"},
				}},
			},
		},
		{
			name:  "any musical letter counts including F and C",
			chart: "{sot}\nF|--1--|\nC|--3--|\nB|--2--|\nE|--0--|\n{eot}",
			wantSecs: []Section{
				{Name: "tab", Lines: []ParsedLine{
					{Type: LineTypeTab, Raw: "F|--1--|"},
					{Type: LineTypeTab, Raw: "C|--3--|"},
					{Type: LineTypeTab, Raw: "B|--2--|"},
					{Type: LineTypeTab, Raw: "E|--0--|"},
				}},
			},
		},
		{
			name:  "qualifying block keeps non-string lines verbatim",
			chart: "{sot}\nCHORDS\n" + sixStrings + "\n{eot}",
			wantSecs: []Section{
				{Name: "tab", Lines: []ParsedLine{
					{Type: LineTypeTab, Raw: "CHORDS"},
					{Type: LineTypeTab, Raw: "E|--0--|"},
					{Type: LineTypeTab, Raw: "B|--2--|"},
					{Type: LineTypeTab, Raw: "G|--2--|"},
					{Type: LineTypeTab, Raw: "D|--2--|"},
					{Type: LineTypeTab, Raw: "A|--0--|"},
					{Type: LineTypeTab, Raw: "E|--X--|"},
				}},
			},
		},
		{
			name:  "unclosed qualifying block flushed at EOF",
			chart: "{sot}\n" + sixStrings,
			wantSecs: []Section{
				{Name: "tab", Lines: []ParsedLine{
					{Type: LineTypeTab, Raw: "E|--0--|"},
					{Type: LineTypeTab, Raw: "B|--2--|"},
					{Type: LineTypeTab, Raw: "G|--2--|"},
					{Type: LineTypeTab, Raw: "D|--2--|"},
					{Type: LineTypeTab, Raw: "A|--0--|"},
					{Type: LineTypeTab, Raw: "E|--X--|"},
				}},
			},
		},
		{
			name:     "unclosed decorative block dropped at EOF",
			chart:    "{sot}\n------\n......",
			wantSecs: []Section{},
		},
		{
			name:  "decorative block abandoned by new environment dropped",
			chart: "{sot}\n------\n{sov}\nA lyric line\n{eov}",
			wantSecs: []Section{
				{Name: "verse", Lines: []ParsedLine{
					{Type: LineTypeLyric, Raw: "A lyric line", Lyrics: "A lyric line"},
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseChart(t, tt.chart)
			if !reflect.DeepEqual(doc.Sections, tt.wantSecs) {
				t.Errorf("Sections = %#v, want %#v", doc.Sections, tt.wantSecs)
			}
		})
	}
}

func TestParseArbitraryEnvironmentNames(t *testing.T) {
	chart := `{start_of_intro: Intro}
Four bars of [D]nothing
{end_of_intro}`
	doc := parseChart(t, chart)

	want := []Section{
		{
			Name: "Intro",
			Lines: []ParsedLine{
				{Type: LineTypeChordAndLyric, Raw: "Four bars of [D]nothing", Lyrics: "Four bars of nothing", Chords: []ChordToken{{Name: "D", Position: 13}}},
			},
		},
	}
	if !reflect.DeepEqual(doc.Sections, want) {
		t.Errorf("Sections = %#v, want %#v (arbitrary env names are spec-legal)", doc.Sections, want)
	}
}

func TestParseIgnoredDirectivesSkipped(t *testing.T) {
	chart := `{define: Am base-fret 1 frets x 0 2 2 1 0}
{new_page}
{transpose: +2}
{start_of_chorus}
Only [G]this renders
{end_of_chorus}`
	doc := parseChart(t, chart)

	if len(doc.Sections) != 1 {
		t.Fatalf("Sections = %#v, want 1 section", doc.Sections)
	}
	if len(doc.Sections[0].Lines) != 1 {
		t.Errorf("Lines = %#v, want only the chorus line", doc.Sections[0].Lines)
	}
	if _, exists := doc.Metadata["define"]; exists {
		t.Error("define leaked into Metadata")
	}
}

func TestParseConditionalSelectors(t *testing.T) {
	chart := `{title-soprano: Selected Song}
{start_of_verse-soprano}
A [D]line
{end_of_verse}
{comment-alto: softly}`
	doc := parseChart(t, chart)

	if doc.Title != "Selected Song" {
		t.Errorf("Title = %q, want %q (selector stripped)", doc.Title, "Selected Song")
	}
	if len(doc.Sections) != 2 {
		t.Fatalf("Sections = %#v, want 2", doc.Sections)
	}
	if doc.Sections[0].Name != "verse" {
		t.Errorf("section name = %q, want verse", doc.Sections[0].Name)
	}
	last := doc.Sections[1].Lines
	if len(last) != 1 || last[0].Type != LineTypeComment || last[0].Lyrics != "softly" {
		t.Errorf("comment section lines = %#v, want one comment 'softly'", last)
	}
}

func TestParseUnclosedEnvironmentFlushed(t *testing.T) {
	doc := parseChart(t, "{soc}\nSwing low")

	want := []Section{
		{
			Name: "chorus",
			Lines: []ParsedLine{
				{Type: LineTypeLyric, Raw: "Swing low", Lyrics: "Swing low"},
			},
		},
	}
	if !reflect.DeepEqual(doc.Sections, want) {
		t.Errorf("Sections = %#v, want %#v (unclosed env flushed at end of parse)", doc.Sections, want)
	}
}

func TestParseBareLyricsFlushed(t *testing.T) {
	doc := parseChart(t, "Swing low\nSweet chariot")

	want := []Section{
		{
			Name: "",
			Lines: []ParsedLine{
				{Type: LineTypeLyric, Raw: "Swing low", Lyrics: "Swing low"},
				{Type: LineTypeLyric, Raw: "Sweet chariot", Lyrics: "Sweet chariot"},
			},
		},
	}
	if !reflect.DeepEqual(doc.Sections, want) {
		t.Errorf("Sections = %#v, want %#v (bare lyrics flushed as unnamed section)", doc.Sections, want)
	}
}

func TestParseBlankLinesDoNotFlush(t *testing.T) {
	chart := "{soc}\nSwing low\n\nSweet chariot\n{eoc}"
	doc := parseChart(t, chart)

	if len(doc.Sections) != 1 {
		t.Fatalf("Sections = %#v, want 1 section", doc.Sections)
	}
	lines := doc.Sections[0].Lines
	if len(lines) != 2 {
		t.Fatalf("Lines = %#v, want 2 lines (blank line does not split section)", lines)
	}
	if lines[0].Lyrics != "Swing low" || lines[1].Lyrics != "Sweet chariot" {
		t.Errorf("Lyrics = %q, %q, want %q, %q", lines[0].Lyrics, lines[1].Lyrics, "Swing low", "Sweet chariot")
	}
}

func TestParseBlankLinesFlushOutsideEnvironment(t *testing.T) {
	chart := "Swing low\n\nSweet chariot"
	doc := parseChart(t, chart)

	want := []Section{
		{
			Name: "",
			Lines: []ParsedLine{
				{Type: LineTypeLyric, Raw: "Swing low", Lyrics: "Swing low"},
			},
		},
		{
			Name: "",
			Lines: []ParsedLine{
				{Type: LineTypeLyric, Raw: "Sweet chariot", Lyrics: "Sweet chariot"},
			},
		},
	}
	if !reflect.DeepEqual(doc.Sections, want) {
		t.Errorf("Sections = %#v, want %#v (blank line outside env splits sections)", doc.Sections, want)
	}
}

func TestParseBlankLineAfterEnvCloseDoesNotCreateSection(t *testing.T) {
	chart := "{soc}\nline1\n{eoc}\n\nline2"
	doc := parseChart(t, chart)

	want := []Section{
		{
			Name: "chorus",
			Lines: []ParsedLine{
				{Type: LineTypeLyric, Raw: "line1", Lyrics: "line1"},
			},
		},
		{
			Name: "",
			Lines: []ParsedLine{
				{Type: LineTypeLyric, Raw: "line2", Lyrics: "line2"},
			},
		},
	}
	if !reflect.DeepEqual(doc.Sections, want) {
		t.Errorf("Sections = %#v, want %#v (blank after {eoc} must not create a spurious section)", doc.Sections, want)
	}
}

func TestParseConsecutiveBlankLinesSingleFlush(t *testing.T) {
	chart := "Swing low\n\n\nSweet chariot"
	doc := parseChart(t, chart)

	if len(doc.Sections) != 2 {
		t.Fatalf("Sections = %#v, want 2 sections (consecutive blank lines flush once)", doc.Sections)
	}
	if len(doc.Sections[0].Lines) != 1 || len(doc.Sections[1].Lines) != 1 {
		t.Errorf("Sections = %#v, want one line per section", doc.Sections)
	}
	if doc.Sections[0].Lines[0].Lyrics != "Swing low" || doc.Sections[1].Lines[0].Lyrics != "Sweet chariot" {
		t.Errorf("Lyrics = %q, %q, want %q, %q", doc.Sections[0].Lines[0].Lyrics, doc.Sections[1].Lines[0].Lyrics, "Swing low", "Sweet chariot")
	}
}

func TestParseCommentsCaptured(t *testing.T) {
	chart := `# a comment
{comment: Chorus}
{c: intro}
{highlight: solo}
{foo: bar}
{title: Swing Low}
{start_of_chorus}
Swing [D]low
{eoc}`
	doc := parseChart(t, chart)

	if doc.Title != "Swing Low" {
		t.Errorf("Title = %q, want %q", doc.Title, "Swing Low")
	}
	if len(doc.Sections) != 2 {
		t.Fatalf("Sections = %#v, want 2 sections", doc.Sections)
	}

	first := doc.Sections[0]
	if first.Name != "" {
		t.Errorf("first section Name = %q, want empty", first.Name)
	}
	if len(first.Lines) != 2 {
		t.Fatalf("first section Lines = %#v, want 2 comment lines", first.Lines)
	}
	for i, want := range []string{"Chorus", "intro"} {
		ln := first.Lines[i]
		if ln.Type != LineTypeComment || ln.Lyrics != want {
			t.Errorf("first.Lines[%d] = %#v, want comment line %q", i, ln, want)
		}
	}

	second := doc.Sections[1]
	if second.Name != "chorus" {
		t.Errorf("second section Name = %q, want %q", second.Name, "chorus")
	}
	if len(second.Lines) != 1 {
		t.Errorf("second section Lines = %#v, want 1 line", second.Lines)
	}
}

func TestParseEmptyChart(t *testing.T) {
	doc, err := NewParser(ModeChordPro).Parse("")
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

func TestParseUnknownMode(t *testing.T) {
	doc, err := NewParser(ParserMode(99)).Parse("Swing low")
	if doc != nil {
		t.Errorf("Parse() doc = %#v, want nil", doc)
	}
	if err == nil {
		t.Fatal("Parse() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown or unsupported parser mode") {
		t.Errorf("Parse() error %q does not contain %q", err.Error(), "unknown or unsupported parser mode")
	}
}

func TestParseBasicMode(t *testing.T) {
	doc, err := NewParser(ModeBasic).Parse("Swing low")
	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}
	if doc == nil {
		t.Fatal("Parse() returned nil document")
	}
	if doc.Metadata == nil {
		t.Error("Metadata is nil, want initialized map")
	}
	if len(doc.Sections) != 1 {
		t.Fatalf("Sections = %#v, want 1 section", doc.Sections)
	}
	if len(doc.Sections[0].Lines) != 1 {
		t.Fatalf("Lines = %#v, want 1 line", doc.Sections[0].Lines)
	}
	if got := doc.Sections[0].Lines[0]; got.Type != LineTypeLyric || got.Lyrics != "Swing low" {
		t.Errorf("line = %#v, want Lyric line 'Swing low'", got)
	}
}

func TestParseFullChart(t *testing.T) {
	chart := `# A simple ChordPro song.

{title: Swing Low Sweet Chariot}

{start_of_chorus}
Swing [D]low, sweet [G]chari[D]ot,
Comin' for to carry me [A7]home.
{end_of_chorus}

{comment: Chorus}`

	want := &Document{
		Title:    "Swing Low Sweet Chariot",
		Metadata: map[string][]string{},
		Sections: []Section{
			{
				Name: "chorus",
				Lines: []ParsedLine{
					{
						Type:   LineTypeChordAndLyric,
						Raw:    "Swing [D]low, sweet [G]chari[D]ot,",
						Lyrics: "Swing low, sweet chariot,",
						Chords: []ChordToken{
							{Name: "D", Position: 6},
							{Name: "G", Position: 17},
							{Name: "D", Position: 22},
						},
					},
					{
						Type:   LineTypeChordAndLyric,
						Raw:    "Comin' for to carry me [A7]home.",
						Lyrics: "Comin' for to carry me home.",
						Chords: []ChordToken{{Name: "A7", Position: 23}},
					},
				},
			},
			{
				Name: "",
				Lines: []ParsedLine{
					{
						Type:   LineTypeComment,
						Raw:    "{comment: Chorus}",
						Lyrics: "Chorus",
					},
				},
			},
		},
	}

	doc := parseChart(t, chart)
	if !reflect.DeepEqual(doc, want) {
		t.Errorf("Document = %#v, want %#v", doc, want)
	}
}

func TestDocumentString(t *testing.T) {
	doc := &Document{
		Title:    "Swing Low Sweet Chariot",
		Artist:   "Green Day",
		Metadata: map[string][]string{},
		Sections: []Section{
			{
				Name: "chorus",
				Lines: []ParsedLine{
					{Type: LineTypeChordAndLyric, Raw: "Swing [D]low"},
				},
			},
		},
	}

	want := "Title: Swing Low Sweet Chariot\n" +
		"Artist: Green Day\n" +
		"----------------------------------------\n" +
		"[chorus]\n" +
		"  Type: 3          Raw: Swing [D]low\n"

	if got := doc.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
