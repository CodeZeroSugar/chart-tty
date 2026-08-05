package parser

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
