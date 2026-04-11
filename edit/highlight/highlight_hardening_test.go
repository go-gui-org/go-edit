package highlight

import (
	"testing"

	"github.com/mike-ward/go-edit/edit/buffer"
)

func TestDecorate_NegativeViewport(t *testing.T) {
	buf := buffer.FromBytes([]byte("hello"))
	buf.Props.FilePath = "test.go"
	h := New(buf, "", nil)
	if h == nil {
		t.Skip("no Go lexer")
	}
	defer h.Close()

	decos := h.Decorate(buffer.Viewport{FirstLine: -5, LastLine: -1}, nil)
	if len(decos) != 0 {
		t.Fatalf("expected no decos, got %d", len(decos))
	}
}

func TestDecorate_InvertedViewport(t *testing.T) {
	buf := buffer.FromBytes([]byte("hello"))
	buf.Props.FilePath = "test.go"
	h := New(buf, "", nil)
	if h == nil {
		t.Skip("no Go lexer")
	}
	defer h.Close()

	decos := h.Decorate(buffer.Viewport{FirstLine: 5, LastLine: 0}, nil)
	if len(decos) != 0 {
		t.Fatalf("expected no decos, got %d", len(decos))
	}
}

func TestClose_StopsObserver(t *testing.T) {
	buf := buffer.FromBytes([]byte("var x = 1"))
	buf.Props.FilePath = "test.go"
	h := New(buf, "", nil)
	if h == nil {
		t.Skip("no Go lexer")
	}

	// Tokenize once.
	h.Decorate(buffer.Viewport{FirstLine: 0, LastLine: 0}, nil)

	h.Close()

	// Edit after close — should not panic or affect highlighter.
	buf.Apply(buffer.Edit{
		Range: buffer.Range{
			Start: buffer.Position{Line: 0, ByteCol: 8},
			End:   buffer.Position{Line: 0, ByteCol: 9},
		},
		NewBytes: []byte("2"),
	})

	// Decorate after close+edit — should still return cached
	// tokens (observer removed, so valid flag unchanged).
	decos := h.Decorate(buffer.Viewport{FirstLine: 0, LastLine: 0}, nil)
	_ = decos // no panic = pass
}

func TestClose_Double(t *testing.T) {
	buf := buffer.FromBytes([]byte("hello"))
	buf.Props.FilePath = "test.go"
	h := New(buf, "", nil)
	if h == nil {
		t.Skip("no Go lexer")
	}
	h.Close()
	h.Close() // double close — should not panic
}

// TestDecorate_ZeroAllocOnCachedValid confirms that a second
// Decorate call into a pre-sized out slice does not allocate
// once tokenization has run and the token cache is valid.
func TestDecorate_ZeroAllocOnCachedValid(t *testing.T) {
	buf := buffer.FromBytes([]byte("package main\nfunc f() {}"))
	buf.Props.FilePath = "test.go"
	h := New(buf, "", nil)
	if h == nil {
		t.Skip("no Go lexer")
	}
	defer h.Close()

	vp := buffer.Viewport{FirstLine: 0, LastLine: 1}
	// Prime: tokenize + size the scratch buffer.
	scratch := h.Decorate(vp, nil)
	if len(scratch) == 0 {
		t.Fatal("expected non-empty decorations on priming call")
	}
	// The steady-state call must not allocate.
	n := testing.AllocsPerRun(50, func() {
		out := h.Decorate(vp, scratch[:0])
		_ = out
	})
	if n != 0 {
		t.Errorf("Decorate allocated %v times on cached valid call, want 0", n)
	}
}
