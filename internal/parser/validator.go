package parser

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

//go:embed spec.json
var embeddedSpec []byte

type ChordProSpec struct {
	MetaDirectives        []string `json:"meta_directives"`
	FormattingDirectives  []string `json:"formatting_directives"`
	EnvironmentDirectives []string `json:"environment_directives"`
}

type Validator struct {
	spec               ChordProSpec
	AllowUnkDirectives bool
}

func NewValidator() (*Validator, error) {
	var spec ChordProSpec
	if err := json.Unmarshal(embeddedSpec, &spec); err != nil {
		return nil, fmt.Errorf("failed to initialize Validator: %w", err)
	}
	if len(spec.EnvironmentDirectives) == 0 || len(spec.FormattingDirectives) == 0 || len(spec.MetaDirectives) == 0 {
		return nil, errors.New("embeded spec.json contains empty directive lists")
	}
	return &Validator{
			spec:               spec,
			AllowUnkDirectives: false,
		},
		nil
}

func (v *Validator) IsValidDirective(k string) bool {
	key := strings.ToLower(k)
	if v.AllowUnkDirectives {
		return true
	}
	return slices.Contains(v.spec.MetaDirectives, key) || slices.Contains(v.spec.FormattingDirectives, key) || slices.Contains(v.spec.EnvironmentDirectives, key)
}

func (v *Validator) ValidateChart(chart string) (bool, error) {
	if chart == "" {
		return false, errors.New("chart is empty, no data to validate")
	}

	lines := strings.Split(strings.ReplaceAll(chart, "\r\n", "\n"), "\n")

	for n, line := range lines {
		l := strings.TrimSpace(line)
		if strings.Contains(l, "{") && !strings.Contains(l, "}") {
			return false, fmt.Errorf("syntax error: unclosed directive on line %d", n+1)
		}
		if strings.HasPrefix(l, "{") && strings.HasSuffix(l, "}") {
			inner := l[1 : len(l)-1]
			inner = strings.TrimSpace(inner)

			parts := strings.SplitN(inner, ":", 2)
			directive := strings.TrimSpace(parts[0])

			if !v.IsValidDirective(directive) {
				return false, fmt.Errorf("directive error: invalid directive '%s' on line %d", l, n+1)
			}
		}

		if strings.ContainsAny(l, "[]") {
			if !bracketsBalanced(l) {
				return false, fmt.Errorf("syntax error: invalid brackets on line %d", n+1)
			}
		}
	}

	return true, nil
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
