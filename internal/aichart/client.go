package aichart

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/CodeZeroSugar/chart-tty/internal/config"
)

type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

func FromConfig(c config.Config) Client {
	cl := Client{
		BaseURL: c.AI.BaseURL,
		APIKey:  c.AI.APIKey,
		Model:   c.AI.Model,
		HTTP:    ipv4HTTPClient(),
	}
	if v := os.Getenv("CHART_TTY_BASE_URL"); v != "" {
		cl.BaseURL = v
	}
	if v := os.Getenv("CHART_TTY_API_KEY"); v != "" {
		cl.APIKey = v
	}
	if v := os.Getenv("CHART_TTY_MODEL"); v != "" {
		cl.Model = v
	}
	return cl
}

// ipv4HTTPClient returns an HTTP client whose dialer only uses IPv4. Some
// networks route IPv6 to a broken path that answers empty 404s (observed for
// the Google Gemini API), and Go's resolver prefers IPv6 when DNS lists AAAA
// records first — so requests can silently fail without ever falling back.
func ipv4HTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, "tcp4", addr)
			},
		},
	}
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	// NOTE: intentionally no temperature/max_tokens fields. The OpenCode Go
	// gateway hangs (>9 min, zero bytes) on requests carrying them; without
	// them the same request completes in ~2 minutes. See docs/LESSONS.md.
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (c Client) Complete(system, user string) (string, error) {
	reqBody, err := json.Marshal(chatRequest{
		Model:    c.Model,
		Messages: []chatMessage{{Role: "system", Content: system}, {Role: "user", Content: user}},
	})
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}

	url := strings.TrimSuffix(c.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("AI request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("AI endpoint returned status %d: %s", resp.StatusCode, b)
	}

	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return "", fmt.Errorf("decoding AI response: %w", err)
	}
	if len(cr.Choices) == 0 || cr.Choices[0].Message.Content == "" {
		return "", errors.New("AI returned no content")
	}
	return cr.Choices[0].Message.Content, nil
}
