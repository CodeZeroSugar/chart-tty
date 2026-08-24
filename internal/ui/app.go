package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CodeZeroSugar/chart-tty/internal/aichart"
	"github.com/CodeZeroSugar/chart-tty/internal/config"
	"github.com/CodeZeroSugar/chart-tty/internal/db"
	"github.com/CodeZeroSugar/chart-tty/internal/parser"
)

type screen int

const (
	screenViewChart screen = iota
	screenBrowseCharts
	screenBrowseSetlists
	screenViewSetlist
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

	lastConverted string

	screen        screen
	store         *db.Store
	browseCharts  []db.ChartMeta
	browseCursor  int
	setlists      []db.SetlistMeta
	setlistCursor int
	setlist       struct {
		name   string
		charts []db.StoredChart
		index  int
	}
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

func (m Model) SetConverter(client *aichart.Client) Model {
	m.converter = client
	return m
}

// SetSource stores the raw original chart text so it can be imported into
// the library.
func (m Model) SetSource(raw string) Model {
	m.rawChart = raw
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
		if k == "ctrl+c" || k == m.keys.Quit {
			m.quitting = true
			return m, tea.Quit
		}
		return m.updateScreen(k)
	}
	m.clampOffset()
	return m, nil
}

func (m Model) updateScreen(k string) (tea.Model, tea.Cmd) {
	if m.screen == screenBrowseCharts {
		return m.updateBrowseChartsKeys(k)
	}
	return m.updateViewChartKeys(k)
}

func (m Model) updateBrowseChartsKeys(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "down", "j":
		if m.browseCursor < len(m.browseCharts)-1 {
			m.browseCursor++
		}
	case "up", "k":
		if m.browseCursor > 0 {
			m.browseCursor--
		}
	case "enter":
		if len(m.browseCharts) > 0 {
			return m.openStoredChart(m.browseCharts[m.browseCursor].ID)
		}
	case "esc":
		m.screen = screenViewChart
	}
	return m, nil
}

// openLibraryBrowser switches to the chart library listing.
func (m Model) openLibraryBrowser() (tea.Model, tea.Cmd) {
	if m.store == nil {
		m.message = "no library open"
		return m, nil
	}
	charts, err := m.store.ListCharts()
	if err != nil {
		m.message = fmt.Sprintf("library error: %v", err)
		return m, nil
	}
	m.browseCharts = charts
	m.browseCursor = 0
	m.screen = screenBrowseCharts
	return m, nil
}

// openStoredChart loads a stored library chart into the viewer.
func (m Model) openStoredChart(id int64) (tea.Model, tea.Cmd) {
	stored, err := m.store.GetChart(id)
	if err != nil {
		m.message = fmt.Sprintf("library error: %v", err)
		return m, nil
	}
	doc, perr := parseStored(stored.Content)
	if perr != nil {
		m.message = "stored chart failed to parse"
		return m, nil
	}
	m.screen = screenViewChart
	m.doc = doc
	m.title = doc.Title
	m.transpose = 0
	m.offset = 0
	m.lines = Render(doc, m.cfg)
	m.message = ""
	return m, nil
}

func (m Model) updateViewChartKeys(k string) (tea.Model, tea.Cmd) {
	switch {
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
	case k == "L":
		return m.openLibraryBrowser()
	case k == "pgup" || k == "b":
		m.offset -= max(m.bodyHeight(), 1)
	case k == "pgdown" || k == " ":
		m.offset += max(m.bodyHeight(), 1)
	case k == "home" || k == "g":
		m.offset = 0
	case k == "end" || k == "G":
		m.offset = len(m.lines)
	case k == "c":
		nm, cmd := m.startConversion()
		return nm, cmd
	case k == "i":
		m = m.importCurrentChart()
		return m, nil
	case k == "s":
		m = m.saveConvertedChart()
		return m, nil
	}
	m.clampOffset()
	return m, nil
}

func (m *Model) clampOffset() {
	if m.offset < 0 {
		m.offset = 0
	}
	maxOff := len(m.lines) - m.visibleRows()
	if maxOff < 0 {
		maxOff = 0
	}
	if m.offset > maxOff {
		m.offset = maxOff
	}
}

// bodyHeight is the number of terminal rows available to chart content after
// reserving rows for the title, status message, and help lines.
func (m Model) bodyHeight() int {
	h := m.height
	if m.title != "" || (m.doc != nil && m.doc.Key != "") {
		h--
	}
	if m.message != "" {
		h--
	}
	if m.showHelp {
		h--
	}
	if h < 1 {
		h = 1
	}
	return h
}

// useColumns reports whether the body should render side by side: the
// terminal must be wide enough AND the content must exceed a single column
// page. This is the single source of truth for layout mode — View and
// clampOffset both derive from it so they can never disagree.
func (m Model) useColumns() bool {
	if !shouldUseColumns(m.width) {
		return false
	}
	colWidth := (m.width - columnGap) / 2
	rows := expandRows(m.body(), colWidth)
	return len(rows) > m.bodyHeight()
}

func (m Model) visibleRows() int {
	rows := m.bodyHeight()
	if m.useColumns() {
		rows *= 2
	}
	return rows
}

func (m Model) View() string {
	if m.screen == screenBrowseCharts {
		return m.viewBrowseCharts()
	}
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

	var body []string
	if m.useColumns() {
		divider := lipgloss.NewStyle().Faint(true).Render(dividerGlyph)
		body = layoutColumns(m.body(), m.width, m.bodyHeight(), divider)
	}
	if body == nil {
		top := m.offset
		bottom := m.offset + m.bodyHeight()
		if bottom > len(m.lines) {
			bottom = len(m.lines)
		}
		if top < len(m.lines) {
			body = m.lines[top:bottom]
		}
	}
	for i, line := range body {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(line)
	}

	if m.showHelp {
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Faint(true).Render("j/k scroll · space/b page · +/- transpose · c convert · q quit"))
	}
	return sb.String()
}

// viewBrowseCharts renders the library listing with a cursor.
func (m Model) viewBrowseCharts() string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Library"))
	sb.WriteString("\n\n")

	if len(m.browseCharts) == 0 {
		sb.WriteString("(library is empty — import a chart with i from the viewer)")
		return sb.String()
	}

	bodyHeight := m.bodyHeight() - 2 // header + blank line
	start := max(m.browseCursor-bodyHeight+1, 0)
	end := min(start+bodyHeight, len(m.browseCharts))

	for i := start; i < end; i++ {
		c := m.browseCharts[i]
		marker := "  "
		row := lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf("%s %s — %s [%s]", c.Title, c.Artist, c.Source, c.UpdatedAt.Format("2006-01-02")))
		if i == m.browseCursor {
			marker = "> "
			row = lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%s %s — %s [%s]", c.Title, c.Artist, c.Source, c.UpdatedAt.Format("2006-01-02")))
		}
		sb.WriteString(marker + row)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render("j/k move · enter open · esc back"))
	return sb.String()
}

// parseStored detects the format of stored content and parses it, mirroring
// the CLI's mode detection.
func parseStored(content string) (*parser.Document, error) {
	validator, err := parser.NewValidator()
	if err != nil {
		return nil, err
	}
	isValid, verr := validator.ValidateChart(content)

	mode := parser.ModeChordPro
	if !isValid || parser.LooksLikeBasicChart(content) {
		mode = parser.ModeBasic
		if !isValid && verr != nil {
			_ = verr // basic mode handles it
		}
	}
	return parser.NewParser(mode).Parse(content)
}

// body returns the rendered lines visible at the current offset. In
// single-column mode it is the offset window; in two-column mode it is the
// remaining stream from the offset, which layoutColumns chunks.
func (m Model) body() []string {
	if m.offset >= len(m.lines) {
		return nil
	}
	return m.lines[m.offset:]
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
	m.lastConverted = res.Chart
	m.message = "converted · s: save to library"
	return m
}

// importCurrentChart stores the loaded chart's original text in the library.
func (m Model) importCurrentChart() Model {
	if m.store == nil || m.rawChart == "" {
		m.message = "nothing to import"
		return m
	}
	title, artist := "", ""
	if m.doc != nil {
		title, artist = m.doc.Title, m.doc.Artist
	}
	if _, err := m.store.AddChart(title, artist, "import", m.rawChart); err != nil {
		m.message = fmt.Sprintf("import failed: %v", err)
		return m
	}
	m.message = "imported to library"
	return m
}

// saveConvertedChart stores the most recent AI conversion in the library.
func (m Model) saveConvertedChart() Model {
	if m.store == nil || m.lastConverted == "" {
		m.message = "no conversion to save"
		return m
	}
	title, artist := "", ""
	if m.doc != nil {
		title, artist = m.doc.Title, m.doc.Artist
	}
	if _, err := m.store.AddChart(title, artist, "ai", m.lastConverted); err != nil {
		m.message = fmt.Sprintf("save failed: %v", err)
		return m
	}
	m.message = "saved to library"
	return m
}

func (m Model) Quitting() bool { return m.quitting }

func (m Model) Offset() int { return m.offset }

func (m Model) Transpose() int { return m.transpose }

// Screen reports the current screen, for tests.
func (m Model) Screen() string {
	switch m.screen {
	case screenBrowseCharts:
		return "browseCharts"
	case screenBrowseSetlists:
		return "browseSetlists"
	case screenViewSetlist:
		return "viewSetlist"
	default:
		return "viewChart"
	}
}

// SetStore attaches the chart library.
func (m Model) SetStore(s *db.Store) Model {
	m.store = s
	return m
}

func Run(doc *parser.Document, cfg RenderConfig) error {
	return RunModel(NewDocModel(doc, cfg))
}

func RunModel(m Model) error {
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}
