package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CodeZeroSugar/chart-tty/internal/config"
	"github.com/CodeZeroSugar/chart-tty/internal/parser"
)

type Model struct {
	lines     []string
	doc       *parser.Document
	transpose int
	offset    int
	width     int
	height    int
	title     string
	cfg       RenderConfig
	keys      config.KeyConfig
	quitting  bool
	showHelp  bool
}

func NewModel(lines []string, cfg RenderConfig) Model {
	return Model{lines: lines, cfg: cfg, keys: config.Default().Keys, showHelp: true}
}

func NewDocModel(doc *parser.Document, cfg RenderConfig) Model {
	m := NewModel(Render(doc, cfg), cfg)
	m.doc = doc
	m.title = doc.Title
	return m
}

func (m Model) SetKeys(keys config.KeyConfig) Model {
	m.keys = keys
	return m
}

func (m Model) SetTitle(title string) Model {
	m.title = title
	return m
}

func (m Model) SetShowHelp(show bool) Model {
	m.showHelp = show
	return m
}

func (m Model) SetTranspose(n int) Model {
	if m.doc == nil {
		return m
	}
	m.doc.Transpose(n)
	m.transpose += n
	m.lines = Render(m.doc, m.cfg)
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
case tea.KeyMsg:
		k := msg.String()
		switch {
		case k == "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case k == m.keys.Quit:
			m.quitting = true
			return m, tea.Quit
		case k == m.keys.ScrollUp || k == "up":
			m.offset--
		case k == m.keys.ScrollDown || k == "down":
			m.offset++
		case k == m.keys.TransposeUp:
			if m.doc != nil {
				m.doc.Transpose(1)
				m.transpose++
				m.lines = Render(m.doc, m.cfg)
			}
		case k == m.keys.TransposeDown:
			if m.doc != nil {
				m.doc.Transpose(-1)
				m.transpose--
				m.lines = Render(m.doc, m.cfg)
			}
		case k == "pgup":
			m.offset -= max(m.height, 1)
		case k == "pgdown" || k == " ":
			m.offset += max(m.height, 1)
		case k == "home" || k == "g":
			m.offset = 0
		case k == "end" || k == "G":
			m.offset = len(m.lines)
		}
	}
	m.clampOffset()
	return m, nil
}

func (m *Model) clampOffset() {
	if m.offset < 0 {
		m.offset = 0
	}
	maxOff := len(m.lines) - m.height
	if maxOff < 0 {
		maxOff = 0
	}
	if m.offset > maxOff {
		m.offset = maxOff
	}
}

func (m Model) View() string {
	var sb strings.Builder
	title := m.title
	if k := m.currentKey(); k != "" {
		title += " · Key: " + k
	}
	if title != "" {
		sb.WriteString(lipgloss.NewStyle().Bold(true).Render(title))
		sb.WriteString("\n")
	}
	top := m.offset
	bottom := m.offset + m.height
	if bottom > len(m.lines) {
		bottom = len(m.lines)
	}
	for i := top; i < bottom; i++ {
		if i > top {
			sb.WriteString("\n")
		}
		sb.WriteString(m.lines[i])
	}
	if m.showHelp {
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Faint(true).Render("j/k scroll · +/- transpose · q quit"))
	}
	return sb.String()
}

func (m Model) currentKey() string {
	if m.doc == nil || m.doc.Key == "" {
		return ""
	}
	c, err := parser.ParseChord(m.doc.Key)
	if err != nil {
		return m.doc.Key
	}
	return c.Transpose(m.transpose).String()
}

func (m Model) Quitting() bool { return m.quitting }

func (m Model) Offset() int { return m.offset }

func (m Model) Transpose() int { return m.transpose }

func Run(doc *parser.Document, cfg RenderConfig) error {
	return RunModel(NewDocModel(doc, cfg))
}

func RunModel(m Model) error {
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}