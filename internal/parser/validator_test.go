package parser

import (
	"strings"
	"testing"
)

func TestNewValidator(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() unexpected error: %v", err)
	}
	if v == nil {
		t.Fatal("NewValidator() returned nil validator")
	}

	if v.AllowUnkDirectives {
		t.Error("AllowUnkDirectives should default to false")
	}
}

func TestSpecPopulated(t *testing.T) {
	if len(Spec.MetaDirectives) == 0 {
		t.Error("Spec.MetaDirectives is empty")
	}
	if len(Spec.FormattingDirectives) == 0 {
		t.Error("Spec.FormattingDirectives is empty")
	}
	if len(Spec.EnvironmentDirectives) == 0 {
		t.Error("Spec.EnvironmentDirectives is empty")
	}
}

func TestIsValidDirective(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() unexpected error: %v", err)
	}

	tests := []struct {
		name      string
		directive string
		want      bool
	}{
		{"meta full name", "title", true},
		{"meta short alias", "t", true},
		{"meta artist", "artist", true},
		{"formatting comment", "comment", true},
		{"formatting short alias", "c", true},
		{"formatting highlight", "highlight", true},
		{"environment soc", "soc", true},
		{"environment start_of_chorus", "start_of_chorus", true},
		{"environment eog", "eog", true},
		{"environment chorus", "chorus", true},
		{"unknown", "foo", false},
		{"near-miss environment", "chorusx", false},
		{"empty string", "", false},
		{"uppercase meta directive", "Title", true},
		{"uppercase env directive", "SOC", true},
		{"x_ custom directive ignored per spec", "x_mspro_pedal_setting", true},
		{"x_ bare", "x_", true},
		{"conditional selector stripped", "title-soprano", true},
		{"conditional env selector", "start_of_verse-soprano", true},
		{"arbitrary environment name", "start_of_intro", true},
		{"arbitrary environment end", "end_of_intro", true},
		{"ignored spec-legal define", "define", true},
		{"ignored output directive", "new_page", true},
		{"ignored font legacy", "textfont", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.IsValidDirective(tt.directive); got != tt.want {
				t.Errorf("IsValidDirective(%q) = %v, want %v", tt.directive, got, tt.want)
			}
		})
	}
}

func TestIsValidDirectiveAllowUnknown(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() unexpected error: %v", err)
	}
	v.AllowUnkDirectives = true

	for _, d := range []string{"foo", "anything", "", "soc", "title"} {
		if !v.IsValidDirective(d) {
			t.Errorf("IsValidDirective(%q) with AllowUnkDirectives=true = false, want true", d)
		}
	}
}

func TestValidateChart(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() unexpected error: %v", err)
	}

	tests := []struct {
		name           string
		chart          string
		wantValid      bool
		wantErrContain string
	}{
		{
			name: "simple valid chart",
			chart: `# A simple ChordPro song.

{title: Swing Low Sweet Chariot}

{start_of_chorus}
Swing [D]low, sweet [G]chari[D]ot,
Comin' for to carry me [A7]home.
{end_of_chorus}

{comment: Chorus}`,
			wantValid: true,
		},
		{
			name:      "directive without space after colon",
			chart:     `{t:Take Me Home, Country Roads}`,
			wantValid: true,
		},
		{
			name:      "directive with inner whitespace",
			chart:     `{ title : Foo }`,
			wantValid: true,
		},
		{
			name:      "directive with leading/trailing whitespace",
			chart:     "  {st:John Denver}  ",
			wantValid: true,
		},
		{
			name:      "complex chord names",
			chart:     `[Dm7] [G/B] [F#maj7] [C#m] [Eaug] [Gsus4] [Bbm7]`,
			wantValid: true,
		},
		{
			name:      "half-diminished seventh",
			chart:     `[C#m7b5]`,
			wantValid: true,
		},
		{
			name:      "seventh suspended fourth",
			chart:     `[A7sus4]`,
			wantValid: true,
		},
		{
			name:      "altered and extended dominants",
			chart:     `[C7b5] [C7#9] [C7alt] [C9] [C11] [C13]`,
			wantValid: true,
		},
		{
			name:      "major and minor extensions",
			chart:     `[Cmaj9] [Cmmaj7] [Am9] [Amadd9] [Dm9maj7]`,
			wantValid: true,
		},
		{
			name:      "six-nine forms",
			chart:     `[C69] [G6/9]`,
			wantValid: true,
		},
		{
			name:      "suspended variants",
			chart:     `[C7sus4] [C6sus2] [C13sus4] [Csus2]`,
			wantValid: true,
		},
		{
			name:      "chord-only lines",
			chart:     "[A] [F#m] [E] [E7] [D] [G] \n[A] [A] [A]",
			wantValid: true,
		},
		{
			name:      "tab grid with pipe characters",
			chart:     "{sot}\nE|--0--|--2--|\n{eot}",
			wantValid: true,
		},
		{
			name:      "asterisk prefixed bracket content",
			chart:     "[*N.C.]",
			wantValid: true,
		},
		{
			name:      "asterisk escaped riff marker",
			chart:     "[*----]",
			wantValid: true,
		},
		{
			name:      "lyrics only no directives or chords",
			chart:     "Swing low, sweet chariot,\nComin' for to carry me home.",
			wantValid: true,
		},
		{
			name:      "uppercase directives",
			chart:     "{Title: A Song}\n{SOC}",
			wantValid: true,
		},
		{
			name:      "empty chord lines",
			chart:     "{title: Foo}\n\n[G]\n\n[D]",
			wantValid: true,
		},
		{
			name:      "crlf line endings",
			chart:     "{title: Swing Low}\r\n\r\nSwing [D]low, sweet [G]chariot.\r\n{comment: Chorus}\r\n",
			wantValid: true,
		},
		{
			name:           "unclosed directive",
			chart:          "{title: Foo",
			wantValid:      false,
			wantErrContain: "unclosed directive",
		},
		{
			name:           "unclosed directive mid-line",
			chart:          "lyric {comment",
			wantValid:      false,
			wantErrContain: "unclosed directive",
		},
		{
			name:           "invalid directive",
			chart:          "{foobar: x}",
			wantValid:      false,
			wantErrContain: "invalid directive",
		},
		{
			name:           "unknown directive no colon",
			chart:          "{bogus}",
			wantValid:      false,
			wantErrContain: "invalid directive",
		},
		{
			name:           "empty braces",
			chart:          "{ }",
			wantValid:      false,
			wantErrContain: "invalid directive",
		},
		{
			name:           "unclosed square bracket",
			chart:          "[D",
			wantValid:      false,
			wantErrContain: "invalid brackets",
		},
		{
			name:           "lone closing square bracket",
			chart:          "D]",
			wantValid:      false,
			wantErrContain: "invalid brackets",
		},
		{
			name:           "closing before opening",
			chart:          "][D]",
			wantValid:      false,
			wantErrContain: "invalid brackets",
		},
		{
			name:           "extra closing bracket",
			chart:          "[D]]",
			wantValid:      false,
			wantErrContain: "invalid brackets",
		},
		{
			name:           "nested brackets",
			chart:          "[[D]]",
			wantValid:      false,
			wantErrContain: "invalid brackets",
		},
		{
			name:           "open bracket on chord line",
			chart:          "[A] [G",
			wantValid:      false,
			wantErrContain: "invalid brackets",
		},
		{
			name:           "no-chord marker rejected without asterisk",
			chart:          "[N.C.]",
			wantValid:      false,
			wantErrContain: "invalid bracket content",
		},
		{
			name:           "riff marker rejected without asterisk",
			chart:          "[----]",
			wantValid:      false,
			wantErrContain: "invalid bracket content",
		},
		{
			name:           "non-chord text rejected",
			chart:          "[foo]",
			wantValid:      false,
			wantErrContain: "invalid bracket content",
		},
		{
			name:      "german H root accepted",
			chart:     "[H]",
			wantValid: true,
		},
		{
			name:      "relaxed mode coda",
			chart:     "[Coda]",
			wantValid: true,
		},
		{
			name:      "relaxed mode star extension",
			chart:     "[Gm*]",
			wantValid: true,
		},
		{
			name:      "relaxed mode bracketed section word",
			chart:     "[Chorus]",
			wantValid: true,
		},
		{
			name:           "empty bracket rejected",
			chart:          "[]",
			wantValid:      false,
			wantErrContain: "invalid bracket content",
		},
		{
			name:           "numeric-only bracket rejected",
			chart:          "[1]",
			wantValid:      false,
			wantErrContain: "invalid bracket content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := v.ValidateChart(tt.chart)
			if valid != tt.wantValid {
				t.Errorf("ValidateChart() valid = %v, want %v", valid, tt.wantValid)
			}
			if tt.wantValid {
				if err != nil {
					t.Errorf("ValidateChart() unexpected error for valid chart: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("ValidateChart() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrContain) {
				t.Errorf("ValidateChart() error %q does not contain %q", err.Error(), tt.wantErrContain)
			}
		})
	}
}

func TestValidateChartXUnderscoreIgnored(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() unexpected error: %v", err)
	}
	valid, err := v.ValidateChart("{x_mspro_pedal: 4}\n{title: Song}\nSwing [D]low")
	if !valid {
		t.Errorf("ValidateChart() valid = false for x_ directive, want true (spec: ignore silently); err=%v", err)
	}
}

func TestValidateChartEmpty(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() unexpected error: %v", err)
	}

	valid, err := v.ValidateChart("")
	if valid {
		t.Error("ValidateChart(\"\") valid = true, want false")
	}
	if err == nil {
		t.Fatal("ValidateChart(\"\") expected error, got nil")
	}
	if !strings.Contains(err.Error(), "chart is empty") {
		t.Errorf("ValidateChart(\"\") error %q does not contain %q", err.Error(), "chart is empty")
	}
}

func TestValidateChartAllowUnknown(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() unexpected error: %v", err)
	}
	v.AllowUnkDirectives = true

	chart := "{custom_directive: hello}\n[G] custom stuff"
	valid, err := v.ValidateChart(chart)
	if !valid {
		t.Errorf("ValidateChart() valid = false with AllowUnkDirectives, want true")
	}
	if err != nil {
		t.Errorf("ValidateChart() unexpected error with AllowUnkDirectives: %v", err)
	}
}

func TestValidateChartCurrentBehaviorGaps(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() unexpected error: %v", err)
	}

	behavior := []struct {
		name      string
		chart     string
		wantValid bool
	}{
		{
			name:      "directive without colon passes",
			chart:     "{title}",
			wantValid: true,
		},
		{
			name:      "unknown directive with leading text bypasses check",
			chart:     "hello {invalid: x} world",
			wantValid: true,
		},
		{
			name:      "lone closing brace passes",
			chart:     "just a } brace",
			wantValid: true,
		},
		{
			name:      "unmatched soc without eoc passes",
			chart:     "{start_of_chorus}\nSwing [D]low.",
			wantValid: true,
		},
		{
			name:      "multiple directives on one line fail",
			chart:     "{a} {b}",
			wantValid: false,
		},
		{
			name:      "directive line with trailing text bypasses check",
			chart:     "{invalid: x} trailing",
			wantValid: true,
		},
	}

	for _, tt := range behavior {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := v.ValidateChart(tt.chart)
			if valid != tt.wantValid {
				t.Errorf("ValidateChart() valid = %v, want %v", valid, tt.wantValid)
			}
			if tt.wantValid && err != nil {
				t.Errorf("ValidateChart() unexpected error: %v", err)
			}
		})
	}
}

func TestBracketsBalanced(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"no brackets", "Swing low", true},
		{"single chord", "[D]", true},
		{"multiple chords", "[D][G][A7]", true},
		{"chords and lyrics", "Swing [D]low, sweet [G]chariot", true},
		{"empty chord", "[]", true},
		{"unclosed opening", "[D", false},
		{"lone closing", "D]", false},
		{"closing before opening", "][D]", false},
		{"extra closing", "[D]]", false},
		{"nested brackets", "[[D]]", false},
		{"two openings", "[[D]", false},
		{"open only", "[", false},
		{"close only", "]", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bracketsBalanced(tt.in); got != tt.want {
				t.Errorf("bracketsBalanced(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
