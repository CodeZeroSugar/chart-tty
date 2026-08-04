// Package parser contains validation and parsing logic for chord sheets
package parser

import (
	"errors"
	"strings"
)

type ParserMode int

const (
	ModeChordPro = iota
	ModeBasic
)

type Parser struct {
	Mode ParserMode
}

func NewParser(mode ParserMode) *Parser {
	return &Parser{
		Mode: mode,
	}
}

func DetectParserMode(valid bool, err error) ParserMode {
	if valid && err == nil {
		return ModeChordPro
	}
	return ModeBasic
}

func (p *Parser) Parse(chart string) (*Document, error) {
	switch p.Mode {
	case ModeChordPro:
		return p.parseChordPro(chart)
	case ModeBasic:
		return p.parseBasic(chart)
	default:
		return nil, errors.New("parser error: unknown or unsupported parser mode")
	}
}

func (p *Parser) parseChordPro(chart string) (*Document, error) {
	if len(chart) == 0 {
		return nil, errors.New("chart is empty, nothing to parse")
	}
	lines := strings.Split(strings.ReplaceAll(chart, "\r\n", "\n"), "\n")
	for _, line := range lines {
		p.parseLine(line)
	}
	return nil, nil
}

func (p *Parser) parseBasic(chart string) (*Document, error) {
	return nil, nil
}

func (p *Parser) parseLine(line string) *ParsedLine {
	l := strings.TrimSpace(line)
	if l == "" {
		return &ParsedLine{
			Type:   LineTypeEmpty,
			Raw:    l,
			Lyrics: "",
			Chords: nil,
		}
	}
	if strings.HasPrefix(l, "{") {
		return &ParsedLine{
			Type:   LineTypeDirective,
			Raw:    l,
			Lyrics: "",
			Chords: nil,
		}
	}
	if strings.HasPrefix(l, "#") {
		return &ParsedLine{
			Type:   LineTypeComment,
			Raw:    l,
			Lyrics: "",
			Chords: nil,
		}
	}
	if isTabLine(l) {
		return &ParsedLine{
			Type:   LineTypeTab,
			Raw:    l,
			Lyrics: "",
			Chords: nil,
		}
	}

	var lyrics strings.Builder
	var chord strings.Builder
	var chords []ChordToken

	inBrackets := false
	for _, char := range l {
		switch char {
		case ']':
			chords = append(chords, ChordToken{Name: chord.String(), Position: lyrics.Len()})
			inBrackets = false
		case '[':
			inBrackets = true
			chord.Reset()
		default:
			if inBrackets {
				chord.WriteRune(char)
			} else {
				lyrics.WriteRune(char)
			}
		}
	}
	cleanLyrics := lyrics.String()
	lineType := LineTypeLyric

	if len(chords) > 0 {
		if strings.TrimSpace(cleanLyrics) != "" {
			lineType = LineTypeChordAndLyric
		} else {
			lineType = LineTypeChord
		}
	}

	return &ParsedLine{
		Type:   LineType(lineType),
		Raw:    l,
		Lyrics: cleanLyrics,
		Chords: chords,
	}
}

func isTabLine(s string) bool {
	return strings.Contains(s, "--") || (strings.Contains(s, "|") && strings.Contains(s, "-"))
}
