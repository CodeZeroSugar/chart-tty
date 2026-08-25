package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/CodeZeroSugar/chart-tty/internal/config"
	"github.com/CodeZeroSugar/chart-tty/internal/parser"
)

type RenderConfig struct {
	HeaderStyle    lipgloss.Style
	CommentStyle   lipgloss.Style
	BannerStyle    lipgloss.Style
	TitleStyle     lipgloss.Style
	HighlightStyle lipgloss.Style

	// SuppressMetaTitleKey omits the title line and the Key: segment from the
	// body metadata block. The TUI sets it because the persistent header row
	// already shows the title and the live transposed key.
	SuppressMetaTitleKey bool
}

func RenderConfigFromConfig(cfg config.Config) RenderConfig {
	return RenderConfig{
		HeaderStyle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(resolveColor(cfg.Theme.HeaderColor))),
		CommentStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color(resolveColor(cfg.Theme.CommentColor))),
		BannerStyle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(resolveColor(cfg.Theme.HeaderColor))),
		TitleStyle:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(resolveColor(cfg.Theme.HeaderColor))),
		HighlightStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(resolveColor(cfg.Theme.HighlightColor))),
	}
}

func Render(doc *parser.Document, cfg RenderConfig) []string {
	rows := renderRows(doc, cfg)
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.text
	}
	return out
}

// renderLines returns the rendered text lines plus a parallel keep slice:
// keep[i] is true when a page break is forbidden between lines i and i+1
// (the chord row of a chord/lyric pair and its lyric must never be split).
func renderLines(doc *parser.Document, cfg RenderConfig) ([]string, []bool) {
	rows := renderRows(doc, cfg)
	lines := make([]string, len(rows))
	keep := make([]bool, len(rows))
	for i, r := range rows {
		lines[i] = r.text
		keep[i] = r.keepWithNext
	}
	return lines, keep
}

// renderedRow is one output line plus its keep-with-next flag.
type renderedRow struct {
	text         string
	keepWithNext bool
}

func renderRows(doc *parser.Document, cfg RenderConfig) []renderedRow {
	var out []renderedRow
	appendRow := func(text string, keep bool) {
		out = append(out, renderedRow{text: text, keepWithNext: keep})
	}
	if block := renderMetaBlock(doc, cfg); len(block) > 0 {
		for _, b := range block {
			appendRow(b, false)
		}
		appendRow("", false)
	}
	for i, sec := range doc.Sections {
		if i > 0 {
			appendRow("", false)
		}
		if sec.Name != "" {
			appendRow(applyStyle("["+sec.Name+"]", cfg.HeaderStyle), false)
		}
		for li, line := range sec.Lines {
			switch line.Type {
			case parser.LineTypeChordAndLyric:
				// The chord row and its lyric row are one unbreakable pair.
				appendRow(chordRow(line.Lyrics, line.Chords), true)
				appendRow(line.Lyrics, false)
			case parser.LineTypeChord:
				appendRow(chordNames(line.Chords), false)
			case parser.LineTypeLyric:
				appendRow(line.Lyrics, false)
			case parser.LineTypeTab:
				// A tab block is one unbreakable unit: chain every tab row to
				// the next tab row, leaving only the last row breakable.
				keep := li+1 < len(sec.Lines) && sec.Lines[li+1].Type == parser.LineTypeTab
				appendRow(line.Raw, keep)
			case parser.LineTypeComment:
				appendRow(applyStyle(line.Lyrics, cfg.CommentStyle), false)
			}
		}
	}
	return out
}

// renderMetaBlock returns the lines shown before the song starts: the title,
// the artist (subtitle), and the performance metadata that is present. It
// returns nil when the document has no metadata to show.
func renderMetaBlock(doc *parser.Document, cfg RenderConfig) []string {
	var out []string
	faint := lipgloss.NewStyle().Faint(true)
	if doc.Title != "" && !cfg.SuppressMetaTitleKey {
		out = append(out, applyStyle(doc.Title, cfg.HeaderStyle))
	}
	if doc.Artist != "" {
		out = append(out, faint.Render(doc.Artist))
	}
	if meta := metaLine(doc, cfg.SuppressMetaTitleKey); meta != "" {
		out = append(out, faint.Render(meta))
	}
	return out
}

// metaLine joins the present performance metadata into a single line:
// Capo · Key · Tempo · Time · Duration, in that order. When skipKey is set the
// Key: segment is omitted (the TUI header already shows the transposed key).
func metaLine(doc *parser.Document, skipKey bool) string {
	parts := make([]string, 0, 5)
	if doc.Capo != "" {
		parts = append(parts, "Capo: "+doc.Capo)
	}
	if doc.Key != "" && !skipKey {
		parts = append(parts, "Key: "+doc.Key)
	}
	if doc.Tempo != "" {
		parts = append(parts, "Tempo: "+doc.Tempo)
	}
	if t := strings.Join(doc.Metadata["time"], ", "); t != "" {
		parts = append(parts, "Time: "+t)
	}
	if d := strings.Join(doc.Metadata["duration"], ", "); d != "" {
		parts = append(parts, "Duration: "+d)
	}
	return strings.Join(parts, " · ")
}

func chordRow(lyrics string, chords []parser.ChordToken) string {
	var sb strings.Builder
	for _, ch := range chords {
		for sb.Len() < ch.Position {
			sb.WriteByte(' ')
		}
		sb.WriteString(ch.Name)
	}
	for sb.Len() < len(lyrics) {
		sb.WriteByte(' ')
	}
	return sb.String()
}

func chordNames(chords []parser.ChordToken) string {
	names := make([]string, len(chords))
	for i, ch := range chords {
		names[i] = ch.Name
	}
	return strings.Join(names, " ")
}

func applyStyle(s string, st lipgloss.Style) string {
	return st.Render(s)
}
