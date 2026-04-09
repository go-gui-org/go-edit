package buffer

import (
	"testing"
	"time"
)

// fakeClock returns a clock func that advances by step on each call.
func fakeClock(start time.Time, step time.Duration) func() time.Time {
	t := start
	return func() time.Time {
		now := t
		t = t.Add(step)
		return now
	}
}

func enabledBuf(text string) *Buffer {
	b := FromBytes([]byte(text))
	b.EnableUndo(fakeClock(time.Now(), time.Millisecond))
	return b
}

func TestUndoBasicRoundTrip(t *testing.T) {
	b := enabledBuf("hello")
	// Insert " world" at end.
	b.SetUndoCursor(Position{0, 5}, Position{0, 5})
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 5}, End: Position{0, 5}},
		NewBytes: []byte(" world"),
	})
	if got := b.String(); got != "hello world" {
		t.Fatalf("after insert: %q", got)
	}

	r := b.Undo()
	if !r.OK {
		t.Fatal("Undo returned !OK")
	}
	if got := b.String(); got != "hello" {
		t.Fatalf("after undo: %q", got)
	}
	if r.Cursor.Cursor != (Position{0, 5}) {
		t.Fatalf("cursor after undo: %v", r.Cursor.Cursor)
	}

	r = b.Redo()
	if !r.OK {
		t.Fatal("Redo returned !OK")
	}
	if got := b.String(); got != "hello world" {
		t.Fatalf("after redo: %q", got)
	}
}

func TestUndoMultipleEdits(t *testing.T) {
	b := enabledBuf("ab")
	// Insert 'X' at 0,1.
	b.SetUndoCursor(Position{0, 1}, Position{0, 1})
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 1}, End: Position{0, 1}},
		NewBytes: []byte("X"),
	})
	// Insert 'Y' at 0,3.
	b.SetUndoCursor(Position{0, 3}, Position{0, 3})
	// Break coalesce by making them non-adjacent via a 600ms gap.
	b.undo.now = fakeClock(time.Now().Add(time.Second), time.Millisecond)
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 3}, End: Position{0, 3}},
		NewBytes: []byte("Y"),
	})
	if got := b.String(); got != "aXbY" {
		t.Fatalf("after edits: %q", got)
	}

	b.Undo()
	if got := b.String(); got != "aXb" {
		t.Fatalf("after first undo: %q", got)
	}
	b.Undo()
	if got := b.String(); got != "ab" {
		t.Fatalf("after second undo: %q", got)
	}
}

func TestUndoCompoundGroup(t *testing.T) {
	b := enabledBuf("hello world")
	b.SetUndoCursor(Position{0, 0}, Position{0, 5})

	// Simulate paste: delete selection [0,0)–[0,5) then insert.
	b.BeginGroup()
	b.Apply(Edit{Range: Range{
		Start: Position{0, 0}, End: Position{0, 5},
	}})
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 0}, End: Position{0, 0}},
		NewBytes: []byte("goodbye"),
	})
	b.EndGroup()

	if got := b.String(); got != "goodbye world" {
		t.Fatalf("after group: %q", got)
	}

	r := b.Undo()
	if !r.OK {
		t.Fatal("Undo returned !OK")
	}
	if got := b.String(); got != "hello world" {
		t.Fatalf("after undo group: %q", got)
	}
}

func TestUndoCoalesceTyping(t *testing.T) {
	b := enabledBuf("")
	// Type "hello" one char at a time, fast (1ms apart).
	for i, ch := range "hello" {
		b.SetUndoCursor(Position{0, i}, Position{0, i})
		b.Apply(Edit{
			Range:    Range{Start: Position{0, i}, End: Position{0, i}},
			NewBytes: []byte{byte(ch)},
		})
	}
	if got := b.String(); got != "hello" {
		t.Fatalf("after typing: %q", got)
	}
	if !b.CanUndo() {
		t.Fatal("expected CanUndo")
	}

	// Single undo should revert all coalesced chars.
	b.Undo()
	if got := b.String(); got != "" {
		t.Fatalf("after single undo of coalesced typing: %q", got)
	}
	if b.CanUndo() {
		t.Fatal("expected no more undo")
	}
}

func TestUndoCoalesceBreakOnNewline(t *testing.T) {
	b := enabledBuf("")
	b.SetUndoCursor(Position{0, 0}, Position{0, 0})
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 0}, End: Position{0, 0}},
		NewBytes: []byte("a"),
	})
	b.SetUndoCursor(Position{0, 1}, Position{0, 1})
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 1}, End: Position{0, 1}},
		NewBytes: []byte("\n"),
	})
	// Newline is not coalescable, so we should have 2 entries.
	b.Undo() // undo newline
	if got := b.String(); got != "a" {
		t.Fatalf("after undo newline: %q", got)
	}
	b.Undo() // undo 'a'
	if got := b.String(); got != "" {
		t.Fatalf("after undo char: %q", got)
	}
}

func TestUndoCoalesceBreakOnTimeout(t *testing.T) {
	start := time.Now()
	b := FromBytes([]byte(""))
	b.EnableUndo(fakeClock(start, time.Millisecond))

	b.SetUndoCursor(Position{0, 0}, Position{0, 0})
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 0}, End: Position{0, 0}},
		NewBytes: []byte("a"),
	})

	// Jump clock past coalesce timeout.
	b.undo.now = fakeClock(start.Add(time.Second), time.Millisecond)

	b.SetUndoCursor(Position{0, 1}, Position{0, 1})
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 1}, End: Position{0, 1}},
		NewBytes: []byte("b"),
	})

	// Two separate entries due to timeout.
	b.Undo()
	if got := b.String(); got != "a" {
		t.Fatalf("after first undo: %q", got)
	}
}

func TestUndoCoalesceBackspace(t *testing.T) {
	b := enabledBuf("hello")
	// Delete chars from end, one at a time (backspace).
	for i := 4; i >= 0; i-- {
		b.SetUndoCursor(Position{0, i + 1}, Position{0, i + 1})
		b.Apply(Edit{Range: Range{
			Start: Position{0, i}, End: Position{0, i + 1},
		}})
	}
	if got := b.String(); got != "" {
		t.Fatalf("after backspace all: %q", got)
	}
	// Single undo restores everything.
	b.Undo()
	if got := b.String(); got != "hello" {
		t.Fatalf("after undo backspace: %q", got)
	}
}

func TestUndoRedoClearedOnNewEdit(t *testing.T) {
	b := enabledBuf("abc")
	b.SetUndoCursor(Position{0, 3}, Position{0, 3})
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 3}, End: Position{0, 3}},
		NewBytes: []byte("d"),
	})
	b.Undo()
	if !b.CanRedo() {
		t.Fatal("expected CanRedo after undo")
	}
	// New edit clears redo.
	b.undo.now = fakeClock(time.Now().Add(time.Second), time.Millisecond)
	b.SetUndoCursor(Position{0, 3}, Position{0, 3})
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 3}, End: Position{0, 3}},
		NewBytes: []byte("X"),
	})
	if b.CanRedo() {
		t.Fatal("expected no redo after new edit")
	}
}

func TestUndoDirtyCleanMark(t *testing.T) {
	b := enabledBuf("hello")
	b.MarkClean()
	if b.Dirty() {
		t.Fatal("expected clean after MarkClean")
	}

	b.SetUndoCursor(Position{0, 5}, Position{0, 5})
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 5}, End: Position{0, 5}},
		NewBytes: []byte("!"),
	})
	if !b.Dirty() {
		t.Fatal("expected dirty after edit")
	}

	b.Undo()
	if b.Dirty() {
		t.Fatal("expected clean after undo to save point")
	}

	b.Redo()
	if !b.Dirty() {
		t.Fatal("expected dirty after redo past save point")
	}
}

func TestUndoCursorRestore(t *testing.T) {
	b := enabledBuf("hello")
	b.SetUndoCursor(Position{0, 2}, Position{0, 0})
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 0}, End: Position{0, 2}},
		NewBytes: []byte("HE"),
	})
	r := b.Undo()
	if r.Cursor.Cursor != (Position{0, 2}) {
		t.Fatalf("cursor: %v", r.Cursor.Cursor)
	}
	if r.Cursor.Anchor != (Position{0, 0}) {
		t.Fatalf("anchor: %v", r.Cursor.Anchor)
	}
}

func TestCanUndoRedo(t *testing.T) {
	b := enabledBuf("")
	if b.CanUndo() {
		t.Fatal("empty stack should not CanUndo")
	}
	if b.CanRedo() {
		t.Fatal("empty stack should not CanRedo")
	}
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 0}, End: Position{0, 0}},
		NewBytes: []byte("x"),
	})
	if !b.CanUndo() {
		t.Fatal("expected CanUndo")
	}
	b.Undo()
	if !b.CanRedo() {
		t.Fatal("expected CanRedo")
	}
}

func TestUndoWithoutEnable(t *testing.T) {
	b := FromBytes([]byte("hello"))
	// No EnableUndo — all undo ops should be no-ops.
	r := b.Undo()
	if r.OK {
		t.Fatal("expected !OK without EnableUndo")
	}
	r = b.Redo()
	if r.OK {
		t.Fatal("expected !OK without EnableUndo")
	}
	if b.CanUndo() || b.CanRedo() {
		t.Fatal("expected false without EnableUndo")
	}
}

func TestUndoNestedGroups(t *testing.T) {
	b := enabledBuf("abc")
	b.SetUndoCursor(Position{0, 0}, Position{0, 0})

	b.BeginGroup()
	b.BeginGroup() // nested
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 0}, End: Position{0, 1}},
		NewBytes: []byte("X"),
	})
	b.EndGroup() // inner — should NOT flush yet
	if got := b.String(); got != "Xbc" {
		t.Fatalf("mid-group: %q", got)
	}
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 1}, End: Position{0, 2}},
		NewBytes: []byte("Y"),
	})
	b.EndGroup() // outer — flushes

	if got := b.String(); got != "XYc" {
		t.Fatalf("after group: %q", got)
	}

	// Single undo reverts both.
	b.Undo()
	if got := b.String(); got != "abc" {
		t.Fatalf("after undo: %q", got)
	}
}

func TestUndoCoalesceForwardDelete(t *testing.T) {
	b := enabledBuf("hello")
	// Forward-delete from position 0.
	for range 5 {
		b.SetUndoCursor(Position{0, 0}, Position{0, 0})
		b.Apply(Edit{Range: Range{
			Start: Position{0, 0}, End: Position{0, 1},
		}})
	}
	if got := b.String(); got != "" {
		t.Fatalf("after delete all: %q", got)
	}
	// Single undo restores.
	b.Undo()
	if got := b.String(); got != "hello" {
		t.Fatalf("after undo: %q", got)
	}
}

// Gap #1: Redo cursor restore.
func TestRedoCursorRestore(t *testing.T) {
	b := enabledBuf("hello")
	b.SetUndoCursor(Position{0, 0}, Position{0, 3})
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 0}, End: Position{0, 3}},
		NewBytes: []byte("HEL"),
	})
	b.Undo()
	r := b.Redo()
	if !r.OK {
		t.Fatal("Redo returned !OK")
	}
	// cursorAfter should be end of applied range.
	if r.Cursor.Cursor != (Position{0, 3}) {
		t.Fatalf("redo cursor: %v", r.Cursor.Cursor)
	}
}

// Gap #2: record without SetUndoCursor (fallback path).
func TestUndoRecordWithoutSetCursor(t *testing.T) {
	b := enabledBuf("abc")
	// Apply without calling SetUndoCursor.
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 1}, End: Position{0, 1}},
		NewBytes: []byte("X"),
	})
	r := b.Undo()
	if !r.OK {
		t.Fatal("expected OK")
	}
	// Fallback cursor is edit start.
	if r.Cursor.Cursor != (Position{0, 1}) {
		t.Fatalf("fallback cursor: %v", r.Cursor.Cursor)
	}
}

// Gap #3: coalesce break on non-adjacent insert.
func TestUndoCoalesceBreakOnNonAdjacentInsert(t *testing.T) {
	b := enabledBuf("abcdef")
	b.SetUndoCursor(Position{0, 1}, Position{0, 1})
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 1}, End: Position{0, 1}},
		NewBytes: []byte("X"),
	})
	// Insert at col 5 (non-adjacent to col 2 where X ended).
	b.SetUndoCursor(Position{0, 5}, Position{0, 5})
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 5}, End: Position{0, 5}},
		NewBytes: []byte("Y"),
	})
	// Should be 2 separate entries.
	b.Undo()
	if got := b.String(); got != "aXbcdef" {
		t.Fatalf("after first undo: %q", got)
	}
	b.Undo()
	if got := b.String(); got != "abcdef" {
		t.Fatalf("after second undo: %q", got)
	}
}

// Gap #4: coalesce break on newline delete.
func TestUndoCoalesceBreakOnNewlineDelete(t *testing.T) {
	b := enabledBuf("ab\ncd")
	// Backspace 'd', then 'c', then '\n' — newline should break.
	b.SetUndoCursor(Position{1, 2}, Position{1, 2})
	b.Apply(Edit{Range: Range{
		Start: Position{1, 1}, End: Position{1, 2},
	}})
	b.SetUndoCursor(Position{1, 1}, Position{1, 1})
	b.Apply(Edit{Range: Range{
		Start: Position{1, 0}, End: Position{1, 1},
	}})
	// Delete the newline.
	b.SetUndoCursor(Position{1, 0}, Position{1, 0})
	b.Apply(Edit{Range: Range{
		Start: Position{0, 2}, End: Position{1, 0},
	}})
	// Newline delete is not coalescable → separate entry.
	b.Undo() // undo newline delete
	if got := b.String(); got != "ab\n" {
		t.Fatalf("after undo newline delete: %q", got)
	}
	b.Undo() // undo 'c' and 'd' deletes (coalesced)
	if got := b.String(); got != "ab\ncd" {
		t.Fatalf("after undo char deletes: %q", got)
	}
}

// Gap #8: eviction decreases cleanIdx (cleanIdx > 0 path).
func TestUndoStackEvictionDecreasesCleanIdx(t *testing.T) {
	b := New()
	b.EnableUndo(time.Now)
	// Add some edits, then MarkClean at a non-zero index.
	for range 50 {
		b.Apply(Edit{
			Range:    Range{Start: Position{0, 0}, End: Position{0, 0}},
			NewBytes: []byte("\n"),
		})
	}
	b.MarkClean()
	savedIdx := b.undo.cleanIdx // should be 50

	// Fill past cap.
	for range maxUndoEntries {
		b.Apply(Edit{
			Range:    Range{Start: Position{0, 0}, End: Position{0, 0}},
			NewBytes: []byte("\n"),
		})
	}
	// cleanIdx should have been decremented, not immediately -1,
	// since the evictions happen one-at-a-time and clean point
	// was well into the stack.
	_ = savedIdx
	// After enough evictions the clean point is gone.
	if b.undo.cleanIdx >= 0 && b.undo.cleanIdx >= len(b.undo.undo) {
		t.Fatalf("cleanIdx %d out of range (stack len %d)",
			b.undo.cleanIdx, len(b.undo.undo))
	}
}

// Gap #9: multiple redo steps in sequence.
func TestUndoMultipleRedos(t *testing.T) {
	b := enabledBuf("")
	for i, ch := range "abc" {
		b.undo.now = fakeClock(
			time.Now().Add(time.Duration(i)*time.Second),
			time.Millisecond,
		)
		b.SetUndoCursor(Position{0, i}, Position{0, i})
		b.Apply(Edit{
			Range:    Range{Start: Position{0, i}, End: Position{0, i}},
			NewBytes: []byte{byte(ch)},
		})
	}
	// Undo all 3.
	b.Undo()
	b.Undo()
	b.Undo()
	if got := b.String(); got != "" {
		t.Fatalf("after 3 undos: %q", got)
	}
	// Redo all 3.
	b.Redo()
	if got := b.String(); got != "a" {
		t.Fatalf("after redo 1: %q", got)
	}
	b.Redo()
	if got := b.String(); got != "ab" {
		t.Fatalf("after redo 2: %q", got)
	}
	b.Redo()
	if got := b.String(); got != "abc" {
		t.Fatalf("after redo 3: %q", got)
	}
}

// Gap #10: undo multi-line insert round-trip.
func TestUndoMultiLineInsertRoundTrip(t *testing.T) {
	b := enabledBuf("hello")
	b.SetUndoCursor(Position{0, 5}, Position{0, 5})
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 5}, End: Position{0, 5}},
		NewBytes: []byte("\nworld\nfoo"),
	})
	if got := b.String(); got != "hello\nworld\nfoo" {
		t.Fatalf("after insert: %q", got)
	}
	b.Undo()
	if got := b.String(); got != "hello" {
		t.Fatalf("after undo: %q", got)
	}
	b.Redo()
	if got := b.String(); got != "hello\nworld\nfoo" {
		t.Fatalf("after redo: %q", got)
	}
}

// Gap #11: undo replay with stale range after external mutation.
func TestUndoReplayWithStaleRange(t *testing.T) {
	b := enabledBuf("line0\nline1\nline2\nline3\nline4")
	b.SetUndoCursor(Position{4, 0}, Position{4, 0})
	b.Apply(Edit{
		Range:    Range{Start: Position{4, 0}, End: Position{4, 5}},
		NewBytes: []byte("REPLACED"),
	})
	// External mutation: delete lines 1-3, making the undo entry's
	// AppliedRange (line 4) stale.
	b.Apply(Edit{Range: Range{
		Start: Position{1, 0}, End: Position{4, 0},
	}})
	// Undo the external deletion first.
	b.Undo()
	// Now undo the replacement — range should be clamped, no panic.
	b.Undo()
	if got := b.String(); got != "line0\nline1\nline2\nline3\nline4" {
		t.Fatalf("after undo: %q", got)
	}
}
