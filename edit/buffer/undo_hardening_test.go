package buffer

import (
	"testing"
	"time"
)

func TestEnableUndoNilClock(t *testing.T) {
	b := FromBytes([]byte("test"))
	b.EnableUndo(nil) // must not panic; defaults to time.Now
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 0}, End: Position{0, 0}},
		NewBytes: []byte("x"),
	})
	if !b.CanUndo() {
		t.Fatal("expected CanUndo with nil clock")
	}
}

func TestEnableUndoIdempotent(t *testing.T) {
	b := FromBytes([]byte("test"))
	b.EnableUndo(nil)
	b.Apply(Edit{
		Range:    Range{Start: Position{0, 0}, End: Position{0, 0}},
		NewBytes: []byte("x"),
	})
	b.EnableUndo(nil) // second call is no-op
	if !b.CanUndo() {
		t.Fatal("second EnableUndo should not reset stack")
	}
}

func TestUndoRedoWithoutEnable(t *testing.T) {
	b := FromBytes([]byte("test"))
	// Must not panic.
	r := b.Undo()
	if r.OK {
		t.Fatal("expected !OK")
	}
	r = b.Redo()
	if r.OK {
		t.Fatal("expected !OK")
	}
}

func TestBeginEndGroupWithoutEnable(t *testing.T) {
	b := FromBytes([]byte("test"))
	// Must not panic.
	b.BeginGroup()
	b.EndGroup()
}

func TestSetUndoCursorWithoutEnable(t *testing.T) {
	b := FromBytes([]byte("test"))
	// Must not panic.
	b.SetUndoCursor(Position{0, 0}, Position{0, 0})
}

func TestEndGroupWithoutBegin(t *testing.T) {
	b := FromBytes([]byte("test"))
	b.EnableUndo(nil)
	// Must not panic; unmatched EndGroup is no-op.
	b.EndGroup()
}

func TestCanUndoRedoWithoutEnable(t *testing.T) {
	b := FromBytes([]byte("test"))
	if b.CanUndo() || b.CanRedo() {
		t.Fatal("expected false without EnableUndo")
	}
}

func TestUndoStackEvictsOldest(t *testing.T) {
	b := New()
	b.EnableUndo(time.Now)
	// Each edit gets its own entry (newlines break coalescing).
	for range maxUndoEntries + 100 {
		b.Apply(Edit{
			Range:    Range{Start: Position{0, 0}, End: Position{0, 0}},
			NewBytes: []byte("\n"),
		})
	}
	if len(b.undo.undo) > maxUndoEntries {
		t.Fatalf("undo stack %d > cap %d",
			len(b.undo.undo), maxUndoEntries)
	}
}

func TestUndoStackEvictionInvalidatesCleanIdx(t *testing.T) {
	b := New()
	b.EnableUndo(time.Now)
	b.MarkClean()
	// Fill past cap — clean point is evicted.
	for range maxUndoEntries + 10 {
		b.Apply(Edit{
			Range:    Range{Start: Position{0, 0}, End: Position{0, 0}},
			NewBytes: []byte("\n"),
		})
	}
	// Undo all — should not accidentally mark clean.
	for b.CanUndo() {
		b.Undo()
	}
	if !b.Dirty() {
		t.Fatal("expected dirty after clean point evicted")
	}
}

func TestCoalesceCapBreaksChain(t *testing.T) {
	b := New()
	b.EnableUndo(time.Now)
	// Type maxCoalesceLen+100 chars; chain must break.
	for i := range maxCoalesceLen + 100 {
		p := Position{0, i}
		b.Apply(Edit{
			Range:    Range{Start: p, End: p},
			NewBytes: []byte{'a'},
		})
	}
	// Must have more than 1 undo entry.
	if len(b.undo.undo) < 2 {
		t.Fatalf("expected coalesce break, got %d entries",
			len(b.undo.undo))
	}
	// No single entry should exceed the cap.
	for i, e := range b.undo.undo {
		if len(e.changes) > maxCoalesceLen {
			t.Fatalf("entry %d has %d changes > cap %d",
				i, len(e.changes), maxCoalesceLen)
		}
	}
}

func TestBeginGroupNestingCap(t *testing.T) {
	b := New()
	b.EnableUndo(time.Now)
	for range maxGroupNesting + 10 {
		b.BeginGroup()
	}
	// Grouping should be capped.
	if b.undo.grouping > maxGroupNesting {
		t.Fatalf("grouping %d > cap %d",
			b.undo.grouping, maxGroupNesting)
	}
}
