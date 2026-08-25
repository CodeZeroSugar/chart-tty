package aichart

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CodeZeroSugar/chart-tty/internal/config"
)

const validChart = "{title: Converted}\n{start_of_chorus}\nSwing [D]low\n{eoc}"

func jsonBody(content string) string {
	b, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{
			{"message": map[string]any{"content": content}},
		},
	})
	return string(b)
}

func mockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func TestComplete(t *testing.T) {
	var gotModel string
	var gotAuth string
	var gotSystem string
	var gotUser string

	srv := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want /chat/completions", r.URL.Path)
		}
		if gotAuth = r.Header.Get("Authorization"); gotAuth == "" {
			t.Error("missing Authorization header")
		}
		body, _ := io.ReadAll(r.Body)
		var req chatRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("bad request body: %v", err)
		}
		gotModel = req.Model
		gotSystem = req.Messages[0].Content
		gotUser = req.Messages[1].Content
		w.Write([]byte(jsonBody(validChart)))
	})

	cl := Client{BaseURL: srv.URL, APIKey: "sekret", Model: "test-model", HTTP: srv.Client()}
	out, err := cl.Complete("sys", "usr")
	if err != nil {
		t.Fatalf("Complete() unexpected error: %v", err)
	}
	if out != validChart {
		t.Errorf("Complete() = %q, want %q", out, validChart)
	}
	if gotModel != "test-model" {
		t.Errorf("model = %q, want test-model", gotModel)
	}
	if !strings.HasPrefix(gotAuth, "Bearer ") {
		t.Errorf("auth = %q, want Bearer prefix", gotAuth)
	}
	if gotSystem != "sys" || gotUser != "usr" {
		t.Errorf("messages = %q / %q, want sys/usr", gotSystem, gotUser)
	}
}

func TestCompleteHTTPError(t *testing.T) {
	srv := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	cl := Client{BaseURL: srv.URL, Model: "m", HTTP: srv.Client()}
	if _, err := cl.Complete("s", "u"); err == nil {
		t.Fatal("Complete() expected error on HTTP 500")
	}
}

func TestCompleteNoChoices(t *testing.T) {
	srv := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[]}`))
	})
	cl := Client{BaseURL: srv.URL, Model: "m", HTTP: srv.Client()}
	if _, err := cl.Complete("s", "u"); err == nil {
		t.Fatal("Complete() expected error on empty choices")
	}
}

func TestCompleteMalformedJSON(t *testing.T) {
	srv := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	})
	cl := Client{BaseURL: srv.URL, Model: "m", HTTP: srv.Client()}
	if _, err := cl.Complete("s", "u"); err == nil {
		t.Fatal("Complete() expected error on malformed JSON")
	}
}

func TestFromConfigEnvOverrides(t *testing.T) {
	t.Setenv("CHART_TTY_BASE_URL", "http://env.example/v1")
	t.Setenv("CHART_TTY_API_KEY", "env-key")
	t.Setenv("CHART_TTY_MODEL", "env-model")

	cfg := config.Default()
	cl := FromConfig(cfg)
	if cl.BaseURL != "http://env.example/v1" {
		t.Errorf("BaseURL = %q, want env override", cl.BaseURL)
	}
	if cl.APIKey != "env-key" {
		t.Errorf("APIKey = %q, want env override", cl.APIKey)
	}
	if cl.Model != "env-model" {
		t.Errorf("Model = %q, want env override", cl.Model)
	}
}

func TestFromConfigDefaults(t *testing.T) {
	cfg := config.Default()
	cl := FromConfig(cfg)
	if cl.BaseURL != cfg.AI.BaseURL || cl.Model != cfg.AI.Model {
		t.Errorf("client = %#v, want config defaults", cl)
	}
}

func TestFromConfigUsesIPv4Dialer(t *testing.T) {
	cl := FromConfig(config.Default())
	tr, ok := cl.HTTP.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client transport = %T, want *http.Transport", cl.HTTP.Transport)
	}
	if tr.DialContext == nil {
		t.Fatal("transport has no DialContext (IPv4-forcing dialer not wired)")
	}
}
