package parser

import "testing"

func TestParseChordMode(t *testing.T) {
	tests := []struct {
		in    string
		want  ChordMode
		isErr bool
	}{
		{"strict", StrictChords, false},
		{"STRICT", StrictChords, false},
		{"relaxed", RelaxedChords, false},
		{"Relaxed", RelaxedChords, false},
		{"  strict  ", StrictChords, false},
		{"bogus", StrictChords, true},
		{"", StrictChords, true},
	}
	for _, tt := range tests {
		got, err := ParseChordMode(tt.in)
		if tt.isErr {
			if err == nil {
				t.Errorf("ParseChordMode(%q) expected error", tt.in)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("ParseChordMode(%q) = %v, %v; want %v", tt.in, got, err, tt.want)
		}
	}
}

func TestChordModeString(t *testing.T) {
	if StrictChords.String() != "strict" || RelaxedChords.String() != "relaxed" {
		t.Errorf("String() = %q / %q, want strict / relaxed", StrictChords.String(), RelaxedChords.String())
	}
}

func TestParseChord(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want Chord
		err  bool
	}{
		{"plain root", "C", Chord{Root: "C"}, false},
		{"root g", "G", Chord{Root: "G"}, false},
		{"sharp root", "F#", Chord{Root: "F#"}, false},
		{"flat root", "Bb", Chord{Root: "Bb"}, false},
		{"double sharp root", "B#", Chord{Root: "B#"}, false},
		{"double sharp root e", "E#", Chord{Root: "E#"}, false},
		{"flat root c", "Cb", Chord{Root: "Cb"}, false},
		{"german H root", "H", Chord{Root: "H"}, false},
		{"german Hb root", "Hb", Chord{Root: "Hb"}, false},
		{"german H seventh", "H7", Chord{Root: "H", Ext: "7"}, false},
		{"minor", "Cm", Chord{Root: "C", Qual: "m"}, false},
		{"major", "Cmaj", Chord{Root: "C", Qual: "maj"}, false},
		{"minor full", "Cmin", Chord{Root: "C", Qual: "min"}, false},
		{"minor short", "Cmi", Chord{Root: "C", Qual: "mi"}, false},
		{"diminished", "Cdim", Chord{Root: "C", Qual: "dim"}, false},
		{"augmented", "Caug", Chord{Root: "C", Qual: "aug"}, false},
		{"suspended", "Csus", Chord{Root: "C", Qual: "sus"}, false},
		{"added", "Cadd", Chord{Root: "C", Qual: "add"}, false},
		{"half diminished", "Ch", Chord{Root: "C", Qual: "h"}, false},
		{"seventh", "C7", Chord{Root: "C", Ext: "7"}, false},
		{"major seventh", "Cmaj7", Chord{Root: "C", Qual: "maj", Ext: "7"}, false},
		{"ninth", "C9", Chord{Root: "C", Ext: "9"}, false},
		{"six nine", "C69", Chord{Root: "C", Ext: "69"}, false},
		{"minor flat five", "Cm7b5", Chord{Root: "C", Qual: "m", Ext: "7b5"}, false},
		{"sharp minor flat five", "C#m7b5", Chord{Root: "C#", Qual: "m", Ext: "7b5"}, false},
		{"seventh sus", "A7sus4", Chord{Root: "A", Ext: "7sus4"}, false},
		{"seventh sus g", "G7sus4", Chord{Root: "G", Ext: "7sus4"}, false},
		{"dominant flat five", "C7b5", Chord{Root: "C", Ext: "7b5"}, false},
		{"dominant sharp five", "C7#5", Chord{Root: "C", Ext: "7#5"}, false},
		{"dominant flat nine", "C7b9", Chord{Root: "C", Ext: "7b9"}, false},
		{"dominant sharp nine", "C7#9", Chord{Root: "C", Ext: "7#9"}, false},
		{"altered dominant", "C7alt", Chord{Root: "C", Ext: "7alt"}, false},
		{"minor major seventh", "Cmmaj7", Chord{Root: "C", Qual: "m", Ext: "maj7"}, false},
		{"minor added ninth", "Amadd9", Chord{Root: "A", Qual: "m", Ext: "add9"}, false},
		{"minor ninth major seventh", "Dm9maj7", Chord{Root: "D", Qual: "m", Ext: "9maj7"}, false},
		{"six sus two", "C6sus2", Chord{Root: "C", Ext: "6sus2"}, false},
		{"thirteenth sus", "C13sus4", Chord{Root: "C", Ext: "13sus4"}, false},
		{"sus two", "Csus2", Chord{Root: "C", Qual: "sus", Ext: "2"}, false},
		{"diminished seventh", "Cdim7", Chord{Root: "C", Qual: "dim", Ext: "7"}, false},
		{"caret major", "C^7", Chord{Root: "C", Ext: "^7"}, false},
		{"plus augmented", "C+", Chord{Root: "C", Ext: "+"}, false},
		{"half diminished seventh", "Ch7", Chord{Root: "C", Qual: "h", Ext: "7"}, false},
		{"slash bass", "C/G", Chord{Root: "C", Bass: "G"}, false},
		{"slash bass g", "G/B", Chord{Root: "G", Bass: "B"}, false},
		{"slash bass flat", "F7/Bb", Chord{Root: "F", Ext: "7", Bass: "Bb"}, false},
		{"minor slash", "Am/G", Chord{Root: "A", Qual: "m", Bass: "G"}, false},
		{"sharp slash", "G#/B", Chord{Root: "G#", Bass: "B"}, false},
		{"number slash bass", "G6/9", Chord{Root: "G", Ext: "6", Bass: "9"}, false},
		{"relaxed mode coda", "Coda", Chord{Root: "C", Ext: "oda"}, false},
		{"relaxed mode star extension", "Gm*", Chord{Root: "G", Qual: "m", Ext: "*"}, false},
		{"relaxed mode section word", "Chorus", Chord{Root: "C", Qual: "h", Ext: "orus"}, false},
		{"empty", "", Chord{}, true},
		{"no chord marker", "N.C.", Chord{}, true},
		{"garbage", "foo", Chord{}, true},
		{"empty", "", Chord{}, true},
		{"garbage", "foo", Chord{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseChord(tt.in)
			if tt.err {
				if err == nil {
					t.Fatalf("ParseChord(%q) expected error, got %#v", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseChord(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ParseChord(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}

func TestChordStringRoundtrip(t *testing.T) {
	names := []string{
		"C", "G", "F#", "Bb", "Cm", "Cmaj7", "Cmin7", "Cdim", "Caug",
		"C7", "C9", "C69", "Cm7b5", "C#m7b5", "A7sus4", "G7sus4",
		"C7b5", "C7#5", "C7b9", "C7#9", "C7alt", "Cmmaj7", "Amadd9",
		"Dm9maj7", "C6sus2", "C13sus4", "Csus2", "Cdim7", "C^7", "C+",
		"C/G", "G/B", "F7/Bb", "Am/G", "G#/B", "G6/9",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			c, err := ParseChord(name)
			if err != nil {
				t.Fatalf("ParseChord(%q) unexpected error: %v", name, err)
			}
			if got := c.String(); got != name {
				t.Errorf("String() = %q, want %q", got, name)
			}
		})
	}
}

func TestTransposeRoot(t *testing.T) {
	tests := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"no change", "C", 0, "C"},
		{"up whole step", "C", 2, "D"},
		{"up fourth", "C", 4, "E"},
		{"g to a", "G", 2, "A"},
		{"a up one", "A", 1, "A#"},
		{"a down one", "A", -1, "G#"},
		{"f up one", "F", 1, "F#"},
		{"c down one", "C", -1, "B"},
		{"e down four", "E", -4, "C"},
		{"octave identity", "C", 12, "C"},
		{"octave plus step", "C", 13, "C#"},
		{"flat bb up one", "Bb", 1, "B"},
		{"flat bb up two", "Bb", 2, "C"},
		{"flat eb up one", "Eb", 1, "E"},
		{"flat eb up two", "Eb", 2, "F"},
		{"flat ab up one", "Ab", 1, "A"},
		{"flat gb up one", "Gb", 1, "G"},
		{"flat db up one", "Db", 1, "D"},
		{"sharp f# up two", "F#", 2, "G#"},
		{"sharp c# up one", "C#", 1, "D"},
		{"sharp a# up one", "A#", 1, "B"},
		{"b up one", "B", 1, "C"},
		{"b double sharp normalizes", "B#", 0, "C"},
		{"flat cb normalizes", "Cb", 0, "B"},
		{"german H up one", "H", 1, "C"},
		{"german H down one", "H", -1, "A#"},
		{"german Hb up one", "Hb", 1, "B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := ParseChord(tt.in)
			if err != nil {
				t.Fatalf("ParseChord(%q) unexpected error: %v", tt.in, err)
			}
			if got := c.Transpose(tt.n).Root; got != tt.want {
				t.Errorf("Transpose(%d).Root = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestTransposeChord(t *testing.T) {
	tests := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"minor seventh", "Cm7", 2, "Dm7"},
		{"seventh sus", "A7sus4", 1, "A#7sus4"},
		{"flat seventh sus", "Bb7sus4", 2, "C7sus4"},
		{"six nine keeps number bass", "G6/9", 2, "A6/9"},
		{"slash bass transposed", "C/G", 2, "D/A"},
		{"slash bass flat", "F7/Bb", 1, "F#7/B"},
		{"minor slash", "Am/G", 2, "Bm/A"},
		{"sharp slash", "G#/B", 2, "A#/C#"},
		{"major seventh down", "Cmaj7", -2, "A#maj7"},
		{"major seventh sharp up", "F#maj7", 1, "Gmaj7"},
		{"octave identity", "Dm7/G", 12, "Dm7/G"},
		{"relaxed coda transposes by root", "Coda", 2, "Doda"},
		{"relaxed chorus transposes by root", "Chorus", -2, "A#horus"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := ParseChord(tt.in)
			if err != nil {
				t.Fatalf("ParseChord(%q) unexpected error: %v", tt.in, err)
			}
			if got := c.Transpose(tt.n).String(); got != tt.want {
				t.Errorf("Transpose(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestTransposePreservesSuffix(t *testing.T) {
	c, err := ParseChord("C#m7b5")
	if err != nil {
		t.Fatalf("ParseChord error: %v", err)
	}
	got := c.Transpose(3)
	if got.Qual != c.Qual || got.Ext != c.Ext {
		t.Errorf("Transpose altered suffix: got %#v, want qual=%q ext=%q", got, c.Qual, c.Ext)
	}
	if got.Root != "E" {
		t.Errorf("Root = %q, want %q (C#+3)", got.Root, "E")
	}
}

func TestDocumentTranspose(t *testing.T) {
	doc := &Document{
		Title:    "T",
		Key:      "C",
		Metadata: map[string][]string{},
		Sections: []Section{
			{
				Name: "",
				Lines: []ParsedLine{
					{
						Type:   LineTypeChordAndLyric,
						Raw:    "[C] home",
						Lyrics: "home",
						Chords: []ChordToken{{Name: "C", Position: 0}, {Name: "G", Position: 5}, {Name: "Riff", Position: 9}, {Name: "", Position: 11}},
					},
				},
			},
		},
	}

	doc.Transpose(2)

	chords := doc.Sections[0].Lines[0].Chords
	want := []ChordToken{{Name: "D", Position: 0}, {Name: "A", Position: 5}, {Name: "Riff", Position: 9}, {Name: "", Position: 11}}
	for i := range want {
		if chords[i].Name != want[i].Name {
			t.Errorf("chords[%d].Name = %q, want %q", i, chords[i].Name, want[i].Name)
		}
		if chords[i].Position != want[i].Position {
			t.Errorf("chords[%d].Position = %d, want %d", i, chords[i].Position, want[i].Position)
		}
	}
	if doc.Key != "D" {
		t.Errorf("Key = %q, want transposed %q", doc.Key, "D")
	}

	doc.Transpose(-2)
	if chords[0].Name != "C" || chords[1].Name != "G" {
		t.Errorf("roundtrip transpose chords = %q, %q, want %q, %q", chords[0].Name, chords[1].Name, "C", "G")
	}
	if doc.Key != "C" {
		t.Errorf("roundtrip transpose key = %q, want %q", doc.Key, "C")
	}
}
