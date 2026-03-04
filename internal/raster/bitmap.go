// Package raster handles text and image rendering to monochrome bitmaps
// suitable for the P-Touch raster protocol.
package raster

import (
	"image"
	"image/color"
	"image/png"
	"io"
)

// Bitmap is a 1-bit monochrome image. Pixels are packed MSB-first:
// bit 7 of byte 0 is the leftmost pixel. 1 = black, 0 = white.
type Bitmap struct {
	Width  int
	Height int
	Stride int    // bytes per row = (Width+7)/8
	Data   []byte // row-major packed pixel data
}

// NewBitmap creates a white (all zeros) bitmap of the given dimensions.
func NewBitmap(w, h int) *Bitmap {
	stride := (w + 7) / 8
	return &Bitmap{
		Width:  w,
		Height: h,
		Stride: stride,
		Data:   make([]byte, stride*h),
	}
}

// SetPixel sets the pixel at (x, y) to black (true) or white (false).
func (b *Bitmap) SetPixel(x, y int, black bool) {
	if x < 0 || x >= b.Width || y < 0 || y >= b.Height {
		return
	}
	byteIdx := y*b.Stride + x/8
	bitIdx := uint(7 - x%8) // MSB first
	if black {
		b.Data[byteIdx] |= 1 << bitIdx
	} else {
		b.Data[byteIdx] &^= 1 << bitIdx
	}
}

// GetPixel returns true if the pixel at (x, y) is black.
func (b *Bitmap) GetPixel(x, y int) bool {
	if x < 0 || x >= b.Width || y < 0 || y >= b.Height {
		return false
	}
	byteIdx := y*b.Stride + x/8
	bitIdx := uint(7 - x%8)
	return b.Data[byteIdx]&(1<<bitIdx) != 0
}

// Transpose returns a transposed bitmap (W×H becomes H×W).
// Maps (x, y) → (y, x) so that:
//   - each column of the original becomes a raster row (tape feed direction)
//   - top of the original (y=0) maps to MSB of byte 0 (top edge of tape)
// This produces correctly oriented, non-mirrored output on P-Touch printers.
func (b *Bitmap) Transpose() *Bitmap {
	dst := NewBitmap(b.Height, b.Width)
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			if b.GetPixel(x, y) {
				dst.SetPixel(y, x, true)
			}
		}
	}
	return dst
}

// PadCenter returns a new bitmap of targetWidth, with the receiver's content
// centered horizontally. Used after transpose to align tape content on the
// printhead — narrower tapes don't start at pixel 0 of the printhead.
func (b *Bitmap) PadCenter(targetWidth int) *Bitmap {
	if targetWidth <= b.Width {
		return b
	}
	dst := NewBitmap(targetWidth, b.Height)
	offset := (targetWidth - b.Width) / 2
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			if b.GetPixel(x, y) {
				dst.SetPixel(x+offset, y, true)
			}
		}
	}
	return dst
}

// ToRasterRows splits the bitmap into byte rows suitable for SendRasterLine.
// Each row is padded to maxPixels/8 bytes. Returns one []byte per bitmap row.
func (b *Bitmap) ToRasterRows(maxPixels int) [][]byte {
	rowBytes := maxPixels / 8
	if rowBytes < b.Stride {
		rowBytes = b.Stride
	}
	rows := make([][]byte, b.Height)
	for y := 0; y < b.Height; y++ {
		row := make([]byte, rowBytes)
		srcStart := y * b.Stride
		srcEnd := srcStart + b.Stride
		if srcEnd > len(b.Data) {
			srcEnd = len(b.Data)
		}
		copy(row, b.Data[srcStart:srcEnd])
		rows[y] = row
	}
	return rows
}

// ToPNG writes the bitmap as a black-and-white PNG.
func (b *Bitmap) ToPNG(w io.Writer) error {
	img := image.NewGray(image.Rect(0, 0, b.Width, b.Height))
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			if b.GetPixel(x, y) {
				img.SetGray(x, y, color.Gray{Y: 0})
			} else {
				img.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
	return png.Encode(w, img)
}

// FromImage converts any image.Image to a 1-bit Bitmap using the given
// luminance threshold (0–255). Pixels darker than threshold become black.
// Transparent pixels are composited onto a white background.
func FromImage(img image.Image, threshold uint8) *Bitmap {
	bounds := img.Bounds()
	bm := NewBitmap(bounds.Dx(), bounds.Dy())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			// Composite onto white: out = src*alpha + white*(1-alpha)
			// RGBA() returns pre-multiplied 16-bit values; a=0xFFFF means opaque.
			if a == 0 {
				continue // fully transparent = white background
			}
			if a < 0xFFFF {
				// Blend with white (0xFFFF).
				inv := 0xFFFF - a
				r = r + inv
				g = g + inv
				b = b + inv
			}
			// Standard luminance: 0.299R + 0.587G + 0.114B
			lum := (299*r + 587*g + 114*b) / 1000
			// RGBA returns 16-bit values; threshold is 8-bit.
			if uint8(lum>>8) < threshold {
				bm.SetPixel(x-bounds.Min.X, y-bounds.Min.Y, true)
			}
		}
	}
	return bm
}
