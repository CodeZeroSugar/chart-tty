package parser

import (
	"fmt"
	"strings"
)

type LineType int

const (
	LineTypeEmpty LineType = iota
	LineTypeDirective
	LineTypeComment
	LineTypeChordAndLyric
	LineTypeChord
	LineTypeLyric
	LineTypeTab
)

type ChordToken struct {
	Name     string
	Position int
}

type Document struct {
	Title    string
	Artist   string
	Key      string
	Tempo    string
	Capo     string
	Metadata map[string][]string
	Sections []Section
}

func (d *Document) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Title: %s\n", d.Title))
	sb.WriteString(fmt.Sprintf("Artist: %s\n", d.Artist))
	sb.WriteString("----------------------------------------\n")

	for _, sec := range d.Sections {
		sb.WriteString(fmt.Sprintf("[%s]\n", sec.Name))
		for _, line := range sec.Lines {
			sb.WriteString(fmt.Sprintf("  Type: %-10v Raw: %s\n", line.Type, line.Raw))
		}
	}
	return sb.String()
}

type Section struct {
	Name  string
	Type  string
	Lines []ParsedLine
}

type ParsedLine struct {
	Type   LineType
	Raw    string
	Lyrics string
	Chords []ChordToken
}
