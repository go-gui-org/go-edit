# Changelog

## Unreleased

- **BREAKING: event callbacks take a single `gui.EventCtx`.** deps: bump go-gui
  to v0.52.0. Callbacks that took `(*gui.Layout, *gui.Event, *gui.Window)` now
  take `func(gui.EventCtx)`, exposing the three as `ctx.Layout`, `ctx.Event` and
  `ctx.Window`.
- **Consume-class callbacks are handled by default.** `OnClick`, `OnChar` and
  `OnFileDrop` are marked handled by dispatch before the callback runs. Five
  editor paths that previously fell through without consuming now say so
  explicitly with `ctx.Bubble()`: a read-only editor rejecting typed input (two
  paths), a non-printable character, a click before the first layout pass, and
  an empty file drop. Without those the editor would have swallowed Tab
  traversal and accelerators.
- Migration guide upstream: `docs/migration-eventctx.md` in go-gui.

## v0.10.3 — 2026-05-24

- scroll: keep cursor 8px from horizontal viewport edge.
- deps: bump go-gui to v0.20.2, go-glyph to v1.8.0.

## v0.10.2 — 2026-05-17

- deps: bump go-gui to v0.19.1 (scroll phase bridge, context menu focus fix).
- deps: bump go-gui to v0.19.0, go-glyph to v1.7.1 (animation heartbeat, Metal
  autorelease fix).
- lint: use `slices.Backward` for reverse loops in undo and fold.

## v0.10.1 — 2026-05-01

- buffer/watcher: emit `WatchDeleted` once and unwatch missing paths.
- buffer/watcher: detect external edits by `(modTime, size)`.
- buffer/save: stream `(*Buffer).WriteTo` via `io.Copy`; surface short writes as
  `io.ErrShortWrite`.
- buffer/save: snapshot + recheck symlink target before atomic commit; fail on
  mid-save target changes.
- buffer: cut multiline insert allocations via in-place splice.

## v0.10.0 — 2026-04-30

- editor: harden `EditorCfg.Font` override. Empty `Family` borrows
  `theme.M5.Family`; NaN / Inf / non-positive `Size` borrows `theme.M5.Size`;
  oversized `Size` clamped to 1024. Prevents proportional-font fallback and
  huge-glyph allocations from hostile or partially-populated configs.
- editor: per-widget `Font` override (`EditorCfg.Font`) for callers that want a
  non-theme monospace style (e.g. npad uses SF Mono Terminal).
- deps: bump go-gui v0.12.5 → v0.17.0, go-glyph → v1.7.0; drop local `replace`
  directives now that go-gui carries the upstream `TextMeasurer` surface.

## v0.9.0

Initial tagged release.
