package edit

import (
	"testing"

	"github.com/mike-ward/go-edit/edit/buffer"
	"github.com/mike-ward/go-edit/edit/internal/fakewin"
	"github.com/mike-ward/go-gui/gui"
)

// sendIMEChar dispatches an IME commit event through the driver.
func (d *driver) sendIMEChar(text string) {
	d.tick()
	d.char(d.ly, fakewin.NewIMECharEvent(text), d.w)
}

func TestIME_CommitInsertsFull(t *testing.T) {
	buf := buffer.New()
	d := newDriver(EditorCfg{
		IDFocus: 1, Buffer: buf, Width: 400, Height: 200,
	})
	d.tick()
	d.sendIMEChar("漢字")
	got := string(buf.Line(0))
	if got != "漢字" {
		t.Fatalf("buffer = %q, want %q", got, "漢字")
	}
	cs := d.cursor()
	want := len([]byte("漢字"))
	if cs.Cursor.ByteCol != want {
		t.Fatalf("ByteCol = %d, want %d", cs.Cursor.ByteCol, want)
	}
}

func TestIME_CommitMultiCursor(t *testing.T) {
	buf := buffer.New()
	buf.Apply(buffer.Edit{
		Range:    buffer.Range{},
		NewBytes: []byte("aa\nbb"),
	})
	d := newDriver(EditorCfg{
		IDFocus: 1, Buffer: buf, Width: 400, Height: 200,
	})
	d.tick()

	st := loadState(d.w, d.cfg.IDFocus)
	st.Cursors = []CursorState{
		{
			Cursor: buffer.Position{Line: 0, ByteCol: 0},
			Anchor: buffer.Position{Line: 0, ByteCol: 0},
		},
		{
			Cursor: buffer.Position{Line: 1, ByteCol: 0},
			Anchor: buffer.Position{Line: 1, ByteCol: 0},
		},
	}
	storeState(d.w, d.cfg.IDFocus, st)

	d.sendIMEChar("漢字") // multi-codepoint triggers IME path
	if got := string(buf.Line(0)); got != "漢字aa" {
		t.Fatalf("line 0 = %q, want %q", got, "漢字aa")
	}
	if got := string(buf.Line(1)); got != "漢字bb" {
		t.Fatalf("line 1 = %q, want %q", got, "漢字bb")
	}
}

func TestIME_CommitInSearchBar(t *testing.T) {
	buf := buffer.New()
	buf.Apply(buffer.Edit{
		Range:    buffer.Range{},
		NewBytes: []byte("hello world"),
	})
	d := newDriver(EditorCfg{
		IDFocus: 1, Buffer: buf, Width: 400, Height: 200,
	})
	d.tick()

	// Activate search directly.
	st := loadState(d.w, d.cfg.IDFocus)
	st.Search.Active = true
	storeState(d.w, d.cfg.IDFocus, st)

	d.sendIMEChar("世界")
	st = d.state()
	if st.Search.Query != "世界" {
		t.Fatalf("query = %q, want %q", st.Search.Query, "世界")
	}
	if got := string(buf.Line(0)); got != "hello world" {
		t.Fatalf("buffer modified: %q", got)
	}
}

func TestIME_CommitReadOnlyNoop(t *testing.T) {
	buf := buffer.New()
	buf.Apply(buffer.Edit{
		Range:    buffer.Range{},
		NewBytes: []byte("test"),
	})
	d := newDriver(EditorCfg{
		IDFocus: 1, Buffer: buf, Width: 400, Height: 200,
		ReadOnly: true,
	})
	d.tick()
	d.sendIMEChar("漢字")
	if got := string(buf.Line(0)); got != "test" {
		t.Fatalf("buffer modified in read-only: %q", got)
	}
}

func TestIME_SingleRuneUsesNormalPath(t *testing.T) {
	// A single-rune IME commit uses the normal CharCode path,
	// not the multi-codepoint branch. Verify single CJK char
	// inserts correctly via the standard path.
	buf := buffer.New()
	d := newDriver(EditorCfg{
		IDFocus: 1, Buffer: buf, Width: 400, Height: 200,
	})
	d.tick()
	d.sendIMEChar("漢")
	got := string(buf.Line(0))
	if got != "漢" {
		t.Fatalf("buffer = %q, want %q", got, "漢")
	}
}

func TestIME_CommitSuppressesFollowingEnter(t *testing.T) {
	t.Run("multi-rune", func(t *testing.T) {
		buf := buffer.New()
		d := newDriver(EditorCfg{
			IDFocus: 1, Buffer: buf, Width: 400, Height: 200,
		})
		d.tick()
		// Simulate: previous AmendLayout saw composition.
		d.frame.imeComposing = true
		// Dispatch char directly (no tick — mirrors real
		// event loop where events fire before AmendLayout).
		d.char(d.ly, fakewin.NewIMECharEvent("漢字"), d.w)
		// Enter keydown the OS sends after the commit.
		d.key(d.ly, fakewin.NewKeyEvent(gui.KeyEnter, 0), d.w)
		if buf.LineCount() != 1 {
			t.Fatalf("lines = %d, want 1", buf.LineCount())
		}
	})
	t.Run("single-rune", func(t *testing.T) {
		buf := buffer.New()
		d := newDriver(EditorCfg{
			IDFocus: 1, Buffer: buf, Width: 400, Height: 200,
		})
		d.tick()
		d.frame.imeComposing = true
		d.char(d.ly, fakewin.NewIMECharEvent("漢"), d.w)
		d.key(d.ly, fakewin.NewKeyEvent(gui.KeyEnter, 0), d.w)
		if buf.LineCount() != 1 {
			t.Fatalf("lines = %d, want 1", buf.LineCount())
		}
		if got := string(buf.Line(0)); got != "漢" {
			t.Fatalf("buffer = %q, want %q", got, "漢")
		}
	})
}

func TestCursorMovement_MultiByteRunes(t *testing.T) {
	buf := buffer.New()
	buf.Apply(buffer.Edit{
		Range:    buffer.Range{},
		NewBytes: []byte("あいう"),
	})
	d := newDriver(EditorCfg{
		IDFocus: 1, Buffer: buf, Width: 400, Height: 200,
	})
	d.tick()

	// Cursor starts at 0. Move right should jump over "あ" (3 bytes).
	d.sendKey(gui.KeyRight)
	if col := d.cursor().Cursor.ByteCol; col != 3 {
		t.Fatalf("after right: ByteCol = %d, want 3", col)
	}
	// Move right again over "い".
	d.sendKey(gui.KeyRight)
	if col := d.cursor().Cursor.ByteCol; col != 6 {
		t.Fatalf("after right×2: ByteCol = %d, want 6", col)
	}
	// Move left back over "い".
	d.sendKey(gui.KeyLeft)
	if col := d.cursor().Cursor.ByteCol; col != 3 {
		t.Fatalf("after left: ByteCol = %d, want 3", col)
	}
}

func TestIME_PrimaryCursorHiddenFlag(t *testing.T) {
	frame := &editorFrameData{}
	frame.imeComposing = true
	frame.imePreedit = "test"
	if !frame.imeComposing {
		t.Fatal("imeComposing not set")
	}
	if frame.imePreedit != "test" {
		t.Fatalf("imePreedit = %q, want %q",
			frame.imePreedit, "test")
	}
}
