package parser

import (
	"regexp"
	"strings"
)

var (
	headerCleanRegex = regexp.MustCompile(`^[\[\s]*([^\]:]+)[\]:\s]*$`)
	xMult            = regexp.MustCompile(`\s*x\d+\s*$`)
)

type SongSectionType int

const (
	TypeGeneric = iota
	TypeIntro
	TypeVerse
	TypeChorus
	TypePostChorus
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
	Title            string
	Artist           string
	Sections         []SongSection
	CompiledSections []SongSection
}

func (s *Song) getHeaderKey(rawHeader string) string {
	h := strings.ToLower(strings.TrimSpace(rawHeader))

	h = strings.TrimPrefix(h, "[")
	h = strings.TrimPrefix(h, "(")
	h = strings.TrimPrefix(h, "{")
	h = strings.TrimSuffix(h, "]")
	h = strings.TrimSuffix(h, ")")
	h = strings.TrimSuffix(h, "{")
	h = strings.TrimSuffix(h, ":")
	h = strings.TrimSuffix(h, "x")
	h = xMult.ReplaceAllString(h, "")

	h = strings.Split(h, " x")[0]
	h = strings.Split(h, "repeat")[0]

	return strings.TrimSpace(h)
}

func (s *Song) CompileLayout() {
	if len(s.Sections) == 0 {
		return
	}
	songMap := make(map[string]SongSection, 0)

	for _, section := range s.Sections {
		key := s.getHeaderKey(section.RawHeader)

		if len(section.Rows) > 0 && key != "" {
			if _, ok := songMap[key]; !ok {
				songMap[key] = section
			}
		}
	}

	songLayout := make([]SongSection, 0)

	for _, section := range s.Sections {
		key := s.getHeaderKey(section.RawHeader)
		if len(section.Rows) == 1 {
			if master, ok := songMap[key]; ok {
				recalledSection := SongSection{
					CleanHeader: section.CleanHeader,
					RawHeader:   section.RawHeader,
					Type:        master.Type,
					Rows:        make([]string, len(master.Rows)),
				}
				copy(recalledSection.Rows, master.Rows)

				songLayout = append(songLayout, recalledSection)
				continue
			}
		}
		songLayout = append(songLayout, section)
	}
	s.CompiledSections = songLayout
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
	case strings.HasPrefix(clean, "post chorus"):
		return TypePostChorus
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
