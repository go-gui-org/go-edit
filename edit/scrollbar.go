package edit

// ScrollbarMode controls when the vertical scrollbar is displayed.
type ScrollbarMode int

const (
	// ScrollbarAuto shows the scrollbar only when content overflows the viewport.
	ScrollbarAuto ScrollbarMode = iota
	// ScrollbarAlways always shows the scrollbar.
	ScrollbarAlways
	// ScrollbarNever never shows the scrollbar.
	ScrollbarNever
)

const (
	scrollbarWidth    float32 = 8
	scrollbarMinThumb float32 = 20
)

// scrollbarVisible reports whether the scrollbar track should be drawn.
func scrollbarVisible(mode ScrollbarMode, totalVisRows int, lineHeight, viewportH float32) bool {
	switch mode {
	case ScrollbarNever:
		return false
	case ScrollbarAlways:
		return true
	default: // ScrollbarAuto
		if lineHeight <= 0 || totalVisRows <= 0 {
			return false
		}
		return float32(totalVisRows)*lineHeight > viewportH
	}
}

// scrollbarHorizVisible reports whether the horizontal scrollbar
// track should be drawn. Wrapping mode disables horizontal scroll.
func scrollbarHorizVisible(
	mode ScrollbarMode, wrapActive bool, contentW, viewportW float32,
) bool {
	if wrapActive {
		return false
	}
	switch mode {
	case ScrollbarNever:
		return false
	case ScrollbarAlways:
		return true
	default: // ScrollbarAuto
		return contentW > viewportW
	}
}

// scrollbarGeometry computes the thumb top Y and height. Returns
// hasThumb=false when content fits entirely in the viewport.
func scrollbarGeometry(
	totalVisRows int, lineHeight, viewportH, scrollY, trackH float32,
) (thumbY, thumbH float32, hasThumb bool) {
	if lineHeight <= 0 || totalVisRows <= 0 || trackH <= 0 {
		return 0, 0, false
	}
	contentH := float32(totalVisRows) * lineHeight
	if contentH <= viewportH {
		return 0, 0, false
	}
	thumbH = viewportH / contentH * trackH
	if thumbH < scrollbarMinThumb {
		thumbH = scrollbarMinThumb
	}
	if thumbH > trackH {
		thumbH = trackH
	}
	thumbRange := trackH - thumbH
	maxScroll := contentH - viewportH
	if thumbRange > 0 && maxScroll > 0 {
		thumbY = scrollY / maxScroll * thumbRange
	}
	return thumbY, thumbH, true
}
