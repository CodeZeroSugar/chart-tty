# AGENTS.md

Guide for agentic workflows on the chart-tty project. Read fully before starting work.

## Project overview

chart-tty is a terminal chord chart viewer. It reads song charts in ChordPro (`.pro`) format and basic chord-over-lyrics text format, converts them into a structured `Document`, and renders them in an interactive TUI.

The project is agent-driven: agents implement features end-to-end (production code plus tests), review their own work, and commit at green checkpoints. The human reviews diffs and steers scope.

## Tech stack & policies

- Go 1.26+, module `github.com/CodeZeroSugar/chart-tty`
- Package ownership and dependency rules:
  - `internal/parser` — **stdlib-pure**. No external imports ever. This is the core domain logic.
  - `internal/ui` — bubbletea + lipgloss (added at M3/M4)
  - `internal/config` — BurntSushi/toml (added at M6)
  - `internal/aichart` — stdlib `net/http` only (no LLM SDK)
- Tests: stdlib `testing` only, table-driven subtests. No test frameworks.
- All directives come from `internal/parser/spec.json`, embedded via `go:embed` and exposed through the package-level `Spec` global (`internal/parser/spec.go`). `init()` panics if any directive list is empty — never ship an empty spec list.

## Architecture map

```
cmd/chart-tty/main.go      CLI entrypoint: flags, file loading, mode detection, hand-off to UI
internal/parser/           validation + parsing -> Document
  spec.go / spec.json      embedded ChordPro directive lists (meta / formatting / environment)
  validator.go             ValidateChart() + chordRegex (strict chord grammar)
  chord.go                 Chord model: ParseChord / Transpose / String; Document.Transpose
  helpers.go               extractDirective, parseEnvDirective, category lookup, tab/chord-line detection
  document.go              Document / Section / ParsedLine / ChordToken model
  parser.go                Parse() -> parseChordPro() / parseBasic(); env + meta handlers
internal/ui/               render.go (headless Render + RenderConfig), app.go (bubbletea Model + keybinds)
internal/config/           TOML config: theme colors, keybindings, AI settings
internal/aichart/          LLM conversion pipeline (OpenAI-compatible client) -> Validator retry loop
charts/                    local sample charts (gitignored, NOT committed) - smoke-test fixtures
```

## Locked domain rules

Settled decisions. Do not re-litigate; if a task contradicts one, stop and flag it.

**Format authority**: for questions about the ChordPro format itself, obey
`docs/chordpro-spec.md` (distilled from chordpro.org with per-section citations). The
"chart-tty deltas" section there lists where this project deliberately diverges.

- **Directive handling is case-insensitive** everywhere (validation, category dispatch, meta/env handlers, env aliases).
- `{subtitle}` and `{st}` map to the **Artist** field.
- **Strict chord grammar** (validator + anything parsing chord names): root `[A-G]` with optional `b`/`#`; optional qualifier (`maj`, `min`, `mi`, `m`, `dim`, `aug`, `sus`, `add`, `h`); extension tokens (`7`, `69`, `m7b5`, `7sus4`, `7b5`, `7#9`, `9`, `11`, `13`, `alt`, `+`, alterations, `sus/add/maj/^` forms); bass is a root note (`C/G`) **or a number** (`G6/9`).
- **Non-chord bracket content is invalid** unless escaped with a leading `*`: `[*N.C.]` and `[*----]` are valid; `[N.C.]`, `[----]`, `[H]`, `[Coda]`, `[Gm*]` are invalid.
- **Blank-line flush rules** (parser):
  - Outside an open environment: blank lines always flush the current section.
  - Inside an open environment (`{start_of_*}`...`{end_of_*}`): blank lines never flush.
  - The trailing section is always flushed at end of parse (unclosed envs and bare lyrics are preserved, not dropped).
- **Section naming**: the directive label if provided (`{start_of_verse: Verse 1}` → `"Verse 1"`), else the env name (`{start_of_chorus}` → `"chorus"`). Bare lines outside any env produce unnamed sections (`Name: ""`).
- **`parseLine` classification order** (first match wins): empty → directive (`{`) → comment (`#`) → tab (`isTabLine`) → chord extraction.
- `{chorus}` alone opens a chorus environment.
- An `{end_of_*}` directive flushes whatever environment is currently active, even if the names don't match.
- **Tab gate**: a `{sot}` block counts as tablature only if ≥4 of its lines match `^[A-Ga-g][ \t]*\|` (any musical letter, mandatory pipe). Qualifying blocks render verbatim; non-qualifying blocks (decorative line breaks) are dropped from the Document entirely.

## Workflow

### Task lifecycle

1. **Pre-task**: read `docs/LESSONS.md` first. Restate the plan. Enumerate edge cases against the locked domain rules before writing code.
2. **Implement**: features end-to-end (production code + tests). Keep the parser stdlib-pure; keep tests table-driven.
3. **Self-review checklist** (before handoff):
   - `go test ./... -count=1` green
   - `go vet ./...` clean
   - Smoke run over every chart in `charts/`: `go run ./cmd/chart-tty <chart>`
   - Diff matches the stated plan; no scope creep
   - Conventions kept (table-driven, stdlib-pure parser, spec-driven directives)
   - Errors wrapped correctly (`%w` only in `Errorf`, with an argument)
4. **Post-milestone retro**: re-read the full diff end-to-end and actively challenge it — edge cases, error paths, test gaps. Append newly discovered pitfalls to `docs/LESSONS.md`.

### Testing philosophy

- **Tests encode intended behavior.** A failing (red) test is a TODO marker for unfinished work — resolve it by implementing the feature, never by deleting or weakening the test.
- Add table-driven cases for every new behavior, including edge cases and regressions.
- Always run tests with `-count=1`; Go caches results and a cached "ok" hides changes after non-test edits.

### Auto-commit policy

Agents commit automatically at **green checkpoints**: a completed task or milestone with a fully green suite and clean vet.

- Only ever commit green. Never commit a red suite — history must stay bisectable.
- Conventional Commits format: `feat(scope):`, `fix(scope):`, `test:`, `docs:`, `chore:`.
- One logical change per commit; production code and its tests go together.
- Never mix unrelated changes; never commit secrets; stage deliberately.
- Do not commit `charts/` (gitignored, local fixtures) or `output.txt`.

## Commands

```sh
go build ./...
go test ./... -count=1
go vet ./...
go run ./cmd/chart-tty charts/long_road_home.pro
```

## Known gotchas

- `fmt.Errorf("...%w")` with no argument, or `%w` inside `fmt.Sprintf`, fails `go vet` — which blocks `go test` entirely. Only use `%w` in `Errorf` and always pass the wrapped error.
- Unused variables/loop indexes fail compilation (e.g. `for n, line := range` with unused `n`).
- `reflect.DeepEqual`: a nil slice is not equal to an empty slice (`[]Section(nil)` vs `[]Section{}`). Know which one the code under test produces.
- `%-10v` on an int pads to width+1 (observed); `LineType` is printed as its raw int by `Document.String()`.
- Test caching: use `-count=1` or stale results will mask failures.
- Fixture charts must not contain blank lines inside environment blocks, or they violate the parser's flush rules.
- Hardcoded `ExampleFile` in `cmd/chart-tty/main.go` is dead cruft; remove during CLI polish (M7).
- `spec.json` has no `define` directive — `{define: ...}` charts will fail validation by design.

## Roadmap

Each milestone exits green: suite + vet clean, smoke run over all `charts/`, committed.

- **M1 — Basic-format parser**: implement `parseBasic` (chord-over-lyrics two-line pairs → same `Document` model). Tests first.
- **M2 — Chord model**: structured chord type (root/qual/ext/bass) parsed from chord names; `Transpose(n semitones)`; serializer back to a name string. Pure logic, heavily unit-tested (reuse the strict grammar; cover the edge chords already in the validator tests).
- **M3 — Renderer**: pure `render(doc, cfg) -> lines` — chords aligned above lyrics via `ChordToken.Position`, section headers, tab blocks verbatim, styled comments. Testable headless (no TTY).
- **M4 — TUI shell**: bubbletea model wrapping the renderer; viewport scrolling, quit/resize key handling. Add bubbletea + lipgloss deps.
- **M5 — Transpose UX**: `+`/`-` keybinds in the TUI and a `--transpose N` CLI flag; show the current key.
- **M6 — Config**: `internal/config` — TOML load with defaults at `~/.config/chart-tty/config.toml`; theme colors, keybind remap. Add BurntSushi/toml dep.
- **M7 — CLI polish + README**: `--help`, `--version`, `--no-color`, `--transpose`, `--ai-convert`, `--write`; proper exit codes; remove hardcoded `ExampleFile`; rewrite README with usage and format docs. Also add a mode-detection heuristic so basic (chord-over-lyrics) charts route to `ModeBasic` (validator passes them vacuously today — see LESSONS.md).
- **M8 — AI-driven chart converter**: `internal/aichart` — takes most existing chord charts (messy, non-compliant) and reformats them to fully ChordPro-compliant output.
  - OpenAI-compatible client (hand-rolled `net/http`): `base_url`, `api_key`, `model` from TOML config, overridable via env `CHART_TTY_API_KEY`, `CHART_TTY_BASE_URL`, `CHART_TTY_MODEL`. Works with OpenAI, compatible gateways, and local models (Ollama/LM Studio).
  - Conversion loop: build prompt (raw chart + ChordPro rule summary, "output only compliant ChordPro") → call model → run the existing `Validator.ValidateChart` on the output → on failure feed the validation errors back and retry (bounded, ~3 attempts) → surface original + errors if still failing.
  - Triggers: `--ai-convert` CLI flag (`--write` emits `<name>.pro` next to source) and a TUI keybind (e.g. `c`) converting the loaded chart in-session.
  - Privacy stance: explicit opt-in only, never auto-send; document that non-local endpoints mean charts leave the machine; local-model option is supported.
  - Tests: `httptest` mock server for the client, prompt-builder unit tests, retry-loop tests with scripted responses.