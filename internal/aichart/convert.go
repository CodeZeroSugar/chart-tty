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

type ProgressEvent struct {
	Attempt     int
	MaxAttempts int
	Message     string
}

func (c Client) Convert(chart string) (Result, error) {
	return c.ConvertProgress(chart, nil)
}

func (c Client) ConvertProgress(chart string, onEvent func(ProgressEvent)) (Result, error) {
	emit := func(attempt int, msg string) {
		if onEvent != nil {
			onEvent(ProgressEvent{Attempt: attempt, MaxAttempts: MaxAttempts, Message: msg})
		}
	}

	res := Result{}
	system, _ := BuildPrompt(chart)
	user := chart

	for attempt := 1; attempt <= MaxAttempts; attempt++ {
		emit(attempt, "contacting model")
		out, err := c.Complete(system, user)
		if err != nil {
			res.Errors = append(res.Errors, err.Error())
			res.Attempts = attempt
			emit(attempt, fmt.Sprintf("model request failed: %v", err))
			continue
		}
		cleaned := StripCodeFences(out)

		emit(attempt, "validating output")
		valid, verr := validate(cleaned)
		if valid {
			res.Chart = cleaned
			res.Attempts = attempt
			emit(attempt, "chart validated")
			return res, nil
		}
		res.Errors = append(res.Errors, verr.Error())
		res.Attempts = attempt
		emit(attempt, fmt.Sprintf("invalid output, retrying: %v", verr))
		user = fmt.Sprintf("%s\n\nThe previous attempt failed validation with error: %v\nHere is the previous attempt; fix these problems and return the complete corrected chart:\n%s", chart, verr.Error(), cleaned)
	}

	return res, fmt.Errorf("AI conversion failed after %d attempts", MaxAttempts)
}

func validate(chart string) (bool, error) {
	v, err := parser.NewValidator()
	if err != nil {
		return false, err
	}
	// AI conversion always uses the strict chord grammar, regardless of the
	// user's strict/relaxed preference for their own charts.
	v.ChordMode = parser.StrictChords
	return v.ValidateChart(chart)
}
