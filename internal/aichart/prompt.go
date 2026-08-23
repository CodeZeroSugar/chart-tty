package aichart

import "strings"

const systemRules = `You convert chord charts into strictly compliant ChordPro output.

Rules:
- Chords go inline in [brackets], e.g. [C] [F#m7b5].
- Directives use {name: value} or {name}. Valid directives: title/t, subtitle/st, artist, composer, lyricist, album, year, key, time, tempo, capo, comment/c, comment_italic/ci, comment_box/cb, start_of_verse/sov, end_of_verse/eov, start_of_chorus/soc, end_of_chorus/eoc, start_of_bridge/sob, end_of_bridge/eob, start_of_tab/sot, end_of_tab/eot, chorus.
- Chord names: root A-G with optional b/#, optional qualifier (maj, min, mi, m, dim, aug, sus, add, h), extensions (7, 69, m7b5, 7sus4, 7b5, 7#9, 9, 11, 13, alt, alterations), optional slash bass like C/G.
- Non-chord bracket content must be escaped with a leading asterisk: [*N.C.], [*Riff], [*Repeat verse 2].
- Wrap verses/choruses/bridges/tabs in {start_of_X}...{end_of_X} directives.
- Preserve the original lyrics and chord names exactly; never drop or invent content.

Transformations for common messy patterns:
- Chord runs written as comma- or space-separated lists on their own line (e.g. "Fm, G# Eb Bb") become individually bracketed chords on one line: [Fm] [G#] [Eb] [Bb]. Never merge them into one bracket or turn them into plain text.
- Repetition markers like "x2" or "x3" become escaped annotations on their own line: [*x2].
- Bracketed text that is not a chord name ([Riff], [Chorus], [Post chorus]) becomes escaped text: [*Riff], [*Chorus], [*Post chorus].
- Section labels like "Verse 1:" or "CHORUS:" are removed as loose lines and become labeled environment directives instead: {start_of_verse: Verse 1} ... {end_of_verse}.
- Lines consisting only of dashes or riff marks become escaped annotations: [*------].
- If chords sit on their own line above lyrics (chord-over-lyrics layout), merge each chord line into its lyric line as inline [chords] placed before the syllable they sit over.
- Never truncate the song. Output the complete chart from first line to last.

Example conversion:

Input:
C          G
Swing low, sweet chariot,

CHORUS:
Fm, Bb x2
[Repeat intro]
------

Output:
[C]Swing low, sweet [G]chariot,

{start_of_chorus}
[Fm] [Bb]
[*x2]
[*Repeat intro]
[*------]
{end_of_chorus}

Output ONLY the converted chart. No commentary, no code fences, no explanations.`

func BuildPrompt(chart string) (system, user string) {
	return systemRules, chart
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