# go-edit Architecture

## Construction & frame flow

```
         user code
            │
            ▼
     Editor(EditorCfg) ───── allocates ─────► *editorFrameData (closure-shared)
            │                                         ▲
            │ returns gui.View                        │ captured by closures
            ▼                                         │
     ┌──────────────────┐                             │
     │  gui.View        │ callbacks ──────────────────┘
     │  ───────         │
     │  AmendLayout(w)  │──► loads editorState from StateMap[cfg.IDFocus]
     │  OnDraw(dc)      │    builds text.Measurer lazily, fills frame struct
     │  OnKeyDown(w,e)  │──► keymap.Dispatch → action → Buffer.Apply
     │  OnChar(w,r)     │──► acceptChar → insert via Buffer.Apply
     │  OnMouseScroll   │──► clampScroll → writes ScrollY to StateMap
     └──────────────────┘
```

## Data model

```
    Buffer ([]*line)                    Edit (single-shape)
    ┌─────────────────┐                ┌───────────────────┐
    │ line 0: []byte  │                │ Range (l,c)-(l,c) │
    │ line 1: []byte  │◄── Apply ──────│ NewBytes  []byte  │
    │ line 2: []byte  │    (Edit)      └───────────────────┘
    │    ...          │        │
    └─────────────────┘        ▼
            ▲             ┌─────────┐
            │             │ Change  │──► undo stack (Phase 3)
            │             └─────────┘
            │                 │
            │       ┌─────────┴─────────┐
            │       ▼                   ▼
         EditFilter chain         PostEditFunc observers
                                        │
                                        ▼
                              highlight cache invalidation
                              MarkSet updates
```

## Render pipeline

```
  editorState (StateMap)          Measurer (text/)
  ┌─────────────────┐             ┌────────────────┐
  │ Cursor (l,c)    │             │ advance, lineH │
  │ ScrollY         │             │ XForColumn     │
  │ Selection       │             │ ColumnForX     │
  │ Search          │             │ (ASCII fast +  │
  │ Marks, Folds    │             │ glyph fallback)│
  └─────────────────┘             └────────────────┘
            │                              │
            └──────────────┬───────────────┘
                           ▼
                     OnDraw(*DrawContext)
                           │
             ┌─────────────┼─────────────┐
             ▼             ▼             ▼
         gutter       text + tokens   cursor/sel
        (diagnostics)  (DecorationProvider
                        → highlight/chroma)
                           │
                           ▼
                   DrawCanvas (ID:"" — no cache,
                   viewport-sized, scroll owned by editor)
```

## Package layout

```
  edit/                EditorCfg, Editor(), closures, keymap, actions
   ├── buffer/         Buffer, line, Edit/Change, EditFilter, MarkSet
   ├── highlight/      chroma DecorationProvider + per-line cache
   ├── text/           Measurer (caches advance + lineH)
   └── internal/
       └── fakewin/    headless test fixture (8px advance, 16px lineH)

  examples/
   ├── basic/          minimal CLI demo (CGO)
   └── npad/           showcase editor
```

## Key constraints

- `OnDraw` has no `*Window` / `TextMeasurer` / `Theme` — all captured in closure at `AmendLayout`.
- DrawCanvas cache bypassed (`ID:""`) because buffer/cursor/scroll change every frame.
- Editor owns `ScrollY`; does not use go-gui's `Column(IDScroll)`.
- `Position` = byte offsets `{Line, ByteCol}`; cursor movement is grapheme-aware via `text.Measurer` (go-glyph layout), rune fallback.
- Single mutation choke point: `Buffer.Apply(Edit) Change`.
