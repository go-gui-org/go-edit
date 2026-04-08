package edit

import (
	"github.com/mike-ward/go-edit/edit/buffer"
	"github.com/mike-ward/go-gui/gui"
)

// EditorCfg configures an Editor widget instance.
//
// IDFocus is the focus/state key. Width and Height define the fixed
// viewport size — the Editor manages scrolling inside this rectangle
// and never virtualizes through go-gui's Column-scroll mechanism
// (DrawCanvas caches the full draw output, which defeats line
// virtualization).
type EditorCfg struct {
	IDFocus         uint32
	Buffer          *buffer.Buffer
	Width           float32
	Height          float32
	ShowLineNumbers bool
	ReadOnly        bool
}

// Editor returns a go-gui View rendering a scrollable monospace
// code editor backed by cfg.Buffer.
func Editor(cfg EditorCfg) gui.View {
	frame := &editorFrameData{}

	canvas := gui.DrawCanvas(gui.DrawCanvasCfg{
		// ID empty → skip draw cache; OnDraw runs every frame.
		Width:          cfg.Width,
		Height:         cfg.Height,
		Clip:           true,
		OnDraw:         editorOnDraw(cfg, frame),
		OnMouseScroll:  editorOnMouseScroll(cfg, frame),
	})

	return gui.Column(gui.ContainerCfg{
		IDFocus:     cfg.IDFocus,
		Width:       cfg.Width,
		Height:      cfg.Height,
		Clip:        true,
		OnKeyDown:   editorOnKeyDown(cfg, frame),
		OnChar:      editorOnChar(cfg, frame),
		AmendLayout: editorAmendLayout(cfg, frame),
		Content:     []gui.View{canvas},
	})
}
