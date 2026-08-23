package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CodeZeroSugar/chart-tty/internal/parser"
)

type Model struct {
	lines    []string
	offset   int
	width    int
	height   int
	title    string
	cfg      RenderConfig
	quitting bool
	showHelp bool
}

func NewModel(lines []string, cfg RenderConfig) Model {
	return Model{lines: lines, cfg: cfg, showHelp: true}
}

func NewDocModel(doc *parser.Document, cfg RenderConfig) Model {
	m := NewModel(Render(doc, cfg), cfg)
	m.title = doc.Title
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

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			m.offset--
		case "down", "j":
			m.offset++
		case "pgup":
			m.offset -= max(m.height, 1)
		case "pgdown", " ":
			m.offset += max(m.height, 1)
		case "home", "g":
			m.offset = 0
		case "end", "G":
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
	if m.title != "" {
		sb.WriteString(lipgloss.NewStyle().Bold(true).Render(m.title))
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
		sb.WriteString(lipgloss.NewStyle().Faint(true).Render("j/k scroll · q quit"))
	}
	return sb.String()
}

func (m Model) Quitting() bool { return m.quitting }

func (m Model) Offset() int { return m.offset }

func Run(doc *parser.Document, cfg RenderConfig) error {
	m := NewDocModel(doc, cfg)
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}