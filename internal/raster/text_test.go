package raster

import (
	"testing"
)

func TestRenderTextBasic(t *testing.T) {
	cfg := TextConfig{
		Lines:    []string{"Hello"},
		FontSize: 24,
	}
	result, err := RenderText(cfg, 128, 128) // 24mm tape, P-750W
	if err != nil {
		t.Fatalf("RenderText error: %v", err)
	}
	if result.Bitmap == nil {
		t.Fatal("Bitmap is nil")
	}
	if len(result.RasterRows) == 0 {
		t.Fatal("no raster rows")
	}
	// Rotated bitmap height = number of raster rows.
	if len(result.RasterRows) != result.Bitmap.Height {
		t.Errorf("raster rows = %d, bitmap height = %d", len(result.RasterRows), result.Bitmap.Height)
	}
	// Each row should be 16 bytes (128 pixels / 8).
	for i, row := range result.RasterRows {
		if len(row) != 16 {
			t.Errorf("row %d len = %d, want 16", i, len(row))
		}
	}
	// At least some rows should have non-zero data (text pixels).
	hasData := false
	for _, row := range result.RasterRows {
		for _, b := range row {
			if b != 0 {
				hasData = true
				break
			}
		}
		if hasData {
			break
		}
	}
	if !hasData {
		t.Error("all raster rows are empty — text not rendered?")
	}
}

func TestRenderTextMultiLine(t *testing.T) {
	cfg := TextConfig{
		Lines:    []string{"Line 1", "Line 2", "Line 3"},
		FontSize: 12,
	}
	result, err := RenderText(cfg, 128, 128)
	if err != nil {
		t.Fatalf("RenderText error: %v", err)
	}
	if result.HeightPx < 1 {
		t.Error("HeightPx should be positive")
	}
	if len(result.RasterRows) == 0 {
		t.Fatal("no raster rows for multi-line")
	}
}

func TestRenderTextAutoSize(t *testing.T) {
	cfg := TextConfig{
		Lines:    []string{"Auto"},
		FontSize: 0, // auto-fit
	}
	result, err := RenderText(cfg, 128, 128)
	if err != nil {
		t.Fatalf("RenderText error: %v", err)
	}
	// Auto-fit should produce a bitmap that's roughly 128px tall (tape height).
	if result.HeightPx != 128 {
		t.Logf("HeightPx = %d (auto-fit, expected ~128)", result.HeightPx)
	}
	if len(result.RasterRows) == 0 {
		t.Fatal("no raster rows")
	}
}

func TestRenderTextBold(t *testing.T) {
	cfg := TextConfig{
		Lines:    []string{"Bold"},
		FontSize: 24,
		Bold:     true,
	}
	result, err := RenderText(cfg, 128, 128)
	if err != nil {
		t.Fatalf("RenderText error: %v", err)
	}
	if len(result.RasterRows) == 0 {
		t.Fatal("no raster rows")
	}
}

func TestRenderTextAlignment(t *testing.T) {
	for _, align := range []Alignment{AlignLeft, AlignCenter, AlignRight} {
		cfg := TextConfig{
			Lines:    []string{"Short", "A longer line here"},
			FontSize: 14,
			Align:    align,
		}
		result, err := RenderText(cfg, 128, 128)
		if err != nil {
			t.Fatalf("RenderText align=%d error: %v", align, err)
		}
		if len(result.RasterRows) == 0 {
			t.Fatalf("no raster rows for align=%d", align)
		}
	}
}

func TestRenderTextEmptyLines(t *testing.T) {
	cfg := TextConfig{
		Lines: []string{},
	}
	_, err := RenderText(cfg, 128, 128)
	if err == nil {
		t.Fatal("expected error for empty lines")
	}
}

func TestRenderTextSmallTape(t *testing.T) {
	cfg := TextConfig{
		Lines:    []string{"Hi"},
		FontSize: 0, // auto-fit
	}
	// 6mm tape = 32 pixels.
	result, err := RenderText(cfg, 32, 128)
	if err != nil {
		t.Fatalf("RenderText error: %v", err)
	}
	if len(result.RasterRows) == 0 {
		t.Fatal("no raster rows")
	}
	// Rows should still be 16 bytes (padded to 128px printhead).
	for i, row := range result.RasterRows {
		if len(row) != 16 {
			t.Errorf("row %d len = %d, want 16", i, len(row))
		}
	}
}

func TestRenderTextPreview(t *testing.T) {
	cfg := TextConfig{
		Lines:    []string{"Preview"},
		FontSize: 20,
	}
	result, err := RenderText(cfg, 128, 128)
	if err != nil {
		t.Fatalf("RenderText error: %v", err)
	}
	if result.Bitmap.Width == 0 || result.Bitmap.Height == 0 {
		t.Error("bitmap dimensions should be non-zero")
	}
}

func TestRenderTextFixedWidth(t *testing.T) {
	// 50mm at 180 DPI = ~354 pixels.
	widthPx := 354
	cfg := TextConfig{
		Lines:      []string{"Fixed"},
		FontSize:   20,
		MaxWidthPx: widthPx,
	}
	result, err := RenderText(cfg, 128, 128)
	if err != nil {
		t.Fatalf("RenderText error: %v", err)
	}
	if result.WidthPx != widthPx {
		t.Errorf("WidthPx = %d, want %d", result.WidthPx, widthPx)
	}
}

func TestRenderTextFixedWidthAutoShrink(t *testing.T) {
	// Very narrow width should auto-shrink the font.
	cfg := TextConfig{
		Lines:      []string{"This is a long text line"},
		FontSize:   0, // auto-fit
		MaxWidthPx: 100,
	}
	result, err := RenderText(cfg, 128, 128)
	if err != nil {
		t.Fatalf("RenderText error: %v", err)
	}
	// Canvas should not exceed the fixed width.
	if result.WidthPx != 100 {
		t.Errorf("WidthPx = %d, want 100", result.WidthPx)
	}
}

func TestRenderTextMultiLineAutoFit(t *testing.T) {
	// 4 lines should auto-size to fit tape height.
	cfg := TextConfig{
		Lines:    []string{"Line 1", "Line 2", "Line 3", "Line 4"},
		FontSize: 0, // auto-fit
	}
	result, err := RenderText(cfg, 128, 128)
	if err != nil {
		t.Fatalf("RenderText error: %v", err)
	}
	if result.HeightPx != 128 {
		t.Logf("HeightPx = %d (expected 128)", result.HeightPx)
	}
	if len(result.RasterRows) == 0 {
		t.Fatal("no raster rows")
	}
}

func TestRenderTextMultiLineFixedWidthCentered(t *testing.T) {
	cfg := TextConfig{
		Lines:      []string{"Short", "A much longer line"},
		FontSize:   0,
		Align:      AlignCenter,
		MaxWidthPx: 400,
	}
	result, err := RenderText(cfg, 128, 128)
	if err != nil {
		t.Fatalf("RenderText error: %v", err)
	}
	if result.WidthPx != 400 {
		t.Errorf("WidthPx = %d, want 400", result.WidthPx)
	}
}

func TestRenderTextMultiLineFixedWidthAlignments(t *testing.T) {
	for _, align := range []Alignment{AlignLeft, AlignCenter, AlignRight} {
		cfg := TextConfig{
			Lines:      []string{"Top", "Bottom"},
			FontSize:   0,
			Align:      align,
			MaxWidthPx: 300,
		}
		result, err := RenderText(cfg, 128, 128)
		if err != nil {
			t.Fatalf("align=%d: RenderText error: %v", align, err)
		}
		if result.WidthPx != 300 {
			t.Errorf("align=%d: WidthPx = %d, want 300", align, result.WidthPx)
		}
	}
}
