package aichart

import (
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	chart := "C  G\nSwing low"
	system, user := BuildPrompt(chart)

	if !strings.Contains(system, "ChordPro") {
		t.Error("system prompt does not mention ChordPro")
	}
	if !strings.Contains(system, "start_of_verse") {
		t.Error("system prompt lacks directive rules")
	}
	if !strings.Contains(system, "[*N.C.]") {
		t.Error("system prompt lacks asterisk escape rule")
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