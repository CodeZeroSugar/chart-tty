package parser

import (
	"reflect"
	"testing"
)

func TestParseLineType(t *testing.T) {
	tests := []struct {
		name string
		line string
		want LineType
	}{
		{"empty line", "", LineTypeEmpty},
		{"whitespace-only line", "   ", LineTypeEmpty},
		{"directive", "{title: Swing Low}", LineTypeDirective},
		{"directive with leading/trailing whitespace", "  {t:Take Me Home}  ", LineTypeDirective},
		{"comment", "# a comment", LineTypeComment},
		{"comment with leading whitespace", "  # indented comment", LineTypeComment},
		{"tab grid", "E|--0--|--2--|", LineTypeTab},
		{"dash-only tab line", "--------", LineTypeTab},
		{"riff marker is tab", "[----]", LineTypeTab},
		{"lyric", "Swing low, sweet chariot", LineTypeLyric},
		{"lyric with single dash", "Well-behaved lyric", LineTypeLyric},
		{"chord-only line", "[A] [F#m] [E]", LineTypeChord},
		{"empty chord", "[]", LineTypeChord},
		{"chord and lyric", "Swing [D]low, sweet [G]chariot", LineTypeChordAndLyric},
		{"chord at start of lyric", "[D]low, sweet chariot", LineTypeChordAndLyric},
		{"directive beats tab", "{----}", LineTypeDirective},
		{"comment beats tab", "# --", LineTypeComment},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(ModeChordPro)
			got := p.parseLine(tt.line)
			if got.Type != tt.want {
				t.Errorf("parseLine(%q).Type = %v, want %v", tt.line, got.Type, tt.want)
			}
		})
	}
}

func TestParseLineRaw(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"raw preserved for lyric", "Swing low", "Swing low"},
		{"empty line raw", "", ""},
		{"whitespace-only raw", "   ", ""},
		{"directive raw trimmed", "  {title: Foo}  ", "{title: Foo}"},
		{"comment raw trimmed", "  # hi", "# hi"},
		{"tab raw trimmed", "  E|--0--|", "E|--0--|"},
		{"chord and lyric raw trimmed", "  [D]low  ", "[D]low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(ModeChordPro)
			got := p.parseLine(tt.line)
			if got.Raw != tt.want {
				t.Errorf("parseLine(%q).Raw = %q, want %q", tt.line, got.Raw, tt.want)
			}
		})
	}
}

func TestParseLineEmptyFields(t *testing.T) {
	p := NewParser(ModeChordPro)

	got := p.parseLine("   ")
	if got.Type != LineTypeEmpty {
		t.Errorf("parseLine(whitespace).Type = %v, want %v", got.Type, LineTypeEmpty)
	}
	if got.Raw != "" {
		t.Errorf("parseLine(whitespace).Raw = %q, want empty", got.Raw)
	}
	if got.Lyrics != "" {
		t.Errorf("parseLine(whitespace).Lyrics = %q, want empty", got.Lyrics)
	}
	if got.Chords != nil {
		t.Errorf("parseLine(whitespace).Chords = %v, want nil", got.Chords)
	}
}

func TestParseLineDirective(t *testing.T) {
	p := NewParser(ModeChordPro)

	got := p.parseLine("{title: Swing Low Sweet Chariot}")
	if got.Type != LineTypeDirective {
		t.Errorf("parseLine().Type = %v, want %v", got.Type, LineTypeDirective)
	}
	if got.Raw != "{title: Swing Low Sweet Chariot}" {
		t.Errorf("parseLine().Raw = %q", got.Raw)
	}
	if got.Lyrics != "" {
		t.Errorf("parseLine().Lyrics = %q, want empty", got.Lyrics)
	}
	if got.Chords != nil {
		t.Errorf("parseLine().Chords = %v, want nil", got.Chords)
	}
}

func TestParseLineChords(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantType LineType
		wantLy   string
		wantCh   []ChordToken
	}{
		{
			name:     "chords inline between lyrics",
			line:     "Swing [D]low, sweet [G]chariot",
			wantType: LineTypeChordAndLyric,
			wantLy:   "Swing low, sweet chariot",
			wantCh:   []ChordToken{{Name: "D", Position: 6}, {Name: "G", Position: 17}},
		},
		{
			name:     "chord at start and middle",
			line:     "[D]low, sweet [G]chari[D]ot",
			wantType: LineTypeChordAndLyric,
			wantLy:   "low, sweet chariot",
			wantCh:   []ChordToken{{Name: "D", Position: 0}, {Name: "G", Position: 11}, {Name: "D", Position: 16}},
		},
		{
			name:     "chord at end of line",
			line:     "Comin' for to carry me [A7]home.",
			wantType: LineTypeChordAndLyric,
			wantLy:   "Comin' for to carry me home.",
			wantCh:   []ChordToken{{Name: "A7", Position: 23}},
		},
		{
			name:     "chord-only line keeps lyric spaces",
			line:     "[A] [F#m] [E]",
			wantType: LineTypeChord,
			wantLy:   "  ",
			wantCh:   []ChordToken{{Name: "A", Position: 0}, {Name: "F#m", Position: 1}, {Name: "E", Position: 2}},
		},
		{
			name:     "complex chord names",
			line:     "[C#m7b5] [G/B] [N.C.] [F#maj7]",
			wantType: LineTypeChord,
			wantLy:   "   ",
			wantCh: []ChordToken{
				{Name: "C#m7b5", Position: 0},
				{Name: "G/B", Position: 1},
				{Name: "N.C.", Position: 2},
				{Name: "F#maj7", Position: 3},
			},
		},
		{
			name:     "empty chord",
			line:     "[]",
			wantType: LineTypeChord,
			wantLy:   "",
			wantCh:   []ChordToken{{Name: "", Position: 0}},
		},
		{
			name:     "stray closing bracket",
			line:     "D]",
			wantType: LineTypeChordAndLyric,
			wantLy:   "D",
			wantCh:   []ChordToken{{Name: "", Position: 1}},
		},
		{
			name:     "unclosed bracket drops chord",
			line:     "[D",
			wantType: LineTypeLyric,
			wantLy:   "",
			wantCh:   nil,
		},
		{
			name:     "unclosed bracket mid-line keeps lyric trailing space",
			line:     "hello [D",
			wantType: LineTypeLyric,
			wantLy:   "hello ",
			wantCh:   nil,
		},
		{
			name:     "nested brackets produce duplicate chords",
			line:     "[[D]]",
			wantType: LineTypeChord,
			wantLy:   "",
			wantCh:   []ChordToken{{Name: "D", Position: 0}, {Name: "D", Position: 0}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(ModeChordPro)
			got := p.parseLine(tt.line)
			if got.Type != tt.wantType {
				t.Errorf("parseLine(%q).Type = %v, want %v", tt.line, got.Type, tt.wantType)
			}
			if got.Lyrics != tt.wantLy {
				t.Errorf("parseLine(%q).Lyrics = %q, want %q", tt.line, got.Lyrics, tt.wantLy)
			}
			if !reflect.DeepEqual(got.Chords, tt.wantCh) {
				t.Errorf("parseLine(%q).Chords = %#v, want %#v", tt.line, got.Chords, tt.wantCh)
			}
		})
	}
}

func TestParseLineComment(t *testing.T) {
	p := NewParser(ModeChordPro)

	got := p.parseLine("# a comment")
	if got.Type != LineTypeComment {
		t.Errorf("parseLine().Type = %v, want %v", got.Type, LineTypeComment)
	}
	if got.Raw != "# a comment" {
		t.Errorf("parseLine().Raw = %q", got.Raw)
	}
	if got.Lyrics != "" {
		t.Errorf("parseLine().Lyrics = %q, want empty", got.Lyrics)
	}
	if got.Chords != nil {
		t.Errorf("parseLine().Chords = %v, want nil", got.Chords)
	}
}

func TestParseLineTab(t *testing.T) {
	p := NewParser(ModeChordPro)

	got := p.parseLine("E|--0--|--2--|")
	if got.Type != LineTypeTab {
		t.Errorf("parseLine().Type = %v, want %v", got.Type, LineTypeTab)
	}
	if got.Raw != "E|--0--|--2--|" {
		t.Errorf("parseLine().Raw = %q", got.Raw)
	}
	if got.Lyrics != "" {
		t.Errorf("parseLine().Lyrics = %q, want empty", got.Lyrics)
	}
	if got.Chords != nil {
		t.Errorf("parseLine().Chords = %v, want nil", got.Chords)
	}
}

func TestIsTabLine(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"tab grid", "E|--0--|--2--|", true},
		{"double dash only", "--------", true},
		{"short double dash", "--", true},
		{"double dash inside text", "ab--cd", true},
		{"pipe with dash", "foo|-bar", true},
		{"riff marker", "[----]", true},
		{"pipe without dash", "foo|bar", false},
		{"single dash", "single-dash", false},
		{"single dash with space", "foo - bar", false},
		{"plain lyric", "Swing low", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTabLine(tt.in); got != tt.want {
				t.Errorf("isTabLine(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
