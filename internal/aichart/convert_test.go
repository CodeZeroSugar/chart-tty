package aichart

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestConvertValidOnFirstAttempt(t *testing.T) {
	srv := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(jsonBody(validChart)))
	})
	cl := Client{BaseURL: srv.URL, Model: "m", HTTP: srv.Client()}

	res, err := cl.Convert("messy input")
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}
	if res.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", res.Attempts)
	}
	if len(res.Errors) != 0 {
		t.Errorf("Errors = %#v, want none", res.Errors)
	}
	if res.Chart != validChart {
		t.Errorf("Chart = %q, want %q", res.Chart, validChart)
	}
}

func TestConvertInvalidThenValid(t *testing.T) {
	calls := 0
	var retryUserPrompt string
	srv := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		var req chatRequest
		json.Unmarshal(body, &req)
		if calls == 1 {
			w.Write([]byte(jsonBody("[not a valid] chart")))
			return
		}
		retryUserPrompt = req.Messages[1].Content
		w.Write([]byte(jsonBody(validChart)))
	})
	cl := Client{BaseURL: srv.URL, Model: "m", HTTP: srv.Client()}

	res, err := cl.Convert("messy input")
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}
	if res.Attempts != 2 {
		t.Errorf("Attempts = %d, want 2", res.Attempts)
	}
	if len(res.Errors) != 1 {
		t.Errorf("Errors = %#v, want 1", res.Errors)
	}
	if res.Chart != validChart {
		t.Errorf("Chart = %q, want %q", res.Chart, validChart)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
	for _, want := range []string{"failed validation", "invalid bracket content", "[not a valid] chart", "messy input"} {
		if !strings.Contains(retryUserPrompt, want) {
			t.Errorf("retry prompt missing %q:\n%s", want, retryUserPrompt)
		}
	}
}

func TestConvertAlwaysInvalid(t *testing.T) {
	srv := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(jsonBody("[not a valid] chart")))
	})
	cl := Client{BaseURL: srv.URL, Model: "m", HTTP: srv.Client()}

	res, err := cl.Convert("messy input")
	if err == nil {
		t.Fatal("Convert() expected error")
	}
	if !strings.Contains(err.Error(), "3 attempts") {
		t.Errorf("error %q does not mention attempt limit", err.Error())
	}
	if res.Attempts != MaxAttempts {
		t.Errorf("Attempts = %d, want %d", res.Attempts, MaxAttempts)
	}
	if len(res.Errors) != MaxAttempts {
		t.Errorf("Errors len = %d, want %d", len(res.Errors), MaxAttempts)
	}
}

func TestConvertFencedValid(t *testing.T) {
	srv := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		body := "```chordpro\n" + validChart + "\n```"
		io.WriteString(w, jsonBody(body))
	})
	cl := Client{BaseURL: srv.URL, Model: "m", HTTP: srv.Client()}

	res, err := cl.Convert("messy")
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}
	if res.Chart != validChart {
		t.Errorf("Chart = %q, want %q (fences stripped)", res.Chart, validChart)
	}
}

func TestConvertProgressEvents(t *testing.T) {
	calls := 0
	srv := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Write([]byte(jsonBody("[not a valid] chart")))
			return
		}
		w.Write([]byte(jsonBody(validChart)))
	})
	cl := Client{BaseURL: srv.URL, Model: "m", HTTP: srv.Client()}

	var events []ProgressEvent
	res, err := cl.ConvertProgress("messy input", func(e ProgressEvent) {
		events = append(events, e)
	})
	if err != nil {
		t.Fatalf("ConvertProgress() unexpected error: %v", err)
	}
	if res.Attempts != 2 || res.Chart != validChart {
		t.Fatalf("result = %#v, want 2 attempts with valid chart", res)
	}

	if len(events) < 4 {
		t.Fatalf("events = %#v, want at least 4", events)
	}
	first, last := events[0], events[len(events)-1]
	if first.Attempt != 1 || first.Message != "contacting model" {
		t.Errorf("first event = %#v, want attempt 1 contacting model", first)
	}
	if last.Attempt != 2 || last.Message != "chart validated" {
		t.Errorf("last event = %#v, want attempt 2 chart validated", last)
	}
	retried := false
	for _, e := range events {
		if e.MaxAttempts != MaxAttempts {
			t.Errorf("event %#v has wrong MaxAttempts", e)
		}
		if strings.Contains(e.Message, "invalid output, retrying") {
			retried = true
		}
	}
	if !retried {
		t.Errorf("no retry event in %#v", events)
	}
}

func TestConvertStrictRejectsRelaxedChords(t *testing.T) {
	// Strict-only validation: a relaxed chord like [Coda] must be rejected and
	// retried, while the [*...] annotation form is accepted.
	calls := 0
	srv := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Write([]byte(jsonBody("{title: X}\n[Coda]")))
			return
		}
		w.Write([]byte(jsonBody(validChart)))
	})
	cl := Client{BaseURL: srv.URL, Model: "m", HTTP: srv.Client()}

	res, err := cl.Convert("messy")
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}
	if res.Attempts != 2 {
		t.Errorf("Attempts = %d, want 2 (relaxed [Coda] must fail strict validation once)", res.Attempts)
	}
	if res.Chart != validChart {
		t.Errorf("Chart = %q, want %q", res.Chart, validChart)
	}
}

func TestConvertNilCallbackStillWorks(t *testing.T) {
	srv := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(jsonBody(validChart)))
	})
	cl := Client{BaseURL: srv.URL, Model: "m", HTTP: srv.Client()}
	res, err := cl.ConvertProgress("messy", nil)
	if err != nil {
		t.Fatalf("ConvertProgress(nil) unexpected error: %v", err)
	}
	if res.Chart != validChart {
		t.Errorf("Chart = %q, want %q", res.Chart, validChart)
	}
}
