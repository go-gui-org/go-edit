package edit

import (
	"testing"

	"github.com/go-gui-org/go-edit/edit/buffer"
	"github.com/go-gui-org/go-edit/edit/internal/fakewin"
	"github.com/go-gui-org/go-gui/gui"
)

// sendIMEChar dispatches an IME commit event through the driver.
func (d *driver) sendIMEChar(text string) {
	d.tick()
	d.char(d.ctx(fakewin.NewIMECharEvent(text)))
}

func TestIME_CommitInsertsFull(t *testing.T) {
	buf := buffer.New()
	d := newDriver(EditorCfg{
		ID: "e1", Buffer: buf, Width: 400, Height: 200,
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
		ID: "e1", Buffer: buf, Width: 400, Height: 200,
	})
	d.tick()

	st := loadState(d.w, d.cfg.ID)
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
	storeState(d.w, d.cfg.ID, st)

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
		ID: "e1", Buffer: buf, Width: 400, Height: 200,
	})
	d.tick()

	// Activate search directly.
	st := loadState(d.w, d.cfg.ID)
	st.Search.Active = true
	storeState(d.w, d.cfg.ID, st)

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
		ID: "e1", Buffer: buf, Width: 400, Height: 200,
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
		ID: "e1", Buffer: buf, Width: 400, Height: 200,
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
			ID: "e1", Buffer: buf, Width: 400, Height: 200,
		})
		d.tick()
		// Simulate: previous AmendLayout saw composition.
		d.frame.imeComposing = true
		// Dispatch char directly (no tick — mirrors real
		// event loop where events fire before AmendLayout).
		d.char(d.ctx(fakewin.NewIMECharEvent("漢字")))
		// Enter keydown the OS sends after the commit.
		d.key(d.ctx(fakewin.NewKeyEvent(gui.KeyEnter, 0)))
		if buf.LineCount() != 1 {
			t.Fatalf("lines = %d, want 1", buf.LineCount())
		}
	})
	t.Run("single-rune", func(t *testing.T) {
		buf := buffer.New()
		d := newDriver(EditorCfg{
			ID: "e1", Buffer: buf, Width: 400, Height: 200,
		})
		d.tick()
		d.frame.imeComposing = true
		d.char(d.ctx(fakewin.NewIMECharEvent("漢")))
		d.key(d.ctx(fakewin.NewKeyEvent(gui.KeyEnter, 0)))
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
		ID: "e1", Buffer: buf, Width: 400, Height: 200,
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

// The root container must opt in to the platform input method, or
// go-gui never starts it for a DrawCanvas editor and CJK input
// cannot be typed (go-gui #999). A read-only editor must stay out:
// an active input method claims keys such as Option+I on macOS.
func TestIME_RootAcceptsTextInput(t *testing.T) {
	for _, tc := range []struct {
		name     string
		readOnly bool
		want     bool
	}{
		{"editable", false, true},
		{"read-only", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := EditorCfg{ID: "e1", ReadOnly: tc.readOnly}
			got := editorRootCfg(cfg, &editorFrameData{}, "", nil)
			if got.AcceptsTextInput != tc.want {
				t.Fatalf("AcceptsTextInput = %v, want %v",
					got.AcceptsTextInput, tc.want)
			}
			if !got.Focusable {
				t.Fatal("root container must stay focusable")
			}
		})
	}
}

// Space in a Japanese input method sends U+3000 IDEOGRAPHIC SPACE,
// not U+0020. unicode.IsPrint accepts only the ASCII space, so the
// editor dropped it. Other Unicode spaces (NBSP, em space) are text
// too and must also insert.
func TestIME_UnicodeSpacesInsert(t *testing.T) {
	for _, r := range []rune{0x3000, 0xA0, 0x2003} {
		buf := buffer.New()
		d := newDriver(EditorCfg{
			ID: "e1", Buffer: buf, Width: 400, Height: 200,
		})
		d.tick()
		d.sendIMEChar(string(r))
		if got := string(buf.Line(0)); got != string(r) {
			t.Errorf("%U: buffer = %q, want %q", r, got, string(r))
		}
	}
}

// drawOverlay runs the cursor overlay's OnDraw against a fresh
// DrawContext that measures with the fake 8 px advance.
func (d *driver) drawOverlay() *gui.DrawContext {
	dc := gui.NewDrawContext(d.cfg.Width, d.cfg.Height,
		d.w.TextMeasurer())
	editorOnDrawCursor(d.cfg, d.frame)(dc)
	return dc
}

// composeAt sets the frame's IME state as AmendLayout would read it
// from the window mid-composition.
func (d *driver) composeAt(preedit string, cursor, selLen int) {
	d.frame.imeComposing = true
	d.frame.imePreedit = preedit
	d.frame.imeCursor = cursor
	d.frame.imeSelLen = selLen
}

// preeditEntry returns the text entry that draws want, if any.
func preeditEntry(dc *gui.DrawContext, want string) (gui.DrawCanvasTextEntry, bool) {
	for _, e := range dc.Texts() {
		if e.Text == want {
			return e, true
		}
	}
	return gui.DrawCanvasTextEntry{}, false
}

// Text being composed must be visible at the caret. Before this the
// editor read the preedit every frame but never drew it, so a
// composition was invisible until commit.
func TestIME_PreeditDrawnAtCaret(t *testing.T) {
	buf := buffer.New()
	buf.Apply(buffer.Edit{NewBytes: []byte("ab\ncd")})
	d := newDriver(EditorCfg{
		ID: "e1", Buffer: buf, Width: 400, Height: 200,
	})
	d.tick()
	st := loadState(d.w, d.cfg.ID)
	st.Cursors = []CursorState{{
		Cursor: buffer.Position{Line: 1, ByteCol: 1},
		Anchor: buffer.Position{Line: 1, ByteCol: 1},
	}}
	storeState(d.w, d.cfg.ID, st)
	d.tick()
	d.composeAt("にほん", 3, 0)

	e, ok := preeditEntry(d.drawOverlay(), "にほん")
	if !ok {
		t.Fatal("preedit not drawn")
	}
	f := d.frame
	wantX := f.gutterW + f.padLeft +
		f.state.Measurer.XForColumn([]byte("cd"), 1)
	wantY := 1 * f.lineHeight
	if e.X != wantX || e.Y != wantY {
		t.Fatalf("preedit at (%v, %v), want (%v, %v)",
			e.X, e.Y, wantX, wantY)
	}
}

// The overlay skips the caret while the blink is off. The preedit
// shares that canvas but must not blink with it.
func TestIME_PreeditDrawnWhileCaretBlinkedOff(t *testing.T) {
	d := newDriver(EditorCfg{
		ID: "e1", Buffer: buffer.New(), Width: 400, Height: 200,
	})
	d.tick()
	d.frame.cursorVisible = false
	d.composeAt("かな", 2, 0)
	if _, ok := preeditEntry(d.drawOverlay(), "かな"); !ok {
		t.Fatal("preedit hidden while caret blinked off")
	}
}

// No composition, no preedit text in the overlay.
func TestIME_NoPreeditWhenNotComposing(t *testing.T) {
	d := newDriver(EditorCfg{
		ID: "e1", Buffer: buffer.New(), Width: 400, Height: 200,
	})
	d.tick()
	if n := len(d.drawOverlay().Texts()); n != 0 {
		t.Fatalf("overlay drew %d text entries, want 0", n)
	}
}
