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
	got := layoutColumns(lines, nil, nil, 83, 3, dividerGlyph)

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
	lines := []string{"l1", "l2", "l3", "l4", "r1", "r2"}
	got := layoutColumns(lines, nil, nil, 83, 4, dividerGlyph)
	if len(got) != 4 {
		t.Fatalf("rows = %d, want 4", len(got))
	}
	for i, row := range got {
		if !strings.Contains(row, " ║ ") {
			t.Errorf("row %d = %q missing divider (frame must stay complete)", i, row)
		}
	}
	checks := []struct{ left, right string }{{"l1", "r1"}, {"l2", "r2"}, {"l3", ""}, {"l4", ""}}
	for i, want := range checks {
		parts := strings.SplitN(got[i], " ║ ", 2)
		if len(parts) != 2 {
			t.Fatalf("row %d = %q does not split on divider", i, got[i])
		}
		if !strings.Contains(parts[0], want.left) {
			t.Errorf("row %d left = %q, want it to contain %q", i, parts[0], want.left)
		}
		if want.right == "" {
			if strings.TrimSpace(parts[1]) != "" {
				t.Errorf("row %d right = %q, want blank filler", i, parts[1])
			}
		} else if !strings.Contains(parts[1], want.right) {
			t.Errorf("row %d right = %q, want it to contain %q", i, parts[1], want.right)
		}
	}
}

func TestLayoutColumnsPadsToEqualWidth(t *testing.T) {
	got := layoutColumns([]string{"x", "y"}, nil, nil, 83, 2, dividerGlyph)
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
	lines := append([]string{tab}, []string{"a", "b", "c", "d"}...)
	got := layoutColumns(lines, nil, nil, 83, 3, dividerGlyph)
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "E|---3-3-4") || !strings.Contains(joined, "-4|---8-8-") {
		t.Errorf("wrapped tab content lost:\n%s", joined)
	}
}

func TestLayoutColumnsStyledLinesKeepDividerAligned(t *testing.T) {
	styled := "\x1b[36;1m[chorus]\x1b[0m" // 7 printable cells, 17 runes
	lines := []string{"plain lyric line here", styled, "another plain line", "more content", "even more", "last line"}
	got := layoutColumns(lines, nil, nil, 83, 3, dividerGlyph)

	first := -1
	for i, row := range got {
		d := strings.Index(row, dividerGlyph)
		if d < 0 {
			t.Fatalf("row %d = %q missing divider", i, row)
		}
		// The divider must sit at the same DISPLAY column on every row —
		// styled rows carry invisible escape runes, so compare cells, not runes.
		w := displayWidth(row[:d])
		if first < 0 {
			first = w
			continue
		}
		if w != first {
			t.Errorf("row %d divider at display column %d, want %d (styled row shifted it)\nrow: %q", i, w, first, row)
		}
	}
	if !strings.Contains(got[1], styled) {
		t.Errorf("styled line lost its escape codes:\n%s", strings.Join(got, "\n"))
	}
}

func TestLayoutColumnsShortContentStaysSingle(t *testing.T) {
	lines := []string{"a", "b", "c"}
	if got := layoutColumns(lines, nil, nil, 83, 3, dividerGlyph); got != nil {
		t.Errorf("content fitting one column must not split, got %#v", got)
	}
}

func TestLayoutColumnsNarrowFallback(t *testing.T) {
	if got := layoutColumns([]string{"a"}, nil, nil, 40, 10, dividerGlyph); got != nil {
		t.Errorf("layoutColumns(40) = %#v, want nil fallback", got)
	}
}

func TestLayoutColumnsDoesNotSplitPairAtDivider(t *testing.T) {
	lines := []string{"a", "b", "cc", "ly", "d", "e"}
	keep := []bool{false, false, true, false, false, false} // cc↔ly pair
	got := layoutColumns(lines, keep, nil, 83, 3, dividerGlyph)
	if len(got) != 3 {
		t.Fatalf("rows = %d, want 3", len(got))
	}
	// The pair (cc, ly) must both land in the right column, not straddle the
	// divider: cc is the first right-column row, ly the second.
	parts0 := strings.SplitN(got[0], " ║ ", 2)
	parts1 := strings.SplitN(got[1], " ║ ", 2)
	if !strings.Contains(parts0[1], "cc") {
		t.Errorf("row0 right = %q, want chord cc pushed into the right column", parts0[1])
	}
	if !strings.Contains(parts1[1], "ly") {
		t.Errorf("row1 right = %q, want lyric ly directly under its chord in the right column", parts1[1])
	}
}

func TestLayoutColumnsDoesNotEndPageOnPairStart(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e", "f", "g"}
	keep := []bool{false, false, false, false, false, true, false} // f↔g pair
	got := layoutColumns(lines, keep, nil, 83, 3, dividerGlyph)
	for _, row := range got {
		if strings.Contains(row, "f") {
			t.Errorf("row %q ends a page on dangling chord f; pair must move to the next page", row)
		}
	}
}

func TestLayoutColumnsDoesNotSplitTabBlockAtDivider(t *testing.T) {
	// A tab block (T0-T2) that fits one column must not straddle the divider:
	// the whole block lands in the right column.
	lines := []string{"a", "b", "T0", "T1", "T2", "d"}
	keepTab := []bool{false, false, true, true, false, false} // T0..T2 chained
	got := layoutColumns(lines, nil, keepTab, 83, 3, dividerGlyph)
	if len(got) != 3 {
		t.Fatalf("rows = %d, want 3", len(got))
	}
	parts0 := strings.SplitN(got[0], " ║ ", 2)
	parts1 := strings.SplitN(got[1], " ║ ", 2)
	parts2 := strings.SplitN(got[2], " ║ ", 2)
	if !strings.Contains(parts0[1], "T0") || !strings.Contains(parts1[1], "T1") || !strings.Contains(parts2[1], "T2") {
		t.Errorf("tab block not kept together in right column: %q / %q / %q", parts0[1], parts1[1], parts2[1])
	}
}

func TestLayoutColumnsTabBlockTallerThanColumnStillRenders(t *testing.T) {
	// A tab block taller than one column cannot be kept whole; the divider
	// split must be allowed rather than dropping the content.
	lines := []string{"a", "T0", "T1", "T2", "T3", "T4", "d"}
	keepTab := []bool{false, true, true, true, true, true, false} // T0..T4 chained (5 rows)
	got := layoutColumns(lines, nil, keepTab, 83, 3, dividerGlyph)
	if len(got) != 3 {
		t.Fatalf("rows = %d, want 3", len(got))
	}
	found := false
	for _, row := range got {
		if strings.Contains(row, "T0") || strings.Contains(row, "T1") {
			found = true
		}
	}
	if !found {
		t.Errorf("over-tall tab block dropped entirely: %#v", got)
	}
}

func TestLayoutColumnsHeaderWithFirstRowNotStraddling(t *testing.T) {
	// A section header with its first chord/lyric row (a run) must not
	// straddle the divider: the whole unit moves to the right column.
	lines := []string{"a", "b", "[S]", "Cc", "Ly", "d", "e"}
	keep := []bool{false, false, true, true, false, false, false} // [S]→Cc→Ly
	got := layoutColumns(lines, keep, nil, 83, 3, dividerGlyph)
	if len(got) != 3 {
		t.Fatalf("rows = %d, want 3", len(got))
	}
	parts0 := strings.SplitN(got[0], " ║ ", 2)
	parts1 := strings.SplitN(got[1], " ║ ", 2)
	parts2 := strings.SplitN(got[2], " ║ ", 2)
	if !strings.Contains(parts0[1], "[S]") || !strings.Contains(parts1[1], "Cc") || !strings.Contains(parts2[1], "Ly") {
		t.Errorf("header+first row not kept together in right column: %q / %q / %q", parts0[1], parts1[1], parts2[1])
	}
}

func TestLayoutColumnsSectionBodyMayStraddle(t *testing.T) {
	// Beyond the header + first row, a section may flow across the divider.
	lines := []string{"[S]", "Cc", "Ly", "d", "e", "f"}
	keep := []bool{true, true, false, false, false, false} // [S]→Cc→Ly run only
	got := layoutColumns(lines, keep, nil, 83, 3, dividerGlyph)
	if len(got) != 3 {
		t.Fatalf("rows = %d, want 3", len(got))
	}
	parts0 := strings.SplitN(got[0], " ║ ", 2)
	if !strings.Contains(parts0[0], "[S]") {
		t.Errorf("row0 left = %q, want the section header in the left column", parts0[0])
	}
	if !strings.Contains(parts0[1], "d") {
		t.Errorf("row0 right = %q, want the section body flowing into the right column", parts0[1])
	}
}
