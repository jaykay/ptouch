package protocol

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// StatusPacketSize is the fixed size of a printer status response.
const StatusPacketSize = 32

// MediaType represents the tape/label media type.
type MediaType byte

const (
	MediaNone         MediaType = 0x00
	MediaLaminated    MediaType = 0x01
	MediaNonLaminated MediaType = 0x03
	MediaFabric       MediaType = 0x04
	MediaHeatShrink21 MediaType = 0x11
	MediaHeatShrink31 MediaType = 0x17
	MediaFlexi        MediaType = 0x13
	MediaFlexID       MediaType = 0x14
	MediaSatin        MediaType = 0x15
	MediaIncompatible MediaType = 0xFF
)

func (m MediaType) String() string {
	switch m {
	case MediaNone:
		return "no media"
	case MediaLaminated:
		return "laminated"
	case MediaNonLaminated:
		return "non-laminated"
	case MediaFabric:
		return "fabric"
	case MediaHeatShrink21:
		return "heat-shrink 2:1"
	case MediaHeatShrink31:
		return "heat-shrink 3:1"
	case MediaFlexi:
		return "flexible ID"
	case MediaFlexID:
		return "flex ID"
	case MediaSatin:
		return "satin"
	case MediaIncompatible:
		return "incompatible tape"
	default:
		return fmt.Sprintf("unknown (0x%02X)", byte(m))
	}
}

// TapeColor represents the tape background color.
type TapeColor byte

const (
	TapeWhite      TapeColor = 0x01
	TapeOther      TapeColor = 0x02
	TapeClear      TapeColor = 0x03
	TapeRed        TapeColor = 0x04
	TapeBlue       TapeColor = 0x05
	TapeYellow     TapeColor = 0x06
	TapeGreen      TapeColor = 0x07
	TapeBlack      TapeColor = 0x08
	TapeMatteWhite TapeColor = 0x20
)

func (c TapeColor) String() string {
	switch c {
	case TapeWhite:
		return "white"
	case TapeOther:
		return "other"
	case TapeClear:
		return "clear/transparent"
	case TapeRed:
		return "red"
	case TapeBlue:
		return "blue"
	case TapeYellow:
		return "yellow"
	case TapeGreen:
		return "green"
	case TapeBlack:
		return "black"
	case TapeMatteWhite:
		return "matte white"
	default:
		return fmt.Sprintf("unknown (0x%02X)", byte(c))
	}
}

// TextColor represents the text/ink color.
type TextColor byte

const (
	TextWhite TextColor = 0x01
	TextRed   TextColor = 0x04
	TextBlue  TextColor = 0x05
	TextBlack TextColor = 0x08
	TextGold  TextColor = 0x0A
)

func (c TextColor) String() string {
	switch c {
	case TextWhite:
		return "white"
	case TextRed:
		return "red"
	case TextBlue:
		return "blue"
	case TextBlack:
		return "black"
	case TextGold:
		return "gold"
	default:
		return fmt.Sprintf("unknown (0x%02X)", byte(c))
	}
}

// StatusType identifies the kind of status response.
type StatusType byte

const (
	StatusReply        StatusType = 0x00
	StatusPrintDone    StatusType = 0x01
	StatusError        StatusType = 0x02
	StatusNotification StatusType = 0x05
	StatusPhaseChange  StatusType = 0x06
)

func (s StatusType) String() string {
	switch s {
	case StatusReply:
		return "reply"
	case StatusPrintDone:
		return "print done"
	case StatusError:
		return "error"
	case StatusNotification:
		return "notification"
	case StatusPhaseChange:
		return "phase change"
	default:
		return fmt.Sprintf("unknown (0x%02X)", byte(s))
	}
}

// ErrorFlags represents the two-byte error bitfield from the status packet.
type ErrorFlags uint16

const (
	ErrNoMedia       ErrorFlags = 1 << 0
	ErrCutterJam     ErrorFlags = 1 << 2
	ErrLowBattery    ErrorFlags = 1 << 3
	ErrInUse         ErrorFlags = 1 << 4
	ErrCoverOpen     ErrorFlags = 1 << 8
	ErrOverheat      ErrorFlags = 1 << 9
	ErrTapeNotLoaded ErrorFlags = 1 << 10
)

// HasError returns true if any error flag is set.
func (e ErrorFlags) HasError() bool { return e != 0 }

func (e ErrorFlags) String() string {
	if e == 0 {
		return "none"
	}
	var parts []string
	flags := []struct {
		flag ErrorFlags
		name string
	}{
		{ErrNoMedia, "no media"},
		{ErrCutterJam, "cutter jam"},
		{ErrLowBattery, "low battery"},
		{ErrInUse, "in use"},
		{ErrCoverOpen, "cover open"},
		{ErrOverheat, "overheat"},
		{ErrTapeNotLoaded, "tape not loaded"},
	}
	for _, f := range flags {
		if e&f.flag != 0 {
			parts = append(parts, f.name)
		}
	}
	// Report any unknown bits.
	known := ErrNoMedia | ErrCutterJam | ErrLowBattery | ErrInUse | ErrCoverOpen | ErrOverheat | ErrTapeNotLoaded
	if unknown := e &^ known; unknown != 0 {
		parts = append(parts, fmt.Sprintf("unknown(0x%04X)", uint16(unknown)))
	}
	return strings.Join(parts, ", ")
}

// Status represents a parsed 32-byte printer status packet.
type Status struct {
	PrintHeadMark byte
	Size          byte
	BrotherCode   byte
	SeriesCode    byte
	Model         byte
	Country       byte
	Error         ErrorFlags
	MediaWidth    byte // mm
	MediaType     MediaType
	MediaLength   byte // mm, 0 = continuous
	StatusType    StatusType
	PhaseType     byte
	PhaseNumber   uint16
	Notification  byte
	TapeColor     TapeColor
	TextColor     TextColor
	HWSettings    [4]byte
}

// ParseStatus decodes a 32-byte status packet.
func ParseStatus(data []byte) (Status, error) {
	if len(data) != StatusPacketSize {
		return Status{}, &ProtocolError{
			Op:  "ParseStatus",
			Msg: fmt.Sprintf("expected %d bytes, got %d", StatusPacketSize, len(data)),
		}
	}
	if data[0] != 0x80 {
		return Status{}, &ProtocolError{
			Op:  "ParseStatus",
			Msg: fmt.Sprintf("invalid printhead mark: 0x%02X", data[0]),
		}
	}
	if data[1] != 0x20 {
		return Status{}, &ProtocolError{
			Op:  "ParseStatus",
			Msg: fmt.Sprintf("unexpected size byte: 0x%02X", data[1]),
		}
	}

	s := Status{
		PrintHeadMark: data[0],
		Size:          data[1],
		BrotherCode:   data[2],
		SeriesCode:    data[3],
		Model:         data[4],
		Country:       data[5],
		Error:         ErrorFlags(binary.LittleEndian.Uint16(data[8:10])),
		MediaWidth:    data[10],
		MediaType:     MediaType(data[11]),
		MediaLength:   data[17],
		StatusType:    StatusType(data[18]),
		PhaseType:     data[19],
		PhaseNumber:   binary.LittleEndian.Uint16(data[20:22]),
		Notification:  data[22],
		TapeColor:     TapeColor(data[24]),
		TextColor:     TextColor(data[25]),
	}
	copy(s.HWSettings[:], data[26:30])
	return s, nil
}

// HasError returns true if the printer reported any errors.
func (s Status) HasError() bool { return s.Error.HasError() }

// TapeWidthMM returns the tape width in millimeters.
func (s Status) TapeWidthMM() int { return int(s.MediaWidth) }

// IsReady returns true if there are no errors and media is loaded.
func (s Status) IsReady() bool {
	return !s.HasError() && s.MediaType != MediaNone && s.MediaType != MediaIncompatible
}
