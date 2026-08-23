# chart-tty

A terminal chord chart viewer. Reads song charts in **ChordPro** (`.pro`) format and basic
chord-over-lyrics text format, converts them into a structured document, and renders them in
an interactive TUI with chords aligned above their lyrics.

## Install

Requires Go 1.26+.

```sh
go build ./cmd/chart-tty
```

## Usage

```sh
chart-tty [flags] <chart file>
```

On a terminal, `chart-tty` opens an interactive viewer. When stdout is piped, it prints the
rendered chart as plain text instead.

### Flags

| Flag | Description |
|------|-------------|
| `--transpose N` | Transpose all chords by N semitones (negative for down) |
| `--config PATH` | Config file path (default `~/.config/chart-tty/config.toml`) |
| `--no-color` | Disable colored output (also honors `NO_COLOR` env var) |
| `--version` | Print version and exit |
| `-h`, `--help` | Show usage and exit |

### TUI keys

| Key | Action |
|-----|--------|
| `q` / `ctrl+c` | Quit |
| `j` / `k` or arrows | Scroll down / up |
| `PgUp` / `PgDn` / space | Page scroll |
| `g` / `G` or home / end | Jump to top / bottom |
| `+` / `-` | Transpose up / down |

Keys are remappable via config.

## Chart formats

### ChordPro

The ChordPro format uses inline chords in brackets and `{directives}` for metadata and sections:

```chordpro
{title: Swing Low Sweet Chariot}
{key: G}

{start_of_verse: Verse 1}
The [G]morning sun is ris[D]ing high
{end_of_verse}

{start_of_chorus}
And I'm [G]heading [D]homeward, [Em]homeward [C]bound
{end_of_chorus}
```

Supported directive families (all case-insensitive):

- **Meta**: `title`/`t`, `subtitle`/`st` (maps to artist), `artist`, `composer`, `lyricist`,
  `copyright`, `album`, `year`, `key`, `time`, `tempo`, `duration`, `capo`, `tag`, `meta`, `sorttitle`, `sortartist`
- **Formatting**: `comment`/`c`, `comment_italic`/`ci`, `comment_box`/`cb`, `highlight`/`h`, `image`
- **Environment**: `start_of_verse`/`sov`, `end_of_verse`/`eov`, `start_of_chorus`/`soc`,
  `end_of_chorus`/`eoc`, `start_of_bridge`/`sob`, `end_of_bridge`/`eob`,
  `start_of_tab`/`sot`, `end_of_tab`/`eot`, `start_of_grid`/`sog`, `end_of_grid`/`eog`, `chorus`

Rules:

- Blank lines inside an open environment never split the section; blank lines outside an
  environment always do.
- Non-chord bracket content must be escaped with a leading `*`: `[*N.C.]`, `[*----]`.
- Chords follow the ChordPro chord grammar: root `A`-`G` with optional `b`/`#`, optional
  qualifier (`maj`, `min`, `mi`, `m`, `dim`, `aug`, `sus`, `add`, `h`), extension tokens
  (`7`, `69`, `m7b5`, `7sus4`, `7b5`, `7#9`, `9`, `11`, `13`, `alt`, alterations, `^`/`+`),
  and an optional slash bass (`C/G`, or a number for `G6/9`).

### Basic format

Plain chord-over-lyrics pairs, where each chord line sits directly above its lyric line:

```
C                G
Swing low, sweet chariot,

        D7          G
Comin' for to carry me home
```

Line pairs are recognized automatically. Blank lines separate stanzas, and `#` lines are
treated as comments (not rendered).

## Configuration

`chart-tty` reads a TOML config from `~/.config/chart-tty/config.toml` (or `--config`).
All fields are optional; unset fields fall back to defaults.

```toml
[theme]
header_color = "cyan"    # lipgloss color name for section headers
comment_color = "yellow" # lipgloss color name for comments

[keys]
quit = "q"
scroll_down = "j"
scroll_up = "k"
transpose_up = "+"
transpose_down = "-"

[ai]
base_url = "https://api.openai.com/v1"
api_key = ""             # or CHART_TTY_API_KEY env var
model = "gpt-4o-mini"
```

## Development

```sh
go build ./...
go test ./... -count=1
go vet ./...
```

See `AGENTS.md` for the agentic workflow guide and `docs/LESSONS.md` for known pitfalls.