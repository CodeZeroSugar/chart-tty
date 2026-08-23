# LESSONS.md

Cumulative pitfalls and decisions discovered during chart-tty work. **Read this before every task** (mandatory pre-task step in AGENTS.md).

Agents: append new entries here when you hit a pitfall or make a non-obvious decision. Keep entries terse and concrete — a future agent should recognize the trap from this list.

## Formatting / code issues

- `fmt.Errorf("...%w")` with no argument, or `%w` inside `fmt.Sprintf`, fails `go vet` — which **blocks `go test` entirely** (vet runs as part of `go test`). Only use `%w` in `Errorf`, always with the wrapped error as an argument.
- Unused variables and loop indexes fail compilation: `for n, line := range` with `n` never used won't compile.
- `fmt.Sprintf("%-10v", intValue)` pads to width+1 (observed: `%-10v` of `3` yields `"3"` + 10 spaces). `LineType` prints as its raw int in `Document.String()`.

## Testing traps

- **Always `-count=1`.** Go caches test results; a cached "ok" hides regressions after non-test edits.
- `reflect.DeepEqual` treats a nil slice as different from an empty slice (`[]Section(nil)` vs `[]Section{}`). Know which one the code produces before writing expectations — parser initializes `Sections: make([]Section, 0)` (non-nil empty); `extractBracketContents` returns `nil` when nothing found.
- Tests encode intended behavior. A failing test is a TODO marker, not a bug to work around — resolve by implementing the feature. Never delete or weaken a red test.
- When flipping a test from "current behavior" to "intended behavior", expect it to stay red until the code changes land. That's correct workflow, not a broken suite.

## Parser / ChordPro specifics

- Sample charts must not contain blank lines inside environment blocks (`{start_of_*}`...`{end_of_*}`) — the parser never flushes on blank lines inside an env, but outside an env blank lines always flush. Blank lines in a fixture can silently merge/split sections you didn't expect.
- `spec.json` has no `define` directive; `{define: ...}` charts fail validation by design.
- Hardcoded `ExampleFile` const in `cmd/chart-tty/main.go` is dead cruft, to be removed during CLI polish (M7).
- `charts/` is gitignored — fixtures are local, never commit them. `output.txt` is also gitignored.
- `ValidateChart` passes bracketless/braiceless charts vacuously. Fixed in M7: `LooksLikeBasicChart` (chord-over-lyrics lines with no `{`/`[`) routes such charts to basic mode in `main.go`.
- Detecting an interactive terminal: use `golang.org/x/term`'s `IsTerminal` on both stdin and stdout — `os.ModeCharDevice` is fooled by `/dev/null` (it's a char device), which wrongly triggers TUI mode and errors out.
- lipgloss auto-disables ANSI when stdout isn't a TTY — piped output is plain text; `--no-color` forces this off in the TUI too.

## Decisions worth remembering

- Directive handling is case-insensitive end-to-end (validation, category dispatch, meta handlers, env aliases).
- `{subtitle}`/`{st}` → Artist field (not a separate subtitle field).
- Blank-line flush rule: outside env = flush; inside env = never; trailing section always flushed at end of parse (so bare lyrics and unclosed envs survive).
- Chord grammar accepts `G6/9` (slash-number bass) but rejects `[----]`, `[N.C.]`, `[H]`, `[Coda]`, `[Gm*]` unless escaped as `[*...]`.
- Comment-family directives (`{comment:}`/`{c:}`/`{comment_italic}`/`{ci}`/`{comment_box}`/`{cb}`) are emitted into the Document as `LineTypeComment` lines so the renderer can style them. `#` comment lines stay ignored in both modes.
- Chord transpose mutates `Document` in place via `Document.Transpose(n)`; the `Key` field stays original and the UI computes the displayed key from a transpose counter, so re-keying doesn't corrupt metadata.
- Renderer chord rows pad to the lyric width; chords past the lyric end keep their absolute column (the chord row simply extends longer).
- TTY detection must use `golang.org/x/term` `IsTerminal` on stdin+stdout; `os.ModeCharDevice` is fooled by `/dev/null`.
- lipgloss auto-disables ANSI when stdout isn't a TTY, so piped output is plain text — config colors only show in the TUI (or when forced).
- AJSON responses in httptest mocks: never embed newlines in a raw JSON string literal — marshal a struct/map instead, or the decoder chokes.
- Basic-mode mode detection: a chart with no `{`/`[` constructs but with chord-over-lyrics lines routes to `ModeBasic` via `LooksLikeBasicChart` (M7).
- AI conversion is strictly opt-in: `--ai-convert` / `--write`, or the `c` TUI key. The validate-retry loop caps at 3 attempts and strips markdown code fences from model output.
- Converter prompt accuracy levers (in order of impact): few-shot worked example covering observed failure modes, explicit messy-pattern transformation rules, retry prompts that include the rejected attempt alongside the validator error. These took the boulevard.pro conversion from "chords dropped, loose labels" to fully validator-clean with labeled environments.
- **Do NOT send `temperature` or `max_tokens` to the OpenCode Go gateway** (`opencode.ai/zen/go/v1`): requests carrying either field hang >9 min with zero bytes; the identical request without them completes in ~2 min. chatRequest deliberately omits them — re-add only per-provider if ever needed.
- The validate-retry loop is not theoretical: on boulevard.pro the model's first attempt left overlapping environments ({start_of_verse: Solo} unclosed before {start_of_tab}); attempt 2, fed the validator error plus the rejected chart, fixed it.
- DeepSeek V4 flash via OpenCode Go: ~2 min per single-attempt conversion of a 131-line chart; a full convert-with-retry cycle ran 3m12s. Budget timeouts accordingly when testing live conversions.
- `--write` emits `<name>.pro` next to source — a `.pro` source gets overwritten in place. Copy to `/tmp` first when converting repo fixtures.