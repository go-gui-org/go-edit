package edit

import (
	"bytes"

	"github.com/mike-ward/go-edit/edit/buffer"
)

// maxIndentWidth caps the indent width to prevent pathological
// allocations from a corrupt IndentStyle.
const maxIndentWidth = 16

// indentUnit returns the bytes for one indent level per style.
func indentUnit(style buffer.IndentStyle) []byte {
	if style.UseTabs {
		return []byte{'\t'}
	}
	w := style.Width
	if w <= 0 {
		w = 4
	}
	if w > maxIndentWidth {
		w = maxIndentWidth
	}
	return bytes.Repeat([]byte{' '}, w)
}

// leadingWhitespace returns the leading tabs/spaces of line.
func leadingWhitespace(line []byte) []byte {
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	// Return a copy to avoid retaining the buffer's line slice.
	out := make([]byte, i)
	copy(out, line[:i])
	return out
}

// indentAction inserts an indent unit at cursor (no selection) or
// prepends an indent to each line in the selection.
func indentAction(st *editorState, buf *buffer.Buffer) {
	unit := indentUnit(buf.Props.IndentStyle)

	if !hasSelection(st) {
		pos := st.Cursor
		c := buf.Apply(buffer.Edit{
			Range:    buffer.Range{Start: pos, End: pos},
			NewBytes: unit,
		})
		st.Cursor = c.AppliedRange.End
		clearSelection(st)
		return
	}

	sel := selectionRange(st)
	// Iterate last line → first to avoid invalidating positions.
	for li := sel.End.Line; li >= sel.Start.Line; li-- {
		// Skip the last line if cursor is at col 0 (not really
		// part of the selection content).
		if li == sel.End.Line && sel.End.ByteCol == 0 && li > sel.Start.Line {
			continue
		}
		pos := buffer.Position{Line: li, ByteCol: 0}
		buf.Apply(buffer.Edit{
			Range:    buffer.Range{Start: pos, End: pos},
			NewBytes: unit,
		})
	}

	// Adjust anchor/cursor columns on affected lines.
	w := len(unit)
	adjustIndent(&st.Anchor, sel, w)
	adjustIndent(&st.Cursor, sel, w)
}

// dedentAction removes one indent unit from the start of the
// current line (no selection) or each line in the selection.
func dedentAction(st *editorState, buf *buffer.Buffer) {
	if !hasSelection(st) {
		removed := dedentLine(buf, st.Cursor.Line)
		st.Cursor.ByteCol -= removed
		if st.Cursor.ByteCol < 0 {
			st.Cursor.ByteCol = 0
		}
		clearSelection(st)
		return
	}

	sel := selectionRange(st)
	for li := sel.End.Line; li >= sel.Start.Line; li-- {
		if li == sel.End.Line && sel.End.ByteCol == 0 && li > sel.Start.Line {
			continue
		}
		removed := dedentLine(buf, li)
		if st.Anchor.Line == li {
			st.Anchor.ByteCol -= removed
			if st.Anchor.ByteCol < 0 {
				st.Anchor.ByteCol = 0
			}
		}
		if st.Cursor.Line == li {
			st.Cursor.ByteCol -= removed
			if st.Cursor.ByteCol < 0 {
				st.Cursor.ByteCol = 0
			}
		}
	}
}

// dedentLine removes one indent unit from the start of line li.
// Returns the number of bytes removed.
func dedentLine(buf *buffer.Buffer, li int) int {
	line := buf.Line(li)
	if len(line) == 0 {
		return 0
	}

	var remove int
	switch line[0] {
	case '\t':
		remove = 1
	case ' ':
		w := buf.Props.IndentStyle.Width
		if w <= 0 {
			w = 4
		}
		if w > maxIndentWidth {
			w = maxIndentWidth
		}
		for remove < w && remove < len(line) && line[remove] == ' ' {
			remove++
		}
	}

	if remove == 0 {
		return 0
	}

	start := buffer.Position{Line: li, ByteCol: 0}
	end := buffer.Position{Line: li, ByteCol: remove}
	buf.Apply(buffer.Edit{Range: buffer.Range{Start: start, End: end}})
	return remove
}

// adjustIndent shifts a position's ByteCol after indent insertion.
func adjustIndent(p *buffer.Position, sel buffer.Range, w int) {
	if p.Line < sel.Start.Line || p.Line > sel.End.Line {
		return
	}
	if p.Line == sel.End.Line && sel.End.ByteCol == 0 &&
		p.Line > sel.Start.Line {
		return
	}
	p.ByteCol += w
}
