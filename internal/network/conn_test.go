package network

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/jaykay/ptouch/internal/protocol"
)

// Compile-time check: Connection must satisfy io.ReadWriter.
var _ io.ReadWriter = (*Connection)(nil)

// validStatusPacket returns a valid 32-byte status packet.
func validStatusPacket() [protocol.StatusPacketSize]byte {
	var p [protocol.StatusPacketSize]byte
	p[0] = 0x80  // printhead mark
	p[1] = 0x20  // size
	p[2] = 'B'   // brother code
	p[3] = '0'   // series code
	p[4] = 0x64  // model
	p[5] = '0'   // country
	p[10] = 24   // media width mm
	p[11] = 0x01 // media type: laminated
	p[18] = 0x00 // status type: reply
	p[24] = 0x01 // tape color: white
	p[25] = 0x08 // text color: black
	return p
}

// testServer starts a TCP listener on localhost with a handler function.
// Returns the listener. The caller must defer ln.Close().
func testServer(t *testing.T, handler func(conn net.Conn)) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("testServer listen: %v", err)
	}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		handler(conn)
	}()
	return ln
}

func TestDialAndHealthCheck(t *testing.T) {
	pkt := validStatusPacket()
	ln := testServer(t, func(conn net.Conn) {
		// Read Init (102 bytes) + StatusRequest (3 bytes).
		buf := make([]byte, 105)
		if _, err := io.ReadFull(conn, buf); err != nil {
			t.Errorf("server read: %v", err)
			return
		}
		// Verify Init: first 100 bytes zero, then 0x1B 0x40.
		for i := 0; i < 100; i++ {
			if buf[i] != 0x00 {
				t.Errorf("init byte %d = 0x%02X, want 0x00", i, buf[i])
				return
			}
		}
		if buf[100] != 0x1B || buf[101] != 0x40 {
			t.Errorf("init tail = [0x%02X, 0x%02X], want [0x1B, 0x40]", buf[100], buf[101])
			return
		}
		// Verify StatusRequest: 0x1B 0x69 0x53.
		if buf[102] != 0x1B || buf[103] != 0x69 || buf[104] != 0x53 {
			t.Errorf("status request = %X, want 1B6953", buf[102:105])
			return
		}
		// Respond with status packet.
		_, _ = conn.Write(pkt[:])
	})
	defer ln.Close()

	conn, status, err := Dial(ln.Addr().String(),
		WithConnectTimeout(2*time.Second),
		WithReadWriteTimeout(2*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	if status.MediaWidth != 24 {
		t.Errorf("status.MediaWidth = %d, want 24", status.MediaWidth)
	}
	if !status.IsReady() {
		t.Error("status.IsReady() = false, want true")
	}
}

func TestDialWithoutHealthCheck(t *testing.T) {
	ln := testServer(t, func(conn net.Conn) {
		// Do nothing — no data exchange expected.
	})
	defer ln.Close()

	conn, status, err := Dial(ln.Addr().String(), WithoutHealthCheck())
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	// Status should be zero value.
	if status.MediaWidth != 0 {
		t.Errorf("status.MediaWidth = %d, want 0 (no health check)", status.MediaWidth)
	}
}

func TestDialHealthCheckFailure(t *testing.T) {
	ln := testServer(t, func(conn net.Conn) {
		// Accept but close immediately — health check should fail.
		conn.Close()
	})
	defer ln.Close()

	_, _, err := Dial(ln.Addr().String(),
		WithReadWriteTimeout(1*time.Second),
	)
	if err == nil {
		t.Fatal("expected error when server closes immediately")
	}
}

func TestDialConnectTimeout(t *testing.T) {
	// 192.0.2.1 is TEST-NET — non-routable, will timeout.
	start := time.Now()
	_, _, err := Dial("192.0.2.1:9100",
		WithConnectTimeout(500*time.Millisecond),
		WithoutHealthCheck(),
	)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed > 3*time.Second {
		t.Errorf("took %v, expected ~500ms timeout", elapsed)
	}
}

func TestDialDefaultPort(t *testing.T) {
	// Just test normalizeAddr — we can't bind to port 9100.
	got := normalizeAddr("192.168.1.50")
	want := "192.168.1.50:9100"
	if got != want {
		t.Errorf("normalizeAddr('192.168.1.50') = %q, want %q", got, want)
	}

	// With explicit port, should pass through.
	got = normalizeAddr("192.168.1.50:1234")
	want = "192.168.1.50:1234"
	if got != want {
		t.Errorf("normalizeAddr('192.168.1.50:1234') = %q, want %q", got, want)
	}
}

func TestConnectionReadTimeout(t *testing.T) {
	ln := testServer(t, func(conn net.Conn) {
		// Never send anything — reader should timeout.
		time.Sleep(5 * time.Second)
	})
	defer ln.Close()

	conn, _, err := Dial(ln.Addr().String(),
		WithoutHealthCheck(),
		WithReadWriteTimeout(200*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	buf := make([]byte, 32)
	_, err = conn.Read(buf)
	if err == nil {
		t.Fatal("expected timeout error on Read")
	}
	if netErr, ok := err.(net.Error); ok && !netErr.Timeout() {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestConnectionReadWriteRoundTrip(t *testing.T) {
	echo := []byte("hello printer")
	ln := testServer(t, func(conn net.Conn) {
		buf := make([]byte, len(echo))
		if _, err := io.ReadFull(conn, buf); err != nil {
			return
		}
		_, _ = conn.Write(buf)
	})
	defer ln.Close()

	conn, _, err := Dial(ln.Addr().String(),
		WithoutHealthCheck(),
		WithReadWriteTimeout(2*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write(echo); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	buf := make([]byte, len(echo))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if string(buf) != string(echo) {
		t.Fatalf("echo = %q, want %q", buf, echo)
	}
}
