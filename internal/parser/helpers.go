package parser

import (
	"slices"
	"strings"
)

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

func validateBracketContent(raw string) bool {
	r := strings.TrimSpace(raw)
	if r == "" {
		return false
	}
	if r[0] == '*' {
		return true
	}
	return chordRegex.MatchString(r)
}

func isTabLine(s string) bool {
	return strings.Contains(s, "--") || (strings.Contains(s, "|") && strings.Contains(s, "-"))
}

func isChordLine(line string) bool {
	l := strings.TrimSpace(line)
	if l == "" {
		return false
	}
	for _, tok := range strings.Fields(l) {
		if !validateBracketContent(tok) {
			return false
		}
	}
	return true
}

func extractBasicChords(line string) []ChordToken {
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
		if tok := line[start:i]; validateBracketContent(tok) {
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

func LooksLikeBasicChart(chart string) bool {
	sawChordLine := false
	for _, line := range strings.Split(chart, "\n") {
		l := strings.TrimSpace(line)
		if strings.Contains(l, "{") || strings.Contains(l, "[") {
			return false
		}
		if isChordLine(l) {
			sawChordLine = true
		}
	}
	return sawChordLine
}

func extractDirective(directiveLine string) (directive string, data string) {
	inner := directiveLine[1 : len(directiveLine)-1]
	inner = strings.TrimSpace(inner)

	parts := strings.SplitN(inner, ":", 2)

	if len(parts) > 1 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(strings.Join(parts[1:], " "))
	}
	return strings.TrimSpace(parts[0]), ""
}

func getDirectiveCategory(directive string) DirectiveCategory {
	d := strings.ToLower(directive)
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
