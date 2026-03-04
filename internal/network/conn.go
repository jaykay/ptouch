// Package network handles TCP communication with network-enabled
// P-Touch printers (typically on port 9100).
package network

import (
	"fmt"
	"net"
	"time"

	"github.com/jaykay/ptouch/internal/protocol"
)

// DefaultPort is the raw TCP port used by Brother network printers.
const DefaultPort = 9100

// DefaultConnectTimeout is the default timeout for establishing a TCP connection.
const DefaultConnectTimeout = 5 * time.Second

// DefaultReadWriteTimeout is the default timeout for individual read/write operations.
const DefaultReadWriteTimeout = 10 * time.Second

// Option configures a Connection.
type Option func(*connectionConfig)

type connectionConfig struct {
	connectTimeout  time.Duration
	rwTimeout       time.Duration
	skipHealthCheck bool
}

// WithConnectTimeout sets the timeout for establishing the TCP connection.
func WithConnectTimeout(d time.Duration) Option {
	return func(c *connectionConfig) { c.connectTimeout = d }
}

// WithReadWriteTimeout sets the deadline for each read or write operation.
func WithReadWriteTimeout(d time.Duration) Option {
	return func(c *connectionConfig) { c.rwTimeout = d }
}

// WithoutHealthCheck disables the automatic health check on connect.
func WithoutHealthCheck() Option {
	return func(c *connectionConfig) { c.skipHealthCheck = true }
}

// Connection wraps a TCP connection to a P-Touch printer with per-operation timeouts.
// It implements io.ReadWriter and can be passed directly to protocol.NewSession.
type Connection struct {
	conn      net.Conn
	rwTimeout time.Duration
}

// Read reads from the printer, applying the read/write timeout as a deadline.
func (c *Connection) Read(p []byte) (int, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.rwTimeout)); err != nil {
		return 0, fmt.Errorf("network: set read deadline: %w", err)
	}
	return c.conn.Read(p)
}

// Write writes to the printer, applying the read/write timeout as a deadline.
func (c *Connection) Write(p []byte) (int, error) {
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.rwTimeout)); err != nil {
		return 0, fmt.Errorf("network: set write deadline: %w", err)
	}
	return c.conn.Write(p)
}

// Close closes the underlying TCP connection.
func (c *Connection) Close() error {
	return c.conn.Close()
}

// RemoteAddr returns the remote address of the connection.
func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Dial connects to a P-Touch printer at the given address.
// If the address lacks a port, DefaultPort (9100) is used.
// After connecting, it performs a health check (Init + RequestStatus) unless
// disabled with WithoutHealthCheck.
// Returns the connection and the status from the health check (zero value if skipped).
func Dial(address string, opts ...Option) (*Connection, protocol.Status, error) {
	cfg := connectionConfig{
		connectTimeout: DefaultConnectTimeout,
		rwTimeout:      DefaultReadWriteTimeout,
	}
	for _, o := range opts {
		o(&cfg)
	}

	addr := normalizeAddr(address)

	conn, err := net.DialTimeout("tcp", addr, cfg.connectTimeout)
	if err != nil {
		return nil, protocol.Status{}, fmt.Errorf("network: dial %s: %w", addr, err)
	}

	c := &Connection{
		conn:      conn,
		rwTimeout: cfg.rwTimeout,
	}

	var status protocol.Status
	if !cfg.skipHealthCheck {
		status, err = healthCheck(c)
		if err != nil {
			c.conn.Close()
			return nil, protocol.Status{}, fmt.Errorf("network: health check %s: %w", addr, err)
		}
	}

	return c, status, nil
}

// normalizeAddr ensures the address has a port component.
func normalizeAddr(address string) string {
	_, _, err := net.SplitHostPort(address)
	if err != nil {
		// No port specified, append default.
		return net.JoinHostPort(address, fmt.Sprintf("%d", DefaultPort))
	}
	return address
}

// healthCheck sends Init + RequestStatus using a temporary zero-flag session.
// Init and RequestStatus are flag-independent — they send universal ESC commands.
func healthCheck(c *Connection) (protocol.Status, error) {
	s := protocol.NewSession(c, protocol.FlagNone)
	if err := s.Init(); err != nil {
		return protocol.Status{}, fmt.Errorf("init: %w", err)
	}
	status, err := s.RequestStatus()
	if err != nil {
		return protocol.Status{}, fmt.Errorf("request status: %w", err)
	}
	return status, nil
}
