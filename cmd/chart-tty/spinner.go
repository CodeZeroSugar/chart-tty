package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// statusLine shows live progress on stderr during AI conversion. On a
// terminal it renders an animated braille spinner with elapsed time and the
// latest milestone; when stderr is piped it prints plain milestone lines.
type statusLine struct {
	mu       sync.Mutex
	message  string
	start    time.Time
	animated bool
	done     chan struct{}
	once     sync.Once
}

func newStatusLine(initial string) *statusLine {
	s := &statusLine{
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

func (s *statusLine) update(msg string) {
	s.mu.Lock()
	s.message = msg
	s.mu.Unlock()
	if !s.animated {
		fmt.Fprintf(os.Stderr, "AI conversion: %s\n", msg)
	}
}

func (s *statusLine) animate() {
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

func (s *statusLine) finish(final string) {
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
