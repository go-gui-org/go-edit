// Package text wraps go-glyph for the editor: shaping, metrics,
// hit-test, glyph-run cache. Single choke point so the rest of the
// editor never imports go-glyph directly.
package text
