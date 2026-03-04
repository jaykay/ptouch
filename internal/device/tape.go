package device

// Tape describes the physical and printable dimensions of a tape cartridge.
type Tape struct {
	WidthMM  int     // tape width in mm (status byte 10 reports this)
	Pixels   int     // printable pixel width at 180 DPI
	MarginMM float64 // margin on each side in mm
}

// tapes is the tape database at 180 DPI, from the C reference.
// The width in mm matches what the printer reports in status byte 10.
// Note: 3.5mm tape is reported as width=4 by the printer.
var tapes = []Tape{
	{WidthMM: 4, Pixels: 24, MarginMM: 0.5},   // 3.5mm tape
	{WidthMM: 6, Pixels: 32, MarginMM: 1.0},    // 6mm tape
	{WidthMM: 9, Pixels: 52, MarginMM: 1.0},    // 9mm tape
	{WidthMM: 12, Pixels: 76, MarginMM: 2.0},   // 12mm tape
	{WidthMM: 18, Pixels: 120, MarginMM: 3.0},  // 18mm tape
	{WidthMM: 24, Pixels: 128, MarginMM: 3.0},  // 24mm tape
	{WidthMM: 36, Pixels: 192, MarginMM: 4.5},  // 36mm tape
}

// LookupTape returns the tape spec for the given width in mm.
// Returns nil if the width is not recognized.
func LookupTape(widthMM int) *Tape {
	for i := range tapes {
		if tapes[i].WidthMM == widthMM {
			return &tapes[i]
		}
	}
	return nil
}

// Tapes returns all known tape specifications.
func Tapes() []Tape {
	out := make([]Tape, len(tapes))
	copy(out, tapes)
	return out
}
