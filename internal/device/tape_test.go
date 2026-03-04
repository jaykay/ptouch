package device

import "testing"

func TestLookupTape(t *testing.T) {
	tests := []struct {
		widthMM    int
		wantPixels int
		wantNil    bool
	}{
		{4, 24, false},   // 3.5mm
		{6, 32, false},
		{9, 52, false},
		{12, 76, false},
		{18, 120, false},
		{24, 128, false},
		{36, 192, false},
		{0, 0, true},     // unknown
		{10, 0, true},    // unknown
		{48, 0, true},    // unknown
	}
	for _, tt := range tests {
		tape := LookupTape(tt.widthMM)
		if tt.wantNil {
			if tape != nil {
				t.Errorf("LookupTape(%d) = %+v, want nil", tt.widthMM, tape)
			}
			continue
		}
		if tape == nil {
			t.Fatalf("LookupTape(%d) = nil, want pixels=%d", tt.widthMM, tt.wantPixels)
		}
		if tape.Pixels != tt.wantPixels {
			t.Errorf("LookupTape(%d).Pixels = %d, want %d", tt.widthMM, tape.Pixels, tt.wantPixels)
		}
		if tape.WidthMM != tt.widthMM {
			t.Errorf("LookupTape(%d).WidthMM = %d", tt.widthMM, tape.WidthMM)
		}
	}
}

func TestTapes(t *testing.T) {
	all := Tapes()
	if len(all) != 7 {
		t.Fatalf("Tapes() returned %d entries, want 7", len(all))
	}
	// Verify it's a copy, not a reference to the internal slice.
	all[0].Pixels = 9999
	if LookupTape(4).Pixels == 9999 {
		t.Fatal("Tapes() returned a reference, not a copy")
	}
}

func TestTapeMargins(t *testing.T) {
	tape := LookupTape(24)
	if tape.MarginMM != 3.0 {
		t.Errorf("24mm tape margin = %f, want 3.0", tape.MarginMM)
	}
	tape = LookupTape(4)
	if tape.MarginMM != 0.5 {
		t.Errorf("3.5mm tape margin = %f, want 0.5", tape.MarginMM)
	}
}
