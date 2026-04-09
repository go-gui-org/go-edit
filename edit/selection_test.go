package edit

import (
	"testing"

	"github.com/mike-ward/go-edit/edit/buffer"
)

func TestHasSelection(t *testing.T) {
	st := editorState{
		Cursor: buffer.Position{Line: 0, ByteCol: 3},
		Anchor: buffer.Position{Line: 0, ByteCol: 3},
	}
	if hasSelection(&st) {
		t.Error("same pos should not be selection")
	}
	st.Anchor.ByteCol = 0
	if !hasSelection(&st) {
		t.Error("different pos should be selection")
	}
}

func TestSelectionRange_Ordered(t *testing.T) {
	// Cursor before anchor.
	st := editorState{
		Cursor: buffer.Position{Line: 0, ByteCol: 0},
		Anchor: buffer.Position{Line: 1, ByteCol: 5},
	}
	r := selectionRange(&st)
	if r.Start != st.Cursor || r.End != st.Anchor {
		t.Errorf("got %+v", r)
	}

	// Anchor before cursor (reversed).
	st.Cursor, st.Anchor = st.Anchor, st.Cursor
	r = selectionRange(&st)
	if r.Start != st.Anchor || r.End != st.Cursor {
		t.Errorf("reversed: got %+v", r)
	}
}

func TestDeleteSelection(t *testing.T) {
	buf := buffer.FromBytes([]byte("hello world"))
	st := editorState{
		Cursor: buffer.Position{Line: 0, ByteCol: 5},
		Anchor: buffer.Position{Line: 0, ByteCol: 0},
	}
	ok := deleteSelection(&st, buf)
	if !ok {
		t.Error("should return true")
	}
	if buf.String() != " world" {
		t.Errorf("buf=%q", buf.String())
	}
	if st.Cursor != (buffer.Position{}) {
		t.Errorf("cursor=%+v", st.Cursor)
	}
	if hasSelection(&st) {
		t.Error("selection should be cleared")
	}
}

func TestDeleteSelection_NoSelection(t *testing.T) {
	buf := buffer.FromBytes([]byte("abc"))
	pos := buffer.Position{Line: 0, ByteCol: 1}
	st := editorState{Cursor: pos, Anchor: pos}
	ok := deleteSelection(&st, buf)
	if ok {
		t.Error("should return false for no selection")
	}
	if buf.String() != "abc" {
		t.Errorf("buf=%q (should be unchanged)", buf.String())
	}
}
