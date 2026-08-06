// Package parser contains validation and parsing logic for chord sheets
package parser

import (
	"errors"
	"fmt"
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
		if err := p.parseChordPro(chart); err != nil {
			return nil, fmt.Errorf("failure during ChordPro parsing: %w", err)
		}
		return p.doc, nil
	case ModeBasic:
		if err := p.parseBasic(chart); err != nil {
			return nil, fmt.Errorf("failure during Basic parsing: %w", err)
		}
	default:
		return nil, errors.New("parser error: unknown or unsupported parser mode")
	}
	return p.doc, nil
}

func (p *Parser) parseChordPro(chart string) error {
	if len(chart) == 0 {
		return errors.New("chart is empty, nothing to parse")
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
			if p.shouldFlushOnBlankLine() {
				p.flushSection()
			}
		case LineTypeComment:
		default:
			p.appendLine(line)
		}
	}
	p.flushSection()
	return nil
}

func (p *Parser) parseBasic(chart string) error {
	return nil
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
	directiveLower := strings.ToLower(directive)
	directiveCategory := getDirectiveCategory(directiveLower)
	switch directiveCategory {
	case CategoryEnvironment:
		p.handleEnvironmentDirective(directiveLower, data)
	case CategoryFormatting:
	case CategoryMeta:
		p.handleMetaDirective(directiveLower, data)
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
	switch action {
	case "start":
		p.flushSection()
		p.activeEnv = env
		p.currentSection = &Section{Name: label}
	case "end":
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
	return p.currentSection != nil && p.activeEnv == ""
}

func (p *Parser) appendLine(line ParsedLine) {
	if p.currentSection == nil {
		p.currentSection = &Section{
			Name:  "",
			Lines: make([]ParsedLine, 0),
		}
	}
	p.currentSection.Lines = append(p.currentSection.Lines, line)
}
