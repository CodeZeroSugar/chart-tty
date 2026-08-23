package aichart

import "strings"

const systemRules = `You convert chord charts into strictly compliant ChordPro output.

Rules:
- Chords go inline in [brackets], e.g. [C] [F#m7b5].
- Directives use {name: value} or {name}. Valid directives: title/t, subtitle/st, artist, composer, lyricist, album, year, key, time, tempo, capo, comment/c, comment_italic/ci, comment_box/cb, start_of_verse/sov, end_of_verse/eov, start_of_chorus/soc, end_of_chorus/eoc, start_of_bridge/sob, end_of_bridge/eob, start_of_tab/sot, end_of_tab/eot, chorus.
- Chord names: root A-G with optional b/#, optional qualifier (maj, min, mi, m, dim, aug, sus, add, h), extensions (7, 69, m7b5, 7sus4, 7b5, 7#9, 9, 11, 13, alt, alterations), optional slash bass like C/G.
- Non-chord bracket content must be escaped with a leading asterisk: [*N.C.], [*----].
- Wrap verses/choruses/bridges/tabs in {start_of_X}...{end_of_X} directives.
- Preserve the original lyrics, chord names, and structure as closely as possible.
- Output ONLY the converted chart. No commentary, no code fences, no explanations.`

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