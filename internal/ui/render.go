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
	var out []string
	if block := renderMetaBlock(doc, cfg); len(block) > 0 {
		out = append(out, block...)
		out = append(out, "")
	}
	for i, sec := range doc.Sections {
		if i > 0 {
			out = append(out, "")
		}
		if sec.Name != "" {
			out = append(out, applyStyle("["+sec.Name+"]", cfg.HeaderStyle))
		}
		for _, line := range sec.Lines {
			switch line.Type {
			case parser.LineTypeChordAndLyric:
				out = append(out, chordRow(line.Lyrics, line.Chords), line.Lyrics)
			case parser.LineTypeChord:
				out = append(out, chordNames(line.Chords))
			case parser.LineTypeLyric:
				out = append(out, line.Lyrics)
			case parser.LineTypeTab:
				out = append(out, line.Raw)
			case parser.LineTypeComment:
				out = append(out, applyStyle(line.Lyrics, cfg.CommentStyle))
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
