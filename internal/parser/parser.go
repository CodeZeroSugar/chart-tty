// Package parser contains validation and parsing logic for chord sheets
package parser

import (
	"errors"
	"strings"
)

type ParserMode int

const (
	ModeChordPro ParserMode = iota
	ModeBasic
)

type Parser struct {
	Mode ParserMode

	doc            *Document
	currentSection *Section
	activeEnv      string
}

func NewParser(mode ParserMode) *Parser {
	return &Parser{
		Mode: mode,
	}
}

func (p *Parser) Parse(chart string) (*Document, error) {
	p.doc = &Document{
		Metadata: make(map[string][]string, 0),
		Sections: make([]Section, 0),
	}
	p.currentSection = nil

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
	cleanChart := strings.ReplaceAll(chart, "\r\n", "\n")

	var parsedLines []ParsedLine
	for line := range strings.SplitSeq(cleanChart, "\n") {
		parsedLines = append(parsedLines, p.parseLine(line))
	}

	for _, line := range parsedLines {
		switch line.Type {
		case LineTypeDirective:
			p.handleDirective(extractDirective(line.Raw))
		case LineTypeEmpty:
			p.shouldFlushOnBlankLine()
		case LineTypeComment:
		default:
			p.currentSection.Lines = append(p.currentSection.Lines, line)
		}
	}

	return nil, nil
}

func (p *Parser) parseBasic(chart string) (*Document, error) {
	return nil, nil
}

func (p *Parser) parseLine(line string) ParsedLine {
	l := strings.TrimSpace(line)
	if l == "" {
		return ParsedLine{
			Type:   LineTypeEmpty,
			Raw:    l,
			Lyrics: "",
			Chords: nil,
		}
	}
	if strings.HasPrefix(l, "{") {
		return ParsedLine{
			Type:   LineTypeDirective,
			Raw:    l,
			Lyrics: "",
			Chords: nil,
		}
	}
	if strings.HasPrefix(l, "#") {
		return ParsedLine{
			Type:   LineTypeComment,
			Raw:    l,
			Lyrics: "",
			Chords: nil,
		}
	}
	if isTabLine(l) {
		return ParsedLine{
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

	return ParsedLine{
		Type:   LineType(lineType),
		Raw:    l,
		Lyrics: cleanLyrics,
		Chords: chords,
	}
}

func (p *Parser) handleDirective(directive, data string) {
	directiveCategory := getDirectiveCategory(directive)
	switch directiveCategory {
	case CategoryEnvironment:
	case CategoryFormatting:
	case CategoryMeta:
		p.handleMetaDirective(directive, data)
	case CategoryUnknown:
	}
}

func (p *Parser) handleEnvironmentDirective(directive, data string) {
	action, env, ok := parseEnvDirective(directive)
	if !ok {
		return
	}
	label := data
	if label == "" {
		label = env
	}
	if action == "start" {
		p.flushSection()
		p.activeEnv = env
		p.currentSection = &Section{Name: label}
	} else if action == "end" {
		if p.activeEnv == env || p.activeEnv != "" {
			p.flushSection()
		}
	}
}

func (p *Parser) handleMetaDirective(directive, data string) {
	switch directive {
	case "title", "t":
		p.doc.Title = data
	case "subtitle", "st", "artist":
		p.doc.Artist = data
	case "tempo":
		p.doc.Tempo = data
	case "key":
		p.doc.Key = data
	case "capo":
		p.doc.Capo = data
	default:
		if _, exists := p.doc.Metadata[directive]; !exists && data != "" {
			p.doc.Metadata[directive] = append(p.doc.Metadata[directive], data)
		} else {
			if data != "" {
				p.doc.Metadata[directive] = append(p.doc.Metadata[directive], data)
			}
		}
	}
}

func (p *Parser) flushSection() {
	if p.currentSection != nil && len(p.currentSection.Lines) > 0 {
		p.doc.Sections = append(p.doc.Sections, *p.currentSection)
	}
	p.currentSection = nil
	p.activeEnv = ""
}

func (p *Parser) shouldFlushOnBlankLine() bool {
	return p.currentSection != nil && p.activeEnv != ""
}
