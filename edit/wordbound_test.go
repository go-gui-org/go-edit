package edit

// Word selection delegates to go-glyph's WordBoundsInString (the same
// segmenter go-gui and go-shirei use), so go-edit must agree with them
// by construction. These tests pin the two regressions the local copy
// had: combining marks detaching from their base, and all CJK scripts
// collapsing into one word. They also mirror go-glyph's own cases at
// the boundary go-edit actually calls.

import (
	"testing"

	"github.com/go-gui-org/go-glyph"
)

func TestWordBoundsCJK_SplitsByScript(t *testing.T) {
	// Han, Hiragana, and Katakana are separate classes, so a script
	// transition ends the word.
	line := "日本語のテキスト"
	start, end := glyph.WordBoundsInString(line, 0)
	if start != 0 || end != 9 {
		t.Errorf("col=0 got [%d,%d) want [0,9)", start, end)
	}
	// 'の' (Hiragana) is a word of its own.
	start, end = glyph.WordBoundsInString(line, 9)
	if start != 9 || end != 12 {
		t.Errorf("col=9 got [%d,%d) want [9,12)", start, end)
	}
}

func TestWordBoundsCombiningMark_StaysAttached(t *testing.T) {
	// U+0301 is unicode.IsMark, folded into classWord, so double-click
	// on 'e' selects "café" whole. The old 3-class segmenter classified
	// the mark as punct and left it behind as its own "word".
	line := "cafe\u0301" // 'e' + U+0301 combining acute
	start, end := glyph.WordBoundsInString(line, 2)
	if start != 0 || end != 6 {
		t.Errorf("got [%d,%d) want [0,6)", start, end)
	}
}

func TestWordBoundsASCII(t *testing.T) {
	tests := []struct {
		col       int
		wantStart int
		wantEnd   int
	}{
		{0, 0, 5},   // 'h' → "hello"
		{3, 0, 5},   // 'l' → "hello"
		{5, 5, 6},   // ' ' → space run
		{6, 6, 11},  // 'w' → "world"
		{10, 6, 11}, // 'd' → "world"
	}
	for _, tt := range tests {
		s, e := glyph.WordBoundsInString("hello world", tt.col)
		if s != tt.wantStart || e != tt.wantEnd {
			t.Errorf("col=%d got [%d,%d) want [%d,%d)",
				tt.col, s, e, tt.wantStart, tt.wantEnd)
		}
	}
}

func TestWordBoundsPunctuation(t *testing.T) {
	s, e := glyph.WordBoundsInString("a+=b", 1) // '+' is its own word
	if s != 1 || e != 3 {
		t.Errorf("got [%d,%d) want [1,3)", s, e)
	}
}

func TestWordBoundsEndOfLine(t *testing.T) {
	// Index one past the end resolves to the last run.
	s, e := glyph.WordBoundsInString("hello", 5)
	if s != 0 || e != 5 {
		t.Errorf("got [%d,%d) want [0,5)", s, e)
	}
}

func TestWordBoundsUnderscore(t *testing.T) {
	s, e := glyph.WordBoundsInString("foo_bar baz", 3) // '_' stays in word
	if s != 0 || e != 7 {
		t.Errorf("got [%d,%d) want [0,7)", s, e)
	}
}

func TestWordBoundsWhitespaceOnlyLine(t *testing.T) {
	// Whitespace-only text has no words: empty range at the query
	// index, which makes double-click a no-op instead of selecting
	// the whitespace run.
	s, e := glyph.WordBoundsInString("   ", 1)
	if s != 1 || e != 1 {
		t.Errorf("got [%d,%d) want [1,1)", s, e)
	}
}
