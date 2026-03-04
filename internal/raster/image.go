package raster

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"golang.org/x/image/draw"
)

// LoadImage loads an image file, scales it to fit the tape height,
// converts to monochrome, rotates for tape orientation, and returns
// raster rows ready for printing.
//
// Supported formats: PNG, JPEG, GIF.
// tapePixels is the printable height (from Tape.Pixels).
// maxPixels is the printhead width for row padding (from Model.MaxPixels).
// marginPx is the margin in pixels on each side (0 = edge-to-edge).
func LoadImage(path string, tapePixels, maxPixels, marginPx int) (*RenderResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("raster: open image: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("raster: decode image: %w", err)
	}

	return RenderImage(img, tapePixels, maxPixels, marginPx), nil
}

// RenderImage converts an already-decoded image to a print-ready RenderResult.
// The image is scaled to fit tapePixels height (maintaining aspect ratio),
// converted to 1-bit monochrome, and transposed for tape orientation.
// marginPx adds padding on all sides (0 = no margin).
func RenderImage(img image.Image, tapePixels, maxPixels, marginPx int) *RenderResult {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	// Scale to fit tape height minus margins, maintaining aspect ratio.
	innerH := tapePixels - 2*marginPx
	if innerH < 1 {
		innerH = 1
	}
	dstW := srcW * innerH / srcH
	if dstW == 0 {
		dstW = 1
	}

	scaled := scaleImage(img, dstW, innerH)
	inner := FromImage(scaled, 127)

	// Place on canvas with margins.
	canvasW := dstW + 2*marginPx
	canvasH := tapePixels
	bm := inner
	if marginPx > 0 {
		bm = NewBitmap(canvasW, canvasH)
		for y := 0; y < inner.Height; y++ {
			for x := 0; x < inner.Width; x++ {
				if inner.GetPixel(x, y) {
					bm.SetPixel(x+marginPx, y+marginPx, true)
				}
			}
		}
	}

	// Transpose for tape orientation and center on printhead.
	rotated := bm.Transpose()
	rotated = rotated.PadCenter(maxPixels)

	return &RenderResult{
		Bitmap:     rotated,
		Preview:    bm,
		RasterRows: rotated.ToRasterRows(maxPixels),
		WidthPx:    canvasW,
		HeightPx:   canvasH,
	}
}

// scaleImage scales an image to the given dimensions using bilinear interpolation.
func scaleImage(src image.Image, dstW, dstH int) image.Image {
	srcBounds := src.Bounds()
	if srcBounds.Dx() == dstW && srcBounds.Dy() == dstH {
		return src // no scaling needed
	}
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, srcBounds, draw.Over, nil)
	return dst
}
