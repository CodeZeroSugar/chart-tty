package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CodeZeroSugar/chart-tty/internal/aichart"
	"github.com/CodeZeroSugar/chart-tty/internal/config"
	"github.com/CodeZeroSugar/chart-tty/internal/parser"
)

type Model struct {
	lines      []string
	doc        *parser.Document
	transpose  int
	offset     int
	width      int
	height     int
	title      string
	cfg        RenderConfig
	keys       config.KeyConfig
	quitting   bool
	showHelp   bool
	rawChart   string
	converter  *aichart.Client
	message    string
	converting bool
}

type convertDoneMsg struct {
	result aichart.Result
	err    error
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

func (m Model) SetConverter(rawChart string, client *aichart.Client) Model {
	m.rawChart = rawChart
	m.converter = client
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
	case convertDoneMsg:
		m = m.applyConversion(msg.result, msg.err)
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
		case k == "c":
			nm, cmd := m.startConversion()
			return nm, cmd
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
	if m.message != "" {
		sb.WriteString(lipgloss.NewStyle().Faint(true).Render(m.message))
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
		sb.WriteString(lipgloss.NewStyle().Faint(true).Render("j/k scroll · +/- transpose · c convert · q quit"))
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

func (m Model) startConversion() (Model, tea.Cmd) {
	if m.converter == nil || m.rawChart == "" || m.converting {
		return m, nil
	}
	m.converting = true
	m.message = "converting…"
	client := m.converter
	raw := m.rawChart
	return m, func() tea.Msg {
		res, err := client.Convert(raw)
		return convertDoneMsg{result: res, err: err}
	}
}

func (m Model) applyConversion(res aichart.Result, cerr error) Model {
	m.converting = false
	if cerr != nil {
		m.message = "AI conversion failed"
		return m
	}
	nd, err := parser.NewParser(parser.ModeChordPro).Parse(res.Chart)
	if err != nil {
		m.message = "converted chart failed to parse"
		return m
	}
	m.doc = nd
	m.title = nd.Title
	m.transpose = 0
	m.lines = Render(nd, m.cfg)
	m.offset = 0
	m.message = "converted"
	return m
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
