package edit

import (
	"testing"

	"github.com/mike-ward/go-edit/edit/buffer"
)

// ---------- clampCursor ----------

func TestClampCursor_LineOOB(t *testing.T) {
	st := editorState{Cursor: buffer.Position{Line: 99, ByteCol: 0}}
	clampCursor(&st, mkBuf("a\nb\nc"))
	if st.Cursor.Line != 2 {
		t.Errorf("Line=%d want 2", st.Cursor.Line)
	}
}

func TestClampCursor_ColOOB(t *testing.T) {
	st := editorState{Cursor: buffer.Position{Line: 0, ByteCol: 99}}
	clampCursor(&st, mkBuf("abc"))
	if st.Cursor.ByteCol != 3 {
		t.Errorf("ByteCol=%d want 3", st.Cursor.ByteCol)
	}
}

func TestClampCursor_NegativeCursor(t *testing.T) {
	st := editorState{Cursor: buffer.Position{Line: -5, ByteCol: -3}}
	clampCursor(&st, mkBuf("abc"))
	if st.Cursor.Line != 0 || st.Cursor.ByteCol != 0 {
		t.Errorf("got %+v", st.Cursor)
	}
}

func TestClampCursor_EmptyBuffer(t *testing.T) {
	st := editorState{Cursor: buffer.Position{Line: 3, ByteCol: 7}}
	clampCursor(&st, buffer.New())
	if st.Cursor.Line != 0 || st.Cursor.ByteCol != 0 {
		t.Errorf("got %+v", st.Cursor)
	}
}

// ---------- clampScroll ----------

func TestClampScroll_LargeBuffer(t *testing.T) {
	cfg := EditorCfg{Buffer: mkBuf("a\nb\nc\nd\ne"), Height: 20}
	st := editorState{ScrollY: 1000}
	clampScroll(&st, cfg, 10) // 5 lines * 10 = 50; 50-20 = 30 max
	if st.ScrollY != 30 {
		t.Errorf("ScrollY=%v want 30", st.ScrollY)
	}
}

func TestClampScroll_BufferFitsInViewport(t *testing.T) {
	cfg := EditorCfg{Buffer: mkBuf("a\nb"), Height: 100}
	st := editorState{ScrollY: 50}
	clampScroll(&st, cfg, 10)
	if st.ScrollY != 0 {
		t.Errorf("ScrollY=%v want 0", st.ScrollY)
	}
}

func TestClampScroll_NegativeIn(t *testing.T) {
	cfg := EditorCfg{Buffer: mkBuf("a\nb\nc"), Height: 10}
	st := editorState{ScrollY: -50}
	clampScroll(&st, cfg, 10)
	if st.ScrollY != 0 {
		t.Errorf("ScrollY=%v want 0", st.ScrollY)
	}
}

// ---------- pageLines ----------

func TestPageLines_ExactFit(t *testing.T) {
	fr := &editorFrameData{lineHeight: 10}
	if n := pageLines(fr, 100); n != 10 {
		t.Errorf("got %d want 10", n)
	}
}

func TestPageLines_ZeroLineHeight(t *testing.T) {
	fr := &editorFrameData{lineHeight: 0}
	if n := pageLines(fr, 100); n != 1 {
		t.Errorf("got %d want 1 (safe fallback)", n)
	}
}

func TestPageLines_SubLineViewport(t *testing.T) {
	fr := &editorFrameData{lineHeight: 10}
	if n := pageLines(fr, 5); n != 1 {
		t.Errorf("got %d want 1", n)
	}
}

// ---------- acceptChar ----------

func TestAcceptChar_AllowsPrintableAndTab(t *testing.T) {
	allowed := []rune{'a', 'Z', '5', '!', ' ', '\t', 'é', '日', '€'}
	for _, r := range allowed {
		if !acceptChar(r) {
			t.Errorf("rejected %q", r)
		}
	}
}

func TestAcceptChar_RejectsControl(t *testing.T) {
	rejected := []rune{0, '\n', '\r', '\x01', '\x1f', 0x7f}
	for _, r := range rejected {
		if acceptChar(r) {
			t.Errorf("accepted %q (%U)", r, r)
		}
	}
}

// ---------- movement edges ----------

func TestMoveUp_AtTop(t *testing.T) {
	st := mkState(0, 2)
	moveUp(&st, mkBuf("abc\ndef"), 1)
	if st.Cursor.Line != 0 {
		t.Errorf("Line=%d want 0", st.Cursor.Line)
	}
}

func TestMoveDown_AtBottom(t *testing.T) {
	st := mkState(1, 2)
	moveDown(&st, mkBuf("abc\ndef"), 1)
	if st.Cursor.Line != 1 {
		t.Errorf("Line=%d want 1 (clamped)", st.Cursor.Line)
	}
}

// ---------- edit edges ----------

func TestBackspace_EmptyBufferNoop(t *testing.T) {
	buf := buffer.New()
	st := mkState(0, 0)
	backspace(&st, buf)
	if buf.String() != "" || st.Cursor != (buffer.Position{}) {
		t.Errorf("content=%q cursor=%+v", buf.String(), st.Cursor)
	}
}

func TestDeleteForward_EmptyBufferNoop(t *testing.T) {
	buf := buffer.New()
	st := mkState(0, 0)
	deleteForward(&st, buf)
	if buf.String() != "" {
		t.Errorf("content=%q", buf.String())
	}
}

func TestInsertNewline_AtLineStart(t *testing.T) {
	buf := mkBuf("hello")
	st := mkState(0, 0)
	insertNewline(EditorCfg{Buffer: buf}, &st, buf)
	if buf.String() != "\nhello" {
		t.Errorf("content=%q", buf.String())
	}
	if st.Cursor.Line != 1 || st.Cursor.ByteCol != 0 {
		t.Errorf("cursor=%+v", st.Cursor)
	}
}

func TestInsertNewline_AtLineEnd(t *testing.T) {
	buf := mkBuf("hello")
	st := mkState(0, 5)
	insertNewline(EditorCfg{Buffer: buf}, &st, buf)
	if buf.String() != "hello\n" {
		t.Errorf("content=%q", buf.String())
	}
	if st.Cursor.Line != 1 || st.Cursor.ByteCol != 0 {
		t.Errorf("cursor=%+v", st.Cursor)
	}
}

// ---------- ensureCursorVisible edges ----------

func TestEnsureCursorVisible_TinyViewport(t *testing.T) {
	// Viewport smaller than a line: cursor should still resolve to
	// a non-negative scroll.
	st := editorState{Cursor: buffer.Position{Line: 5, ByteCol: 0}}
	fr := &editorFrameData{lineHeight: 10, valid: true}
	ensureCursorVisible(&st, fr, 5)
	if st.ScrollY < 0 {
		t.Errorf("ScrollY=%v negative", st.ScrollY)
	}
}

func TestEnsureCursorVisible_InvalidFrame(t *testing.T) {
	st := editorState{Cursor: buffer.Position{Line: 10, ByteCol: 0}, ScrollY: 7}
	fr := &editorFrameData{lineHeight: 10, valid: false}
	ensureCursorVisible(&st, fr, 100)
	if st.ScrollY != 7 {
		t.Errorf("ScrollY=%v want 7 (unchanged)", st.ScrollY)
	}
}
