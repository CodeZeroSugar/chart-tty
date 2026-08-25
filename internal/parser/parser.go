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
	// ChordMode governs basic-format chord-line detection. ChordPro mode
	// extraction is mode-agnostic (bracket names are captured regardless).
	ChordMode ChordMode

	doc            *Document
	currentSection *Section
	activeEnv      string
	tabBuffer      []string
}

func NewParser(mode ParserMode) *Parser {
	return &Parser{
		Mode:      mode,
		ChordMode: defaultChordMode,
	}
}

func (p *Parser) Parse(chart string) (*Document, error) {
	p.doc = &Document{
		Metadata: make(map[string][]string, 0),
		Sections: make([]Section, 0),
	}
	p.currentSection = nil
	p.tabBuffer = nil

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
			dir, data := extractDirective(line.Raw)
			if isCommentDirective(dir) {
				p.appendLine(ParsedLine{
					Type:   LineTypeComment,
					Raw:    line.Raw,
					Lyrics: data,
					Chords: nil,
				})
			} else {
				p.handleDirective(dir, data)
			}
		case LineTypeEmpty:
			if p.activeEnv == "tab" {
				p.tabBuffer = append(p.tabBuffer, "")
			} else if p.shouldFlushOnBlankLine() {
				p.flushSection()
			}
		case LineTypeComment:
		default:
			if p.activeEnv == "tab" {
				p.tabBuffer = append(p.tabBuffer, line.Raw)
			} else {
				p.appendLine(line)
			}
		}
	}
	p.flushTabBuffer()
	p.flushSection()
	return nil
}

func (p *Parser) parseBasic(chart string) error {
	if len(chart) == 0 {
		return errors.New("chart is empty, nothing to parse")
	}
	cleanChart := strings.ReplaceAll(chart, "\r\n", "\n")
	lines := strings.Split(cleanChart, "\n")

	type lineKind int
	const (
		kindBlank lineKind = iota
		kindComment
		kindTab
		kindChord
		kindLyric
	)

	class := make([]lineKind, len(lines))
	for i, ln := range lines {
		switch {
		case strings.TrimSpace(ln) == "":
			class[i] = kindBlank
		case strings.HasPrefix(strings.TrimSpace(ln), "#"):
			class[i] = kindComment
		case isTabLine(strings.TrimSpace(ln)):
			class[i] = kindTab
		case isChordLine(ln, p.ChordMode):
			class[i] = kindChord
		default:
			class[i] = kindLyric
		}
	}

	for i := 0; i < len(lines); i++ {
		switch class[i] {
		case kindBlank:
			p.flushSection()
		case kindComment:
		case kindTab:
			p.appendLine(ParsedLine{
				Type:   LineTypeTab,
				Raw:    strings.TrimSpace(lines[i]),
				Lyrics: "",
				Chords: nil,
			})
		case kindChord:
			if i+1 < len(lines) && class[i+1] == kindLyric {
				p.appendLine(ParsedLine{
					Type:   LineTypeChordAndLyric,
					Raw:    lines[i+1],
					Lyrics: lines[i+1],
					Chords: extractBasicChords(lines[i], p.ChordMode),
				})
				i++
			} else {
				p.appendLine(ParsedLine{
					Type:   LineTypeChord,
					Raw:    strings.TrimSpace(lines[i]),
					Lyrics: "",
					Chords: extractBasicChords(lines[i], p.ChordMode),
				})
			}
		case kindLyric:
			p.appendLine(ParsedLine{
				Type:   LineTypeLyric,
				Raw:    lines[i],
				Lyrics: lines[i],
				Chords: nil,
			})
		}
	}
	p.flushSection()
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
	case CategoryIgnored:
		// Spec-legal, unsupported: silently skipped.
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
		p.flushTabBuffer()
		p.flushSection()
		p.activeEnv = env
		p.currentSection = &Section{Name: label}
	case "end":
		p.flushTabBuffer()
		if p.activeEnv == env || p.activeEnv != "" {
			p.flushSection()
		}
	}
}

// flushTabBuffer evaluates a pending {sot} block when the parser leaves the
// tab environment (by {eot}, an unrelated directive, or end of input).
// Qualifying blocks are appended verbatim as tab lines; decorative blocks are
// discarded entirely.
func (p *Parser) flushTabBuffer() {
	defer func() { p.tabBuffer = nil }()
	if p.activeEnv != "tab" || len(p.tabBuffer) == 0 {
		return
	}
	if !isRealTabBlock(p.tabBuffer) {
		return
	}
	for _, raw := range p.tabBuffer {
		p.appendLine(ParsedLine{
			Type:   LineTypeTab,
			Raw:    raw,
			Lyrics: "",
			Chords: nil,
		})
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
