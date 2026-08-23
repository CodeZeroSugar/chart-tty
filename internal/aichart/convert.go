package aichart

import (
	"fmt"

	"github.com/CodeZeroSugar/chart-tty/internal/parser"
)

const MaxAttempts = 3

type Result struct {
	Chart    string
	Attempts int
	Errors   []string
}

func (c Client) Convert(chart string) (Result, error) {
	res := Result{}
	system, _ := BuildPrompt(chart)
	user := chart

	for attempt := 1; attempt <= MaxAttempts; attempt++ {
		out, err := c.Complete(system, user)
		if err != nil {
			res.Errors = append(res.Errors, err.Error())
			res.Attempts = attempt
			continue
		}
		cleaned := StripCodeFences(out)

		valid, verr := validate(cleaned)
		if valid {
			res.Chart = cleaned
			res.Attempts = attempt
			return res, nil
		}
		res.Errors = append(res.Errors, verr.Error())
		res.Attempts = attempt
		user = fmt.Sprintf("%s\n\nThe previous attempt failed validation with error: %v\nHere is the previous attempt; fix these problems and return the complete corrected chart:\n%s", chart, verr.Error(), cleaned)
	}

	return res, fmt.Errorf("AI conversion failed after %d attempts", MaxAttempts)
}

func validate(chart string) (bool, error) {
	v, err := parser.NewValidator()
	if err != nil {
		return false, err
	}
	return v.ValidateChart(chart)
}