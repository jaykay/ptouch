package protocol

import (
	"bytes"
	"io"
	"testing"
)

// testRW is a test transport that captures writes and provides canned reads.
type testRW struct {
	written bytes.Buffer
	reader  io.Reader
}

func (t *testRW) Write(p []byte) (int, error) { return t.written.Write(p) }
func (t *testRW) Read(p []byte) (int, error)  { return t.reader.Read(p) }

func newTestRW(readData []byte) *testRW {
	return &testRW{reader: bytes.NewReader(readData)}
}

func TestSessionInit(t *testing.T) {
	rw := newTestRW(nil)
	s := NewSession(rw, FlagNone)

	if err := s.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got := rw.written.Bytes()
	if len(got) != 102 {
		t.Fatalf("Init wrote %d bytes, want 102", len(got))
	}
	if got[100] != 0x1B || got[101] != 0x40 {
		t.Fatalf("Init tail = [0x%02X, 0x%02X], want [0x1B, 0x40]", got[100], got[101])
	}
}

func TestSessionRequestStatus(t *testing.T) {
	pkt := validPacket() // from status_test.go
	rw := newTestRW(pkt[:])
	s := NewSession(rw, FlagNone)

	status, err := s.RequestStatus()
	if err != nil {
		t.Fatalf("RequestStatus() error = %v", err)
	}

	// Verify the status request command was sent.
	written := rw.written.Bytes()
	want := []byte{0x1B, 0x69, 0x53}
	if !bytes.Equal(written, want) {
		t.Fatalf("sent %X, want %X", written, want)
	}

	if status.MediaWidth != 24 {
		t.Errorf("MediaWidth = %d, want 24", status.MediaWidth)
	}
	if !status.IsReady() {
		t.Error("IsReady() = false, want true")
	}
}

func TestSessionStartRasterStandard(t *testing.T) {
	rw := newTestRW(nil)
	s := NewSession(rw, FlagNone)

	if err := s.StartRaster(); err != nil {
		t.Fatalf("StartRaster() error = %v", err)
	}

	want := []byte{0x1B, 0x69, 0x52, 0x01}
	if !bytes.Equal(rw.written.Bytes(), want) {
		t.Fatalf("sent %X, want %X", rw.written.Bytes(), want)
	}
}

func TestSessionStartRasterP700(t *testing.T) {
	rw := newTestRW(nil)
	s := NewSession(rw, FlagP700Init)

	if err := s.StartRaster(); err != nil {
		t.Fatalf("StartRaster() error = %v", err)
	}

	want := []byte{0x1B, 0x69, 0x61, 0x01}
	if !bytes.Equal(rw.written.Bytes(), want) {
		t.Fatalf("sent %X, want %X", rw.written.Bytes(), want)
	}
}

func TestSessionSendRasterLineStandard(t *testing.T) {
	rw := newTestRW(nil)
	s := NewSession(rw, FlagNone)

	data := []byte{0xFF, 0x00}
	if err := s.SendRasterLine(data); err != nil {
		t.Fatalf("SendRasterLine() error = %v", err)
	}

	// Standard: 0x47 [len] 0x00 [data]
	want := []byte{0x47, 0x02, 0x00, 0xFF, 0x00}
	if !bytes.Equal(rw.written.Bytes(), want) {
		t.Fatalf("sent %X, want %X", rw.written.Bytes(), want)
	}
}

func TestSessionSendRasterLinePackBits(t *testing.T) {
	rw := newTestRW(nil)
	s := NewSession(rw, FlagRasterPackBits)

	data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	if err := s.SendRasterLine(data); err != nil {
		t.Fatalf("SendRasterLine() error = %v", err)
	}

	got := rw.written.Bytes()
	if got[0] != 0x47 {
		t.Fatalf("got[0] = 0x%02X, want 0x47", got[0])
	}
	// Verify the compressed payload decompresses back to original.
	compressed := PackBits(data)
	payload := got[3:]
	if !bytes.Equal(payload, compressed) {
		t.Fatalf("payload %X != compressed %X", payload, compressed)
	}
}

func TestSessionSetCompressionEnable(t *testing.T) {
	rw := newTestRW(nil)
	s := NewSession(rw, FlagRasterPackBits)

	if err := s.SetCompression(true); err != nil {
		t.Fatalf("SetCompression(true) error = %v", err)
	}
	want := []byte{0x4D, 0x02}
	if !bytes.Equal(rw.written.Bytes(), want) {
		t.Fatalf("sent %X, want %X", rw.written.Bytes(), want)
	}
}

func TestSessionSetCompressionDisable(t *testing.T) {
	rw := newTestRW(nil)
	s := NewSession(rw, FlagNone)

	if err := s.SetCompression(false); err != nil {
		t.Fatalf("SetCompression(false) error = %v", err)
	}
	want := []byte{0x4D, 0x00}
	if !bytes.Equal(rw.written.Bytes(), want) {
		t.Fatalf("sent %X, want %X", rw.written.Bytes(), want)
	}
}

func TestSessionSetMargin(t *testing.T) {
	rw := newTestRW(nil)
	s := NewSession(rw, FlagNone)

	if err := s.SetMargin(14); err != nil {
		t.Fatalf("SetMargin(14) error = %v", err)
	}
	want := []byte{0x1B, 0x69, 0x64, 0x0E, 0x00}
	if !bytes.Equal(rw.written.Bytes(), want) {
		t.Fatalf("sent %X, want %X", rw.written.Bytes(), want)
	}
}

func TestSessionSetPrecutWithFlag(t *testing.T) {
	rw := newTestRW(nil)
	s := NewSession(rw, FlagHasPrecut)

	if err := s.SetPrecut(true); err != nil {
		t.Fatalf("SetPrecut(true) error = %v", err)
	}
	want := []byte{0x1B, 0x69, 0x4D, 0x40}
	if !bytes.Equal(rw.written.Bytes(), want) {
		t.Fatalf("sent %X, want %X", rw.written.Bytes(), want)
	}
}

func TestSessionSetPrecutNoFlag(t *testing.T) {
	rw := newTestRW(nil)
	s := NewSession(rw, FlagNone) // no FlagHasPrecut

	if err := s.SetPrecut(true); err != nil {
		t.Fatalf("SetPrecut(true) error = %v", err)
	}
	// Should be a no-op.
	if rw.written.Len() != 0 {
		t.Fatalf("expected no data written, got %X", rw.written.Bytes())
	}
}

func TestSessionEndPageEject(t *testing.T) {
	rw := newTestRW(nil)
	s := NewSession(rw, FlagNone)

	if err := s.EndPage(true); err != nil {
		t.Fatalf("EndPage(true) error = %v", err)
	}
	if !bytes.Equal(rw.written.Bytes(), []byte{0x1A}) {
		t.Fatalf("sent %X, want [1A]", rw.written.Bytes())
	}
}

func TestSessionEndPageChain(t *testing.T) {
	rw := newTestRW(nil)
	s := NewSession(rw, FlagNone)

	if err := s.EndPage(false); err != nil {
		t.Fatalf("EndPage(false) error = %v", err)
	}
	if !bytes.Equal(rw.written.Bytes(), []byte{0x0C}) {
		t.Fatalf("sent %X, want [0C]", rw.written.Bytes())
	}
}

func TestSessionSendEmptyLine(t *testing.T) {
	rw := newTestRW(nil)
	s := NewSession(rw, FlagNone)

	if err := s.SendEmptyLine(); err != nil {
		t.Fatalf("SendEmptyLine() error = %v", err)
	}
	if !bytes.Equal(rw.written.Bytes(), []byte{0x5A}) {
		t.Fatalf("sent %X, want [5A]", rw.written.Bytes())
	}
}

func TestSessionSetMediaInfo(t *testing.T) {
	rw := newTestRW(nil)
	s := NewSession(rw, FlagNone)

	if err := s.SetMediaInfo(0x01, 24, 0, 100); err != nil {
		t.Fatalf("SetMediaInfo() error = %v", err)
	}

	got := rw.written.Bytes()
	if len(got) != 13 {
		t.Fatalf("SetMediaInfo wrote %d bytes, want 13", len(got))
	}
	if got[0] != 0x1B || got[1] != 0x69 || got[2] != 0x7A {
		t.Fatalf("prefix = %X, want 1B697A", got[:3])
	}
}

func TestSessionFlags(t *testing.T) {
	rw := newTestRW(nil)
	flags := FlagP700Init | FlagRasterPackBits
	s := NewSession(rw, flags)
	if s.Flags() != flags {
		t.Fatalf("Flags() = %v, want %v", s.Flags(), flags)
	}
}
