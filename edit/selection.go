package edit

import "github.com/mike-ward/go-edit/edit/buffer"

// hasSelection reports whether the editor has an active selection.
func hasSelection(st *editorState) bool {
	return st.Anchor != st.Cursor
}

// selectionRange returns the ordered [Start, End) range of the
// current selection. If no selection, returns an empty range at
// Cursor.
func selectionRange(st *editorState) buffer.Range {
	return orderedRange(st.Anchor, st.Cursor)
}

// orderedRange returns a range with Start <= End.
func orderedRange(a, b buffer.Position) buffer.Range {
	if a.After(b) {
		a, b = b, a
	}
	return buffer.Range{Start: a, End: b}
}

// clearSelection collapses the selection by moving Anchor to Cursor.
func clearSelection(st *editorState) {
	st.Anchor = st.Cursor
}

// deleteSelection deletes the selected text and places the cursor at
// the start of the deleted range. Returns true if a selection existed
// and was deleted.
func deleteSelection(st *editorState, buf *buffer.Buffer) bool {
	if !hasSelection(st) {
		return false
	}
	sel := selectionRange(st)
	buf.Apply(buffer.Edit{Range: sel})
	st.Cursor = sel.Start
	st.Anchor = sel.Start
	return true
}
