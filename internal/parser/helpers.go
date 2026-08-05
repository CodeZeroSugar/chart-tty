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

func isTabLine(s string) bool {
	return strings.Contains(s, "--") || (strings.Contains(s, "|") && strings.Contains(s, "-"))
}

func DetectParserMode(valid bool, err error) ParserMode {
	if valid && err == nil {
		return ModeChordPro
	}
	return ModeBasic
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
	if slices.Contains(Spec.EnvironmentDirectives, directive) {
		return CategoryEnvironment
	}
	if slices.Contains(Spec.FormattingDirectives, directive) {
		return CategoryFormatting
	}
	if slices.Contains(Spec.MetaDirectives, directive) {
		return CategoryMeta
	}
	return CategoryUnknown
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
