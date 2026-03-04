package device

import (
	"testing"

	"github.com/jaykay/ptouch/internal/protocol"
)

func TestLookupByProductID(t *testing.T) {
	tests := []struct {
		pid      uint16
		wantName string
		wantNil  bool
	}{
		{0x2062, "PT-P750W", false},
		{0x2061, "PT-P700", false},
		{0x2074, "PT-D600", false},
		{0x0000, "", true},
		{0xFFFF, "", true},
	}
	for _, tt := range tests {
		m := LookupByProductID(tt.pid)
		if tt.wantNil {
			if m != nil {
				t.Errorf("LookupByProductID(0x%04X) = %q, want nil", tt.pid, m.Name)
			}
			continue
		}
		if m == nil {
			t.Fatalf("LookupByProductID(0x%04X) = nil, want %q", tt.pid, tt.wantName)
		}
		if m.Name != tt.wantName {
			t.Errorf("LookupByProductID(0x%04X).Name = %q, want %q", tt.pid, m.Name, tt.wantName)
		}
	}
}

func TestLookupByName(t *testing.T) {
	tests := []struct {
		name    string
		wantPID uint16
		wantNil bool
	}{
		{"PT-P750W", 0x2062, false},
		{"pt-p750w", 0x2062, false}, // case insensitive
		{"PT-P700", 0x2061, false},
		{"UNKNOWN", 0, true},
	}
	for _, tt := range tests {
		m := LookupByName(tt.name)
		if tt.wantNil {
			if m != nil {
				t.Errorf("LookupByName(%q) = %q, want nil", tt.name, m.Name)
			}
			continue
		}
		if m == nil {
			t.Fatalf("LookupByName(%q) = nil, want PID 0x%04X", tt.name, tt.wantPID)
		}
		if m.ProductID != tt.wantPID {
			t.Errorf("LookupByName(%q).ProductID = 0x%04X, want 0x%04X", tt.name, m.ProductID, tt.wantPID)
		}
	}
}

func TestLookupByNameSkipsPLite(t *testing.T) {
	// PT-P750W has a P-Lite entry (0x2065) — LookupByName should skip it.
	m := LookupByName("PT-P750W")
	if m == nil {
		t.Fatal("LookupByName(PT-P750W) = nil")
	}
	if m.Flags.Has(protocol.FlagPLite) {
		t.Errorf("LookupByName returned P-Lite entry (PID=0x%04X)", m.ProductID)
	}
	if m.ProductID != 0x2062 {
		t.Errorf("ProductID = 0x%04X, want 0x2062", m.ProductID)
	}
}

func TestP750WFlags(t *testing.T) {
	m := LookupByName("PT-P750W")
	if m == nil {
		t.Fatal("PT-P750W not found")
	}
	if !m.Flags.Has(protocol.FlagRasterPackBits) {
		t.Error("PT-P750W should have FlagRasterPackBits")
	}
	if !m.Flags.Has(protocol.FlagP700Init) {
		t.Error("PT-P750W should have FlagP700Init")
	}
	if m.Flags.Has(protocol.FlagHasPrecut) {
		t.Error("PT-P750W should NOT have FlagHasPrecut")
	}
	if m.MaxPixels != 128 {
		t.Errorf("MaxPixels = %d, want 128", m.MaxPixels)
	}
	if m.DPI != 180 {
		t.Errorf("DPI = %d, want 180", m.DPI)
	}
}

func TestModels(t *testing.T) {
	all := Models()
	if len(all) == 0 {
		t.Fatal("Models() returned empty")
	}
	for _, m := range all {
		if m.Flags.Has(protocol.FlagPLite) {
			t.Errorf("Models() includes P-Lite entry: %s (0x%04X)", m.Name, m.ProductID)
		}
	}
}
