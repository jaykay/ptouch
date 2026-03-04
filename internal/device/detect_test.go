package device

import (
	"testing"

	"github.com/jaykay/ptouch/internal/protocol"
)

// makeStatus builds a minimal protocol.Status for testing.
func makeStatus(model byte, mediaWidth byte, mediaType protocol.MediaType) protocol.Status {
	return protocol.Status{
		PrintHeadMark: 0x80,
		Size:          0x20,
		BrotherCode:   'B',
		Model:         model,
		MediaWidth:    mediaWidth,
		MediaType:     mediaType,
	}
}

func TestDetectP750W(t *testing.T) {
	// PT-P750W has PID 0x2062, so model byte = 0x62.
	status := makeStatus(0x62, 24, protocol.MediaLaminated)
	result := Detect(status)

	if result.Model == nil {
		t.Fatal("Detect() Model = nil, want PT-P750W")
	}
	if result.Model.Name != "PT-P750W" {
		t.Errorf("Model.Name = %q, want PT-P750W", result.Model.Name)
	}
	if !result.Model.Flags.Has(protocol.FlagRasterPackBits) {
		t.Error("expected FlagRasterPackBits")
	}
	if !result.Model.Flags.Has(protocol.FlagP700Init) {
		t.Error("expected FlagP700Init")
	}

	if result.Tape == nil {
		t.Fatal("Detect() Tape = nil, want 24mm")
	}
	if result.Tape.Pixels != 128 {
		t.Errorf("Tape.Pixels = %d, want 128", result.Tape.Pixels)
	}
}

func TestDetectP700(t *testing.T) {
	status := makeStatus(0x61, 12, protocol.MediaLaminated)
	result := Detect(status)

	if result.Model == nil {
		t.Fatal("Detect() Model = nil, want PT-P700")
	}
	if result.Model.Name != "PT-P700" {
		t.Errorf("Model.Name = %q, want PT-P700", result.Model.Name)
	}

	if result.Tape == nil {
		t.Fatal("Detect() Tape = nil, want 12mm")
	}
	if result.Tape.Pixels != 76 {
		t.Errorf("Tape.Pixels = %d, want 76", result.Tape.Pixels)
	}
}

func TestDetectUnknownModel(t *testing.T) {
	status := makeStatus(0xFF, 24, protocol.MediaLaminated)
	result := Detect(status)

	if result.Model != nil {
		t.Errorf("Detect() Model = %q, want nil for unknown model", result.Model.Name)
	}
	// Tape should still be detected.
	if result.Tape == nil {
		t.Fatal("Detect() Tape = nil, want 24mm even with unknown model")
	}
}

func TestDetectNoMedia(t *testing.T) {
	status := makeStatus(0x62, 0, protocol.MediaNone)
	result := Detect(status)

	if result.Model == nil {
		t.Fatal("Detect() Model = nil, want PT-P750W")
	}
	if result.Tape != nil {
		t.Errorf("Detect() Tape = %+v, want nil (no media)", result.Tape)
	}
}

func TestDetectUnknownTapeWidth(t *testing.T) {
	status := makeStatus(0x62, 99, protocol.MediaLaminated)
	result := Detect(status)

	if result.Model == nil {
		t.Fatal("Detect() Model = nil")
	}
	if result.Tape != nil {
		t.Errorf("Detect() Tape = %+v, want nil for unknown width 99mm", result.Tape)
	}
}

func TestDetectSkipsPLite(t *testing.T) {
	// PID 0x2065 is PT-P750W in P-Lite mode.
	status := makeStatus(0x65, 24, protocol.MediaLaminated)
	result := Detect(status)

	if result.Model != nil {
		t.Errorf("Detect() should not match P-Lite entry, got %q (0x%04X)",
			result.Model.Name, result.Model.ProductID)
	}
}

func TestDetectVariousTapes(t *testing.T) {
	tests := []struct {
		widthMM    byte
		wantPixels int
	}{
		{6, 32},
		{9, 52},
		{12, 76},
		{18, 120},
		{24, 128},
	}
	for _, tt := range tests {
		status := makeStatus(0x62, tt.widthMM, protocol.MediaLaminated)
		result := Detect(status)
		if result.Tape == nil {
			t.Fatalf("Detect() Tape = nil for %dmm", tt.widthMM)
		}
		if result.Tape.Pixels != tt.wantPixels {
			t.Errorf("Detect() %dmm: Tape.Pixels = %d, want %d",
				tt.widthMM, result.Tape.Pixels, tt.wantPixels)
		}
	}
}
