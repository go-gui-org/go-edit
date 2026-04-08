package buffer

import "testing"

// FuzzBufferApply feeds arbitrary byte sequences as both initial
// content and edit payload. Asserts: Apply never panics, the resulting
// buffer's String() round-trips through FromBytes, and per-line byte
// sum + newline count match String().
func FuzzBufferApply(f *testing.F) {
	f.Add([]byte("hello\nworld"), []byte("X"), 0, 2, 0, 5)
	f.Add([]byte(""), []byte("\n\n"), 0, 0, 0, 0)
	f.Add([]byte("\x00\x01\xff"), []byte{}, 0, 0, 0, 3)
	f.Add([]byte("a\nb\nc"), []byte("multi\nline\ninsert"), 0, 1, 2, 0)

	f.Fuzz(func(t *testing.T,
		initial, payload []byte,
		sl, sc, el, ec int,
	) {
		b := FromBytes(initial)
		b.Apply(Edit{
			Range:    Range{Start: Position{sl, sc}, End: Position{el, ec}},
			NewBytes: payload,
		})
		s := b.String()
		b2 := FromBytes([]byte(s))
		if b2.String() != s {
			t.Fatalf("round-trip failed: %q -> %q", s, b2.String())
		}
		checkInvariants(t, b, 0)
	})
}
