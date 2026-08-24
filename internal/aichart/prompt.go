package aichart

import (
	"strings"

	"github.com/CodeZeroSugar/chart-tty/internal/parser"
)

// conversionRules wraps the embedded ChordPro spec reference with the
// conversion-specific instructions: output discipline, messy-pattern
// transformations earned against real charts, and the never-emit blacklist of
// constructs chart-tty does not enforce.
var conversionRules = `You are converting chord charts into strictly compliant ChordPro.

Output discipline:
- Output ONLY the converted chart. No commentary, no code fences, no explanations.
- Never truncate the song. Output the complete chart from first line to last.
- Preserve the original lyrics and chord names exactly; never drop or invent content.
- Directives must be alone on their own line.
- Use only these environment types: verse, chorus, bridge, tab, grid.
- Non-chord bracket content must be escaped with a leading asterisk: [*N.C.], [*----].

Never emit any of the following constructs — this project does not support them:
- {define: ...} or {chord: ...} chord definitions
- {transpose: ...}
- Delegated environments: {start_of_abc}, {start_of_ly}, {start_of_svg}, {start_of_textblock}
- Markup tags in lyrics or directives
- Conditional directive selectors (a "-suffix" after a directive name)
- Any directive starting with x_

Transformations for common messy patterns:
- Chord runs written as comma- or space-separated lists on their own line (e.g. "Fm, G# Eb Bb") become individually bracketed chords on one line: [Fm] [G#] [Eb] [Bb]. Never merge them into one bracket or turn them into plain text.
- Repetition markers like "x2" or "x3" become escaped annotations on their own line: [*x2].
- Bracketed text that is not a chord name ([Riff], [Chorus], [Post chorus]) becomes escaped text: [*Riff], [*Chorus], [*Repeat intro].
- Section labels like "Verse 1:" or "CHORUS:" are removed as loose lines and become labeled environment directives instead: {start_of_verse: Verse 1} ... {end_of_verse}.
- Lines consisting only of dashes or riff marks become escaped annotations: [*------].
- If chords sit on their own line above lyrics (chord-over-lyrics layout), merge each chord line into its lyric line as inline [chords] placed before the syllable they sit over.

The relevant specification sections follow:

` + promptSpecCore(parser.SpecReference)

// promptSpecCore extracts the sections of the spec reference that are safe
// and useful to send to the model. The full document stays canonical for
// agents; the gateway, however, reproducibly hangs or 500s when the system
// prompt exceeds ~2.5K tokens with a full chart as user input, so sections
// that are redundant with the wrapper rules (tab verbatim semantics,
// sources, unsupported families — already covered by the never-emit list)
// are excluded. See docs/LESSONS.md.
func promptSpecCore(ref string) string {
	ranges := []struct{ start, end string }{
		{start: "## 1.", end: "\n## 3."},
		{start: "## 4.", end: "\n## 5."},
	}
	var sb strings.Builder
	for _, r := range ranges {
		i := strings.Index(ref, r.start)
		if i < 0 {
			continue
		}
		seg := ref[i:]
		if j := strings.Index(seg, r.end); j >= 0 {
			seg = seg[:j]
		}
		sb.WriteString(strings.TrimRight(seg, "\n"))
		sb.WriteString("\n\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func BuildPrompt(chart string) (system, user string) {
	return conversionRules + promptSpecCore(parser.SpecReference), chart
}

func StripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimPrefix(s, "chordpro")
		s = strings.TrimSpace(s)
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	}
	return strings.TrimSpace(s)
}