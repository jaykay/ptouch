package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestInit(t *testing.T) {
	got := Init()
	if len(got) != 102 {
		t.Fatalf("Init() length = %d, want 102", len(got))
	}
	for i := 0; i < 100; i++ {
		if got[i] != 0x00 {
			t.Fatalf("Init()[%d] = 0x%02X, want 0x00", i, got[i])
		}
	}
	if got[100] != 0x1B || got[101] != 0x40 {
		t.Fatalf("Init() tail = [0x%02X, 0x%02X], want [0x1B, 0x40]", got[100], got[101])
	}
}

func TestStatusRequest(t *testing.T) {
	got := StatusRequest()
	want := []byte{0x1B, 0x69, 0x53}
	if !bytes.Equal(got, want) {
		t.Fatalf("StatusRequest() = %X, want %X", got, want)
	}
}

func TestRasterStart(t *testing.T) {
	tests := []struct {
		name string
		p700 bool
		want []byte
	}{
		{"standard", false, []byte{0x1B, 0x69, 0x52, 0x01}},
		{"P700", true, []byte{0x1B, 0x69, 0x61, 0x01}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RasterStart(tt.p700)
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("RasterStart(%v) = %X, want %X", tt.p700, got, tt.want)
			}
		})
	}
}

func TestMediaInfo(t *testing.T) {
	tests := []struct {
		name        string
		mediaType   byte
		width       byte
		length      byte
		rasterLines uint32
	}{
		{"24mm_100lines", 0x01, 24, 0, 100},
		{"12mm_500lines", 0x01, 12, 0, 500},
		{"18mm_large", 0x03, 18, 0, 65536},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MediaInfo(tt.mediaType, tt.width, tt.length, tt.rasterLines)
			if len(got) != 13 {
				t.Fatalf("MediaInfo() length = %d, want 13", len(got))
			}
			if got[0] != 0x1B || got[1] != 0x69 || got[2] != 0x7A {
				t.Fatalf("MediaInfo() prefix = %X, want 1B697A", got[:3])
			}
			if got[3] != 0x86 {
				t.Fatalf("MediaInfo() validity = 0x%02X, want 0x86", got[3])
			}
			if got[4] != tt.mediaType {
				t.Fatalf("MediaInfo() mediaType = 0x%02X, want 0x%02X", got[4], tt.mediaType)
			}
			if got[5] != tt.width {
				t.Fatalf("MediaInfo() width = %d, want %d", got[5], tt.width)
			}
			if got[6] != tt.length {
				t.Fatalf("MediaInfo() length = %d, want %d", got[6], tt.length)
			}
			gotLines := binary.LittleEndian.Uint32(got[7:11])
			if gotLines != tt.rasterLines {
				t.Fatalf("MediaInfo() rasterLines = %d, want %d", gotLines, tt.rasterLines)
			}
		})
	}
}

func TestCompression(t *testing.T) {
	tests := []struct {
		mode byte
		want []byte
	}{
		{CompressionNone, []byte{0x4D, 0x00}},
		{CompressionPackBits, []byte{0x4D, 0x02}},
	}
	for _, tt := range tests {
		got := Compression(tt.mode)
		if !bytes.Equal(got, tt.want) {
			t.Fatalf("Compression(0x%02X) = %X, want %X", tt.mode, got, tt.want)
		}
	}
}

func TestPrecut(t *testing.T) {
	tests := []struct {
		enable bool
		want   []byte
	}{
		{false, []byte{0x1B, 0x69, 0x4D, 0x00}},
		{true, []byte{0x1B, 0x69, 0x4D, 0x40}},
	}
	for _, tt := range tests {
		got := Precut(tt.enable)
		if !bytes.Equal(got, tt.want) {
			t.Fatalf("Precut(%v) = %X, want %X", tt.enable, got, tt.want)
		}
	}
}

func TestSingleByteCommands(t *testing.T) {
	tests := []struct {
		name string
		fn   func() []byte
		want []byte
	}{
		{"EmptyLine", EmptyLine, []byte{0x5A}},
		{"FormFeed", FormFeed, []byte{0x0C}},
		{"PrintEject", PrintEject, []byte{0x1A}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("%s() = %X, want %X", tt.name, got, tt.want)
			}
		})
	}
}

func TestFlagHas(t *testing.T) {
	f := FlagRasterPackBits | FlagP700Init
	if !f.Has(FlagRasterPackBits) {
		t.Fatal("expected Has(FlagRasterPackBits) = true")
	}
	if !f.Has(FlagP700Init) {
		t.Fatal("expected Has(FlagP700Init) = true")
	}
	if f.Has(FlagHasPrecut) {
		t.Fatal("expected Has(FlagHasPrecut) = false")
	}
}
