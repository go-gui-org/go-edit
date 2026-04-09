package edit

import (
	"github.com/mike-ward/go-edit/edit/buffer"
	"github.com/mike-ward/go-gui/gui"
)

// defaultActions maps action IDs to their implementations.
// This is the single source of truth for built-in editor
// actions; the default keymap and any user keymaps reference
// these by string ID.
//
// Actions without PreservesAnchor have Anchor = Cursor applied
// automatically after execution by the dispatch in editorOnKeyDown.
var defaultActions = map[string]Action{
	// ---- cursor movement ----

	"cursor.left": {
		ID: "cursor.left",
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			if hasSelection(st) {
				st.Cursor = selectionRange(st).Start
				return
			}
			moveLeft(st, buf)
		},
	},
	"cursor.right": {
		ID: "cursor.right",
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			if hasSelection(st) {
				st.Cursor = selectionRange(st).End
				return
			}
			moveRight(st, buf)
		},
	},
	"cursor.up": {
		ID: "cursor.up",
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			moveUp(st, buf, 1)
		},
		PreservesDesiredCol: true,
	},
	"cursor.down": {
		ID: "cursor.down",
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			moveDown(st, buf, 1)
		},
		PreservesDesiredCol: true,
	},
	"cursor.home": {
		ID: "cursor.home",
		Execute: func(_ EditorCfg, st *editorState, _ *buffer.Buffer, _ *gui.Window) {
			st.Cursor.ByteCol = 0
		},
	},
	"cursor.end": {
		ID: "cursor.end",
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			st.Cursor.ByteCol = len(buf.Line(st.Cursor.Line))
		},
	},

	// ---- selection (extends from Anchor) ----

	"select.left": {
		ID:              "select.left",
		PreservesAnchor: true,
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			moveLeft(st, buf)
		},
	},
	"select.right": {
		ID:              "select.right",
		PreservesAnchor: true,
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			moveRight(st, buf)
		},
	},
	"select.up": {
		ID:                  "select.up",
		PreservesAnchor:     true,
		PreservesDesiredCol: true,
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			moveUp(st, buf, 1)
		},
	},
	"select.down": {
		ID:                  "select.down",
		PreservesAnchor:     true,
		PreservesDesiredCol: true,
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			moveDown(st, buf, 1)
		},
	},
	"select.home": {
		ID:              "select.home",
		PreservesAnchor: true,
		Execute: func(_ EditorCfg, st *editorState, _ *buffer.Buffer, _ *gui.Window) {
			st.Cursor.ByteCol = 0
		},
	},
	"select.end": {
		ID:              "select.end",
		PreservesAnchor: true,
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			st.Cursor.ByteCol = len(buf.Line(st.Cursor.Line))
		},
	},
	"select.all": {
		ID:              "select.all",
		PreservesAnchor: true,
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			st.Anchor = buffer.Position{}
			lastLine := buf.LineCount() - 1
			st.Cursor = buffer.Position{
				Line:    lastLine,
				ByteCol: len(buf.Line(lastLine)),
			}
		},
	},

	// ---- editing ----

	"edit.backspace": {
		ID: "edit.backspace",
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			if hasSelection(st) {
				buf.BeginGroup()
				deleteSelection(st, buf)
				buf.EndGroup()
				return
			}
			backspace(st, buf)
		},
	},
	"edit.delete": {
		ID: "edit.delete",
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			if hasSelection(st) {
				buf.BeginGroup()
				deleteSelection(st, buf)
				buf.EndGroup()
				return
			}
			deleteForward(st, buf)
		},
	},
	"edit.newline": {
		ID: "edit.newline",
		Execute: func(cfg EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			buf.BeginGroup()
			insertNewline(cfg, st, buf)
			buf.EndGroup()
		},
	},
	"edit.undo": {
		ID: "edit.undo",
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			r := buf.Undo()
			if r.OK {
				st.Cursor = r.Cursor.Cursor
				st.Anchor = r.Cursor.Anchor
			}
		},
	},
	"edit.redo": {
		ID: "edit.redo",
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			r := buf.Redo()
			if r.OK {
				st.Cursor = r.Cursor.Cursor
				st.Anchor = r.Cursor.Anchor
			}
		},
	},

	// ---- clipboard ----

	"edit.copy": {
		ID: "edit.copy",
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, w *gui.Window) {
			if !hasSelection(st) {
				return
			}
			w.SetClipboard(buf.TextInRange(selectionRange(st)))
		},
		PreservesAnchor: true,
	},
	"edit.cut": {
		ID: "edit.cut",
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, w *gui.Window) {
			if !hasSelection(st) {
				return
			}
			w.SetClipboard(buf.TextInRange(selectionRange(st)))
			buf.BeginGroup()
			deleteSelection(st, buf)
			buf.EndGroup()
		},
	},
	"edit.paste": {
		ID: "edit.paste",
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, w *gui.Window) {
			text := w.GetClipboard()
			if len(text) == 0 {
				return
			}
			// Cap paste at MaxLoadBytes to prevent OOM from a
			// pathological clipboard.
			if len(text) > buffer.MaxLoadBytes {
				text = text[:buffer.MaxLoadBytes]
			}
			buf.BeginGroup()
			deleteSelection(st, buf)
			pos := st.Cursor
			c := buf.Apply(buffer.Edit{
				Range:    buffer.Range{Start: pos, End: pos},
				NewBytes: []byte(text),
			})
			st.Cursor = c.AppliedRange.End
			buf.EndGroup()
		},
	},

	// ---- indent ----

	"edit.indent": {
		ID: "edit.indent",
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			indentAction(st, buf)
		},
		PreservesAnchor: true,
	},
	"edit.dedent": {
		ID: "edit.dedent",
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			dedentAction(st, buf)
		},
		PreservesAnchor: true,
	},
}

// pageUpAction and pageDownAction need EditorCfg for viewport
// height, so they're registered separately as closures.
func pageUpAction(cfg EditorCfg, frame *editorFrameData) Action {
	return Action{
		ID: "cursor.pageup",
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			moveUp(st, buf, pageLines(frame, cfg.Height))
		},
		PreservesDesiredCol: true,
	}
}

func pageDownAction(cfg EditorCfg, frame *editorFrameData) Action {
	return Action{
		ID: "cursor.pagedown",
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			moveDown(st, buf, pageLines(frame, cfg.Height))
		},
		PreservesDesiredCol: true,
	}
}

// selectPageUpAction and selectPageDownAction extend selection.
func selectPageUpAction(cfg EditorCfg, frame *editorFrameData) Action {
	return Action{
		ID:              "select.pageup",
		PreservesAnchor: true,
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			moveUp(st, buf, pageLines(frame, cfg.Height))
		},
		PreservesDesiredCol: true,
	}
}

func selectPageDownAction(cfg EditorCfg, frame *editorFrameData) Action {
	return Action{
		ID:              "select.pagedown",
		PreservesAnchor: true,
		Execute: func(_ EditorCfg, st *editorState, buf *buffer.Buffer, _ *gui.Window) {
			moveDown(st, buf, pageLines(frame, cfg.Height))
		},
		PreservesDesiredCol: true,
	}
}
