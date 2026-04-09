package edit

import (
	"time"

	"github.com/mike-ward/go-edit/edit/buffer"
	"github.com/mike-ward/go-gui/gui"
)

const doubleClickThresholdMs = 400

// hitTestPosition converts mouse event coordinates to a buffer
// Position, clamped to valid line/col.
func hitTestPosition(
	e *gui.Event,
	frame *editorFrameData,
	buf *buffer.Buffer,
) buffer.Position {
	mx := e.MouseX - frame.gutterW - frame.padLeft
	my := e.MouseY

	// Guard NaN / negative / absurd values.
	if mx != mx || mx < 0 {
		mx = 0
	}
	if my != my || my < 0 {
		my = 0
	}

	lh := frame.lineHeight
	if lh <= 0 || frame.state.Measurer == nil {
		return buffer.Position{}
	}

	line := max(int((my+frame.state.ScrollY)/lh), 0)
	line = min(line, buf.LineCount()-1)

	lineBytes := buf.Line(line)
	col := min(frame.state.Measurer.ColumnForX(lineBytes, mx), len(lineBytes))

	return buffer.Position{Line: line, ByteCol: col}
}

// editorOnClick returns the OnClick handler for the DrawCanvas.
// OnClick fires on mouse-down in go-gui.
func editorOnClick(
	cfg EditorCfg,
	frame *editorFrameData,
) func(*gui.Layout, *gui.Event, *gui.Window) {
	return func(layout *gui.Layout, e *gui.Event, w *gui.Window) {
		if !frame.valid {
			return
		}
		st := loadState(w, cfg.IDFocus)
		pos := hitTestPosition(e, frame, cfg.Buffer)
		now := time.Now().UnixMilli()

		// Click count detection. Use line-only match so minor
		// horizontal jitter doesn't break double/triple-click.
		if now-st.LastClickTime <= doubleClickThresholdMs &&
			st.LastClickPos.Line == pos.Line {
			st.ClickCount++
			if st.ClickCount > 3 {
				st.ClickCount = 3
			}
		} else {
			st.ClickCount = 1
		}
		st.LastClickTime = now
		st.LastClickPos = pos

		switch st.ClickCount {
		case 2: // double-click: word select
			line := cfg.Buffer.Line(pos.Line)
			start, end := wordBoundsAtByte(line, pos.ByteCol)
			st.Anchor = buffer.Position{Line: pos.Line, ByteCol: start}
			st.Cursor = buffer.Position{Line: pos.Line, ByteCol: end}
		case 3: // triple-click: line select
			lineLen := len(cfg.Buffer.Line(pos.Line))
			st.Anchor = buffer.Position{Line: pos.Line, ByteCol: 0}
			st.Cursor = buffer.Position{Line: pos.Line, ByteCol: lineLen}
		default: // single click
			if e.Modifiers.Has(gui.ModShift) {
				// Shift-click extends selection.
				st.Cursor = pos
			} else {
				st.Cursor = pos
				st.Anchor = pos
			}
		}

		st.DesiredCol = st.Cursor.ByteCol
		ensureCursorVisible(&st, frame, cfg.Height)
		storeState(w, cfg.IDFocus, st)

		// Start drag via MouseLock for single clicks.
		if st.ClickCount == 1 {
			w.MouseLock(gui.MouseLockCfg{
				MouseMove: editorDragMove(cfg, frame),
				MouseUp:   editorDragUp(),
			})
		}

		e.IsHandled = true
	}
}

// editorDragMove updates the cursor during a mouse drag.
func editorDragMove(
	cfg EditorCfg,
	frame *editorFrameData,
) func(*gui.Layout, *gui.Event, *gui.Window) {
	return func(layout *gui.Layout, e *gui.Event, w *gui.Window) {
		if !frame.valid {
			return
		}
		st := loadState(w, cfg.IDFocus)
		st.Cursor = hitTestPosition(e, frame, cfg.Buffer)
		st.DesiredCol = st.Cursor.ByteCol
		ensureCursorVisible(&st, frame, cfg.Height)
		storeState(w, cfg.IDFocus, st)
	}
}

// editorDragUp ends the mouse drag.
func editorDragUp() func(*gui.Layout, *gui.Event, *gui.Window) {
	return func(_ *gui.Layout, _ *gui.Event, w *gui.Window) {
		w.MouseUnlock()
	}
}
