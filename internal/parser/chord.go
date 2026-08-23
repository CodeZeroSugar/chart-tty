package parser

import (
	"fmt"
	"strings"
)

type Chord struct {
	Root string
	Qual string
	Ext  string
	Bass string
}

var chordQualifiers = []string{"maj", "min", "dim", "aug", "sus", "add", "mi", "m", "h"}

func ParseChord(name string) (Chord, error) {
	if !chordRegex.MatchString(name) {
		return Chord{}, fmt.Errorf("invalid chord name %q", name)
	}

	root := name[:1]
	rest := name[1:]
	if len(rest) > 0 && (rest[0] == 'b' || rest[0] == '#') {
		root += rest[:1]
		rest = rest[1:]
	}

	suffix := rest
	bass := ""
	if idx := strings.IndexByte(rest, '/'); idx >= 0 {
		suffix = rest[:idx]
		bass = rest[idx+1:]
	}

	qual := ""
	ext := suffix
	for _, q := range chordQualifiers {
		if strings.HasPrefix(suffix, q) {
			qual = q
			ext = suffix[len(q):]
			break
		}
	}

	return Chord{Root: root, Qual: qual, Ext: ext, Bass: bass}, nil
}

func (c Chord) String() string {
	var sb strings.Builder
	sb.WriteString(c.Root)
	sb.WriteString(c.Qual)
	sb.WriteString(c.Ext)
	if c.Bass != "" {
		sb.WriteString("/")
		sb.WriteString(c.Bass)
	}
	return sb.String()
}

var pitchClasses = map[string]int{
	"C": 0, "C#": 1, "Db": 1, "D": 2, "D#": 3, "Eb": 3,
	"E": 4, "E#": 5, "Fb": 4, "F": 5, "F#": 6, "Gb": 6,
	"G": 7, "G#": 8, "Ab": 8, "A": 9, "A#": 10, "Bb": 10,
	"B": 11, "Cb": 11, "B#": 0,
}

var sharpScale = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

var flatScale = []string{"C", "Db", "D", "Eb", "E", "F", "Gb", "G", "Ab", "A", "Bb", "B"}

func transposeNote(note string, semitones int) string {
	pc := pitchClasses[note]
	pc = ((pc + semitones) % 12 + 12) % 12
	scale := sharpScale
	if strings.Contains(note, "b") {
		scale = flatScale
	}
	return scale[pc]
}

func (c Chord) Transpose(n int) Chord {
	root := transposeNote(c.Root, n)
	bass := c.Bass
	if bass != "" && bass[0] >= 'A' && bass[0] <= 'G' {
		bass = transposeNote(bass, n)
	}
	return Chord{Root: root, Qual: c.Qual, Ext: c.Ext, Bass: bass}
}

func (d *Document) Transpose(n int) {
	for i := range d.Sections {
		for j := range d.Sections[i].Lines {
			line := &d.Sections[i].Lines[j]
			for k := range line.Chords {
				c, err := ParseChord(line.Chords[k].Name)
				if err != nil {
					continue
				}
				line.Chords[k].Name = c.Transpose(n).String()
			}
		}
	}
}