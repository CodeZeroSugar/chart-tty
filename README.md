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
| `--set-api-key KEY` | Store the AI API key in the config file (`-` reads it from stdin) |
| `--no-color` | Disable colored output (also honors `NO_COLOR` env var) |
| `--ai-convert` | Convert chart to compliant ChordPro via AI |
| `--write` | Write converted chart to `<name>.pro` next to source (requires `--ai-convert`) |
| `--version` | Print version and exit |
| `-h`, `--help` | Show usage and exit |

### TUI keys

| Key | Action |
|-----|--------|
| `q` / `ctrl+c` | Quit |
| `j` / `k` or arrows | Scroll down / up |
| `space` / `PgDn` | Page down |
| `b` / `PgUp` | Page up |
| `g` / `G` or home / end | Jump to top / bottom |
| `+` / `-` | Transpose up / down |
| `c` | Convert the loaded chart to ChordPro via AI |

Keys are remappable via config; `c` is not.

### Wide terminals

On terminals wide enough for two 40-character columns (≥83 columns), charts render
side by side with a vertical break down the center — content fills the left column,
then continues in the right, like book pages. The split only kicks in when content
exceeds one full page; shorter charts stay single-column. Narrower terminals use
single-column left-aligned layout. Overlong lines (wide tab blocks) soft-wrap within
their column.

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
- `{sot}` blocks are treated as tablature only when at least four lines lead with a musical
  letter and a pipe (`E|------`); decorative `{sot}` blocks used purely for spacing are dropped.
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

## AI conversion

Messy, non-compliant chord charts can be reformatted into fully compliant ChordPro using an
LLM endpoint (OpenAI-compatible API — works with OpenAI, compatible gateways, and local models
like Ollama / LM Studio).

```sh
chart-tty --ai-convert <messy-chart.txt>          # convert, then view/print
chart-tty --ai-convert --write <messy-chart.txt>  # also write <name>.pro next to source
```

The converter sends your chart plus a ChordPro rules summary to the model, validates the output
with the built-in validator, and retries (up to 3 attempts) feeding validation errors back.

**Privacy:** conversion is strictly opt-in — charts are never sent automatically. Only
`--ai-convert` or the `c` key in the TUI triggers a request. Local endpoints (Ollama/LM Studio)
keep charts on your machine; any other endpoint sends the chart text to that server.

Endpoint, key, and model come from the `[ai]` config section or the env vars
`CHART_TTY_BASE_URL`, `CHART_TTY_API_KEY`, `CHART_TTY_MODEL`.

### Example: OpenCode Go

[OpenCode Go](https://opencode.ai/docs/zen/) exposes an OpenAI-compatible endpoint that
serves DeepSeek, GLM, Kimi and other open models — one key, no code changes needed:

```toml
[ai]
base_url = "https://opencode.ai/zen/go/v1"
api_key = ""                  # set via: chart-tty --set-api-key <key>
model = "deepseek-v4-flash"   # or deepseek-v4-pro
```

```sh
chart-tty --ai-convert --write messy-chart.txt
```

To store the key in your config file (comments and other settings are preserved):

```sh
chart-tty --set-api-key sk-...     # or: chart-tty --set-api-key -  (key read from stdin)
```

The config file is written with `0600` permissions since it holds a secret. Note that a
set `CHART_TTY_API_KEY` env var takes precedence over the stored value, and passing the
key as an argument leaves it in your shell history — prefer the stdin form for shared machines.

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