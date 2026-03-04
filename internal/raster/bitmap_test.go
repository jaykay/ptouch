package raster

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestNewBitmap(t *testing.T) {
	b := NewBitmap(16, 8)
	if b.Width != 16 || b.Height != 8 {
		t.Fatalf("dims = %dx%d, want 16x8", b.Width, b.Height)
	}
	if b.Stride != 2 {
		t.Fatalf("Stride = %d, want 2", b.Stride)
	}
	if len(b.Data) != 16 {
		t.Fatalf("Data len = %d, want 16", len(b.Data))
	}
}

func TestNewBitmapNonAligned(t *testing.T) {
	b := NewBitmap(10, 1)
	if b.Stride != 2 {
		t.Fatalf("Stride for 10px = %d, want 2", b.Stride)
	}
}

func TestSetGetPixel(t *testing.T) {
	b := NewBitmap(16, 8)

	// All pixels start white.
	if b.GetPixel(0, 0) {
		t.Fatal("pixel (0,0) should be white initially")
	}

	// Set and get.
	b.SetPixel(0, 0, true)
	if !b.GetPixel(0, 0) {
		t.Fatal("pixel (0,0) should be black after set")
	}

	// Clear.
	b.SetPixel(0, 0, false)
	if b.GetPixel(0, 0) {
		t.Fatal("pixel (0,0) should be white after clear")
	}

	// Bit position: pixel 7 should be bit 0 of byte 0 (MSB first).
	b.SetPixel(7, 0, true)
	if b.Data[0] != 0x01 {
		t.Fatalf("byte 0 = 0x%02X, want 0x01 for pixel 7", b.Data[0])
	}

	// Pixel 0 should be bit 7 of byte 0.
	b.SetPixel(0, 0, true)
	if b.Data[0] != 0x81 {
		t.Fatalf("byte 0 = 0x%02X, want 0x81 for pixels 0+7", b.Data[0])
	}
}

func TestSetPixelOutOfBounds(t *testing.T) {
	b := NewBitmap(8, 8)
	// Should not panic.
	b.SetPixel(-1, 0, true)
	b.SetPixel(0, -1, true)
	b.SetPixel(8, 0, true)
	b.SetPixel(0, 8, true)
	if b.GetPixel(-1, 0) || b.GetPixel(8, 0) {
		t.Fatal("out-of-bounds GetPixel should return false")
	}
}

func TestTranspose(t *testing.T) {
	// Create a 4x2 bitmap:
	//   Row 0: X . . .
	//   Row 1: . . . X
	b := NewBitmap(4, 2)
	b.SetPixel(0, 0, true)
	b.SetPixel(3, 1, true)

	r := b.Transpose()
	// Transpose: (x, y) -> (y, x). Result should be 2x4.
	//   (0,0) -> (0, 0)
	//   (3,1) -> (1, 3)
	if r.Width != 2 || r.Height != 4 {
		t.Fatalf("transposed dims = %dx%d, want 2x4", r.Width, r.Height)
	}
	if !r.GetPixel(0, 0) {
		t.Error("expected black at transposed (0,0)")
	}
	if !r.GetPixel(1, 3) {
		t.Error("expected black at transposed (1,3)")
	}
	if r.GetPixel(1, 0) {
		t.Error("expected white at transposed (1,0)")
	}
}

func TestTransposeDoubleIsIdentity(t *testing.T) {
	// Transposing twice should return to original.
	b := NewBitmap(8, 4)
	b.SetPixel(2, 1, true)
	b.SetPixel(5, 3, true)

	r := b.Transpose().Transpose()
	if r.Width != b.Width || r.Height != b.Height {
		t.Fatalf("2x transpose dims = %dx%d, want %dx%d", r.Width, r.Height, b.Width, b.Height)
	}
	if !r.GetPixel(2, 1) {
		t.Error("pixel (2,1) lost after 2x transpose")
	}
	if !r.GetPixel(5, 3) {
		t.Error("pixel (5,3) lost after 2x transpose")
	}
}

func TestToRasterRows(t *testing.T) {
	b := NewBitmap(8, 3)
	b.SetPixel(0, 0, true) // byte 0 = 0x80
	b.SetPixel(7, 1, true) // byte 0 = 0x01
	// Row 2 all white.

	rows := b.ToRasterRows(128) // P-750W: 128px = 16 bytes
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	for _, r := range rows {
		if len(r) != 16 {
			t.Fatalf("row len = %d, want 16", len(r))
		}
	}
	if rows[0][0] != 0x80 {
		t.Errorf("row 0 byte 0 = 0x%02X, want 0x80", rows[0][0])
	}
	if rows[1][0] != 0x01 {
		t.Errorf("row 1 byte 0 = 0x%02X, want 0x01", rows[1][0])
	}
	if rows[2][0] != 0x00 {
		t.Errorf("row 2 byte 0 = 0x%02X, want 0x00", rows[2][0])
	}
}

func TestToPNG(t *testing.T) {
	b := NewBitmap(8, 4)
	b.SetPixel(0, 0, true)
	b.SetPixel(7, 3, true)

	var buf bytes.Buffer
	if err := b.ToPNG(&buf); err != nil {
		t.Fatalf("ToPNG error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("PNG output is empty")
	}

	// Decode and verify.
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("png.Decode error: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 8 || bounds.Dy() != 4 {
		t.Fatalf("PNG dims = %dx%d, want 8x4", bounds.Dx(), bounds.Dy())
	}
}

func TestFromImage(t *testing.T) {
	// Create a 4x4 RGBA image: top-left black, rest white.
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.White)
		}
	}
	img.Set(0, 0, color.Black)
	img.Set(3, 3, color.Black)

	bm := FromImage(img, 127)
	if bm.Width != 4 || bm.Height != 4 {
		t.Fatalf("dims = %dx%d, want 4x4", bm.Width, bm.Height)
	}
	if !bm.GetPixel(0, 0) {
		t.Error("pixel (0,0) should be black")
	}
	if !bm.GetPixel(3, 3) {
		t.Error("pixel (3,3) should be black")
	}
	if bm.GetPixel(1, 1) {
		t.Error("pixel (1,1) should be white")
	}
}

func TestFromImageGrayThreshold(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 3, 1))
	img.SetGray(0, 0, color.Gray{Y: 0})   // pure black
	img.SetGray(1, 0, color.Gray{Y: 126}) // just below threshold
	img.SetGray(2, 0, color.Gray{Y: 128}) // just above threshold

	bm := FromImage(img, 127)
	if !bm.GetPixel(0, 0) {
		t.Error("pixel 0 (gray 0) should be black")
	}
	if !bm.GetPixel(1, 0) {
		t.Error("pixel 1 (gray 126) should be black")
	}
	if bm.GetPixel(2, 0) {
		t.Error("pixel 2 (gray 128) should be white")
	}
}
