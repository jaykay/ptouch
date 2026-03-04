package protocol

import (
	"testing"
)

// validPacket returns a valid 32-byte status packet with sensible defaults.
func validPacket() [StatusPacketSize]byte {
	var p [StatusPacketSize]byte
	p[0] = 0x80  // printhead mark
	p[1] = 0x20  // size
	p[2] = 'B'   // brother code
	p[3] = '0'   // series code
	p[4] = 0x64  // model
	p[5] = '0'   // country
	p[10] = 24   // media width mm
	p[11] = 0x01 // media type: laminated
	p[18] = 0x00 // status type: reply
	p[24] = 0x01 // tape color: white
	p[25] = 0x08 // text color: black
	return p
}

func TestParseStatusValid(t *testing.T) {
	pkt := validPacket()
	s, err := ParseStatus(pkt[:])
	if err != nil {
		t.Fatalf("ParseStatus() error = %v", err)
	}
	if s.PrintHeadMark != 0x80 {
		t.Errorf("PrintHeadMark = 0x%02X, want 0x80", s.PrintHeadMark)
	}
	if s.BrotherCode != 'B' {
		t.Errorf("BrotherCode = 0x%02X, want 'B'", s.BrotherCode)
	}
	if s.MediaWidth != 24 {
		t.Errorf("MediaWidth = %d, want 24", s.MediaWidth)
	}
	if s.MediaType != MediaLaminated {
		t.Errorf("MediaType = 0x%02X, want 0x01", s.MediaType)
	}
	if s.TapeColor != TapeWhite {
		t.Errorf("TapeColor = 0x%02X, want 0x01", s.TapeColor)
	}
	if s.TextColor != TextBlack {
		t.Errorf("TextColor = 0x%02X, want 0x08", s.TextColor)
	}
	if s.HasError() {
		t.Errorf("HasError() = true, want false")
	}
	if !s.IsReady() {
		t.Errorf("IsReady() = false, want true")
	}
	if s.TapeWidthMM() != 24 {
		t.Errorf("TapeWidthMM() = %d, want 24", s.TapeWidthMM())
	}
}

func TestParseStatusWithErrors(t *testing.T) {
	pkt := validPacket()
	// Set error flags: cover open (bit 8) + no media (bit 0)
	pkt[8] = 0x01 // low byte: no media
	pkt[9] = 0x01 // high byte: cover open

	s, err := ParseStatus(pkt[:])
	if err != nil {
		t.Fatalf("ParseStatus() error = %v", err)
	}
	if !s.HasError() {
		t.Fatal("HasError() = false, want true")
	}
	if s.Error&ErrNoMedia == 0 {
		t.Error("expected ErrNoMedia flag set")
	}
	if s.Error&ErrCoverOpen == 0 {
		t.Error("expected ErrCoverOpen flag set")
	}
	str := s.Error.String()
	if str == "none" {
		t.Error("Error.String() should not be 'none'")
	}
}

func TestParseStatusNoMedia(t *testing.T) {
	pkt := validPacket()
	pkt[11] = byte(MediaNone) // no media loaded
	s, err := ParseStatus(pkt[:])
	if err != nil {
		t.Fatalf("ParseStatus() error = %v", err)
	}
	if s.IsReady() {
		t.Error("IsReady() = true, want false (no media)")
	}
}

func TestParseStatusWrongLength(t *testing.T) {
	_, err := ParseStatus(make([]byte, 31))
	if err == nil {
		t.Fatal("expected error for 31-byte packet")
	}
	_, err = ParseStatus(make([]byte, 33))
	if err == nil {
		t.Fatal("expected error for 33-byte packet")
	}
}

func TestParseStatusInvalidHeader(t *testing.T) {
	pkt := validPacket()
	pkt[0] = 0x00 // invalid printhead mark
	_, err := ParseStatus(pkt[:])
	if err == nil {
		t.Fatal("expected error for invalid printhead mark")
	}

	pkt = validPacket()
	pkt[1] = 0x10 // invalid size byte
	_, err = ParseStatus(pkt[:])
	if err == nil {
		t.Fatal("expected error for invalid size byte")
	}
}

func TestMediaTypeString(t *testing.T) {
	tests := []struct {
		mt   MediaType
		want string
	}{
		{MediaNone, "no media"},
		{MediaLaminated, "laminated"},
		{MediaNonLaminated, "non-laminated"},
		{MediaFabric, "fabric"},
		{MediaIncompatible, "incompatible tape"},
		{MediaType(0xAB), "unknown (0xAB)"},
	}
	for _, tt := range tests {
		got := tt.mt.String()
		if got != tt.want {
			t.Errorf("MediaType(0x%02X).String() = %q, want %q", byte(tt.mt), got, tt.want)
		}
	}
}

func TestTapeColorString(t *testing.T) {
	if TapeWhite.String() != "white" {
		t.Errorf("TapeWhite.String() = %q", TapeWhite.String())
	}
	if TapeColor(0xFE).String() != "unknown (0xFE)" {
		t.Errorf("unknown tape color string = %q", TapeColor(0xFE).String())
	}
}

func TestTextColorString(t *testing.T) {
	if TextBlack.String() != "black" {
		t.Errorf("TextBlack.String() = %q", TextBlack.String())
	}
	if TextGold.String() != "gold" {
		t.Errorf("TextGold.String() = %q", TextGold.String())
	}
}

func TestStatusTypeString(t *testing.T) {
	if StatusPrintDone.String() != "print done" {
		t.Errorf("StatusPrintDone.String() = %q", StatusPrintDone.String())
	}
	if StatusType(0xFF).String() != "unknown (0xFF)" {
		t.Errorf("unknown status type string = %q", StatusType(0xFF).String())
	}
}

func TestErrorFlagsString(t *testing.T) {
	if ErrorFlags(0).String() != "none" {
		t.Errorf("zero ErrorFlags.String() = %q", ErrorFlags(0).String())
	}
	e := ErrCoverOpen | ErrNoMedia
	s := e.String()
	if s == "none" {
		t.Fatal("combined error string should not be 'none'")
	}
}
