package edit

import (
	"testing"

	"github.com/mike-ward/go-edit/edit/buffer"
)

func mkState(line, col int) editorState {
	return editorState{
		Cursor:     buffer.Position{Line: line, ByteCol: col},
		DesiredCol: col,
	}
}

func mkBuf(s string) *buffer.Buffer {
	return buffer.FromBytes([]byte(s))
}

// ---------- movement ----------

func TestMoveLeftWithinLine(t *testing.T) {
	st := mkState(0, 3)
	moveLeft(&st, mkBuf("abcdef"))
	if st.Cursor.ByteCol != 2 || st.Cursor.Line != 0 {
		t.Errorf("got %+v", st.Cursor)
	}
}

func TestMoveLeftCrossLine(t *testing.T) {
	st := mkState(1, 0)
	moveLeft(&st, mkBuf("abc\ndef"))
	if st.Cursor.Line != 0 || st.Cursor.ByteCol != 3 {
		t.Errorf("got %+v", st.Cursor)
	}
}

func TestMoveLeftAtStart(t *testing.T) {
	st := mkState(0, 0)
	moveLeft(&st, mkBuf("abc"))
	if st.Cursor != (buffer.Position{}) {
		t.Errorf("got %+v", st.Cursor)
	}
}

func TestMoveRightCrossLine(t *testing.T) {
	st := mkState(0, 3)
	moveRight(&st, mkBuf("abc\ndef"))
	if st.Cursor.Line != 1 || st.Cursor.ByteCol != 0 {
		t.Errorf("got %+v", st.Cursor)
	}
}

func TestMoveRightAtEnd(t *testing.T) {
	st := mkState(0, 3)
	moveRight(&st, mkBuf("abc"))
	if st.Cursor.ByteCol != 3 || st.Cursor.Line != 0 {
		t.Errorf("got %+v", st.Cursor)
	}
}

func TestMoveUpDesiredColPreserved(t *testing.T) {
	st := mkState(2, 10)
	st.DesiredCol = 10
	// Line 1 is shorter; cursor should clamp to its length but
	// DesiredCol should survive.
	moveUp(&st, mkBuf("long line here\nshort\nanother long line"), 1)
	if st.Cursor.Line != 1 || st.Cursor.ByteCol != 5 {
		t.Errorf("got %+v", st.Cursor)
	}
	if st.DesiredCol != 10 {
		t.Errorf("DesiredCol=%d want 10", st.DesiredCol)
	}
}

func TestMoveDownPastEnd(t *testing.T) {
	st := mkState(0, 0)
	moveDown(&st, mkBuf("a\nb\nc"), 100)
	if st.Cursor.Line != 2 {
		t.Errorf("got Line=%d want 2", st.Cursor.Line)
	}
}

// ---------- editing ----------

func TestBackspaceMidLine(t *testing.T) {
	buf := mkBuf("hello")
	st := mkState(0, 3)
	backspace(&st, buf)
	if buf.String() != "helo" {
		t.Errorf("content=%q", buf.String())
	}
	if st.Cursor.ByteCol != 2 {
		t.Errorf("cursor=%+v", st.Cursor)
	}
}

func TestBackspaceJoinsLines(t *testing.T) {
	buf := mkBuf("foo\nbar")
	st := mkState(1, 0)
	backspace(&st, buf)
	if buf.String() != "foobar" {
		t.Errorf("content=%q", buf.String())
	}
	if st.Cursor.Line != 0 || st.Cursor.ByteCol != 3 {
		t.Errorf("cursor=%+v", st.Cursor)
	}
}

func TestBackspaceAtStartNoop(t *testing.T) {
	buf := mkBuf("abc")
	st := mkState(0, 0)
	backspace(&st, buf)
	if buf.String() != "abc" {
		t.Errorf("content=%q", buf.String())
	}
}

func TestDeleteForwardJoinsLines(t *testing.T) {
	buf := mkBuf("foo\nbar")
	st := mkState(0, 3)
	deleteForward(&st, buf)
	if buf.String() != "foobar" {
		t.Errorf("content=%q", buf.String())
	}
	if st.Cursor.ByteCol != 3 || st.Cursor.Line != 0 {
		t.Errorf("cursor=%+v", st.Cursor)
	}
}

func TestDeleteForwardAtEOFNoop(t *testing.T) {
	buf := mkBuf("abc")
	st := mkState(0, 3)
	deleteForward(&st, buf)
	if buf.String() != "abc" {
		t.Errorf("content=%q", buf.String())
	}
}

func TestInsertNewlineSplitsLine(t *testing.T) {
	buf := mkBuf("hello")
	st := mkState(0, 3)
	insertNewline(&st, buf)
	if buf.String() != "hel\nlo" {
		t.Errorf("content=%q", buf.String())
	}
	if st.Cursor.Line != 1 || st.Cursor.ByteCol != 0 {
		t.Errorf("cursor=%+v", st.Cursor)
	}
}

// ---------- scroll ----------

func TestEnsureCursorVisibleScrollsDown(t *testing.T) {
	// Viewport = 100px, lineHeight = 10px → 10 visible lines.
	// Cursor at line 15 while ScrollY=0 → should scroll so
	// cursor line is bottom of viewport.
	st := editorState{
		Cursor:  buffer.Position{Line: 15, ByteCol: 0},
		ScrollY: 0,
	}
	fr := &editorFrameData{lineHeight: 10, valid: true}
	ensureCursorVisible(&st, fr, 100)
	// cy+lh = 160; 160 - 100 = 60.
	if st.ScrollY != 60 {
		t.Errorf("ScrollY=%v want 60", st.ScrollY)
	}
}

func TestEnsureCursorVisibleScrollsUp(t *testing.T) {
	st := editorState{
		Cursor:  buffer.Position{Line: 2, ByteCol: 0},
		ScrollY: 100,
	}
	fr := &editorFrameData{lineHeight: 10, valid: true}
	ensureCursorVisible(&st, fr, 100)
	if st.ScrollY != 20 {
		t.Errorf("ScrollY=%v want 20", st.ScrollY)
	}
}

func TestEnsureCursorVisibleNoop(t *testing.T) {
	st := editorState{
		Cursor:  buffer.Position{Line: 5, ByteCol: 0},
		ScrollY: 20,
	}
	fr := &editorFrameData{lineHeight: 10, valid: true}
	ensureCursorVisible(&st, fr, 100)
	if st.ScrollY != 20 {
		t.Errorf("ScrollY=%v want 20 (unchanged)", st.ScrollY)
	}
}

// ---------- integration: key sequence ----------

func TestEditorFactoryBuilds(t *testing.T) {
	// Just verify Editor(cfg) returns a non-nil View without panic.
	v := Editor(EditorCfg{
		IDFocus: 1,
		Buffer:  mkBuf("hello\nworld"),
		Width:   400,
		Height:  200,
	})
	if v == nil {
		t.Fatal("Editor returned nil")
	}
}
