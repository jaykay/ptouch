// Package protocol implements the Brother P-Touch ESC/P command protocol.
//
// The printer communicates via ESC-prefixed byte sequences for initialization,
// mode selection, tape configuration, and raster data transfer.
package protocol

import "encoding/binary"

const esc = 0x1B

// Flag represents printer capability flags.
type Flag uint8

const (
	FlagNone           Flag = 0x00
	FlagUnsupRaster    Flag = 0x01
	FlagRasterPackBits Flag = 0x02
	FlagPLite          Flag = 0x04
	FlagP700Init       Flag = 0x08
	FlagUseInfoCmd     Flag = 0x10
	FlagHasPrecut      Flag = 0x20
	FlagD460BTMagic    Flag = 0x40
)

// Has returns true if f contains the given flag.
func (f Flag) Has(flag Flag) bool { return f&flag != 0 }

// PageFormat represents page formatting flags.
type PageFormat byte

const (
	FeedNone   PageFormat = 0x00
	FeedSmall  PageFormat = 0x08
	FeedMedium PageFormat = 0x0C
	FeedLarge  PageFormat = 0x1A
	AutoCut    PageFormat = 0x40
	Mirror     PageFormat = 0x80
)

// Init returns the 102-byte initialization sequence (100 zero bytes + ESC @).
func Init() []byte {
	buf := make([]byte, 102)
	buf[100] = esc
	buf[101] = 0x40
	return buf
}

// StatusRequest returns the 3-byte status request command.
func StatusRequest() []byte {
	return []byte{esc, 0x69, 0x53}
}

// RasterStart returns the raster-mode start command.
// If p700 is true, uses the P700 variant (ESC i a 0x01).
func RasterStart(p700 bool) []byte {
	if p700 {
		return []byte{esc, 0x69, 0x61, 0x01}
	}
	return []byte{esc, 0x69, 0x52, 0x01}
}

// MediaInfo returns the media info command.
// mediaType: tape type byte, width/length in mm, rasterLines: total raster rows.
func MediaInfo(mediaType byte, width byte, length byte, rasterLines uint32) []byte {
	buf := make([]byte, 13)
	buf[0] = esc
	buf[1] = 0x69
	buf[2] = 0x7A
	buf[3] = 0x86 // validity: width + length + raster count valid
	buf[4] = mediaType
	buf[5] = width
	buf[6] = length
	binary.LittleEndian.PutUint32(buf[7:11], rasterLines)
	buf[11] = 0x00 // starting page
	buf[12] = 0x00 // reserved
	return buf
}

// Compression returns the 2-byte compression mode command.
func Compression(mode byte) []byte {
	return []byte{0x4D, mode}
}

// CompressionNone disables compression.
const CompressionNone byte = 0x00

// CompressionPackBits enables PackBits compression.
const CompressionPackBits byte = 0x02

// DefaultFeedMargin is the feed margin in dots on each side of the label.
// The printer feeds this amount of blank tape before and after the content
// for the cutter. At 180 DPI, 14 dots ≈ 2mm per side ≈ 4mm total.
const DefaultFeedMargin = 14

// Margin returns the feed margin command (ESC i d).
// dots is the margin in printer dots on each side of the label.
func Margin(dots int) []byte {
	return []byte{esc, 0x69, 0x64, byte(dots & 0xFF), byte((dots >> 8) & 0xFF)}
}

// Precut returns the precut command.
func Precut(enable bool) []byte {
	if enable {
		return []byte{esc, 0x69, 0x4D, 0x40}
	}
	return []byte{esc, 0x69, 0x4D, 0x00}
}

// EmptyLine returns the single-byte empty raster line command.
func EmptyLine() []byte {
	return []byte{0x5A}
}

// FormFeed returns the form-feed command (chain print, no cut).
func FormFeed() []byte {
	return []byte{0x0C}
}

// PrintEject returns the print-and-eject command (final cut).
func PrintEject() []byte {
	return []byte{0x1A}
}
