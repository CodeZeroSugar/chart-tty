package aichart

import (
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	chart := "C  G\nSwing low"
	system, user := BuildPrompt(chart)

	// Embedded spec reference content (prompt core sections).
	for _, want := range []string{
		"File format basics",
		"## 2. Chords",
		"start_of_verse",
	} {
		if !strings.Contains(system, want) {
			t.Errorf("system prompt missing spec content %q", want)
		}
	}
	// Excluded sections must stay out of the prompt (gateway size constraint).
	for _, banned := range []string{"Chart-tty deltas", "## 5. Tab", "Sources"} {
		if strings.Contains(system, banned) {
			t.Errorf("system prompt must not include excluded section %q", banned)
		}
	}

	// Conversion wrapper rules.
	for _, want := range []string{
		"Output ONLY the converted chart",
		"Never truncate the song",
		"individually bracketed",
		"[*x2]",
		"[*Repeat intro]",
		"[*N.C.]",
		"Environments must be properly matched",
		"[Fm] [G#] [Eb] [Bb]",
		"{start_of_verse: Verse 1}",
		"chord-over-lyrics",
	} {
		if !strings.Contains(system, want) {
			t.Errorf("system prompt missing conversion rule %q", want)
		}
	}

	// Never-emit blacklist.
	for _, want := range []string{
		"{define:",
		"{transpose:",
		"{start_of_abc}",
		"x_",
		"Conditional directive selectors",
	} {
		if !strings.Contains(system, want) {
			t.Errorf("system prompt missing never-emit entry %q", want)
		}
	}

	if user != chart {
		t.Errorf("user prompt = %q, want raw chart", user)
	}
}

func TestStripCodeFences(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"chordpro fence", "```chordpro\n{title: X}\n```", "{title: X}"},
		{"plain fence", "```\n{title: X}\n```", "{title: X}"},
		{"no fence", "{title: X}", "{title: X}"},
		{"trailing whitespace", "  {title: X}  \n", "{title: X}"},
		{"fence with blank lines", "```chordpro\n\n{title: X}\n\n```", "{title: X}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripCodeFences(tt.in); got != tt.want {
				t.Errorf("StripCodeFences(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
