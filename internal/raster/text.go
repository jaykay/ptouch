package raster

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	xfont "golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Alignment controls horizontal text alignment.
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

// TextConfig configures text rendering.
type TextConfig struct {
	Lines      []string  // text lines to render
	FontData   []byte    // TTF/OTF data (nil = use embedded Go font)
	FontSize   float64   // size in points (0 = auto-fit)
	Bold       bool      // use bold variant of embedded font
	Align      Alignment // horizontal alignment (default: center)
	DPI        int       // dots per inch (0 = 180)
	MaxWidthPx int       // fixed label width in pixels (0 = auto from text width)
}

// RenderResult holds the output of a rendering operation.
type RenderResult struct {
	Bitmap     *Bitmap  // transposed bitmap, ready for printing
	Preview    *Bitmap  // human-readable bitmap (before transpose), for preview
	RasterRows [][]byte // pre-split rows for SendRasterLine
	WidthPx    int      // label width in pixels (before transpose)
	HeightPx   int      // label height in pixels (before transpose = tape pixels)
}

// RenderText renders text lines to a 1-bit bitmap suitable for printing.
// tapePixels is the printable pixel height (from Tape.Pixels).
// maxPixels is the printhead width for row padding (from Model.MaxPixels).
func RenderText(cfg TextConfig, tapePixels, maxPixels int) (*RenderResult, error) {
	if len(cfg.Lines) == 0 {
		return nil, fmt.Errorf("raster: no text lines to render")
	}
	dpi := cfg.DPI
	if dpi == 0 {
		dpi = 180
	}

	fontObj, err := loadFont(cfg)
	if err != nil {
		return nil, err
	}

	// Horizontal padding: ~10% of tape height on each side.
	padPx := tapePixels / 10
	if padPx < 2 {
		padPx = 2
	}

	fontSize := cfg.FontSize
	if fontSize == 0 {
		maxW := cfg.MaxWidthPx
		if maxW > 0 {
			maxW -= 2 * padPx // reserve padding
		}
		fontSize = autoFitSize(fontObj, cfg.Lines, tapePixels, maxW, dpi)
	}

	face, err := opentype.NewFace(fontObj, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     float64(dpi),
		Hinting: xfont.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("raster: create font face: %w", err)
	}
	defer face.Close()

	// Measure all lines (accounting for icons).
	metrics := face.Metrics()
	lineHeight := metrics.Ascent + metrics.Descent
	lineSpacingPx := int(math.Ceil(float64(lineHeight) * 1.2 / 64.0))
	lineHeightPx := int(math.Ceil(float64(lineHeight) / 64.0))
	ascentPx := int(math.Ceil(float64(metrics.Ascent) / 64.0))
	iconSize := lineHeightPx

	useIcons := hasIcons(cfg.Lines)
	iconCache := make(map[string]*image.RGBA)

	var maxTextWidth int
	lineWidths := make([]int, len(cfg.Lines))
	lineSegs := make([][]segment, len(cfg.Lines))
	for i, line := range cfg.Lines {
		if useIcons {
			segs := parseLine(line)
			lineSegs[i] = segs
			w := measureSegments(face, segs, iconSize, iconCache)
			lineWidths[i] = w
		} else {
			adv := xfont.MeasureString(face, line)
			lineWidths[i] = adv.Ceil()
		}
		if lineWidths[i] > maxTextWidth {
			maxTextWidth = lineWidths[i]
		}
	}

	// Canvas width: fixed if MaxWidthPx is set, otherwise natural text width + padding.
	canvasWidth := maxTextWidth + 2*padPx
	if cfg.MaxWidthPx > 0 {
		canvasWidth = cfg.MaxWidthPx
	}
	if canvasWidth < 1 {
		canvasWidth = 1
	}

	// Actual text block height: (N-1) gaps between lines + one line height.
	textBlockHeight := lineSpacingPx*(len(cfg.Lines)-1) + lineHeightPx
	canvasHeight := tapePixels
	if canvasHeight < textBlockHeight {
		canvasHeight = textBlockHeight
	}

	// Create RGBA canvas.
	canvas := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

	// Vertical centering: start y so text block is centered on tape.
	startY := (canvasHeight - textBlockHeight) / 2

	drawer := &xfont.Drawer{
		Dst:  canvas,
		Src:  image.NewUniform(color.Black),
		Face: face,
	}

	for i, line := range cfg.Lines {
		var xOffset int
		switch cfg.Align {
		case AlignCenter:
			xOffset = (canvasWidth - lineWidths[i]) / 2
		case AlignRight:
			xOffset = canvasWidth - lineWidths[i] - padPx
		default:
			xOffset = padPx
		}

		y := startY + i*lineSpacingPx + ascentPx

		if useIcons && lineSegs[i] != nil {
			drawSegments(canvas, drawer, lineSegs[i], xOffset, y, iconSize, iconCache)
		} else {
			drawer.Dot = fixed.Point26_6{
				X: fixed.I(xOffset),
				Y: fixed.I(y),
			}
			drawer.DrawString(line)
		}
	}

	// Convert to 1-bit bitmap.
	bm := FromImage(canvas, 127)

	// Transpose for tape orientation.
	rotated := bm.Transpose()

	// Center on printhead — narrower tapes don't occupy pixel 0 of the
	// printhead, so the content must be offset to the tape's position.
	rotated = rotated.PadCenter(maxPixels)

	// Generate raster rows.
	rasterRows := rotated.ToRasterRows(maxPixels)

	return &RenderResult{
		Bitmap:     rotated,
		Preview:    bm,
		RasterRows: rasterRows,
		WidthPx:    canvasWidth,
		HeightPx:   canvasHeight,
	}, nil
}

// loadFont loads a font from the config, falling back to embedded Go fonts.
func loadFont(cfg TextConfig) (*opentype.Font, error) {
	var data []byte
	if cfg.FontData != nil {
		data = cfg.FontData
	} else if cfg.Bold {
		data = gobold.TTF
	} else {
		data = goregular.TTF
	}
	f, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("raster: parse font: %w", err)
	}
	return f, nil
}

// autoFitSize finds the largest font size (in points) that fits the text
// within the given tape height and optional max width.
func autoFitSize(fontObj *opentype.Font, lines []string, maxHeightPx, maxWidthPx int, dpi int) float64 {
	lo := 1.0
	hi := 200.0

	for i := 0; i < 20; i++ { // binary search iterations
		mid := (lo + hi) / 2.0
		h, w := measureText(fontObj, lines, mid, dpi)
		fits := h <= maxHeightPx
		if maxWidthPx > 0 {
			fits = fits && w <= maxWidthPx
		}
		if fits {
			lo = mid
		} else {
			hi = mid
		}
	}
	return math.Floor(lo)
}

// measureText returns the total height and max width in pixels for the given
// lines at the given font size.
func measureText(fontObj *opentype.Font, lines []string, sizePt float64, dpi int) (height, width int) {
	face, err := opentype.NewFace(fontObj, &opentype.FaceOptions{
		Size:    sizePt,
		DPI:     float64(dpi),
		Hinting: xfont.HintingFull,
	})
	if err != nil {
		return 9999, 9999
	}
	defer face.Close()

	metrics := face.Metrics()
	lineHeight := metrics.Ascent + metrics.Descent
	lineSpacingPx := int(math.Ceil(float64(lineHeight) * 1.2 / 64.0))
	lineHeightPx := int(math.Ceil(float64(lineHeight) / 64.0))

	height = lineSpacingPx*(len(lines)-1) + lineHeightPx

	// Dummy cache for measuring — icons resolved here are discarded.
	measureCache := make(map[string]*image.RGBA)
	for _, line := range lines {
		var w int
		if hasIcons([]string{line}) {
			w = measureSegments(face, parseLine(line), lineHeightPx, measureCache)
		} else {
			w = xfont.MeasureString(face, line).Ceil()
		}
		if w > width {
			width = w
		}
	}
	return height, width
}

// measureSegments returns the total width of a parsed line with mixed text/icons.
// Icon images are resolved and cached in iconCache for reuse during drawing.
func measureSegments(face xfont.Face, segs []segment, iconSize int, iconCache map[string]*image.RGBA) int {
	w := 0
	for _, s := range segs {
		if s.iconName != "" {
			img := resolveIcon(s.iconName, iconSize)
			if img != nil {
				iconCache[s.iconName] = img
				w += img.Bounds().Dx()
			}
		} else {
			w += xfont.MeasureString(face, s.text).Ceil()
		}
	}
	return w
}

// drawSegments renders a parsed line with mixed text and icons onto the canvas.
func drawSegments(dst *image.RGBA, drawer *xfont.Drawer, segs []segment, x, yBaseline, iconSize int, iconCache map[string]*image.RGBA) {
	curX := x
	for _, s := range segs {
		if s.iconName != "" {
			img := iconCache[s.iconName]
			if img == nil {
				continue
			}
			// Vertically center icon on the text baseline.
			iconY := yBaseline - iconSize + iconSize/6
			w := renderIconToImage(dst, img, curX, iconY)
			curX += w
		} else {
			drawer.Dot = fixed.Point26_6{
				X: fixed.I(curX),
				Y: fixed.I(yBaseline),
			}
			drawer.DrawString(s.text)
			curX = drawer.Dot.X.Ceil()
		}
	}
}
