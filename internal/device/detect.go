package device

import "github.com/jaykay/ptouch/internal/protocol"

// DetectResult holds the auto-detected model and tape from a status response.
type DetectResult struct {
	Model *Model // nil if the printer model is not recognized
	Tape  *Tape  // nil if no media is loaded or width is unknown
}

// Detect maps a printer status response to known model and tape specs.
//
// Model detection: the status Model byte (offset 4) is matched against known
// product ID low bytes. This heuristic works for most Brother printers where
// the status model byte corresponds to the low byte of the USB product ID.
//
// Tape detection: status.MediaWidth (mm) is looked up in the tape table.
func Detect(status protocol.Status) DetectResult {
	var result DetectResult

	// Try matching the model byte against known product IDs.
	// The status model byte often equals the low byte of the PID.
	// Try both with 0x20xx prefix (common for P-Touch) and exact.
	pid := uint16(0x2000) | uint16(status.Model)
	result.Model = LookupByProductID(pid)
	if result.Model != nil && result.Model.Flags.Has(protocol.FlagPLite) {
		// Don't auto-detect P-Lite entries.
		result.Model = nil
	}

	// Tape lookup by reported width.
	if status.MediaWidth > 0 {
		result.Tape = LookupTape(int(status.MediaWidth))
	}

	return result
}
