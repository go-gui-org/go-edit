package edit

import "testing"

func TestWordBoundsAtByte_ASCII(t *testing.T) {
	line := []byte("hello world")
	tests := []struct {
		col       int
		wantStart int
		wantEnd   int
	}{
		{0, 0, 5},   // 'h' → "hello"
		{3, 0, 5},   // 'l' → "hello"
		{5, 5, 6},   // ' ' → space
		{6, 6, 11},  // 'w' → "world"
		{10, 6, 11}, // 'd' → "world"
	}
	for _, tt := range tests {
		s, e := wordBoundsAtByte(line, tt.col)
		if s != tt.wantStart || e != tt.wantEnd {
			t.Errorf("col=%d got [%d,%d) want [%d,%d)",
				tt.col, s, e, tt.wantStart, tt.wantEnd)
		}
	}
}

func TestWordBoundsAtByte_Punctuation(t *testing.T) {
	line := []byte("a+=b")
	s, e := wordBoundsAtByte(line, 1) // '+'
	if s != 1 || e != 3 {
		t.Errorf("got [%d,%d) want [1,3)", s, e)
	}
}

func TestWordBoundsAtByte_Empty(t *testing.T) {
	s, e := wordBoundsAtByte(nil, 0)
	if s != 0 || e != 0 {
		t.Errorf("got [%d,%d) want [0,0)", s, e)
	}
}

func TestWordBoundsAtByte_EndOfLine(t *testing.T) {
	line := []byte("hello")
	s, e := wordBoundsAtByte(line, 5) // past end
	if s != 0 || e != 5 {
		t.Errorf("got [%d,%d) want [0,5)", s, e)
	}
}

func TestWordBoundsAtByte_Underscore(t *testing.T) {
	line := []byte("foo_bar baz")
	s, e := wordBoundsAtByte(line, 3) // '_'
	if s != 0 || e != 7 {
		t.Errorf("got [%d,%d) want [0,7)", s, e)
	}
}

func TestClassifyRune(t *testing.T) {
	if classifyRune('a') != classWord {
		t.Error("'a' should be word")
	}
	if classifyRune('0') != classWord {
		t.Error("'0' should be word")
	}
	if classifyRune('_') != classWord {
		t.Error("'_' should be word")
	}
	if classifyRune(' ') != classSpace {
		t.Error("' ' should be space")
	}
	if classifyRune('+') != classPunct {
		t.Error("'+' should be punct")
	}
}
