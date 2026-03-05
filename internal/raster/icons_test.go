package raster

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		input string
		want  []segment
	}{
		{
			"plain text",
			[]segment{{text: "plain text"}},
		},
		{
			"I :ti-heart: you",
			[]segment{{text: "I "}, {iconName: "ti-heart"}, {text: " you"}},
		},
		{
			":bi-star: rating",
			[]segment{{iconName: "bi-star"}, {text: " rating"}},
		},
		{
			"end :ti-check:",
			[]segment{{text: "end "}, {iconName: "ti-check"}},
		},
		{
			":ti-heart::bi-star:",
			[]segment{{iconName: "ti-heart"}, {iconName: "bi-star"}},
		},
		{
			":xx-unknown: stays literal",
			[]segment{{text: ":xx-unknown: stays literal"}},
		},
		{
			"no icons here",
			[]segment{{text: "no icons here"}},
		},
		{
			// Single word with colons (not a valid prefix) stays literal.
			":nope: text",
			[]segment{{text: ":nope: text"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLine(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseLine(%q) = %d segments, want %d\ngot: %+v", tt.input, len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i].text != tt.want[i].text || got[i].iconName != tt.want[i].iconName {
					t.Errorf("segment[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestHasIcons(t *testing.T) {
	if !hasIcons([]string{"I :ti-heart: Go"}) {
		t.Error("expected hasIcons=true for ti- prefix")
	}
	if !hasIcons([]string{":bi-check: ok"}) {
		t.Error("expected hasIcons=true for bi- prefix")
	}
	if hasIcons([]string{"no icons here"}) {
		t.Error("expected hasIcons=false for plain text")
	}
	if hasIcons([]string{":xx-bogus: icon"}) {
		t.Error("expected hasIcons=false for unknown prefix")
	}
}

func TestSplitIconName(t *testing.T) {
	tests := []struct {
		name       string
		wantPrefix string
		wantIcon   string
		wantOK     bool
	}{
		{"ti-heart", "ti", "heart", true},
		{"bi-check-circle", "bi", "check-circle", true},
		{"nope", "", "", false},
		{"-bad", "", "", false},
		{"a-", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, i, ok := splitIconName(tt.name)
			if p != tt.wantPrefix || i != tt.wantIcon || ok != tt.wantOK {
				t.Errorf("splitIconName(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.name, p, i, ok, tt.wantPrefix, tt.wantIcon, tt.wantOK)
			}
		})
	}
}

func TestIconSVGURL(t *testing.T) {
	url, err := iconSVGURL("ti", "heart")
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://raw.githubusercontent.com/tabler/tabler-icons/main/icons/outline/heart.svg" {
		t.Errorf("unexpected tabler URL: %s", url)
	}

	url, err = iconSVGURL("bi", "check")
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://raw.githubusercontent.com/twbs/icons/main/icons/check.svg" {
		t.Errorf("unexpected bootstrap URL: %s", url)
	}

	_, err = iconSVGURL("xx", "nope")
	if err == nil {
		t.Error("expected error for unknown prefix")
	}
}

func TestRasterizeSVG(t *testing.T) {
	// Minimal valid SVG.
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="24" height="24">
		<circle cx="12" cy="12" r="10" fill="black"/>
	</svg>`)

	img, err := rasterizeSVG(svg, 48)
	if err != nil {
		t.Fatalf("rasterizeSVG: %v", err)
	}
	if img.Bounds().Dx() != 48 || img.Bounds().Dy() != 48 {
		t.Errorf("expected 48x48, got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}

	// Should have some non-white pixels (the circle).
	hasDark := false
	for y := range 48 {
		for x := range 48 {
			r, _, _, _ := img.At(x, y).RGBA()
			if r < 0x8000 {
				hasDark = true
			}
		}
	}
	if !hasDark {
		t.Error("expected dark pixels in rasterized circle")
	}
}

func TestRenderIconToImage(t *testing.T) {
	// Create a small "icon" image with a black center.
	icon := image.NewRGBA(image.Rect(0, 0, 12, 12))
	for y := range 12 {
		for x := range 12 {
			icon.Set(x, y, color.White)
		}
	}
	for y := 3; y < 9; y++ {
		for x := 3; x < 9; x++ {
			icon.Set(x, y, color.Black)
		}
	}

	dst := image.NewRGBA(image.Rect(0, 0, 24, 24))
	for i := range dst.Pix {
		dst.Pix[i] = 255
	}

	w := renderIconToImage(dst, icon, 5, 5)
	if w != 12 {
		t.Errorf("renderIconToImage returned width %d, want 12", w)
	}

	// Check that some pixels are now black.
	hasBlack := false
	for y := 5; y < 17; y++ {
		for x := 5; x < 17; x++ {
			r, _, _, _ := dst.At(x, y).RGBA()
			if r == 0 {
				hasBlack = true
			}
		}
	}
	if !hasBlack {
		t.Error("expected black pixels in destination")
	}
}

func TestFetchIconSVGCache(t *testing.T) {
	// Set up a temp cache dir.
	tmpDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpDir)

	// Pre-populate cache with a test SVG.
	cacheDir := filepath.Join(tmpDir, "ptouch", "icons", "ti")
	os.MkdirAll(cacheDir, 0o755)
	testSVG := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10"/></svg>`)
	os.WriteFile(filepath.Join(cacheDir, "test-icon.svg"), testSVG, 0o644)

	// Should load from cache without network.
	data, err := fetchIconSVG("ti", "test-icon")
	if err != nil {
		t.Fatalf("fetchIconSVG from cache: %v", err)
	}
	if string(data) != string(testSVG) {
		t.Error("cached data mismatch")
	}
}
