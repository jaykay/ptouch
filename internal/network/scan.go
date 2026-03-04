package network

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// ScanPort scans all hosts in the given CIDR subnet for an open TCP port.
// It returns the IPs that responded within the per-host timeout.
// Concurrency is capped at 128 goroutines.
func ScanPort(ctx context.Context, cidr string, port int, perHost time.Duration) ([]net.IP, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("network: parse CIDR %q: %w", cidr, err)
	}

	var targets []net.IP
	for addr := ip.Mask(ipNet.Mask); ipNet.Contains(addr); addr = nextIP(addr) {
		targets = append(targets, dupIP(addr))
	}

	// Skip network and broadcast addresses for /24 and larger.
	if len(targets) > 2 {
		targets = targets[1 : len(targets)-1]
	}

	var (
		mu    sync.Mutex
		found []net.IP
		wg    sync.WaitGroup
		sem   = make(chan struct{}, 128)
	)

	for _, target := range targets {
		select {
		case <-ctx.Done():
			break
		default:
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(ip net.IP) {
			defer wg.Done()
			defer func() { <-sem }()

			addr := net.JoinHostPort(ip.String(), fmt.Sprintf("%d", port))
			conn, err := net.DialTimeout("tcp", addr, perHost)
			if err != nil {
				return
			}
			conn.Close()

			mu.Lock()
			found = append(found, ip)
			mu.Unlock()
		}(target)
	}

	wg.Wait()
	return found, nil
}

// LocalSubnets returns the CIDR notation for all non-loopback IPv4 interfaces.
func LocalSubnets() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("network: list interfaces: %w", err)
	}

	var subnets []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			if ipNet.IP.To4() == nil {
				continue // skip IPv6
			}
			subnets = append(subnets, ipNet.String())
		}
	}
	return subnets, nil
}

func nextIP(ip net.IP) net.IP {
	next := dupIP(ip)
	for i := len(next) - 1; i >= 0; i-- {
		next[i]++
		if next[i] != 0 {
			break
		}
	}
	return next
}

func dupIP(ip net.IP) net.IP {
	dup := make(net.IP, len(ip))
	copy(dup, ip)
	return dup
}
