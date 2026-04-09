package edit

import (
	"github.com/mike-ward/go-edit/edit/buffer"
	"github.com/mike-ward/go-edit/edit/text"
	"github.com/mike-ward/go-gui/gui"
)

// nsEdit is the StateMap namespace for persistent editor state
// keyed by IDFocus.
const nsEdit = "edit.state"

// capEdit caps the number of concurrently tracked editor instances
// per window.
const capEdit = 64

// editorState is the persistent per-instance state, stored in the
// window's StateMap across frames.
type editorState struct {
	Cursor     buffer.Position
	Anchor     buffer.Position // selection anchor; Anchor == Cursor → no sel
	DesiredCol int             // sticky col for Up/Down movement
	ScrollY    float32         // scroll offset in pixels
	Measurer   *text.Measurer

	// Mouse click tracking for double/triple-click detection.
	LastClickTime int64           // UnixMilli of last mouse-down
	LastClickPos  buffer.Position // position of last click
	ClickCount    int             // 1=single, 2=double, 3=triple
}

// editorFrameData is the per-frame snapshot shared between the
// AmendLayout callback (which has *Window) and the OnDraw callback
// (which does not). One instance per Editor(cfg) call, discarded at
// end of frame.
type editorFrameData struct {
	state      editorState
	lineHeight float32
	gutterW    float32
	padLeft    float32 // padding between gutter and text
	valid      bool    // set true by AmendLayout; OnDraw no-ops if false
}

func loadState(w *gui.Window, id uint32) editorState {
	m := gui.StateMap[uint32, editorState](w, nsEdit, capEdit)
	s, _ := m.Get(id)
	return s
}

func storeState(w *gui.Window, id uint32, s editorState) {
	m := gui.StateMap[uint32, editorState](w, nsEdit, capEdit)
	m.Set(id, s)
}
