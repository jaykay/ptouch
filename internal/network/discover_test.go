package network

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestPrinterAddr(t *testing.T) {
	tests := []struct {
		name string
		p    Printer
		want string
	}{
		{
			"ipv4 preferred",
			Printer{Host: "printer.local", Port: 9100, AddrV4: net.IPv4(192, 168, 1, 50)},
			"192.168.1.50:9100",
		},
		{
			"hostname fallback",
			Printer{Host: "printer.local", Port: 9100},
			"printer.local:9100",
		},
		{
			"custom port",
			Printer{Host: "printer.local", Port: 1234, AddrV4: net.IPv4(10, 0, 0, 1)},
			"10.0.0.1:1234",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.p.Addr()
			if got != tt.want {
				t.Errorf("Addr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDiscoverTimeout(t *testing.T) {
	// Short timeout — should return quickly with no error.
	// May or may not find real printers on the network.
	printers, err := Discover(context.Background(), 200*time.Millisecond)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	t.Logf("found %d printers in 200ms", len(printers))
}
