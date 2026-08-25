package parser

import (
	"regexp"
	"slices"
	"strings"
)

// minTabStringLines is how many string-note lines a {sot} block must contain
// before it counts as real tablature.
const minTabStringLines = 4

var stringNoteLine = regexp.MustCompile(`^[A-Ga-g][ \t]*\|`)

// isStringNoteLine reports whether a line leads with a musical letter and a
// pipe, e.g. "E|------". The pipe is mandatory; dash count is irrelevant.
func isStringNoteLine(line string) bool {
	return stringNoteLine.MatchString(strings.TrimSpace(line))
}

// isRealTabBlock reports whether the raw lines of a {sot} block contain at
// least minTabStringLines string-note lines. Blocks that fail are decorative
// line breaks pretending to be tabs.
func isRealTabBlock(lines []string) bool {
	count := 0
	for _, l := range lines {
		if isStringNoteLine(l) {
			count++
		}
	}
	return count >= minTabStringLines
}

func bracketsBalanced(l string) bool {
	inBrackets := false
	for _, char := range l {
		switch char {
		case '[':
			if inBrackets {
				return false
			}
			inBrackets = true
		case ']':
			if !inBrackets {
				return false
			}
			inBrackets = false
		}
	}
	return !inBrackets
}

func extractBracketContents(line string) []string {
	var contents []string
	inBracket := false
	var buf []rune

	for _, r := range line {
		switch r {
		case '[':
			inBracket = true
			buf = buf[:0]
		case ']':
			if inBracket {
				inBracket = false
				contents = append(contents, string(buf))
			}
		default:
			if inBracket {
				buf = append(buf, r)
			}
		}
	}
	return contents
}

// validateBracketContent reports whether bracket content is acceptable under
// the given chord mode. A leading '*' marks an annotation and is always
// accepted. Strict mode requires the strict grammar; relaxed mode also
// accepts spec relaxed chords (valid root + any non-empty extension tail).
func validateBracketContent(raw string, mode ChordMode) bool {
	r := strings.TrimSpace(raw)
	if r == "" {
		return false
	}
	if r[0] == '*' {
		return true
	}
	if strictChordRe.MatchString(r) {
		return true
	}
	return mode == RelaxedChords && relaxedChordRe.MatchString(r)
}

func isTabLine(s string) bool {
	return strings.Contains(s, "--") || (strings.Contains(s, "|") && strings.Contains(s, "-"))
}

func isChordLine(line string, mode ChordMode) bool {
	l := strings.TrimSpace(line)
	if l == "" {
		return false
	}
	for _, tok := range strings.Fields(l) {
		if !validateBracketContent(tok, mode) {
			return false
		}
	}
	return true
}

func extractBasicChords(line string, mode ChordMode) []ChordToken {
	var out []ChordToken
	i := 0
	for i < len(line) {
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		if i >= len(line) {
			break
		}
		start := i
		for i < len(line) && line[i] != ' ' && line[i] != '\t' {
			i++
		}
		if tok := line[start:i]; validateBracketContent(tok, mode) {
			out = append(out, ChordToken{Name: tok, Position: start})
		}
	}
	return out
}

func DetectParserMode(valid bool, err error) ParserMode {
	if valid && err == nil {
		return ModeChordPro
	}
	return ModeBasic
}

func LooksLikeBasicChart(chart string, mode ChordMode) bool {
	sawChordLine := false
	for _, line := range strings.Split(chart, "\n") {
		l := strings.TrimSpace(line)
		if strings.Contains(l, "{") || strings.Contains(l, "[") {
			return false
		}
		if isChordLine(l, mode) {
			sawChordLine = true
		}
	}
	return sawChordLine
}

// stripSelector removes a conditional directive selector per spec:
// "{title-soprano}" is the title directive selected for soprano. No canonical
// directive name contains a hyphen, so cutting at the first one is safe.
func stripSelector(name string) string {
	if i := strings.IndexByte(name, '-'); i >= 0 {
		return name[:i]
	}
	return name
}

func extractDirective(directiveLine string) (directive string, data string) {
	inner := directiveLine[1 : len(directiveLine)-1]
	inner = strings.TrimSpace(inner)

	parts := strings.SplitN(inner, ":", 2)

	if len(parts) > 1 {
		return stripSelector(strings.TrimSpace(parts[0])), strings.TrimSpace(strings.Join(parts[1:], " "))
	}

	// No colon: either a bare directive ({soc}) or the spec's bare/attribute
	// argument forms, equivalent to the colon form:
	//   {start_of_verse Verse 1}
	//   {start_of_verse label="Verse 1"}
	name := parts[0]
	if strings.Contains(name, " ") {
		tokens := strings.SplitN(name, " ", 2)
		data = strings.TrimSpace(tokens[1])
		data = strings.TrimPrefix(data, "label=")
		data = strings.Trim(strings.TrimSpace(data), "\"'")
		return stripSelector(strings.TrimSpace(tokens[0])), data
	}
	return stripSelector(name), ""
}

func getDirectiveCategory(directive string) DirectiveCategory {
	d := strings.ToLower(directive)
	// Spec-legal directives chart-tty does not implement are silently
	// skipped (checked first so e.g. start_of_abc stays ignored rather than
	// becoming a generic environment).
	if slices.Contains(Spec.IgnoredDirectives, d) {
		return CategoryIgnored
	}
	// Spec: arbitrary section names are allowed (letters, digits, underscores);
	// unknown environments are treated as part of the song lyrics.
	if strings.HasPrefix(d, "start_of_") || strings.HasPrefix(d, "end_of_") {
		return CategoryEnvironment
	}
	if slices.Contains(Spec.EnvironmentDirectives, d) {
		return CategoryEnvironment
	}
	if slices.Contains(Spec.FormattingDirectives, d) {
		return CategoryFormatting
	}
	if slices.Contains(Spec.MetaDirectives, d) {
		return CategoryMeta
	}
	return CategoryUnknown
}

func isCommentDirective(directive string) bool {
	switch strings.ToLower(directive) {
	case "comment", "c", "comment_italic", "ci", "comment_box", "cb":
		return true
	}
	return false
}

func parseEnvDirective(directive string) (actionType string, envName string, ok bool) {
	d := strings.ToLower(strings.TrimSpace(directive))

	switch {
	case strings.HasPrefix(d, "start_of_"):
		return "start", strings.TrimPrefix(d, "start_of_"), true
	case strings.HasPrefix(d, "end_of_"):
		return "end", strings.TrimPrefix(d, "end_of_"), true
	case strings.HasPrefix(d, "so") && len(d) == 3:
		return "start", resolveAlias(d[2:]), true
	case strings.HasPrefix(d, "eo") && len(d) == 3:
		return "end", resolveAlias(d[2:]), true
	case d == "chorus":
		return "start", "chorus", true
	}
	return "", "", false
}

func resolveAlias(char string) string {
	switch char {
	case "v":
		return "verse"
	case "c":
		return "chorus"
	case "b":
		return "bridge"
	case "t":
		return "tab"
	case "g":
		return "grid"
	default:
		return char
	}
}
