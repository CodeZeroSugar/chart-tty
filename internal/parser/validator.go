package parser

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type Validator struct {
	AllowUnkDirectives bool
	ChordMode          ChordMode
}

func NewValidator() (*Validator, error) {
	return &Validator{
			AllowUnkDirectives: false,
			ChordMode:          defaultChordMode,
		},
		nil
}

func (v *Validator) IsValidDirective(k string) bool {
	key := strings.ToLower(stripSelector(strings.TrimSpace(k)))
	if v.AllowUnkDirectives {
		return true
	}
	// Per spec, custom x_ directives must be completely ignored by
	// applications that do not handle them — no warning, no error.
	if strings.HasPrefix(key, "x_") {
		return true
	}
	// Spec-legal directives chart-tty does not implement pass validation and
	// are skipped at parse time, so charts using them stay in ChordPro mode.
	if slices.Contains(Spec.IgnoredDirectives, key) {
		return true
	}
	// Spec: arbitrary section names (letters, digits, underscores) are legal.
	if strings.HasPrefix(key, "start_of_") || strings.HasPrefix(key, "end_of_") {
		return true
	}
	return slices.Contains(Spec.MetaDirectives, key) || slices.Contains(Spec.FormattingDirectives, key) || slices.Contains(Spec.EnvironmentDirectives, key)
}

func (v *Validator) ValidateChart(chart string) (bool, error) {
	if chart == "" {
		return false, errors.New("chart is empty, no data to validate")
	}

	lines := strings.Split(strings.ReplaceAll(chart, "\r\n", "\n"), "\n")

	var currentEnv string
	for n, line := range lines {
		l := strings.TrimSpace(line)
		if strings.Contains(l, "{") && !strings.Contains(l, "}") {
			return false, fmt.Errorf("syntax error: unclosed directive on line %d", n+1)
		}
		directiveLine := strings.HasPrefix(l, "{") && strings.HasSuffix(l, "}")
		if directiveLine {
			directive, _ := extractDirective(l)
			if !v.IsValidDirective(directive) {
				return false, fmt.Errorf("directive error: invalid directive '%s' on line %d", l, n+1)
			}
			// Environment matching is case-insensitive and covers both the
			// standard environments and arbitrary start_of_*/end_of_* names.
			d := strings.ToLower(directive)
			if slices.Contains(Spec.EnvironmentDirectives, d) || strings.HasPrefix(d, "start_of_") || strings.HasPrefix(d, "end_of_") {
				action, name, _ := parseEnvDirective(d)
				switch action {
				case "start":
					currentEnv = name
				case "end":
					if name != currentEnv {
						return false, fmt.Errorf("directive error: end of current environment not found: %s", currentEnv)
					}
					currentEnv = ""
				}
			}
			// Brackets inside a directive's arguments are text, not chords.
			continue
		}

		if strings.ContainsAny(l, "[]") {
			if !bracketsBalanced(l) {
				return false, fmt.Errorf("syntax error: invalid brackets on line %d", n+1)
			}
			bracketContent := extractBracketContents(l)
			for _, bc := range bracketContent {
				if !validateBracketContent(bc, v.ChordMode) {
					return false, fmt.Errorf("syntax error: invalid bracket content on line %d", n+1)
				}
			}
		}
	}

	if currentEnv != "" {
		return false, fmt.Errorf("directive error: unclosed environment: %s", currentEnv)
	}

	return true, nil
}
