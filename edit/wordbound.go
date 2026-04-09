package edit

import (
	"unicode"
	"unicode/utf8"
)

type charClass int

const (
	classWord  charClass = iota // letter, digit, underscore
	classPunct                  // non-whitespace, non-word
	classSpace                  // whitespace
)

func classifyRune(r rune) charClass {
	switch {
	case r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
		return classWord
	case unicode.IsSpace(r):
		return classSpace
	default:
		return classPunct
	}
}

// classAt decodes the rune at byte offset col in line and returns
// its class and byte width. If col is out of range, returns
// classSpace, 0.
func classAt(line []byte, col int) (charClass, int) {
	if col < 0 || col >= len(line) {
		return classSpace, 0
	}
	r, size := utf8.DecodeRune(line[col:])
	if r == utf8.RuneError && size <= 1 {
		return classPunct, 1
	}
	return classifyRune(r), size
}

// wordBoundsAtByte returns the [start, end) byte range of the word
// at byteCol within lineBytes. Used for double-click word selection.
// If byteCol is at the end of line, the preceding word is selected.
func wordBoundsAtByte(lineBytes []byte, byteCol int) (int, int) {
	if len(lineBytes) == 0 {
		return 0, 0
	}
	// If at end of line, back up one rune.
	if byteCol >= len(lineBytes) {
		byteCol = len(lineBytes)
		_, size := utf8.DecodeLastRune(lineBytes[:byteCol])
		byteCol -= size
	}
	if byteCol < 0 {
		byteCol = 0
	}

	anchor, _ := classAt(lineBytes, byteCol)

	// Scan left.
	start := byteCol
	for start > 0 {
		r, size := utf8.DecodeLastRune(lineBytes[:start])
		if r == utf8.RuneError && size <= 1 {
			break
		}
		if classifyRune(r) != anchor {
			break
		}
		start -= size
	}

	// Scan right.
	end := byteCol
	for end < len(lineBytes) {
		cls, size := classAt(lineBytes, end)
		if cls != anchor {
			break
		}
		end += size
	}

	return start, end
}
