package raster

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// Icon providers with their SVG base URLs.
const (
	PrefixTabler    = "ti"
	PrefixBootstrap = "bi"

	tablerURL    = "https://raw.githubusercontent.com/tabler/tabler-icons/main/icons/outline/%s.svg"
	bootstrapURL = "https://raw.githubusercontent.com/twbs/icons/main/icons/%s.svg"
)

// iconPattern matches :prefix-name: placeholders in text.
// Supports :ti-heart:, :bi-check:, etc.
var iconPattern = regexp.MustCompile(`:([a-z][a-z0-9]*-[a-z][a-z0-9-]*):`)

// segment represents a piece of a text line — either plain text or an icon.
type segment struct {
	text     string // non-empty for text segments
	iconName string // non-empty for icon segments (full shortcode, e.g. "ti-heart")
}

// parseLine splits a line into text and icon segments.
func parseLine(line string) []segment {
	var segs []segment
	last := 0
	for _, loc := range iconPattern.FindAllStringIndex(line, -1) {
		name := line[loc[0]+1 : loc[1]-1]
		prefix, _, ok := splitIconName(name)
		if !ok || (prefix != PrefixTabler && prefix != PrefixBootstrap) {
			continue
		}
		if loc[0] > last {
			segs = append(segs, segment{text: line[last:loc[0]]})
		}
		segs = append(segs, segment{iconName: name})
		last = loc[1]
	}
	if last < len(line) {
		segs = append(segs, segment{text: line[last:]})
	}
	if len(segs) == 0 {
		segs = append(segs, segment{text: line})
	}
	return segs
}

// hasIcons returns true if any line contains a recognized :prefix-name: shortcode.
func hasIcons(lines []string) bool {
	for _, line := range lines {
		for _, loc := range iconPattern.FindAllStringIndex(line, -1) {
			name := line[loc[0]+1 : loc[1]-1]
			prefix, _, ok := splitIconName(name)
			if ok && (prefix == PrefixTabler || prefix == PrefixBootstrap) {
				return true
			}
		}
	}
	return false
}

// splitIconName splits "ti-heart" into ("ti", "heart", true).
func splitIconName(name string) (prefix, icon string, ok bool) {
	idx := strings.IndexByte(name, '-')
	if idx < 1 || idx >= len(name)-1 {
		return "", "", false
	}
	return name[:idx], name[idx+1:], true
}

// iconSVGURL returns the raw SVG URL for a given provider prefix and icon name.
func iconSVGURL(prefix, icon string) (string, error) {
	switch prefix {
	case PrefixTabler:
		return fmt.Sprintf(tablerURL, icon), nil
	case PrefixBootstrap:
		return fmt.Sprintf(bootstrapURL, icon), nil
	default:
		return "", fmt.Errorf("unknown icon provider %q", prefix)
	}
}

// iconCacheDir returns the cache directory for icons.
func iconCacheDir() string {
	dir := os.Getenv("XDG_CACHE_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".cache")
	}
	return filepath.Join(dir, "ptouch", "icons")
}

// iconCachePath returns the file path for a cached icon SVG.
func iconCachePath(prefix, icon string) string {
	return filepath.Join(iconCacheDir(), prefix, icon+".svg")
}

// fetchIconSVG returns the SVG data for an icon, using cache if available.
func fetchIconSVG(prefix, icon string) ([]byte, error) {
	path := iconCachePath(prefix, icon)

	// Check cache first.
	if data, err := os.ReadFile(path); err == nil {
		return data, nil
	}

	url, err := iconSVGURL(prefix, icon)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download icon %s-%s: %w", prefix, icon, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("icon %q not found in %s library", icon, prefix)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download icon %s-%s: HTTP %s", prefix, icon, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read icon %s-%s: %w", prefix, icon, err)
	}

	// Cache for next time (best-effort).
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err == nil {
		_ = os.WriteFile(path, data, 0o644)
	}

	return data, nil
}

// rasterizeSVG renders SVG data to an RGBA image of the given size.
func rasterizeSVG(svgData []byte, size int) (*image.RGBA, error) {
	// Replace currentColor with black — SVG icon libraries use currentColor
	// for stroke/fill so the icon inherits the parent's text color.
	svgStr := strings.ReplaceAll(string(svgData), "currentColor", "black")

	icon, err := oksvg.ReadIconStream(strings.NewReader(svgStr))
	if err != nil {
		return nil, fmt.Errorf("parse SVG: %w", err)
	}

	icon.SetTarget(0, 0, float64(size), float64(size))

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	// Fill with white background.
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

	scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
	dasher := rasterx.NewDasher(size, size, scanner)
	icon.Draw(dasher, 1.0)

	return img, nil
}

// resolveIcon fetches (or loads from cache) and rasterizes an icon at the given size.
// Returns the RGBA image and its width. On error returns nil.
func resolveIcon(fullName string, size int) *image.RGBA {
	prefix, icon, ok := splitIconName(fullName)
	if !ok {
		return nil
	}

	svgData, err := fetchIconSVG(prefix, icon)
	if err != nil {
		return nil
	}

	img, err := rasterizeSVG(svgData, size)
	if err != nil {
		return nil
	}

	return img
}

// renderIconToImage draws a resolved icon image onto a destination canvas at (x, y).
// The icon is drawn as black pixels only (threshold-based), preserving the white background.
// Returns the icon width in pixels.
func renderIconToImage(dst *image.RGBA, iconImg *image.RGBA, x, y int) int {
	if iconImg == nil {
		return 0
	}
	size := iconImg.Bounds().Dx()

	for py := range size {
		for px := range size {
			r, g, b, _ := iconImg.At(px, py).RGBA()
			// Use luminance threshold to convert to black/white.
			lum := (299*r + 587*g + 114*b) / 1000
			if uint8(lum>>8) < 160 {
				dst.Set(x+px, y+py, color.Black)
			}
		}
	}
	return size
}
