# ChordPro Specification Reference

Reference for agents working on chart-tty. Distilled from the official specification at
[chordpro.org](https://www.chordpro.org/chordpro/) (retrieved 2026-08-23). Normative
statements are preserved close to verbatim; each section cites its source page.

**Read this before answering "what does the spec say" questions.** Where chart-tty's
behavior deliberately differs from the spec, see [Chart-tty deltas](#chart-tty-deltas).

## Sources

| Section | Source |
|---|---|
| Format basics | [chordpro-introduction](https://www.chordpro.org/chordpro/chordpro-introduction/) |
| Directives overview | [chordpro-directives](https://www.chordpro.org/chordpro/chordpro-directives/) |
| Environments | [directives-env](https://www.chordpro.org/chordpro/directives-env/) |
| Tab environment | [directives-env_tab](https://www.chordpro.org/chordpro/directives-env_tab/) |
| Chords | [chordpro-chords](https://www.chordpro.org/chordpro/chordpro-chords/) |

## 1. File format basics

The ChordPro file format is a simple text file format. Common filename extensions are
`.cho`, `.crd`, `.chopro`, `.chord` and `.pro`.

A song consists of:

- **Lyrics-and-chords lines**: lyrics interspersed with chords written between `[` and `]`.
  The chords are placed in front of the syllable they belong to.
- **Directives**: lines that start with `{` and end with `}`. As with every ChordPro
  directive, these should be alone on a line.
- **Empty lines**.
- **Remarks**: all lines that start with a `#` are ignored.

Example:

```
{title: Swing Low Sweet Chariot}

{start_of_chorus}
Swing [D]low, sweet [G]chari[D]ot,
Comin' for to carry me [A7]home.
{end_of_chorus}

{comment: Chorus}
```

**Annotations** are textual remarks placed above the lyrics, just like chords, specified
with `[*`*text*`]`. In strict chord parsing, content in brackets that does not form a
valid chord name must be escaped this way.

## 2. Chords

ChordPro can parse chord names in two modes: *strict* and *relaxed*.

In **strict** mode (default), chord names are recognized only if they consist of:

- a **root note**, e.g. `C`, `F#` or `Bb`
- an optional **qualifier**, e.g. `m` (minor), `aug` (augmented)
- an optional **extension**, which must be one of the built-in extension names
- an optional **bass**, a slash `/` followed by another root note

Examples: `C`, `F#`, `Besm`, `Am7`, `C/B`.

When parsed, the constituents register as chord properties `root`, `qual`, `ext`, `bass`
(and `name`). In **relaxed** mode the extension is not required to be known — e.g.
`[Coda]` parses as root `C` plus extension `oda`.

Built-in extensions include (non-exhaustive; full list on the source page):

- Major: `2 3 4 5 6 69 7 7b5 7#5 7#9 7sus4 9 9sus4 11 13 13#11 alt add2 add4 add9 sus2 sus4 maj7 maj7#5 maj13 ^7 ^9 ...`
- Minor (`m`/`mi`/`min`/`-`): `m7b5 mmaj7 m9 m11 m6 madd9 msus4 m7sus4 ...`
- Other: `aug + dim dim7 h h7 h9`

## 3. Directives

Directives control appearance and metadata. A directive has a *name* and optionally an
*argument* separated by a colon: `{title: Swing Low Sweet Chariot}`. Many directives have
long and short names (e.g. `title` / `t`); using the full name is advised.

Arguments may use attribute syntax for multi-argument directives, e.g.
`{image: src="myimage.jpg" scale="50%"}`. Single-value directives also accept bare or
attribute forms equivalently: `{start_of_verse Verse 1}` ≡ `{start_of_verse label="Verse 1"}`.

### Meta-data directives

`title` (t), `sorttitle`, `subtitle` (st), `artist`, `sortartist`, `composer`, `lyricist`,
`copyright`, `album`, `year`, `key`, `time`, `tempo`, `duration`, `capo`, `tag`, `meta`.

### Formatting directives

`comment` (c), `highlight` (h), `comment_italic` (ci), `comment_box` (cb), `image`.

### Environment directives

See section 4.

### Other families (not enforced by chart-tty)

`new_song`; delegated environments (`start_of_abc/ly/svg/textblock`); chord diagrams
(`define`, `chord`); transposition (`transpose`); font/size/colour legacy props;
output directives (`new_page`, `column_break`, ...); legacy (`grid`, `titles`, `columns`).

### Custom extensions

Any directive whose name starts with `x_` must be completely ignored by applications that
do not handle it — no warning generated. Conventionally followed by an app namespace tag.

### Conditional directives

Any directive can be conditionally selected by postfixing `-` plus a selector, e.g.
`{start_of_verse-soprano}` / `{define-guitar: ...}`. Selection matches instrument type,
user name, then meta items; reversed by appending `!`.

## 4. Environments

Environments (also called *sections*) group series of input lines into meaningful units.
They start with a `start_of` directive and end with a corresponding `end_of` directive;
the directives should be alone on a line.

Arbitrary section names are allowed as long as they consist of letters, digits and
underscores. Unknown/unhandled environments should always be treated as part of the song
lyrics. Environments `chorus`, `tab`, and `grid` get predefined special treatment.

All environment directives may include an optional label:
`{start_of_verse: label="Verse 1"}` or, for backward compatibility,
`{start_of_verse: Verse 1}`.

Short forms exist for the standard environments: `soc`/`eoc`, `sov`/`eov`, `sob`/`eob`,
`sot`/`eot`, `sog`/`eog`. The standalone `{chorus}` directive indicates a chorus.

## 5. Tab environments

`{start_of_tab}` (short `sot`) indicates that the following lines form a section of guitar
TAB instructions. **The lines will not be folded or changed. Markup is left as is, and
directives are considered literal text except for** `{end_of_tab}` and `{eot}`.

May include an optional label: `{start_of_tab: Solo}`.

## Chart-tty deltas

Deliberate deviations and stricter enforcement compared to the spec above:

- **Case-insensitive directives**: dispatch and matching lowercase everything; the spec
  treats directive names case-sensitively in practice.
- **`{subtitle}`/`{st}` maps to the Artist field** — a chart-tty convention based on
  real-world charts.
- **Strict bracket-content validation**: non-chord bracket content fails validation unless
  escaped with `[*...]`. Per spec, bracket content is technically free-form ("it doesn't
  matter what you put between the []") unless needed for diagrams/transposition — but
  chart-tty validates to keep rendered output sane.
- **Chord grammar is a fixed subset**: root `[A-G]` + optional `b`/`#`, qualifier set
  (`maj,min,mi,m,dim,aug,sus,add,h`), numeric/altered extension tokens, slash bass that may
  be a root or a number (`G6/9`). No relaxed mode, no German `H`, no Roman/Nashville roots.
- **Tab gate**: a `{sot}` block renders as tablature only if ≥4 lines match
  `^[A-Ga-g][ \t]*\|`; decorative blocks are dropped entirely. The spec says tab lines are
  passed through unchanged but places no structural requirement on them.
- **Blank-line flush rule**: outside any environment, blank lines flush the current
  section; inside one, never; trailing sections flush at end of parse. The spec leaves
  empty-line semantics to implementations.
- **No support** (by design): delegated environments, `{define}`, `{transpose}`,
  conditional selectors, markup, font/size/colour legacy directives. Unknown directives are
  ignored during parsing (spec-compliant for `x_*`).
