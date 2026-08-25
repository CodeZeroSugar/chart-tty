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
