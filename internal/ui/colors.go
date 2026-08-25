package ui

import "strings"

// namedColors maps the color names users write in the config to the ANSI
// color codes lipgloss accepts. lipgloss.Color() only understands hex
// ("#ff8800") and numeric ("196", "6") values — a named string like "cyan"
// resolves to a nil color and renders nothing. This bridges the two.
var namedColors = map[string]string{
	"black":         "0",
	"red":           "1",
	"green":         "2",
	"yellow":        "3",
	"blue":          "4",
	"magenta":       "5",
	"cyan":          "6",
	"white":         "7",
	"gray":          "8",
	"grey":          "8",
	"brightblack":   "8",
	"brightred":     "9",
	"brightgreen":   "10",
	"brightyellow":  "11",
	"brightblue":    "12",
	"brightmagenta": "13",
	"brightcyan":    "14",
	"brightwhite":   "15",
}

// resolveColor maps a config color value to a value lipgloss.Color
// understands: named colors become ANSI codes, hex and numeric values pass
// through unchanged, and anything unknown passes through as-is (lipgloss
// then renders no color, matching an invalid input).
func resolveColor(s string) string {
	if code, ok := namedColors[strings.ToLower(s)]; ok {
		return code
	}
	return s
}
