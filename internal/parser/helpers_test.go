package parser

import (
	"errors"
	"reflect"
	"testing"
)

func TestExtractBracketContents(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []string
	}{
		{"single chord in lyric", "Swing [D]low", []string{"D"}},
		{"multiple chords", "[A] [F#m] [E]", []string{"A", "F#m", "E"}},
		{"adjacent chords", "[A][G]", []string{"A", "G"}},
		{"asterisk escaped content", "[*N.C.]", []string{"*N.C."}},
		{"riff marker", "[--]", []string{"--"}},
		{"empty bracket", "[]", []string{""}},
		{"no brackets", "Swing low", nil},
		{"unclosed bracket yields nothing", "unclosed [D", nil},
		{"closing without opening yields nothing", "stray ]]", nil},
		{"nested brackets collapse", "[[D]]", []string{"D"}},
		{"complex chord names", "[C#m7b5] [G/B]", []string{"C#m7b5", "G/B"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBracketContents(tt.line)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractBracketContents(%q) = %#v, want %#v", tt.line, got, tt.want)
			}
		})
	}
}

func TestValidateBracketContent(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		mode ChordMode
		want bool
	}{
		{"simple chord", "D", StrictChords, true},
		{"seventh chord", "A7", StrictChords, true},
		{"minor with sharp", "C#m", StrictChords, true},
		{"minor with flat", "Bbm7", StrictChords, true},
		{"major seventh", "Dmaj7", StrictChords, true},
		{"major seventh with sharp", "F#maj7", StrictChords, true},
		{"suspended chord", "Asus4", StrictChords, true},
		{"suspended with g", "Gsus4", StrictChords, true},
		{"added chord", "Cadd9", StrictChords, true},
		{"augmented", "Eaug", StrictChords, true},
		{"diminished", "Ddim", StrictChords, true},
		{"slash bass chord", "G/B", StrictChords, true},
		{"slash bass with flat", "F7/Bb", StrictChords, true},
		{"slash bass with sharp", "G#/B", StrictChords, true},
		{"minor slash", "Am/G", StrictChords, true},
		{"flat root", "Db", StrictChords, true},
		{"double sharp root", "B#", StrictChords, true},
		{"minor with whitespace", " Dm ", StrictChords, true},
		{"asterisk alone", "*", StrictChords, true},
		{"asterisk escaped no-chord", "*N.C.", StrictChords, true},
		{"asterisk escaped rest", "*rest", StrictChords, true},
		{"asterisk with whitespace", " *N.C. ", StrictChords, true},
		{"half-diminished", "C#m7b5", StrictChords, true},
		{"minor seventh flat five", "Dm7b5", StrictChords, true},
		{"seventh sus", "A7sus4", StrictChords, true},
		{"seventh sus g", "G7sus4", StrictChords, true},
		{"seventh sus c", "C7sus4", StrictChords, true},
		{"six-nine slash form", "G6/9", StrictChords, true},
		{"six-nine compact form", "C69", StrictChords, true},
		{"dominant flat five", "C7b5", StrictChords, true},
		{"dominant sharp five", "C7#5", StrictChords, true},
		{"dominant flat nine", "C7b9", StrictChords, true},
		{"dominant sharp nine", "C7#9", StrictChords, true},
		{"ninth", "C9", StrictChords, true},
		{"eleventh", "C11", StrictChords, true},
		{"thirteenth", "C13", StrictChords, true},
		{"ninth sharp eleven", "C9#11", StrictChords, true},
		{"thirteenth flat nine", "C13b9", StrictChords, true},
		{"altered dominant", "C7alt", StrictChords, true},
		{"major ninth", "Cmaj9", StrictChords, true},
		{"major thirteenth", "Cmaj13", StrictChords, true},
		{"major seventh sharp five", "Cmaj7#5", StrictChords, true},
		{"minor major seventh", "Cmmaj7", StrictChords, true},
		{"minor ninth", "Am9", StrictChords, true},
		{"minor added ninth", "Amadd9", StrictChords, true},
		{"minor sixth", "Cm6", StrictChords, true},
		{"minor ninth major seventh", "Dm9maj7", StrictChords, true},
		{"diminished seventh", "Cdim7", StrictChords, true},
		{"six suspended second", "C6sus2", StrictChords, true},
		{"thirteenth suspended fourth", "C13sus4", StrictChords, true},
		{"suspended second", "Csus2", StrictChords, true},
		{"no-chord without asterisk rejected", "N.C.", StrictChords, false},
		{"no-chord without asterisk no dot", "N.C", StrictChords, false},
		{"german H root accepted", "H", StrictChords, true},
		{"german H seventh accepted", "H7", StrictChords, true},
		{"relaxed chord rejected in strict mode", "Coda", StrictChords, false},
		{"relaxed star extension rejected in strict mode", "Gm*", StrictChords, false},
		{"relaxed section word rejected in strict mode", "Chorus", StrictChords, false},
		{"relaxed chord accepted in relaxed mode", "Coda", RelaxedChords, true},
		{"relaxed star extension accepted in relaxed mode", "Gm*", RelaxedChords, true},
		{"relaxed section word accepted in relaxed mode", "Chorus", RelaxedChords, true},
		{"lowercase root rejected", "m", StrictChords, false},
		{"numeric rejected", "1", StrictChords, false},
		{"empty string rejected", "", StrictChords, false},
		{"whitespace rejected", "   ", StrictChords, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateBracketContent(tt.raw, tt.mode); got != tt.want {
				t.Errorf("validateBracketContent(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestStripSelector(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no selector", "title", "title"},
		{"instrument selector", "define-guitar", "define"},
		{"voice selector", "comment-alto", "comment"},
		{"env selector", "start_of_verse-soprano", "start_of_verse"},
		{"reversed selector", "comment-tenor!", "comment"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripSelector(tt.in); got != tt.want {
				t.Errorf("stripSelector(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestExtractDirective(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantDir  string
		wantData string
	}{
		{"title with space after colon", "{title: Swing Low}", "title", "Swing Low"},
		{"title without space after colon", "{t:Take Me Home, Country Roads}", "t", "Take Me Home, Country Roads"},
		{"inner whitespace", "{ title : Foo }", "title", "Foo"},
		{"environment no data", "{soc}", "soc", ""},
		{"chorus no data", "{chorus}", "chorus", ""},
		{"multiple colons preserved in data", "{t:A: B}", "t", "A: B"},
		{"empty data", "{title:}", "title", ""},
		{"comment directive", "{comment:Verse 1}", "comment", "Verse 1"},
		{"preserves directive case", "{Title: A Song}", "Title", "A Song"},
		{"conditional selector stripped", "{title-soprano: Hi}", "title", "Hi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDir, gotData := extractDirective(tt.line)
			if gotDir != tt.wantDir || gotData != tt.wantData {
				t.Errorf("extractDirective(%q) = (%q, %q), want (%q, %q)", tt.line, gotDir, gotData, tt.wantDir, tt.wantData)
			}
		})
	}
}

func TestGetDirectiveCategory(t *testing.T) {
	tests := []struct {
		name      string
		directive string
		want      DirectiveCategory
	}{
		{"meta directive", "title", CategoryMeta},
		{"meta alias", "t", CategoryMeta},
		{"formatting directive", "comment", CategoryFormatting},
		{"formatting alias", "c", CategoryFormatting},
		{"environment directive", "soc", CategoryEnvironment},
		{"environment chorus", "chorus", CategoryEnvironment},
		{"unknown directive", "foo", CategoryUnknown},
		{"empty directive", "", CategoryUnknown},
		{"uppercase meta directive", "Title", CategoryMeta},
		{"uppercase meta key", "KEY", CategoryMeta},
		{"uppercase formatting directive", "Comment", CategoryFormatting},
		{"uppercase environment directive", "SOC", CategoryEnvironment},
		{"ignored define", "define", CategoryIgnored},
		{"delegated env ignored before generic env", "start_of_abc", CategoryIgnored},
		{"output directive ignored", "new_page", CategoryIgnored},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDirectiveCategory(tt.directive); got != tt.want {
				t.Errorf("getDirectiveCategory(%q) = %v, want %v", tt.directive, got, tt.want)
			}
		})
	}
}

func TestParseEnvDirective(t *testing.T) {
	tests := []struct {
		name       string
		directive  string
		wantAction string
		wantEnv    string
		wantOK     bool
	}{
		{"start_of full name", "start_of_chorus", "start", "chorus", true},
		{"start_of verse", "start_of_verse", "start", "verse", true},
		{"start_of tab", "start_of_tab", "start", "tab", true},
		{"end_of full name", "end_of_chorus", "end", "chorus", true},
		{"end_of tab", "end_of_tab", "end", "tab", true},
		{"alias soc", "soc", "start", "chorus", true},
		{"alias sov", "sov", "start", "verse", true},
		{"alias sob", "sob", "start", "bridge", true},
		{"alias sot", "sot", "start", "tab", true},
		{"alias sog", "sog", "start", "grid", true},
		{"alias eoc", "eoc", "end", "chorus", true},
		{"alias eov", "eov", "end", "verse", true},
		{"alias eob", "eob", "end", "bridge", true},
		{"alias eot", "eot", "end", "tab", true},
		{"alias eog", "eog", "end", "grid", true},
		{"chorus shorthand", "chorus", "start", "chorus", true},
		{"uppercase", "SOC", "start", "chorus", true},
		{"mixed case", "Start_Of_Chorus", "start", "chorus", true},
		{"unknown alias resolves to itself", "sox", "start", "x", true},
		{"unknown end alias", "eox", "end", "x", true},
		{"plain env name rejected", "verse", "", "", false},
		{"empty rejected", "", "", "", false},
		{"unknown rejected", "foo", "", "", false},
		{"short prefix rejected", "so", "", "", false},
		{"long alias rejected", "socd", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAction, gotEnv, gotOK := parseEnvDirective(tt.directive)
			if gotAction != tt.wantAction || gotEnv != tt.wantEnv || gotOK != tt.wantOK {
				t.Errorf("parseEnvDirective(%q) = (%q, %q, %v), want (%q, %q, %v)", tt.directive, gotAction, gotEnv, gotOK, tt.wantAction, tt.wantEnv, tt.wantOK)
			}
		})
	}
}

func TestResolveAlias(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"verse", "v", "verse"},
		{"chorus", "c", "chorus"},
		{"bridge", "b", "bridge"},
		{"tab", "t", "tab"},
		{"grid", "g", "grid"},
		{"unknown returns input", "x", "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveAlias(tt.in); got != tt.want {
				t.Errorf("resolveAlias(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsStringNoteLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"uppercase with pipe", "E|------", true},
		{"lowercase", "e|--0--", true},
		{"any musical letter F", "F|--1--", true},
		{"any musical letter C", "C|--3--", true},
		{"indented", "   B|--2--", true},
		{"space before pipe", "E |---", true},
		{"pipe only no dashes", "G|", true},
		{"no pipe", "E --0--", false},
		{"dashes only", "------", false},
		{"out of range letter H", "H|--1--", false},
		{"no leading letter", "--3--", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isStringNoteLine(tt.line); got != tt.want {
				t.Errorf("isStringNoteLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestIsRealTabBlock(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  bool
	}{
		{"six string lines", []string{"E|--", "B|--", "G|--", "D|--", "A|--", "E|--"}, true},
		{"four is boundary", []string{"E|--", "B|--", "G|--", "D|--"}, true},
		{"three plus junk", []string{"E|--", "B|--", "G|--", "junk"}, false},
		{"decorative dashes only", []string{"------", "......"}, false},
		{"empty block", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRealTabBlock(tt.lines); got != tt.want {
				t.Errorf("isRealTabBlock(%#v) = %v, want %v", tt.lines, got, tt.want)
			}
		})
	}
}

func TestDetectParserMode(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
		err   error
		want  ParserMode
	}{
		{"valid no error", true, nil, ModeChordPro},
		{"invalid no error", false, nil, ModeBasic},
		{"valid with error", true, errors.New("boom"), ModeBasic},
		{"invalid with error", false, errors.New("boom"), ModeBasic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectParserMode(tt.valid, tt.err); got != tt.want {
				t.Errorf("DetectParserMode(%v, %v) = %v, want %v", tt.valid, tt.err, got, tt.want)
			}
		})
	}
}

func TestLooksLikeBasicChart(t *testing.T) {
	tests := []struct {
		name  string
		chart string
		want  bool
	}{
		{"chord over lyrics", "C  G\nSwing low, sweet chariot", true},
		{"basic with comment", "# intro\nC G\nSwing low", true},
		{"basic single chord line", "Am7", true},
		{"chordpro inline brackets", "{soc}\nSwing [D]low\n{eoc}", false},
		{"chordpro directive and chords", "{title: X}\n[G] home", false},
		{"bare lyrics no chords", "Swing low, sweet chariot", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LooksLikeBasicChart(tt.chart, StrictChords); got != tt.want {
				t.Errorf("LooksLikeBasicChart(%q) = %v, want %v", tt.chart, got, tt.want)
			}
		})
	}
}
