// Package device defines supported printer models, their capabilities,
// and tape specifications.
package device

import (
	"strings"

	"github.com/jaykay/ptouch/internal/protocol"
)

// BrotherVendorID is the USB vendor ID for all Brother printers.
const BrotherVendorID = 0x04F9

// Model describes a supported Brother P-Touch printer.
type Model struct {
	Name      string        // human-readable name, e.g. "PT-P750W"
	ProductID uint16        // USB product ID
	MaxPixels int           // printhead width in pixels
	DPI       int           // dots per inch
	Flags     protocol.Flag // capability flags
}

// models is the database of all supported printers, from the C reference.
var models = []Model{
	{"PT-9200DX", 0x2001, 384, 360, protocol.FlagRasterPackBits | protocol.FlagHasPrecut},
	{"PT-9200DX", 0x2002, 384, 360, protocol.FlagRasterPackBits | protocol.FlagHasPrecut},
	{"PT-2300", 0x2004, 112, 180, protocol.FlagRasterPackBits | protocol.FlagHasPrecut},
	{"PT-2420PC", 0x2007, 128, 180, protocol.FlagRasterPackBits},
	{"PT-2450PC", 0x2011, 128, 180, protocol.FlagRasterPackBits},
	{"PT-1950", 0x2019, 112, 180, protocol.FlagRasterPackBits},
	{"PT-2700", 0x201F, 128, 180, protocol.FlagHasPrecut},
	{"PT-1230PC", 0x202C, 128, 180, protocol.FlagNone},
	{"PT-2430PC", 0x202D, 128, 180, protocol.FlagNone},
	{"PT-1230PC", 0x2030, 128, 180, protocol.FlagPLite},
	{"PT-2430PC", 0x2031, 128, 180, protocol.FlagPLite},
	{"PT-2730", 0x2041, 128, 180, protocol.FlagNone},
	{"PT-H500", 0x205E, 128, 180, protocol.FlagRasterPackBits | protocol.FlagHasPrecut},
	{"PT-E500", 0x205F, 128, 180, protocol.FlagRasterPackBits},
	{"PT-E550W", 0x2060, 128, 180, protocol.FlagUnsupRaster},
	{"PT-P700", 0x2061, 128, 180, protocol.FlagRasterPackBits | protocol.FlagP700Init | protocol.FlagHasPrecut},
	{"PT-P750W", 0x2062, 128, 180, protocol.FlagRasterPackBits | protocol.FlagP700Init},
	{"PT-P700", 0x2064, 128, 180, protocol.FlagPLite},
	{"PT-P750W", 0x2065, 128, 180, protocol.FlagPLite},
	{"PT-D450", 0x2073, 128, 180, protocol.FlagUseInfoCmd},
	{"PT-D600", 0x2074, 128, 180, protocol.FlagRasterPackBits},
	{"PT-P710BT", 0x20AF, 128, 180, protocol.FlagRasterPackBits | protocol.FlagHasPrecut},
	{"PT-D410", 0x20DF, 128, 180, protocol.FlagUseInfoCmd | protocol.FlagHasPrecut | protocol.FlagD460BTMagic},
	{"PT-D460BT", 0x20E0, 128, 180, protocol.FlagP700Init | protocol.FlagUseInfoCmd | protocol.FlagHasPrecut | protocol.FlagD460BTMagic},
	{"PT-D610BT", 0x20E1, 128, 180, protocol.FlagP700Init | protocol.FlagUseInfoCmd | protocol.FlagHasPrecut | protocol.FlagD460BTMagic},
	{"PT-E310BT", 0x2201, 128, 180, protocol.FlagP700Init | protocol.FlagUseInfoCmd | protocol.FlagD460BTMagic},
	{"PT-E560BT", 0x2203, 128, 180, protocol.FlagP700Init | protocol.FlagUseInfoCmd | protocol.FlagD460BTMagic},
}

// LookupByProductID returns the model with the given USB product ID.
// Returns nil if not found.
func LookupByProductID(pid uint16) *Model {
	for i := range models {
		if models[i].ProductID == pid {
			return &models[i]
		}
	}
	return nil
}

// LookupByName returns the first model matching the given name (case-insensitive).
// Skips P-Lite mode entries (FlagPLite).
// Returns nil if not found.
func LookupByName(name string) *Model {
	upper := strings.ToUpper(name)
	for i := range models {
		if models[i].Flags.Has(protocol.FlagPLite) {
			continue
		}
		if strings.ToUpper(models[i].Name) == upper {
			return &models[i]
		}
	}
	return nil
}

// Models returns all known printer models (excluding P-Lite entries).
func Models() []Model {
	var out []Model
	for _, m := range models {
		if !m.Flags.Has(protocol.FlagPLite) {
			out = append(out, m)
		}
	}
	return out
}
