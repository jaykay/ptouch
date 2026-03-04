package protocol

import (
	"bytes"
	"testing"
)

func TestRasterLine(t *testing.T) {
	data := []byte{0xFF, 0x00, 0xAA}
	got := RasterLine(data)
	want := []byte{0x47, 0x03, 0x00, 0xFF, 0x00, 0xAA}
	if !bytes.Equal(got, want) {
		t.Fatalf("RasterLine() = %X, want %X", got, want)
	}
}

func TestRasterLineEmpty(t *testing.T) {
	got := RasterLine([]byte{})
	want := []byte{0x47, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("RasterLine(empty) = %X, want %X", got, want)
	}
}

func TestRasterLinePackBits(t *testing.T) {
	// 4 identical bytes: PackBits compresses to [253, 0xFF] (2 bytes)
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	got := RasterLinePackBits(data)
	compressed := PackBits(data)
	n := len(compressed)

	// Same header format as standard: G [lenLo] [lenHi] [data]
	if got[0] != 0x47 {
		t.Fatalf("got[0] = 0x%02X, want 0x47", got[0])
	}
	if got[1] != byte(n) {
		t.Fatalf("got[1] = %d, want %d", got[1], n)
	}
	if got[2] != byte(n>>8) {
		t.Fatalf("got[2] = 0x%02X, want 0x%02X", got[2], byte(n>>8))
	}
	if !bytes.Equal(got[3:], compressed) {
		t.Fatalf("payload mismatch: got %X, want %X", got[3:], compressed)
	}
}

func TestPackBitsAllZeros(t *testing.T) {
	// 16 zero bytes should compress to a single run.
	data := make([]byte, 16)
	compressed := PackBits(data)
	// Run of 16: control byte = 257-16 = 241, value = 0x00
	want := []byte{241, 0x00}
	if !bytes.Equal(compressed, want) {
		t.Fatalf("PackBits(16 zeros) = %X, want %X", compressed, want)
	}
}

func TestPackBitsAllDifferent(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	compressed := PackBits(data)
	// All different: literal of 5 bytes: control byte = 4, then the 5 bytes.
	want := []byte{4, 1, 2, 3, 4, 5}
	if !bytes.Equal(compressed, want) {
		t.Fatalf("PackBits(all different) = %X, want %X", compressed, want)
	}
}

func TestPackBitsMixed(t *testing.T) {
	// Literal [1, 2], then run [3, 3, 3]
	data := []byte{1, 2, 3, 3, 3}
	compressed := PackBits(data)

	// Decompress to verify round-trip.
	decompressed, err := UnpackBits(compressed)
	if err != nil {
		t.Fatalf("UnpackBits() error = %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Fatalf("round-trip failed: got %X, want %X", decompressed, data)
	}
}

func TestPackBitsMaxRun(t *testing.T) {
	// Exactly 128 identical bytes: single run.
	data := bytes.Repeat([]byte{0xAA}, 128)
	compressed := PackBits(data)
	// Run of 128: control = 257-128 = 129
	want := []byte{129, 0xAA}
	if !bytes.Equal(compressed, want) {
		t.Fatalf("PackBits(128 same) = %X, want %X", compressed, want)
	}
}

func TestPackBitsOverMaxRun(t *testing.T) {
	// 129 identical bytes: two runs (128 + 1 literal).
	data := bytes.Repeat([]byte{0xBB}, 129)
	compressed := PackBits(data)
	decompressed, err := UnpackBits(compressed)
	if err != nil {
		t.Fatalf("UnpackBits() error = %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Fatalf("round-trip for 129 bytes failed: got len=%d, want len=129", len(decompressed))
	}
}

func TestPackBitsEmpty(t *testing.T) {
	compressed := PackBits([]byte{})
	if compressed != nil {
		t.Fatalf("PackBits(empty) = %X, want nil", compressed)
	}
}

func TestPackBitsRoundTrip(t *testing.T) {
	testCases := [][]byte{
		{0x00},
		{0xFF, 0xFF},
		{0x01, 0x02, 0x03},
		{0xAA, 0xAA, 0xAA, 0xBB, 0xCC, 0xDD, 0xDD},
		bytes.Repeat([]byte{0x55}, 200),
		// Simulate a typical 16-byte printhead row (128px).
		{0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0x00},
	}
	for i, data := range testCases {
		compressed := PackBits(data)
		decompressed, err := UnpackBits(compressed)
		if err != nil {
			t.Fatalf("case %d: UnpackBits() error = %v", i, err)
		}
		if !bytes.Equal(decompressed, data) {
			t.Fatalf("case %d: round-trip failed: got %X, want %X", i, decompressed, data)
		}
	}
}

func TestUnpackBitsLiteralOverflow(t *testing.T) {
	// Control byte says 3 literals follow, but only 2 bytes remain.
	_, err := UnpackBits([]byte{0x02, 0xAA, 0xBB})
	if err == nil {
		t.Fatal("expected error for truncated literal")
	}
}

func TestUnpackBitsRunMissing(t *testing.T) {
	// Control byte for run, but no value byte.
	_, err := UnpackBits([]byte{0xFE})
	if err == nil {
		t.Fatal("expected error for missing run byte")
	}
}
