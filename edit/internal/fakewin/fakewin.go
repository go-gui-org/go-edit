// Package fakewin provides a headless *gui.Window with a
// deterministic TextMeasurer, plus event builders for driving
// editor callbacks in unit tests.
//
// It lives under edit/internal so external projects cannot import
// it; it is a test fixture, not a public API.
package fakewin

import (
	"github.com/mike-ward/go-glyph"
	"github.com/mike-ward/go-gui/gui"
)

// Advance is the fixed monospace advance used by the fake measurer,
// in pixels. LineHeight is the fixed line height.
const (
	Advance    float32 = 8
	LineHeight float32 = 16
)

// New returns a headless window with the fake measurer and an
// in-memory clipboard installed.
func New() *gui.Window {
	w := &gui.Window{}
	w.SetTextMeasurer(&fakeMeasurer{})
	var clip string
	w.SetClipboardFn(func(s string) { clip = s })
	w.SetClipboardGetFn(func() string { return clip })
	return w
}

// NewKeyEvent builds a key-down event with the given code + modifiers.
func NewKeyEvent(code gui.KeyCode, mods gui.Modifier) *gui.Event {
	return &gui.Event{
		Type:      gui.EventKeyDown,
		KeyCode:   code,
		Modifiers: mods,
	}
}

// NewCharEvent builds a character-input event for rune r.
func NewCharEvent(r rune) *gui.Event {
	return &gui.Event{
		Type:     gui.EventChar,
		CharCode: uint32(r),
	}
}

// NewScrollEvent builds a mouse-scroll event with the given vertical
// delta (positive = scroll up).
func NewScrollEvent(deltaY float32) *gui.Event {
	return &gui.Event{
		Type:    gui.EventMouseScroll,
		ScrollY: deltaY,
	}
}

// NewClickEvent builds a mouse-down (click) event at the given
// coordinates with optional modifiers.
func NewClickEvent(x, y float32, mods gui.Modifier) *gui.Event {
	return &gui.Event{
		Type:      gui.EventMouseDown,
		MouseX:    x,
		MouseY:    y,
		Modifiers: mods,
	}
}

// fakeMeasurer is a deterministic monospace measurer. Every character
// is Advance pixels wide; line height is LineHeight. LayoutText
// returns a minimal glyph.Layout — the editor's ASCII fast path
// bypasses LayoutText entirely, so most driver tests never exercise
// the returned Layout.
type fakeMeasurer struct{}

func (fakeMeasurer) TextWidth(text string, _ gui.TextStyle) float32 {
	return float32(len(text)) * Advance
}

func (fakeMeasurer) TextHeight(_ string, _ gui.TextStyle) float32 {
	return LineHeight
}

func (fakeMeasurer) FontHeight(_ gui.TextStyle) float32 { return LineHeight }

func (fakeMeasurer) FontAscent(_ gui.TextStyle) float32 { return LineHeight * 0.8 }

// LayoutText assumes ASCII input. It produces one CharRect per byte,
// which is wrong for multibyte UTF-8 runes (continuation bytes get
// phantom rects). Editor driver tests hit the ASCII fast path in
// edit/text.Measurer and never call this with non-ASCII content.
func (fakeMeasurer) LayoutText(text string, _ gui.TextStyle, _ float32) (glyph.Layout, error) {
	rects := make([]glyph.CharRect, len(text))
	idx := make(map[int]int, len(text))
	for i := range text {
		rects[i] = glyph.CharRect{
			Rect: glyph.Rect{
				X:      float32(i) * Advance,
				Y:      0,
				Width:  Advance,
				Height: LineHeight,
			},
			Index: i,
		}
		idx[i] = i
	}
	return glyph.Layout{
		Text:            text,
		CharRects:       rects,
		CharRectByIndex: idx,
		Width:           float32(len(text)) * Advance,
		Height:          LineHeight,
	}, nil
}
