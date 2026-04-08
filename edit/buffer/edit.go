package buffer

// Edit is the single mutation shape applied to a Buffer: replace the
// bytes in Range with NewBytes. Insert is Range.Empty() with non-nil
// NewBytes; delete is non-empty Range with nil NewBytes.
//
// A single Edit type (not a tagged union of Insert/Delete/Replace) keeps
// the undo record and EditFilter surfaces minimal. Decision locked in
// Phase -1.
type Edit struct {
	Range    Range
	NewBytes []byte
}

// Change is the undo record produced by Buffer.Apply. It carries enough
// information to restore prior state by inverting the Edit.
//
// OldBytes is a copy of the bytes replaced (owned by the Change, safe
// to retain). AppliedRange is the range the new bytes now occupy after
// the edit — equal to [Edit.Range.Start, Edit.Range.Start + len of new
// bytes expressed as lines/cols].
type Change struct {
	Applied      Edit
	OldBytes     []byte
	AppliedRange Range
}
