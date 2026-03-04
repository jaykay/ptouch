package main

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/jaykay/ptouch/internal/device"
	"github.com/jaykay/ptouch/internal/network"
	"github.com/jaykay/ptouch/internal/protocol"
	"github.com/jaykay/ptouch/internal/raster"
	"github.com/spf13/cobra"
)

var (
	flagText     []string
	flagImage    string
	flagFont     string
	flagFontSize float64
	flagBold     bool
	flagAlign    string
	flagWidth    float64
	flagCopies   int
	flagCut      bool
	flagNoCut    bool
	flagChain    bool
	flagPreview  string
	flagMargin   float64
)

var printCmd = &cobra.Command{
	Use:   "print",
	Short: "Print a text or image label",
	RunE:  runPrint,
}

func init() {
	printCmd.Flags().StringArrayVar(&flagText, "text", nil, "text line (repeatable)")
	printCmd.Flags().StringVar(&flagImage, "image", "", "image file to print (PNG/JPEG/GIF)")
	printCmd.Flags().StringVar(&flagFont, "font", "", "custom TTF/OTF font file")
	printCmd.Flags().Float64Var(&flagFontSize, "fontsize", 0, "font size in points (0 = auto-fit)")
	printCmd.Flags().BoolVar(&flagBold, "bold", false, "use bold font")
	printCmd.Flags().StringVar(&flagAlign, "align", "center", "text alignment: left|center|right")
	printCmd.Flags().Float64Var(&flagWidth, "width", 0, "label width in mm (0 = auto from text)")
	printCmd.Flags().IntVar(&flagCopies, "copies", 1, "number of copies")
	printCmd.Flags().BoolVar(&flagCut, "cut", true, "cut tape after printing")
	printCmd.Flags().BoolVar(&flagNoCut, "no-cut", false, "don't cut tape (chain print)")
	printCmd.Flags().BoolVar(&flagChain, "chain", false, "chain print (no cut between labels)")
	printCmd.Flags().StringVar(&flagPreview, "preview", "", "save preview PNG instead of printing")
	printCmd.Flags().Float64Var(&flagMargin, "margin", 0, "image margin in mm (default 0, text always has a built-in margin)")
	rootCmd.AddCommand(printCmd)
}

func runPrint(cmd *cobra.Command, args []string) error {
	hasText := len(flagText) > 0
	hasImage := flagImage != ""
	if !hasText && !hasImage {
		return fmt.Errorf("provide --text or --image")
	}
	if hasText && hasImage {
		return fmt.Errorf("--text and --image are mutually exclusive")
	}

	// Determine cut behavior.
	eject := flagCut && !flagNoCut && !flagChain

	// For preview mode, we need model/tape info but no connection.
	if flagPreview != "" {
		return runPreview(cmd)
	}

	// Connect to printer.
	if flagHost == "" {
		return fmt.Errorf("--host is required (or use --preview for offline rendering)")
	}

	// Resolve model.
	model := device.LookupByName(flagModel)
	if model == nil {
		return fmt.Errorf("unknown model %q — use --model with a supported model name", flagModel)
	}
	debugf("model: %s (maxPixels=%d, DPI=%d, flags=0x%02X)", model.Name, model.MaxPixels, model.DPI, model.Flags)

	// Check printer status before printing.
	ws, wsErr := fetchWebStatus(flagHost)
	if wsErr == nil {
		debugf("printer status: %s, media: %s (%s)", ws.DeviceStatus, ws.MediaType, ws.MediaStatus)
		if hint := statusHint(ws); hint != "" {
			return fmt.Errorf("printer not ready: %s", hint)
		}
	} else {
		debugf("web status unavailable: %v", wsErr)
	}

	// Resolve tape width.
	tape, err := resolveTape(model, ws, wsErr)
	if err != nil {
		return err
	}
	debugf("tape: %dmm (%d printable pixels)", tape.WidthMM, tape.Pixels)

	// Render before connecting (fail fast on bad input).
	result, err := renderLabel(model, tape)
	if err != nil {
		return err
	}
	debugf("rendered: %dx%d px, %d raster rows, row size %d bytes",
		result.WidthPx, result.HeightPx, len(result.RasterRows), len(result.RasterRows[0]))

	// Connect without health check — P-Touch printers don't respond
	// to status requests over TCP.
	debugf("connecting to %s (timeout %s)", flagHost, flagTimeout)
	conn, _, err := network.Dial(flagHost,
		network.WithReadWriteTimeout(flagTimeout),
		network.WithoutHealthCheck(),
	)
	if err != nil {
		return fmt.Errorf("connect: %w\n\nHint: make sure the printer is turned on and reachable at %s", err, flagHost)
	}
	defer conn.Close()

	fmt.Fprintf(cmd.OutOrStdout(), "Printing on %s with %dmm tape (%d rows)...\n",
		model.Name, tape.WidthMM, len(result.RasterRows))

	// Print.
	usePackBits := model.Flags.Has(protocol.FlagRasterPackBits)
	debugf("compression: PackBits=%v, eject=%v, copies=%d", usePackBits, eject, flagCopies)

	sess := protocol.NewSession(conn, model.Flags)
	for copy := 0; copy < flagCopies; copy++ {
		if err := printLabel(sess, model, tape, result, eject, copy == flagCopies-1); err != nil {
			return fmt.Errorf("print (copy %d): %w", copy+1, err)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Done (%d %s printed).\n", flagCopies, pluralize("copy", "copies", flagCopies))
	return nil
}

// statusHint returns an actionable error message if the printer is not ready,
// or empty string if it's OK.
func statusHint(ws *webStatus) string {
	if ws == nil {
		return ""
	}
	ds := strings.ToUpper(ws.DeviceStatus)
	ms := strings.ToUpper(ws.MediaStatus)

	switch {
	case strings.Contains(ds, "COVER OPEN"), strings.Contains(ds, "COVER IS OPEN"):
		return "cover is open — close the tape compartment cover"
	case (strings.Contains(ms, "EMPTY") && !strings.Contains(ms, "NOT EMPTY")), strings.Contains(ms, "NO MEDIA"):
		return "no tape loaded — insert a tape cassette"
	case strings.Contains(ds, "CUTTER"), strings.Contains(ds, "JAM"):
		return "cutter jam — open the cover, remove jammed tape, and close again"
	case strings.Contains(ds, "COOLING"), strings.Contains(ds, "OVERHEAT"):
		return "printer is overheated — wait for it to cool down"
	}
	return ""
}

// resolveTape determines the tape spec from flags or the printer's web interface.
func resolveTape(model *device.Model, ws *webStatus, wsErr error) (*device.Tape, error) {
	if flagTape > 0 {
		t := device.LookupTape(flagTape)
		if t == nil {
			return nil, fmt.Errorf("unsupported tape width %dmm (supported: 4, 6, 9, 12, 18, 24, 36)", flagTape)
		}
		debugf("tape: using --tape %dmm", flagTape)
		return t, nil
	}

	// Try to detect tape width from web interface.
	if wsErr == nil && ws != nil && ws.TapeWidthMM > 0 {
		if t := device.LookupTape(ws.TapeWidthMM); t != nil {
			debugf("tape: detected %dmm from web interface", ws.TapeWidthMM)
			return t, nil
		}
	}

	// Fall back to largest tape the model supports.
	if t := defaultTapeForModel(model); t != nil {
		debugf("tape: falling back to model default %dmm", t.WidthMM)
		return t, nil
	}

	return nil, fmt.Errorf("could not determine tape width — use --tape (e.g. --tape 24)")
}

// defaultTapeForModel returns the largest tape that fits the model's printhead.
func defaultTapeForModel(model *device.Model) *device.Tape {
	var best *device.Tape
	for _, t := range device.Tapes() {
		if t.Pixels <= model.MaxPixels {
			tc := t
			best = &tc
		}
	}
	return best
}

func runPreview(cmd *cobra.Command) error {
	model := device.LookupByName(flagModel)
	if model == nil {
		model = &device.Model{Name: "preview", MaxPixels: 128, DPI: 180}
	}

	var tape *device.Tape
	if flagTape > 0 {
		tape = device.LookupTape(flagTape)
	}
	if tape == nil && flagHost != "" {
		if ws, err := fetchWebStatus(flagHost); err == nil && ws.TapeWidthMM > 0 {
			tape = device.LookupTape(ws.TapeWidthMM)
			if tape != nil {
				debugf("preview: detected %dmm tape from printer", ws.TapeWidthMM)
			}
		}
	}
	if tape == nil {
		tape = defaultTapeForModel(model)
	}
	if tape == nil {
		tape = &device.Tape{WidthMM: 24, Pixels: 128, MarginMM: 3.0}
	}

	result, err := renderLabel(model, tape)
	if err != nil {
		return err
	}

	f, err := os.Create(flagPreview)
	if err != nil {
		return fmt.Errorf("create preview file: %w", err)
	}
	defer f.Close()

	preview := result.Preview
	if preview == nil {
		preview = result.Bitmap
	}
	if err := preview.ToPNG(f); err != nil {
		return fmt.Errorf("write preview PNG: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Preview saved to %s (%dx%d px)\n",
		flagPreview, preview.Width, preview.Height)
	return nil
}

func renderLabel(model *device.Model, tape *device.Tape) (*raster.RenderResult, error) {
	if len(flagText) > 0 {
		return renderText(tape.Pixels, model.MaxPixels, model.DPI)
	}
	marginPx := 0
	if flagMargin > 0 {
		dpi := model.DPI
		if dpi == 0 {
			dpi = 180
		}
		marginPx = int(math.Round(flagMargin * float64(dpi) / 25.4))
		debugf("image margin: %.1fmm = %dpx", flagMargin, marginPx)
	}
	return raster.LoadImage(flagImage, tape.Pixels, model.MaxPixels, marginPx)
}

func renderText(tapePixels, maxPixels, dpi int) (*raster.RenderResult, error) {
	var fontData []byte
	if flagFont != "" {
		data, err := os.ReadFile(flagFont)
		if err != nil {
			return nil, fmt.Errorf("read font file: %w", err)
		}
		fontData = data
	}

	align := raster.AlignCenter
	switch strings.ToLower(flagAlign) {
	case "left":
		align = raster.AlignLeft
	case "right":
		align = raster.AlignRight
	case "center":
		// default
	default:
		return nil, fmt.Errorf("invalid alignment %q (use left, center, or right)", flagAlign)
	}

	// Convert --width mm to pixels.
	var maxWidthPx int
	if flagWidth > 0 {
		if dpi == 0 {
			dpi = 180
		}
		maxWidthPx = int(math.Round(flagWidth * float64(dpi) / 25.4))
		debugf("label width: %.1fmm = %dpx at %d DPI", flagWidth, maxWidthPx, dpi)
	}

	cfg := raster.TextConfig{
		Lines:      flagText,
		FontData:   fontData,
		FontSize:   flagFontSize,
		Bold:       flagBold,
		Align:      align,
		MaxWidthPx: maxWidthPx,
	}

	return raster.RenderText(cfg, tapePixels, maxPixels)
}

func printLabel(sess *protocol.Session, model *device.Model, tape *device.Tape, result *raster.RenderResult, eject bool, isLast bool) error {
	if err := sess.Init(); err != nil {
		return fmt.Errorf("init: %w", err)
	}

	// Switch to raster mode (needed before media info on P700 series).
	if err := sess.StartRaster(); err != nil {
		return fmt.Errorf("start raster: %w", err)
	}

	if err := sess.SetMediaInfo(
		0x00,                // media type (auto)
		byte(tape.WidthMM), // width mm
		0,                   // length (0 = continuous)
		uint32(len(result.RasterRows)),
	); err != nil {
		return fmt.Errorf("set media info: %w", err)
	}

	if err := sess.SetCompression(model.Flags.Has(protocol.FlagRasterPackBits)); err != nil {
		return fmt.Errorf("set compression: %w", err)
	}

	if err := sess.SetPrecut(false); err != nil {
		return fmt.Errorf("set precut: %w", err)
	}

	for i, row := range result.RasterRows {
		if err := sess.SendRasterLine(row); err != nil {
			return fmt.Errorf("send row %d: %w", i, err)
		}
	}

	// Eject (cut) on the last copy, or if eject is requested for each.
	shouldEject := eject || isLast
	if err := sess.EndPage(shouldEject); err != nil {
		return fmt.Errorf("end page: %w", err)
	}

	return nil
}

func pluralize(singular, plural string, n int) string {
	if n == 1 {
		return singular
	}
	return plural
}
