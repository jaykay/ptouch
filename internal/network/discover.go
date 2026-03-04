package network

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/mdns"
)

// DefaultDiscoveryTimeout is how long to listen for mDNS responses.
const DefaultDiscoveryTimeout = 5 * time.Second

// BrotherServiceType is the mDNS service type used by Brother network printers.
const BrotherServiceType = "_pdl-datastream._tcp"

// Printer represents a printer discovered via mDNS.
type Printer struct {
	Name   string // mDNS instance name (e.g., "Brother PT-P750W")
	Host   string // hostname
	Port   int    // TCP port (usually 9100)
	AddrV4 net.IP // IPv4 address, if available
	AddrV6 net.IP // IPv6 address, if available
}

// Addr returns the address in "host:port" form, preferring IPv4.
func (p Printer) Addr() string {
	host := p.Host
	if p.AddrV4 != nil {
		host = p.AddrV4.String()
	}
	return net.JoinHostPort(host, fmt.Sprintf("%d", p.Port))
}

// Discover searches the local network for Brother P-Touch printers using mDNS.
// It listens for up to timeout duration (DefaultDiscoveryTimeout if zero).
// The context can be used for cancellation.
func Discover(ctx context.Context, timeout time.Duration) ([]Printer, error) {
	if timeout == 0 {
		timeout = DefaultDiscoveryTimeout
	}

	entriesCh := make(chan *mdns.ServiceEntry, 16)
	var printers []Printer

	// Collect results in background.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for entry := range entriesCh {
			printers = append(printers, Printer{
				Name:   entry.Name,
				Host:   entry.Host,
				Port:   entry.Port,
				AddrV4: entry.AddrV4,
				AddrV6: entry.AddrV6,
			})
		}
	}()

	params := mdns.DefaultParams(BrotherServiceType)
	params.Entries = entriesCh
	params.Timeout = timeout

	// Run query in a goroutine so we can enforce our own timeout.
	// mdns.Query may block longer than expected on some systems.
	queryDone := make(chan error, 1)
	go func() {
		queryDone <- mdns.Query(params)
	}()

	// Wait for query to finish or our hard deadline.
	deadline := time.After(timeout + 2*time.Second)
	select {
	case err := <-queryDone:
		close(entriesCh)
		<-done
		if err != nil {
			return nil, fmt.Errorf("network: mdns query: %w", err)
		}
	case <-ctx.Done():
		close(entriesCh)
		<-done
		return printers, ctx.Err()
	case <-deadline:
		close(entriesCh)
		<-done
		// Query didn't return in time — return whatever we found.
	}

	return printers, nil
}
