package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
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
	screenPickSetlist
	screenViewSetlist
	screenPickFile
)

type Model struct {
	lines        []string
	doc          *parser.Document
	transpose    int
	offset       int
	width        int
	height       int
	title        string
	cfg          RenderConfig
	keys         config.KeyConfig
	quitting     bool
	showHelp     bool
	rawChart     string
	converter    *aichart.Client
	message      string
	converting   bool
	convProgress *conversionProgress

	chordMode parser.ChordMode

	lastConverted string

	screen        screen
	store         *db.Store
	browseCharts  []db.ChartMeta
	browseCursor  int
	deletePending string // title awaiting y/n confirmation
	setlists      []db.SetlistMeta
	setlistCursor int
	inputMode     bool
	nameInput     textinput.Model
	pickDir       string
	pickFiles     []string
	pickCursor    int
	setlist       struct {
		name   string
		charts []setlistChartView
		index  int
	}
}

// setlistChartView is one chart inside an open setlist, pre-rendered.
type setlistChartView struct {
	title   string
	content string
	lines   []string
}

type convertDoneMsg struct {
	result aichart.Result
	err    error
}

// conversionTickMsg prompts the model to refresh the corner status from the
// live conversion progress.
type conversionTickMsg struct{}

// conversionProgress holds the latest AI progress event shared between the
// conversion goroutine and the model's tick handler.
type conversionProgress struct {
	mu  sync.Mutex
	msg string
}

func (p *conversionProgress) set(m string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.msg = m
}

func (p *conversionProgress) get() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.msg
}

func tickConversion() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg { return conversionTickMsg{} })
}

func NewModel(lines []string, cfg RenderConfig) Model {
	return Model{lines: lines, cfg: cfg, keys: config.Default().Keys, showHelp: true, chordMode: parser.DefaultChordMode()}
}

func NewDocModel(doc *parser.Document, cfg RenderConfig) Model {
	m := NewModel(Render(doc, cfg), cfg)
	m.doc = doc
	m.title = doc.Title
	return m
}

// NewLibraryModel launches the TUI straight into the library browser with no
// chart loaded — the entry point for running chart-tty without a file.
func NewLibraryModel(store *db.Store, cfg RenderConfig) Model {
	m := NewModel(nil, cfg)
	m.store = store
	m.screen = screenBrowseCharts
	if charts, err := store.ListCharts(); err == nil {
		m.browseCharts = charts
	}
	ti := textinput.New()
	ti.Placeholder = "setlist name"
	ti.CharLimit = 60
	m.nameInput = ti
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
		m.converting = false
		m.convProgress = nil
		m = m.applyConversion(msg.result, msg.err)
	case conversionTickMsg:
		if m.converting && m.convProgress != nil {
			m.message = "converting… " + m.convProgress.get()
			return m, tickConversion()
		}
		return m, nil
	case tea.KeyMsg:
		k := msg.String()
		if k == "ctrl+c" || k == m.keys.Quit {
			m.quitting = true
			return m, tea.Quit
		}
		return m.updateScreen(msg)
	}
	m.clampOffset()
	return m, nil
}

func (m Model) updateScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return m, nil
	}
	// Typed input (new setlist name) captures everything while active.
	if m.inputMode {
		return m.updateNameInput(msg)
	}
	k := key.String()

	switch m.screen {
	case screenBrowseCharts:
		return m.updateBrowseChartsKeys(k)
	case screenBrowseSetlists:
		return m.updateBrowseSetlistsKeys(k)
	case screenPickSetlist:
		return m.updatePickSetlistKeys(k)
	case screenViewSetlist:
		return m.updateViewSetlistKeys(k)
	case screenPickFile:
		return m.updatePickFileKeys(k)
	default:
		return m.updateViewChartKeys(k)
	}
}

func (m Model) updateNameInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	km := msg.(tea.KeyMsg)
	switch km.String() {
	case "esc":
		m.inputMode = false
		m.nameInput.Blur()
		return m, nil
	case "enter":
		name := strings.TrimSpace(m.nameInput.Value())
		m.inputMode = false
		m.nameInput.Blur()
		if name == "" {
			m.message = "setlist name required"
			return m, nil
		}
		if _, err := m.store.CreateSetlist(name); err != nil {
			if errors.Is(err, db.ErrExists) {
				m.message = "setlist name already exists"
				return m, nil
			}
			m.message = fmt.Sprintf("create failed: %v", err)
			return m, nil
		}
		m.refreshSetlists()
		return m, nil
	}
	nm, cmd := m.nameInput.Update(msg)
	m.nameInput = nm
	return m, cmd
}

func (m *Model) refreshSetlists() {
	if lists, err := m.store.ListSetlists(); err == nil {
		m.setlists = lists
		m.setlistCursor = 0
	}
}

func (m Model) openSetlistBrowser() (tea.Model, tea.Cmd) {
	if m.store == nil {
		m.message = "no library open"
		return m, nil
	}
	m.refreshSetlists()
	ti := textinput.New()
	ti.Placeholder = "setlist name"
	ti.CharLimit = 60
	m.nameInput = ti
	m.screen = screenBrowseSetlists
	return m, nil
}

func (m Model) updateBrowseSetlistsKeys(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "down", "j":
		if m.setlistCursor < len(m.setlists)-1 {
			m.setlistCursor++
		}
	case "up", "k":
		if m.setlistCursor > 0 {
			m.setlistCursor--
		}
	case "n":
		m.inputMode = true
		m.nameInput.SetValue("")
		return m, m.nameInput.Focus()
	case "enter":
		if len(m.setlists) > 0 {
			sl := m.setlists[m.setlistCursor]
			charts, err := m.store.SetlistCharts(sl.ID)
			if err != nil {
				m.message = fmt.Sprintf("library error: %v", err)
				return m, nil
			}
			views := make([]setlistChartView, len(charts))
			for i, c := range charts {
				doc, perr := parseStored(c.Content, m.chordMode)
				if perr != nil {
					views[i] = setlistChartView{title: c.Title, content: c.Content, lines: []string{"(unparsable chart)"}}
					continue
				}
				views[i] = setlistChartView{title: doc.Title, content: c.Content, lines: Render(doc, m.cfg)}
			}
			m.setlist.name = sl.Name
			m.setlist.charts = views
			m.setlist.index = 0
			m.offset = 0
			m.screen = screenViewSetlist
		}
	case "c":
		m.message = "open a setlist first"
	case "esc":
		m.screen = screenViewChart
	}
	return m, nil
}

func (m Model) updatePickSetlistKeys(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "down", "j":
		if m.setlistCursor < len(m.setlists)-1 {
			m.setlistCursor++
		}
	case "up", "k":
		if m.setlistCursor > 0 {
			m.setlistCursor--
		}
	case "enter":
		if len(m.setlists) > 0 && len(m.browseCharts) > 0 {
			chartID := m.browseCharts[m.browseCursor].ID
			sl := m.setlists[m.setlistCursor]
			if err := m.store.AppendSetlistItem(sl.ID, chartID); err != nil {
				m.message = fmt.Sprintf("add failed: %v", err)
				return m, nil
			}
			m.message = fmt.Sprintf("added to %s", sl.Name)
			m.screen = screenBrowseCharts
		}
	case "esc":
		m.screen = screenBrowseCharts
	}
	return m, nil
}

func (m Model) updateBrowseChartsKeys(k string) (tea.Model, tea.Cmd) {
	// Confirm state: only y/n/esc act while a delete is pending.
	if m.deletePending != "" {
		switch k {
		case "y":
			id := m.browseCharts[m.browseCursor].ID
			title := m.browseCharts[m.browseCursor].Title
			if err := m.store.DeleteChart(id); err != nil {
				m.message = fmt.Sprintf("delete failed: %v", err)
			} else {
				m.message = "deleted " + title
			}
			m.deletePending = ""
			m.refreshCharts()
			return m, nil
		case "n", "esc":
			m.deletePending = ""
			return m, nil
		}
		return m, nil
	}

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
	case "d":
		if len(m.browseCharts) > 0 {
			m.deletePending = m.browseCharts[m.browseCursor].Title
		}
	case "c":
		if len(m.browseCharts) > 0 {
			stored, err := m.store.GetChart(m.browseCharts[m.browseCursor].ID)
			if err != nil {
				m.message = fmt.Sprintf("library error: %v", err)
				return m, nil
			}
			m.rawChart = stored.Content
			return m.startConversion()
		}
	case "s":
		if m.store == nil {
			return m, nil
		}
		m.refreshSetlists()
		m.setlistCursor = 0
		if len(m.setlists) == 0 {
			ti := textinput.New()
			ti.Placeholder = "setlist name"
			ti.CharLimit = 60
			m.nameInput = ti
			m.inputMode = true
			return m, m.nameInput.Focus()
		}
		m.screen = screenPickSetlist
	case "esc":
		m.screen = screenViewChart
	}
	return m, nil
}

// refreshCharts reloads the library listing and clamps the cursor.
func (m *Model) refreshCharts() {
	if lists, err := m.store.ListCharts(); err == nil {
		m.browseCharts = lists
		if m.browseCursor >= len(lists) {
			m.browseCursor = max(len(lists)-1, 0)
		}
	}
}

func (m Model) updateViewSetlistKeys(k string) (tea.Model, tea.Cmd) {
	n := len(m.setlist.charts)
	if n == 0 {
		if k == "esc" {
			m.screen = screenViewChart
		}
		return m, nil
	}
	maxOff := m.setlistMaxOff()

	switch k {
	case "down", "j":
		if m.offset < maxOff {
			m.offset++
		} else if m.setlist.index < n-1 {
			m.setlist.index++
			m.offset = 0
		}
	case "up", "k":
		if m.offset > 0 {
			m.offset--
		} else if m.setlist.index > 0 {
			m.setlist.index--
			m.offset = max(0, len(m.setlist.charts[m.setlist.index].lines)-m.bodyHeight())
		}
	case "pgdown", " ":
		if m.offset < maxOff {
			m.offset += max(m.bodyHeight(), 1)
			if m.offset > maxOff {
				m.offset = maxOff
			}
		} else if m.setlist.index < n-1 {
			m.setlist.index++
			m.offset = 0
		}
	case "pgup", "b":
		if m.offset > 0 {
			m.offset -= max(m.bodyHeight(), 1)
			if m.offset < 0 {
				m.offset = 0
			}
		} else if m.setlist.index > 0 {
			m.setlist.index--
			m.offset = max(0, len(m.setlist.charts[m.setlist.index].lines)-m.bodyHeight())
		}
	case "home", "g":
		m.setlist.index = 0
		m.offset = 0
	case "end", "G":
		m.setlist.index = n - 1
		m.offset = max(0, len(m.setlist.charts[n-1].lines)-m.bodyHeight())
	case "esc":
		return m.openSetlistBrowser()
	case "c":
		if idx := m.setlist.index; idx >= 0 && idx < len(m.setlist.charts) {
			m.rawChart = m.setlist.charts[idx].content
		}
		return m.startConversion()
	}
	return m, nil
}

// setlistMaxOff is the maximum scroll offset of the currently open setlist chart.
func (m Model) setlistMaxOff() int {
	if m.setlist.index >= len(m.setlist.charts) {
		return 0
	}
	return max(0, len(m.setlist.charts[m.setlist.index].lines)-m.bodyHeight())
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
	doc, perr := parseStored(stored.Content, m.chordMode)
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
	m.rawChart = stored.Content
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
	case k == "S":
		return m.openSetlistBrowser()
	case k == "o":
		return m.openFilePicker()
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
	case k == "m":
		if m.chordMode == parser.StrictChords {
			m.chordMode = parser.RelaxedChords
		} else {
			m.chordMode = parser.StrictChords
		}
		if m.rawChart != "" {
			if doc, err := parseStored(m.rawChart, m.chordMode); err == nil {
				m.doc = doc
				m.title = doc.Title
				m.transpose = 0
				m.offset = 0
				m.lines = Render(doc, m.cfg)
			}
		}
		m.message = "chord mode: " + m.chordMode.String()
		return m, nil
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
// reserving rows for the title and help lines. The status message shares the
// title row, so it does not consume a row of its own.
func (m Model) bodyHeight() int {
	h := m.height
	if m.title != "" || (m.doc != nil && m.doc.Key != "") {
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
	switch m.screen {
	case screenBrowseCharts:
		return m.viewBrowseCharts()
	case screenBrowseSetlists, screenPickSetlist:
		return m.viewBrowseSetlists(m.screen == screenPickSetlist)
	case screenViewSetlist:
		return m.viewSetlist()
	case screenPickFile:
		return m.viewPickFile()
	}
	if m.doc == nil && len(m.lines) == 0 {
		out := headerRow("chart-tty", m.message, m.width) + "\n\n" +
			lipgloss.NewStyle().Faint(true).Render("No chart loaded.\nL library · S setlists · q quit")
		return out
	}
	var sb strings.Builder
	title := m.title
	if k := m.currentKey(); k != "" {
		title += " · Key: " + k
	}
	if title != "" || m.message != "" {
		sb.WriteString(headerRow(title, m.message, m.width))
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
		sb.WriteString(lipgloss.NewStyle().Faint(true).Render("j/k scroll · space/b page · +/- transpose · i save · o open · L library · S setlists · c convert · q quit"))
	}
	return sb.String()
}

// viewBrowseCharts renders the library listing with a cursor.
func (m Model) viewBrowseCharts() string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Library"))
	sb.WriteString("\n\n")

	if m.deletePending != "" {
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("yellow")).Render(fmt.Sprintf("Delete %q? (y/n)", m.deletePending)))
		sb.WriteString("\n\n")
	} else {
		sb.WriteString(headerRow("Library", m.message, m.width))
		sb.WriteString("\n\n")
	}

	if len(m.browseCharts) == 0 {
		sb.WriteString("(library is empty — import a chart with i from the viewer)")
		return sb.String()
	}

	bodyHeight := m.bodyHeight() - 3 // title row, blank line, and gap before help
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
	hint := "j/k move · enter open · s add to setlist · d delete · esc back"
	if m.deletePending != "" {
		hint = "y confirm delete · n/esc cancel"
	}
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render(hint))
	return sb.String()
}

// parseStored detects the format of stored content and parses it, mirroring
// the CLI's mode detection under the given chord mode.
func parseStored(content string, chordMode parser.ChordMode) (*parser.Document, error) {
	validator, err := parser.NewValidator()
	if err != nil {
		return nil, err
	}
	validator.ChordMode = chordMode
	isValid, verr := validator.ValidateChart(content)

	mode := parser.ModeChordPro
	if !isValid || parser.LooksLikeBasicChart(content, chordMode) {
		mode = parser.ModeBasic
		if !isValid && verr != nil {
			_ = verr // basic mode handles it
		}
	}
	p := parser.NewParser(mode)
	p.ChordMode = chordMode
	return p.Parse(content)
}

// viewBrowseSetlists renders the setlist listing (or the pick-setlist
// variant used when adding the browsed chart to a setlist).
func (m Model) viewBrowseSetlists(picking bool) string {
	var sb strings.Builder
	header := "Setlists"
	if picking {
		header = "Add \"" + m.browseCharts[m.browseCursor].Title + "\" to:"
	}
	sb.WriteString(headerRow(header, m.message, m.width))
	sb.WriteString("\n\n")

	if m.inputMode {
		sb.WriteString("New setlist: " + m.nameInput.View() + "\n")
		return sb.String()
	}

	if len(m.setlists) == 0 {
		sb.WriteString("(no setlists — press n to create one)")
		return sb.String()
	}

	bodyHeight := m.bodyHeight() - 3 // title row, blank line, and gap before help
	start := max(m.setlistCursor-bodyHeight+1, 0)
	end := min(start+bodyHeight, len(m.setlists))

	for i := start; i < end; i++ {
		sl := m.setlists[i]
		marker, style := "  ", lipgloss.NewStyle().Faint(true)
		if i == m.setlistCursor {
			marker, style = "> ", lipgloss.NewStyle().Bold(true)
		}
		sb.WriteString(marker + style.Render(sl.Name))
		sb.WriteString("\n")
	}
	hint := "j/k move · enter open · n new · esc back"
	if picking {
		hint = "j/k choose · enter add · esc cancel"
	}
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render(hint))
	return sb.String()
}

// viewSetlist renders the current chart of an open setlist.
func (m Model) viewSetlist() string {
	var sb strings.Builder

	name := m.setlist.name
	pos := fmt.Sprintf("%d/%d", m.setlist.index+1, len(m.setlist.charts))
	curTitle := ""
	keySuffix := ""
	if k := m.currentKey(); k != "" {
		keySuffix = " · Key: " + k
	}
	if m.setlist.index < len(m.setlist.charts) {
		curTitle = m.setlist.charts[m.setlist.index].title
	}
	header := fmt.Sprintf("%s ─ chart %s ─ %s%s", name, pos, curTitle, keySuffix)
	sb.WriteString(headerRow(strings.TrimPrefix(strings.TrimRight(header, " "), "─ "), m.message, m.width))
	sb.WriteString("\n")

	if m.setlist.index < len(m.setlist.charts) {
		lines := m.setlist.charts[m.setlist.index].lines
		rows := max(m.bodyHeight()-1, 1) // header is always printed
		top := min(m.offset, max(len(lines)-1, 0))
		bottom := min(top+rows, len(lines))
		for i := top; i < bottom; i++ {
			sb.WriteString(lines[i])
			sb.WriteString("\n")
		}
	}

	help := "j/k scroll · space/b next/prev chart · esc back"
	if m.showHelp {
		sb.WriteString(lipgloss.NewStyle().Faint(true).Render(help))
	}
	return sb.String()
}

// pickerExts are the file extensions the disk picker lists.
var pickerExts = map[string]bool{
	".cho": true, ".crd": true, ".chopro": true, ".chord": true, ".pro": true, ".txt": true,
}

func isPickableFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return pickerExts[ext]
}

// openFilePicker lists chart files from the charts directory (falling back to
// the working directory, or to an already-set pickDir for tests) and switches
// to the picker screen.
func (m Model) openFilePicker() (tea.Model, tea.Cmd) {
	dir := m.pickDir
	if dir == "" {
		dir = "charts"
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			dir = "."
		}
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		m.message = fmt.Sprintf("file picker error: %v", err)
		return m, nil
	}
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if isPickableFile(e.Name()) {
			files = append(files, filepath.Join(abs, e.Name()))
		}
	}
	slices.Sort(files)
	m.pickDir = abs
	m.pickFiles = files
	m.pickCursor = 0
	m.screen = screenPickFile
	return m, nil
}

func (m Model) updatePickFileKeys(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "down", "j":
		if m.pickCursor < len(m.pickFiles)-1 {
			m.pickCursor++
		}
	case "up", "k":
		if m.pickCursor > 0 {
			m.pickCursor--
		}
	case "enter":
		if len(m.pickFiles) > 0 {
			return m.openDiskFile(m.pickFiles[m.pickCursor])
		}
	case "c":
		if len(m.pickFiles) > 0 {
			content, err := os.ReadFile(m.pickFiles[m.pickCursor])
			if err != nil {
				m.message = fmt.Sprintf("read failed: %v", err)
				return m, nil
			}
			m.rawChart = string(content)
			return m.startConversion()
		}
	case "esc":
		m.screen = screenViewChart
	}
	return m, nil
}

// openDiskFile reads and loads a chart file from disk into the viewer.
func (m Model) openDiskFile(path string) (tea.Model, tea.Cmd) {
	content, err := os.ReadFile(path)
	if err != nil {
		m.message = fmt.Sprintf("read failed: %v", err)
		return m, nil
	}
	doc, perr := parseStored(string(content), m.chordMode)
	if perr != nil {
		m.message = "file failed to parse"
		return m, nil
	}
	m.screen = screenViewChart
	m.doc = doc
	m.title = doc.Title
	m.transpose = 0
	m.offset = 0
	m.lines = Render(doc, m.cfg)
	m.message = ""
	m.rawChart = string(content)
	return m, nil
}

// viewPickFile renders the filesystem chart list.
func (m Model) viewPickFile() string {
	var sb strings.Builder
	sb.WriteString(headerRow("Pick a chart  "+m.pickDir, m.message, m.width))
	sb.WriteString("\n\n")

	if len(m.pickFiles) == 0 {
		sb.WriteString("(no chart files found)")
		return sb.String()
	}

	rows := max(m.bodyHeight()-3, 1)
	start := max(m.pickCursor-rows+1, 0)
	end := min(start+rows, len(m.pickFiles))
	for i := start; i < end; i++ {
		marker, style := "  ", lipgloss.NewStyle().Faint(true)
		if i == m.pickCursor {
			marker, style = "> ", lipgloss.NewStyle().Bold(true)
		}
		sb.WriteString(marker + style.Render(filepath.Base(m.pickFiles[i])))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render("j/k move · enter open · esc back"))
	return sb.String()
}

// truncateTo cuts s to at most w display cells, appending an ellipsis when
// truncation occurs. Callers apply styles after truncation so escape
// sequences are never split.
func truncateTo(s string, w int) string {
	if w < 1 {
		return ""
	}
	if displayWidth(s) <= w {
		return s
	}
	var sb strings.Builder
	used := 0
	for _, r := range s {
		rw := displayWidth(string(r))
		if used+rw > w-1 { // reserve one cell for the ellipsis
			break
		}
		sb.WriteRune(r)
		used += rw
	}
	return sb.String() + "…"
}

// headerRow lays the title left-aligned and the status message right-aligned
// on the same line, truncating with an ellipsis when they collide. The result
// is exactly width cells wide.
func headerRow(title, msg string, width int) string {
	if width < 1 {
		width = 1
	}
	if msg == "" {
		return lipgloss.NewStyle().Bold(true).Render(truncateTo(title, width))
	}
	const gap = 3
	title = truncateTo(title, width-gap)
	titleW := displayWidth(title)
	msgW := width - titleW - gap
	if msgW < 1 {
		msgW = 1
		title = truncateTo(title, width-gap-1)
		titleW = displayWidth(title)
		msgW = width - titleW - gap
		if msgW < 1 {
			msgW = 1
		}
	}
	msg = truncateTo(msg, msgW)
	pad := width - titleW - displayWidth(msg)
	return lipgloss.NewStyle().Bold(true).Render(title) + strings.Repeat(" ", pad) + lipgloss.NewStyle().Faint(true).Render(msg)
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
	if m.converting {
		return m, nil
	}
	if m.converter == nil {
		m.message = "AI conversion not configured (set an API key)"
		return m, nil
	}
	if m.rawChart == "" {
		m.message = "no chart to convert — open a chart first"
		return m, nil
	}
	prog := &conversionProgress{}
	m.convProgress = prog
	m.converting = true
	m.message = "converting…"
	client := m.converter
	raw := m.rawChart
	convCmd := func() tea.Msg {
		res, err := client.ConvertProgress(raw, func(e aichart.ProgressEvent) {
			prog.set(fmt.Sprintf("attempt %d/%d: %s", e.Attempt, e.MaxAttempts, e.Message))
		})
		return convertDoneMsg{result: res, err: err}
	}
	return m, tea.Batch(convCmd, tickConversion())
}

func (m Model) applyConversion(res aichart.Result, cerr error) Model {
	m.converting = false
	if cerr != nil {
		msg := cerr.Error()
		if len(msg) > 120 {
			msg = msg[:120] + "…"
		}
		m.message = "AI conversion failed: " + msg
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

// ChordMode reports the active strict/relaxed grammar, for tests.
func (m Model) ChordMode() parser.ChordMode { return m.chordMode }

// Screen reports the current screen, for tests.
func (m Model) Screen() string {
	switch m.screen {
	case screenBrowseCharts:
		return "browseCharts"
	case screenBrowseSetlists:
		return "browseSetlists"
	case screenPickSetlist:
		return "pickSetlist"
	case screenViewSetlist:
		return "viewSetlist"
	case screenPickFile:
		return "pickFile"
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
	// Alternate screen: multi-screen TUIs need full-frame repaints; inline
	// mode appends below previous output and corrupts on height changes.
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
