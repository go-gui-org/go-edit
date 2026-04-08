package edit

import (
	"math"
	"testing"

	"github.com/mike-ward/go-edit/edit/buffer"
	"github.com/mike-ward/go-edit/edit/internal/fakewin"
	"github.com/mike-ward/go-gui/gui"
)

// ---------- Editor factory ----------

func TestEditor_NilBufferSubstitutesEmpty(t *testing.T) {
	v := Editor(EditorCfg{
		IDFocus: 100, Buffer: nil, Width: 400, Height: 200,
	})
	if v == nil {
		t.Fatal("Editor returned nil")
	}
	// Drive a frame to confirm no panic reaches AmendLayout.
	d := &driver{
		cfg: EditorCfg{
			IDFocus: 100, Buffer: buffer.New(),
			Width: 400, Height: 200,
		},
		frame: &editorFrameData{},
		w:     fakewin.New(),
		ly:    &gui.Layout{},
	}
	d.amend = editorAmendLayout(d.cfg, d.frame)
	d.tick()
}

func TestEditor_NaNDimensions(t *testing.T) {
	nan := float32(math.NaN())
	v := Editor(EditorCfg{
		IDFocus: 101,
		Buffer:  mkBuf("hello"),
		Width:   nan,
		Height:  nan,
	})
	if v == nil {
		t.Fatal("Editor returned nil")
	}
}

// ---------- sanitizeDim ----------

func TestSanitizeDim_ClampsEdgeValues(t *testing.T) {
	nan := float32(math.NaN())
	pinf := float32(math.Inf(+1))
	ninf := float32(math.Inf(-1))
	cases := []struct {
		in, want float32
	}{
		{nan, minDimension},
		{pinf, maxDimension},
		{ninf, minDimension},
		{-100, minDimension},
		{0, minDimension},
		{0.5, minDimension},
		{1, 1},
		{100, 100},
		{maxDimension, maxDimension},
		{maxDimension + 1, maxDimension},
		{1e20, maxDimension},
	}
	for _, c := range cases {
		if got := sanitizeDim(c.in); got != c.want {
			t.Errorf("sanitizeDim(%v)=%v want %v", c.in, got, c.want)
		}
	}
}

// ---------- clampScroll edge cases ----------

func TestClampScroll_NaNIn(t *testing.T) {
	cfg := EditorCfg{Buffer: mkBuf("a\nb\nc"), Height: 10}
	st := editorState{ScrollY: float32(math.NaN())}
	clampScroll(&st, cfg, 10)
	if st.ScrollY != 0 || st.ScrollY != st.ScrollY {
		t.Errorf("ScrollY=%v want 0", st.ScrollY)
	}
}

func TestClampScroll_ZeroLineHeight(t *testing.T) {
	cfg := EditorCfg{Buffer: mkBuf("a\nb"), Height: 10}
	st := editorState{ScrollY: 500}
	clampScroll(&st, cfg, 0)
	if st.ScrollY != 0 {
		t.Errorf("ScrollY=%v want 0", st.ScrollY)
	}
}

// ---------- ensureCursorVisible edge cases ----------

func TestEnsureCursorVisible_NaNViewport(t *testing.T) {
	st := editorState{ScrollY: 42}
	fr := &editorFrameData{lineHeight: 10, valid: true}
	ensureCursorVisible(&st, fr, float32(math.NaN()))
	if st.ScrollY != 42 {
		t.Errorf("ScrollY=%v want 42 (unchanged)", st.ScrollY)
	}
}

func TestEnsureCursorVisible_ZeroViewport(t *testing.T) {
	st := editorState{ScrollY: 42}
	fr := &editorFrameData{lineHeight: 10, valid: true}
	ensureCursorVisible(&st, fr, 0)
	if st.ScrollY != 42 {
		t.Errorf("ScrollY=%v want 42 (unchanged)", st.ScrollY)
	}
}

// ---------- driver: mouse scroll NaN/absurd ----------

func TestDriver_MouseScrollNaNDropped(t *testing.T) {
	buf := mkBuf("a\nb\nc\nd")
	d := newDriver(EditorCfg{
		IDFocus: 200, Buffer: buf, Width: 400, Height: 200,
	})
	d.tick()
	before := d.state().ScrollY
	d.wheel(d.ly, fakewin.NewScrollEvent(float32(math.NaN())), d.w)
	if d.state().ScrollY != before {
		t.Errorf("ScrollY changed on NaN event")
	}
}

func TestDriver_MouseScrollAbsurdDropped(t *testing.T) {
	buf := mkBuf("a\nb\nc\nd")
	d := newDriver(EditorCfg{
		IDFocus: 201, Buffer: buf, Width: 400, Height: 200,
	})
	d.tick()
	before := d.state().ScrollY
	d.wheel(d.ly, fakewin.NewScrollEvent(1e9), d.w)
	if d.state().ScrollY != before {
		t.Errorf("ScrollY changed on absurd event: %v→%v",
			before, d.state().ScrollY)
	}
}
