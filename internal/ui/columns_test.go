package ui

import (
	"reflect"
	"strings"
	"testing"
)

func TestShouldUseColumns(t *testing.T) {
	tests := []struct {
		name  string
		width int
		want  bool
	}{
		{"narrow terminal", 40, false},
		{"just under threshold", 82, false},
		{"at threshold", 83, true},
		{"wide", 160, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldUseColumns(tt.width); got != tt.want {
				t.Errorf("shouldUseColumns(%d) = %v, want %v", tt.width, got, tt.want)
			}
		})
	}
}

func TestExpandRows(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		colWidth int
		want     []string
	}{
		{
			name:     "short lines pass through",
			lines:    []string{"abc", "de"},
			colWidth: 5,
			want:     []string{"abc", "de"},
		},
		{
			name:     "empty line becomes one blank row",
			lines:    []string{""},
			colWidth: 4,
			want:     []string{""},
		},
		{
			name:     "exact fit not wrapped",
			lines:    []string{"12345"},
			colWidth: 5,
			want:     []string{"12345"},
		},
		{
			name:     "long line wraps",
			lines:    []string{"E|---3-3-4-4|---8-8-6-6"},
			colWidth: 10,
			want:     []string{"E|---3-3-4", "-4|---8-8-", "6-6"},
		},
		{
			name:     "mixed lengths",
			lines:    []string{"short", "averyverylongline", "ok"},
			colWidth: 6,
			want:     []string{"short", "averyv", "erylon", "gline", "ok"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandRows(tt.lines, tt.colWidth)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expandRows() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestLayoutColumnsFlowOrder(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e", "f"}
	got := layoutColumns(lines, 83, 3, dividerGlyph)

	if len(got) != 3 {
		t.Fatalf("rows = %d, want 3", len(got))
	}
	for row, want := range []struct{ left, right string }{{"a", "d"}, {"b", "e"}, {"c", "f"}} {
		if !strings.Contains(got[row], want.left+" ") && !strings.HasPrefix(got[row], want.left) {
			t.Errorf("row %d = %q, missing left %q", row, got[row], want.left)
		}
		if !strings.HasSuffix(strings.TrimRight(got[row], " "), want.right) {
			t.Errorf("row %d = %q, missing right %q", row, got[row], want.right)
		}
		if !strings.Contains(got[row], " ║ ") {
			t.Errorf("row %d = %q, missing divider", row, got[row])
		}
	}
}

func TestLayoutColumnsDividerOnEveryRow(t *testing.T) {
	got := layoutColumns([]string{"only one line"}, 83, 4, dividerGlyph)
	if len(got) != 4 {
		t.Fatalf("rows = %d, want 4", len(got))
	}
	for i, row := range got {
		if !strings.Contains(row, " ║ ") {
			t.Errorf("row %d = %q missing divider (frame must stay complete)", i, row)
		}
	}
	if !strings.Contains(got[0], "only one line") {
		t.Errorf("row 0 = %q missing content", got[0])
	}
}

func TestLayoutColumnsPadsToEqualWidth(t *testing.T) {
	got := layoutColumns([]string{"x", "y"}, 83, 2, dividerGlyph)
	for i, row := range got {
		if len([]rune(row)) != 83 {
			t.Errorf("row %d width = %d, want 83", i, len([]rune(row)))
		}
		parts := strings.Split(row, " ║ ")
		if len(parts) != 2 {
			t.Fatalf("row %d = %q does not split on divider", i, row)
		}
		if len([]rune(parts[0])) != 40 || len([]rune(parts[1])) != 40 {
			t.Errorf("row %d column widths = %d/%d, want 40/40", i, len([]rune(parts[0])), len([]rune(parts[1])))
		}
	}
}

func TestLayoutColumnsWrapsLongLines(t *testing.T) {
	tab := "E|---3-3-4-4|---8-8-6-6|-------3-3-4-4"
	got := layoutColumns([]string{tab}, 83, 3, dividerGlyph)
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "E|---3-3-4-") || !strings.Contains(joined, "4|---8-8-6-") {
		t.Errorf("wrapped tab content lost:\n%s", joined)
	}
}

func TestLayoutColumnsNarrowFallback(t *testing.T) {
	if got := layoutColumns([]string{"a"}, 40, 10, dividerGlyph); got != nil {
		t.Errorf("layoutColumns(40) = %#v, want nil fallback", got)
	}
}