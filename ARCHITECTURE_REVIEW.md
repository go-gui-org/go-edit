# go-edit — architecture review

Scope: `edit/`, `edit/buffer/`, `edit/text/`, `edit/highlight/`,
`ROADMAP.md`, `CLAUDE.md`. Review based on structure, not runtime
profiling. ~22 files in `edit/`, ~20 in `edit/buffer/`, 51 test files,
~10.7K test LOC.

## Strengths

1. **Design tenets stated and followed.** Pure Go, headless-testable,
   allocation-conscious, immediate-mode. Every subsequent decision
   traces back to one of these. Rare discipline.
2. **Single mutation choke point.** `Buffer.Apply(Edit) Change` is
   the only path that mutates the document. Undo, filters, observers,
   mark tracking, and decorations all hang off this. Classic correct
   shape for an editor core.
3. **Extension substrate was built before features.** `EditFilter`
   chain, `PostEditFunc`, `MarkSet`, `DecorationProvider` — plugin
   points existed before syntax highlighting, search, multi-cursor,
   folds consumed them. Features fit the substrate instead of
   bolting on.
4. **Locked type decisions, documented.** `Position = {Line, ByteCol}`,
   `Edit` as a single shape (no tagged union), `Change` as undo
   record. The "Phase -1 — Decisions (locked)" section in ROADMAP is
   the right pattern: pin shapes early, defer semantics.
5. **Closure discipline around the framework constraint.**
   `OnDraw(*DrawContext)` has no `*Window`; `Editor(cfg)` closes over
   `*editorFrameData`, `AmendLayout` populates it, `OnDraw` reads it.
   Documented in CLAUDE.md as a first-class constraint. Good.
6. **Hardening tripwires as tests.** `*_hardening_test.go` files
   codify the trust boundary: nil buffer, NaN dims, absurd scroll
   deltas, nil filter, double-remove. Invariants as executable specs.
7. **Headless fixture in `internal/fakewin`.** Deterministic fake
   TextMeasurer (8 px / 16 px) lets driver tests run without CGO.
   Scoped under `internal/` so the fake cannot leak to consumers.
8. **Fuzz coverage on Buffer.Apply.** `FuzzBufferApply` guards the
   choke point against crash-class bugs. Few editor projects bother.
9. **StateMap namespace convention.** Dotted keys (`edit.state`),
   capacity hint. Matches go-gui's broader convention; no ad-hoc
   globals.
10. **Atomic save via `atomicwrite.go`** with its own tests. Not an
    afterthought.

## Architectural concerns

### 1. Line storage drift and long-line pathology

CLAUDE.md says `Buffer.lines` is `[]byte`; actual code is `[]*line`
(pointer-per-line). Documentation is wrong, and more importantly each
line is an independent heap object — conflicts with the
allocation-conscious tenet. Line splits churn allocations.

Bigger: `MaxLoadBytes = 256 MiB` advertises support that the
slice-of-line-objects model cannot honour well. One 50 MB minified
line (the open question in ROADMAP) makes `XForColumn`, wrap, bracket
scan, and fold detection all go quadratic or worse.

**Recommend:**
- Fix CLAUDE.md immediately.
- Either lower `MaxLoadBytes` to what the current model survives
  (benchmark it) or commit to the per-line gap buffer now. Leaving
  the gap buffer "deferred until benchmarks justify" while advertising
  256 MiB is a latent bug.
- Add a long-line cap (`MaxLineBytes`, e.g. 1 MiB) that triggers a
  "dense line" read-only mode. Protects the measurement + wrap paths
  from adversarial input.

### 2. Per-frame O(buffer) walks

`totalVisualRowsForBuffer` iterates every line (and, for wrapped
lines, every line's break list) on every scroll clamp. `wrapMap`
is rebuilt each `AmendLayout`. Bracket match scans up to 10k bytes
per frame. Fold lookup (`isFolded`, `nextVisible`) is a linear scan.

At the current test sizes this is invisible. At 256 MiB it is
catastrophic. Incremental state is the correct fix and the substrate
is already there (`PostEditFunc` can invalidate cached wrap rows for
the affected range).

**Recommend:**
- Cache `totalVisualRows` keyed by `(Buffer.Version, wrapWidth, folds
  hash)`; recompute only on mismatch.
- Build `wrapMap` incrementally from `PostEditFunc`, not from scratch
  in `AmendLayout`.
- Sort `FoldedRanges` and binary-search. An interval tree is overkill
  until there are thousands of folds.

### 3. DrawCanvas `ID:""` bypass is a permanent cost

The editor uses `ID:""` to defeat go-gui's per-shape draw cache
because cursor/scroll/buffer change every frame. Correct as a
workaround, but it means the editor re-runs its entire draw path on
every frame even when nothing changed (idle with a stable cursor
blink). That is a lot of floor cost.

**Recommend:**
- Push upstream: a DrawCanvas that accepts a `Version uint64` the
  widget computes itself from `(Buffer.Version, ScrollY, cursorHash,
  selectionHash, frameTickForBlink)`. Keep the framework cache and
  invalidate it exactly when something visible changed.
- Short term: gate the cursor blink to only redraw the cursor cell,
  not the whole widget.

### 4. Chroma re-tokenization cost

"Full-buffer tokenization with per-line cache; invalidated on any
edit." For a 1 MB Go file, every keystroke re-lexes 1 MB. The
per-line cache does not actually help if the invalidation is global.

**Recommend:**
- Persist chroma lexer state at line boundaries. Re-tokenize from the
  first affected line to the first line where the persisted state
  matches the cached state (Emacs-style restart heuristic).
- Or: debounce tokenization behind a 16 ms timer and render stale
  tokens in the meantime. Cheap and usually invisible.

### 5. Closure-shared `*editorFrameData` is per-widget, not per-frame

`Editor(cfg)` allocates a single `*editorFrameData` and closes over
it. If the same `Editor` view is realized twice (split view, pop-out,
list rendering the same widget factory) the closure-shared struct is
a write/write race between the two instances.

**Recommend:**
- Document "Editor(cfg) result is single-instance-use; construct a
  new Editor per mount site" as a contract — and check it in
  `AmendLayout` with a sentinel.
- Better: move `editorFrameData` into StateMap keyed by `IDFocus`.
  Then AmendLayout and OnDraw both look it up. One allocation per
  focus ID, not per `Editor(cfg)` call.

### 6. Per-cursor dispatch cost at scale

`dispatchPerCursor` uses a swap-to-index-0 trick and a PostEditFunc
observer that adjusts every other cursor after every Apply. For N
cursors and M edits, that is N×M. 1000 cursors typing a character is
1M adjustments.

**Recommend:**
- Collect all edits from the group, apply them in reverse, then do
  one sweep over the non-edited cursors to re-derive positions from
  the group's total delta map. The observer becomes a no-op inside
  `BeginGroup`/`EndGroup`.

### 7. Keymap dispatch is linear

`Keymap.Bindings` is an ordered slice; `KeymapStack` walks layers
and within each layer scans bindings. Replace each layer with a
`map[key]ActionID` where `key = (KeyCode << 16) | Modifier`. O(1)
dispatch, identical semantics, less code. The ordered slice only
matters for the "help screen" enumeration — keep a parallel slice
for that.

### 8. `DecorationProvider.Decorate` returns a fresh slice

Called once per frame, per visible range, per provider. Each call
allocates. Change to `Decorate(vp Viewport, out []Decoration) []Decoration`
and let the editor pass in a reusable buffer. Matches the
allocation-conscious tenet and costs nothing.

### 9. EditorCfg surface area

`EditorCfg` already has ~12 fields and more are coming (LSP,
minimap, completion…). Cluster into sub-configs:
`EditorCfg{Display DisplayCfg, Input InputCfg, Lang LangCfg, Theme
EditorTheme}`. Additive changes stay confined to one sub-struct.

### 10. Status/ROADMAP drift

CLAUDE.md status block says "Phase 6 is the latest committed";
ROADMAP marks Phases 7 and 8 complete. The divergence is small and
easy, but CLAUDE.md is what future-you (and future me) reads first.
Treat it as the source of truth or delete the status block and link
to ROADMAP.

### 11. Minor

- `findMatchingBracket` 10k cap is documented but silent — make it
  return a `(found, hitCap bool)` so the UI can show "no match" vs
  "search truncated."
- `detectIndent` scans up to 1000 non-empty lines on load. Cheap, but
  it's O(file) — move behind a goroutine triggered from `Load` so
  the first frame does not wait on it.
- `dc.Polyline` stack-allocation claim for squiggles — verify with
  `go build -gcflags='-m'`. Escape analysis regressions are silent.
- Keymap loading from a config file is missing. The stack supports
  it; a JSON/TOML loader would be ~100 lines and unlocks user
  customization without rebuilds.
- `ensureCursorVisible` / `clampScroll` sanitize their inputs. Make
  sure the scroll-delta accumulator (if any) clamps *before*
  accumulation, not after.

## On the AI criticism

The artifact does not read like LLM output dropped into a repo. It
reads like a codebase someone thought about hard:

- Constraints documented as constraints, not as "here's how it
  works."
- Phase -1 locks type decisions with rationale before any feature
  code.
- Hardening invariants are a test class, not a comment.
- Fuzz tests exist at the mutation choke point.
- Upstream-first policy (push to go-gui rather than work around) is
  stated and followed (`(*Window).TextMeasurer()` getter was pushed
  up).

Those are process choices. A model cannot make those without a human
picking them. The output reflects the choices regardless of how the
keystrokes arrived. A reviewer focused on provenance rather than
product is measuring the wrong thing. The honest defense is the
ROADMAP, the hardening tests, the fuzz harness, the locked-decisions
section, and the fact that the constraint list in CLAUDE.md proves
someone read the framework's internals.

That said, two things would strengthen the defense materially:

1. **Fix the documentation drift** (line storage, phase status).
   Drift is a tell that nobody is reading the docs. Keep them live.
2. **Land the per-line gap buffer, or lower `MaxLoadBytes`.** The
   advertised capability gap is the kind of thing a skeptic notices
   first. Closing it demonstrates engineering judgment that an
   LLM-only workflow would not reliably produce.

## Prioritized action list

1. Sync CLAUDE.md with ROADMAP and the actual buffer line type. (trivial)
2. Add `MaxLineBytes` cap + dense-line mode. (small)
3. Move `editorFrameData` to StateMap, drop the closure. (small)
4. Cache `totalVisualRows` and build `wrapMap` incrementally. (medium)
5. Incremental chroma tokenization with line-boundary state. (medium)
6. Map-based keymap dispatch + `Decorate` output-slice API. (small, easy wins)
7. Gap buffer or lowered `MaxLoadBytes`. (large / policy)
8. Upstream DrawCanvas version-keyed cache. (large, coordinate with go-gui)

## Unresolved questions

- split-view / multi-mount semantics for `Editor(cfg)` — contract?
- `MaxLoadBytes` truthful ceiling on current model — benchmarked?
- chroma lexer state serializable at line boundaries for all
  supported languages?
- theme: extend go-gui Theme, or keep `EditorTheme` standalone?
