package ui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// StatusLine shows live AI-conversion progress directly on the terminal via
// stderr, decoupled from bubbletea's render loop. On a terminal it animates a
// braille spinner with elapsed time and the latest milestone; when stderr is
// piped it prints plain milestone lines. The TUI uses this same mechanism as
// the CLI so in-progress work is always visible.
type StatusLine struct {
	mu       sync.Mutex
	message  string
	start    time.Time
	animated bool
	done     chan struct{}
	once     sync.Once
}

func NewStatusLine(initial string) *StatusLine {
	s := &StatusLine{
		message:  initial,
		start:    time.Now(),
		animated: term.IsTerminal(int(os.Stderr.Fd())),
		done:     make(chan struct{}),
	}
	if s.animated {
		go s.animate()
	}
	return s
}

func (s *StatusLine) Update(msg string) {
	s.mu.Lock()
	s.message = msg
	s.mu.Unlock()
	if !s.animated {
		fmt.Fprintf(os.Stderr, "AI conversion: %s\n", msg)
	}
}

func (s *StatusLine) animate() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for i := 0; ; i++ {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			frame := spinnerFrames[i%len(spinnerFrames)]
			elapsed := time.Since(s.start).Round(time.Second)
			fmt.Fprintf(os.Stderr, "\r\033[K%s converting… %s (%s)", frame, s.message, elapsed)
			s.mu.Unlock()
		}
	}
}

// Finish stops the spinner and prints the final status line once.
func (s *StatusLine) Finish(final string) {
	s.once.Do(func() {
		close(s.done)
		elapsed := time.Since(s.start).Round(time.Second)
		if s.animated {
			fmt.Fprintf(os.Stderr, "\r\033[K%s (%s)\n", final, elapsed)
		} else {
			fmt.Fprintf(os.Stderr, "AI conversion: %s (%s)\n", final, elapsed)
		}
	})
}
