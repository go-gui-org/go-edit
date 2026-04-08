package buffer

import (
	"errors"
	"strings"
	"testing"
)

// ---------- Load ----------

type errReader struct{}

func (errReader) Read(p []byte) (int, error) { return 0, errors.New("boom") }

func TestLoadReaderError(t *testing.T) {
	if _, err := Load(errReader{}); err == nil {
		t.Fatal("want error from failing reader")
	}
}

func TestLoadReaderRoundTrip(t *testing.T) {
	b, err := Load(strings.NewReader("a\nb\nc"))
	if err != nil {
		t.Fatal(err)
	}
	if got := b.String(); got != "a\nb\nc" {
		t.Fatalf("got %q", got)
	}
	if b.LineCount() != 3 {
		t.Fatalf("LineCount=%d", b.LineCount())
	}
}

// ---------- Len / metadata ----------

func TestLenMultiLine(t *testing.T) {
	// "ab\ncd\ne" = 2+1+2+1+1 = 7 bytes.
	b := FromBytes([]byte("ab\ncd\ne"))
	if got := b.Len(); got != 7 {
		t.Fatalf("Len=%d want 7", got)
	}
}

func TestFromBytesSingleNewline(t *testing.T) {
	b := FromBytes([]byte("\n"))
	if b.LineCount() != 2 {
		t.Fatalf("LineCount=%d want 2", b.LineCount())
	}
	if len(b.Line(0)) != 0 || len(b.Line(1)) != 0 {
		t.Fatalf("lines not empty: %q %q", b.Line(0), b.Line(1))
	}
}

// ---------- Position / Range ----------

func TestPositionBefore(t *testing.T) {
	cases := []struct {
		a, b Position
		want bool
	}{
		{pos(0, 0), pos(0, 1), true},
		{pos(0, 1), pos(0, 0), false},
		{pos(0, 5), pos(1, 0), true},
		{pos(1, 0), pos(0, 5), false},
		{pos(2, 3), pos(2, 3), false},
	}
	for _, c := range cases {
		if got := c.a.Before(c.b); got != c.want {
			t.Errorf("%+v.Before(%+v)=%v want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestRangeEmpty(t *testing.T) {
	if !(Range{}).Empty() {
		t.Error("zero range not empty")
	}
	if !(Range{Start: pos(3, 4), End: pos(3, 4)}).Empty() {
		t.Error("equal-endpoint range not empty")
	}
	if (Range{Start: pos(0, 0), End: pos(0, 1)}).Empty() {
		t.Error("non-zero range reported empty")
	}
}

// ---------- Apply edge cases ----------

func TestApplyOnEmptyBuffer(t *testing.T) {
	b := New()
	b.Apply(Edit{NewBytes: []byte("hi\nthere")})
	if got := b.String(); got != "hi\nthere" {
		t.Fatalf("got %q", got)
	}
	if b.LineCount() != 2 {
		t.Fatalf("LineCount=%d", b.LineCount())
	}
}

func TestApplyInsertLeadingNewline(t *testing.T) {
	b := FromBytes([]byte("foo"))
	// Insert "\nbar" at col 1 of "foo" → "f", "barbaroo"? No:
	// line0="f"+"" = "f"; segs=["", "bar"]; last="bar"+"oo" → lines
	// ["f", "baroo"]. Verify.
	b.Apply(Edit{Range: rangeOf(0, 1, 0, 1), NewBytes: []byte("\nbar")})
	if got := b.String(); got != "f\nbaroo" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyDeleteAll(t *testing.T) {
	b := FromBytes([]byte("a\nb\nc"))
	last := b.LineCount() - 1
	end := pos(last, len(b.Line(last)))
	b.Apply(Edit{Range: Range{Start: pos(0, 0), End: end}})
	if got := b.String(); got != "" {
		t.Fatalf("got %q want empty", got)
	}
	if b.LineCount() != 1 {
		t.Fatalf("LineCount=%d want 1", b.LineCount())
	}
}

func TestApplyNoop(t *testing.T) {
	b := FromBytes([]byte("hello"))
	c := b.Apply(Edit{Range: rangeOf(0, 2, 0, 2)})
	if got := b.String(); got != "hello" {
		t.Fatalf("content changed: %q", got)
	}
	if len(c.OldBytes) != 0 {
		t.Errorf("OldBytes=%q", c.OldBytes)
	}
	if c.AppliedRange.Start != pos(0, 2) || c.AppliedRange.End != pos(0, 2) {
		t.Errorf("AppliedRange=%+v", c.AppliedRange)
	}
}

func TestApplyReversedRange(t *testing.T) {
	b := FromBytes([]byte("hello"))
	// End before Start; clampRange should swap.
	b.Apply(Edit{
		Range:    Range{Start: pos(0, 4), End: pos(0, 1)},
		NewBytes: []byte("X"),
	})
	if got := b.String(); got != "hXo" {
		t.Fatalf("got %q", got)
	}
}

func TestChangeRecordDelete(t *testing.T) {
	b := FromBytes([]byte("hello\nworld"))
	c := b.Apply(Edit{Range: rangeOf(0, 2, 1, 3)})
	if string(c.OldBytes) != "llo\nwor" {
		t.Errorf("OldBytes=%q", c.OldBytes)
	}
	if c.AppliedRange.Start != pos(0, 2) || c.AppliedRange.End != pos(0, 2) {
		t.Errorf("AppliedRange=%+v want collapsed at start", c.AppliedRange)
	}
	if c.Applied.NewBytes != nil {
		t.Errorf("Applied.NewBytes=%q want nil", c.Applied.NewBytes)
	}
}
