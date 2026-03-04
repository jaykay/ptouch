package raster

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderImage(t *testing.T) {
	// Create a simple 20x10 test image: left half black, right half white.
	img := image.NewRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			if x < 10 {
				img.Set(x, y, color.Black)
			} else {
				img.Set(x, y, color.White)
			}
		}
	}

	result := RenderImage(img, 128, 128, 0) // 24mm tape
	if result.Bitmap == nil {
		t.Fatal("Bitmap is nil")
	}
	if len(result.RasterRows) == 0 {
		t.Fatal("no raster rows")
	}
	// After scaling to 128px height and rotation, check dimensions.
	t.Logf("scaled: %dx%d, rotated bitmap: %dx%d, raster rows: %d",
		result.WidthPx, result.HeightPx, result.Bitmap.Width, result.Bitmap.Height, len(result.RasterRows))

	// Each row should be 16 bytes (128px / 8).
	for i, row := range result.RasterRows {
		if len(row) != 16 {
			t.Errorf("row %d len = %d, want 16", i, len(row))
		}
	}
}

func TestRenderImageSmallTape(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 5))
	for y := 0; y < 5; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.Black)
		}
	}

	result := RenderImage(img, 32, 128, 0) // 6mm tape
	if result.Bitmap == nil {
		t.Fatal("Bitmap is nil")
	}
	if result.HeightPx != 32 {
		t.Errorf("HeightPx = %d, want 32", result.HeightPx)
	}
}

func TestRenderImageAspectRatio(t *testing.T) {
	// 100x50 image scaled to 128px height → should be 256px wide (2:1 ratio).
	img := image.NewRGBA(image.Rect(0, 0, 100, 50))
	result := RenderImage(img, 128, 128, 0)
	if result.WidthPx != 256 {
		t.Errorf("WidthPx = %d, want 256 (aspect ratio preserved)", result.WidthPx)
	}
}

func TestLoadImage(t *testing.T) {
	// Create a temporary PNG file.
	dir := t.TempDir()
	path := filepath.Join(dir, "test.png")

	img := image.NewRGBA(image.Rect(0, 0, 16, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 16; x++ {
			if (x+y)%2 == 0 {
				img.Set(x, y, color.Black)
			} else {
				img.Set(x, y, color.White)
			}
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("encode PNG: %v", err)
	}
	f.Close()

	result, err := LoadImage(path, 128, 128, 0)
	if err != nil {
		t.Fatalf("LoadImage error: %v", err)
	}
	if result.Bitmap == nil {
		t.Fatal("Bitmap is nil")
	}
	if len(result.RasterRows) == 0 {
		t.Fatal("no raster rows")
	}
}

func TestLoadImageNotFound(t *testing.T) {
	_, err := LoadImage("/nonexistent/file.png", 128, 128, 0)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestScaleImageNoOp(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	scaled := scaleImage(img, 10, 10)
	// Should return the same image (no scaling needed).
	if scaled != img {
		t.Error("scaleImage should return original when dimensions match")
	}
}
