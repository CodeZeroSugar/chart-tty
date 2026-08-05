package parser

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var chordRegex = regexp.MustCompile(`^[A-G][b#]?(m|maj|min|dim|aug|sus|add)?[0-9]*(\/[A-G][b#]?)?$`)

type Validator struct {
	AllowUnkDirectives bool
}

func NewValidator() (*Validator, error) {
	return &Validator{
			AllowUnkDirectives: false,
		},
		nil
}

func (v *Validator) IsValidDirective(k string) bool {
	key := strings.ToLower(k)
	if v.AllowUnkDirectives {
		return true
	}
	return slices.Contains(Spec.MetaDirectives, key) || slices.Contains(Spec.FormattingDirectives, key) || slices.Contains(Spec.EnvironmentDirectives, key)
}

func (v *Validator) ValidateChart(chart string) (bool, error) {
	if chart == "" {
		return false, errors.New("chart is empty, no data to validate")
	}

	lines := strings.Split(strings.ReplaceAll(chart, "\r\n", "\n"), "\n")

	for n, line := range lines {
		l := strings.TrimSpace(line)
		if strings.Contains(l, "{") && !strings.Contains(l, "}") {
			return false, fmt.Errorf("syntax error: unclosed directive on line %d\n", n+1)
		}
		if strings.HasPrefix(l, "{") && strings.HasSuffix(l, "}") {
			inner := l[1 : len(l)-1]
			inner = strings.TrimSpace(inner)

			parts := strings.SplitN(inner, ":", 2)
			directive := strings.TrimSpace(parts[0])

			if !v.IsValidDirective(directive) {
				return false, fmt.Errorf("directive error: invalid directive '%s' on line %d\n", l, n+1)
			}
		}

		if strings.ContainsAny(l, "[]") {
			if !bracketsBalanced(l) {
				return false, fmt.Errorf("syntax error: invalid brackets on line %d\n", n+1)
			}
			bracketContent := extractBracketContents(l)
			for _, bc := range bracketContent {
				if !validateBracketContent(bc) {
					return false, fmt.Errorf("syntax error: invalid bracket content on line %d\n", n+1)
				}
			}
		}
	}

	return true, nil
}
