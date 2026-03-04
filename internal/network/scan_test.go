package network

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestLocalSubnets(t *testing.T) {
	subnets, err := LocalSubnets()
	if err != nil {
		t.Fatalf("LocalSubnets() error: %v", err)
	}
	if len(subnets) == 0 {
		t.Skip("no active network interfaces")
	}
	for _, s := range subnets {
		_, _, err := net.ParseCIDR(s)
		if err != nil {
			t.Errorf("invalid CIDR %q: %v", s, err)
		}
	}
	t.Logf("found subnets: %v", subnets)
}

func TestScanPortLocalhost(t *testing.T) {
	// Start a listener to have a known open port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port

	hits, err := ScanPort(context.Background(), "127.0.0.1/32", port, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("ScanPort error: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if !hits[0].Equal(net.IPv4(127, 0, 0, 1)) {
		t.Errorf("expected 127.0.0.1, got %s", hits[0])
	}
}

func TestScanPortNoneOpen(t *testing.T) {
	// Scan a port that's almost certainly not open on localhost.
	hits, err := ScanPort(context.Background(), "127.0.0.1/32", 19999, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("ScanPort error: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("expected 0 hits, got %d", len(hits))
	}
}

func TestNextIP(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"192.168.1.0", "192.168.1.1"},
		{"192.168.1.254", "192.168.1.255"},
		{"192.168.1.255", "192.168.2.0"},
	}
	for _, tt := range tests {
		got := nextIP(net.ParseIP(tt.in).To4())
		if got.String() != tt.want {
			t.Errorf("nextIP(%s) = %s, want %s", tt.in, got, tt.want)
		}
	}
}
