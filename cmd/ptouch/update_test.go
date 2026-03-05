package main

import "testing"

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest  string
		current string
		want    bool
	}{
		{"v1.3.0", "v1.2.0", true},
		{"v1.2.1", "v1.2.0", true},
		{"v2.0.0", "v1.9.9", true},
		{"v1.2.0", "v1.2.0", false},
		{"v1.1.0", "v1.2.0", false},
		{"v1.2.0", "dev", false},
		{"", "v1.2.0", false},
		{"v1.3.0", "v1.2.1-0.20260305-abc123", true},
		{"v1.2.1-0.20260305-abc123", "v1.2.0", true},
	}
	for _, tt := range tests {
		t.Run(tt.latest+"_vs_"+tt.current, func(t *testing.T) {
			if got := isNewer(tt.latest, tt.current); got != tt.want {
				t.Errorf("isNewer(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
			}
		})
	}
}

func TestNormalizeSemver(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"v1.2.3", "1.2.3"},
		{"1.2.3", "1.2.3"},
		{"v1.2", ""},
		{"dev", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeSemver(tt.input); got != tt.want {
				t.Errorf("normalizeSemver(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
