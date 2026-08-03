// Package parser internal/parser/parser.go
package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var headerCleanRegex = regexp.MustCompile(`^[\[\s]*([^\]:]+)[\]:\s]*$`)

type SongSectionType int

const (
	TypeGeneric = iota
	TypeIntro
	TypeVerse
	TypeChorus
	TypeBridge
	TypeInterlude
	TypeRiff
	TypeOutro
)

type SongSection struct {
	RawHeader   string
	CleanHeader string
	Type        SongSectionType
	Rows        []string
}

type Song struct {
	Title    string
	Artist   string
	Sections []SongSection
}

func (s *Song) Normalize() {
	for i := 0; i < len(s.Sections); i++ {
		raw := s.Sections[i].RawHeader
		if strings.TrimSpace(raw) == "" {
			s.Sections[i].Type = TypeGeneric
			continue
		}

		matches := headerCleanRegex.FindStringSubmatch(raw)
		var clean string
		if len(matches) > 1 {
			clean = strings.ToLower(strings.TrimSpace(matches[1]))
		} else {
			clean = strings.ToLower(strings.TrimSpace(raw))
		}

		s.Sections[i].Type = s.detectSectionType(clean)
		s.Sections[i].CleanHeader = clean
	}
	s.organizeSections()
}

func (s *Song) detectSectionType(clean string) SongSectionType {
	switch {
	case strings.HasPrefix(clean, "intro"):
		return TypeIntro
	case strings.HasPrefix(clean, "verse"):
		return TypeVerse
	case strings.HasPrefix(clean, "chorus"):
		return TypeChorus
	case strings.HasPrefix(clean, "bridge"):
		return TypeBridge
	case strings.HasPrefix(clean, "interlude"):
		return TypeInterlude
	case strings.HasPrefix(clean, "riff"):
		return TypeRiff
	case strings.HasPrefix(clean, "outro"):
		return TypeOutro
	default:
		return TypeGeneric
	}
}

func (s *Song) organizeSections() {
	if len(s.Sections) == 0 {
		return
	}

	collapsed := make([]SongSection, 0, len(s.Sections))
	var activeSection *SongSection

	for i := 0; i < len(s.Sections); i++ {
		current := s.Sections[i]
		if current.Type == TypeGeneric {
			if activeSection == nil {
				collapsed = append(collapsed, SongSection{
					RawHeader: "Content",
					Type:      TypeGeneric,
					Rows:      make([]string, 0),
				})
				activeSection = &collapsed[len(collapsed)-1]
			}

			activeSection.Rows = append(activeSection.Rows, current.Rows...)

		} else {
			collapsed = append(collapsed, current)

			activeSection = &collapsed[len(collapsed)-1]
		}

	}
	s.Sections = collapsed
}

type Parser struct {
	IncludeTabs bool
}

func NewParser(includeTabs bool) *Parser {
	return &Parser{IncludeTabs: includeTabs}
}

func (p *Parser) ParseFile(filePath string) (*Song, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return &Song{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	song := &Song{}
	var section SongSection

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "{t:") || strings.HasPrefix(line, "{title:") {
			song.Title = strings.TrimSuffix(strings.Split(line, ":")[1], "}")
			continue
		}
		if strings.HasPrefix(line, "{st:") || strings.HasPrefix(line, "{subtitle:") {
			song.Artist = strings.TrimSuffix(strings.Split(line, ":")[1], "}")
			continue
		}

		if line == "" {
			if len(section.Rows) > 0 {
				section.RawHeader = section.Rows[0]
				song.Sections = append(song.Sections, section)
				section = SongSection{}

			}
			continue
		}
		section.Rows = append(section.Rows, line)

	}

	if len(section.Rows) > 0 {
		song.Sections = append(song.Sections, section)
	}

	return song, scanner.Err()
}
