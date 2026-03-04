package main

import (
	"bytes"
	"io"
	"net"
	"testing"

	"github.com/jaykay/ptouch/internal/protocol"
	"github.com/jaykay/ptouch/internal/raster"
)

// mockPrinter is a TCP server that accepts and discards raster data,
// simulating a P-Touch printer's write-only TCP behavior.
type mockPrinter struct {
	listener net.Listener
	received []byte
	done     chan struct{}
}

func newMockPrinter(t *testing.T) *mockPrinter {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	mp := &mockPrinter{
		listener: ln,
		done:     make(chan struct{}),
	}
	go func() {
		defer close(mp.done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		data, _ := io.ReadAll(conn)
		mp.received = data
	}()
	return mp
}

func (mp *mockPrinter) addr() string {
	return mp.listener.Addr().String()
}

func (mp *mockPrinter) close() {
	mp.listener.Close()
	<-mp.done
}

func TestPrintPipelineText(t *testing.T) {
	mp := newMockPrinter(t)
	defer mp.close()

	// Render a label.
	cfg := raster.TextConfig{
		Lines:    []string{"Test"},
		FontSize: 20,
	}
	result, err := raster.RenderText(cfg, 128, 128)
	if err != nil {
		t.Fatalf("RenderText: %v", err)
	}
	if len(result.RasterRows) == 0 {
		t.Fatal("no raster rows")
	}

	// Send to mock printer using the same pipeline as printLabel.
	conn, err := net.Dial("tcp", mp.addr())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	sess := protocol.NewSession(conn, protocol.FlagP700Init|protocol.FlagRasterPackBits)
	if err := sess.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := sess.StartRaster(); err != nil {
		t.Fatalf("StartRaster: %v", err)
	}
	if err := sess.SetMediaInfo(0x00, 24, 0, uint32(len(result.RasterRows))); err != nil {
		t.Fatalf("SetMediaInfo: %v", err)
	}
	if err := sess.SetCompression(true); err != nil {
		t.Fatalf("SetCompression: %v", err)
	}
	for i, row := range result.RasterRows {
		if err := sess.SendRasterLine(row); err != nil {
			t.Fatalf("SendRasterLine[%d]: %v", i, err)
		}
	}
	if err := sess.EndPage(true); err != nil {
		t.Fatalf("EndPage: %v", err)
	}
	conn.Close()

	// Wait for mock printer to finish receiving.
	<-mp.done

	// Verify we received data.
	if len(mp.received) == 0 {
		t.Fatal("mock printer received no data")
	}

	// Verify the init sequence (100 zeros + ESC @).
	if len(mp.received) < 102 {
		t.Fatalf("received only %d bytes, expected at least 102", len(mp.received))
	}
	for i := 0; i < 100; i++ {
		if mp.received[i] != 0x00 {
			t.Fatalf("init byte %d = 0x%02X, want 0x00", i, mp.received[i])
		}
	}
	if mp.received[100] != 0x1B || mp.received[101] != 0x40 {
		t.Fatalf("init ESC @ = [0x%02X, 0x%02X], want [0x1B, 0x40]", mp.received[100], mp.received[101])
	}

	// Verify P700 raster start (ESC i a 0x01).
	rest := mp.received[102:]
	if !bytes.HasPrefix(rest, []byte{0x1B, 0x69, 0x61, 0x01}) {
		t.Fatalf("raster start = %X..., want 1B696101", rest[:min(4, len(rest))])
	}

	// Verify it ends with PrintEject (0x1A).
	if mp.received[len(mp.received)-1] != 0x1A {
		t.Fatalf("last byte = 0x%02X, want 0x1A (PrintEject)", mp.received[len(mp.received)-1])
	}

	// Verify we have raster data (look for 0x47 'G' commands).
	hasRaster := false
	for _, b := range mp.received {
		if b == 0x47 {
			hasRaster = true
			break
		}
	}
	if !hasRaster {
		t.Fatal("no raster data (0x47 commands) found in output")
	}
}

func TestPrintPipelineChainNoCut(t *testing.T) {
	mp := newMockPrinter(t)
	defer mp.close()

	cfg := raster.TextConfig{
		Lines:    []string{"Chain"},
		FontSize: 20,
	}
	result, err := raster.RenderText(cfg, 128, 128)
	if err != nil {
		t.Fatalf("RenderText: %v", err)
	}

	conn, err := net.Dial("tcp", mp.addr())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	sess := protocol.NewSession(conn, protocol.FlagP700Init|protocol.FlagRasterPackBits)
	if err := sess.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := sess.StartRaster(); err != nil {
		t.Fatalf("StartRaster: %v", err)
	}
	if err := sess.SetMediaInfo(0x00, 24, 0, uint32(len(result.RasterRows))); err != nil {
		t.Fatalf("SetMediaInfo: %v", err)
	}
	if err := sess.SetCompression(true); err != nil {
		t.Fatalf("SetCompression: %v", err)
	}
	for i, row := range result.RasterRows {
		if err := sess.SendRasterLine(row); err != nil {
			t.Fatalf("SendRasterLine[%d]: %v", i, err)
		}
	}
	// Chain print = no eject.
	if err := sess.EndPage(false); err != nil {
		t.Fatalf("EndPage: %v", err)
	}
	conn.Close()

	<-mp.done

	// Should end with FormFeed (0x0C), not PrintEject (0x1A).
	if mp.received[len(mp.received)-1] != 0x0C {
		t.Fatalf("last byte = 0x%02X, want 0x0C (FormFeed)", mp.received[len(mp.received)-1])
	}
}

func TestPrintPipelineImage(t *testing.T) {
	mp := newMockPrinter(t)
	defer mp.close()

	// Create a small test image.
	bm := raster.NewBitmap(20, 10)
	bm.SetPixel(5, 5, true)
	bm.SetPixel(10, 5, true)

	// Simulate what LoadImage does: transpose + pad + raster rows.
	rotated := bm.Transpose()
	rotated = rotated.PadCenter(128)
	rows := rotated.ToRasterRows(128)

	conn, err := net.Dial("tcp", mp.addr())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	sess := protocol.NewSession(conn, protocol.FlagP700Init|protocol.FlagRasterPackBits)
	if err := sess.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := sess.StartRaster(); err != nil {
		t.Fatalf("StartRaster: %v", err)
	}
	if err := sess.SetMediaInfo(0x00, 12, 0, uint32(len(rows))); err != nil {
		t.Fatalf("SetMediaInfo: %v", err)
	}
	if err := sess.SetCompression(true); err != nil {
		t.Fatalf("SetCompression: %v", err)
	}
	for i, row := range rows {
		if err := sess.SendRasterLine(row); err != nil {
			t.Fatalf("SendRasterLine[%d]: %v", i, err)
		}
	}
	if err := sess.EndPage(true); err != nil {
		t.Fatalf("EndPage: %v", err)
	}
	conn.Close()

	<-mp.done

	if len(mp.received) == 0 {
		t.Fatal("mock printer received no data")
	}
	// Ends with PrintEject.
	if mp.received[len(mp.received)-1] != 0x1A {
		t.Fatalf("last byte = 0x%02X, want 0x1A", mp.received[len(mp.received)-1])
	}
}

func TestPrintPipelineMultiCopy(t *testing.T) {
	// Two copies: first ends with FormFeed, second with PrintEject.
	mp := newMockPrinter(t)
	defer mp.close()

	cfg := raster.TextConfig{
		Lines:    []string{"Copy"},
		FontSize: 20,
	}
	result, err := raster.RenderText(cfg, 128, 128)
	if err != nil {
		t.Fatalf("RenderText: %v", err)
	}

	conn, err := net.Dial("tcp", mp.addr())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	sess := protocol.NewSession(conn, protocol.FlagP700Init|protocol.FlagRasterPackBits)
	copies := 2
	for c := 0; c < copies; c++ {
		if err := sess.Init(); err != nil {
			t.Fatalf("Init[%d]: %v", c, err)
		}
		if err := sess.StartRaster(); err != nil {
			t.Fatalf("StartRaster[%d]: %v", c, err)
		}
		if err := sess.SetMediaInfo(0x00, 24, 0, uint32(len(result.RasterRows))); err != nil {
			t.Fatalf("SetMediaInfo[%d]: %v", c, err)
		}
		if err := sess.SetCompression(true); err != nil {
			t.Fatalf("SetCompression[%d]: %v", c, err)
		}
		for i, row := range result.RasterRows {
			if err := sess.SendRasterLine(row); err != nil {
				t.Fatalf("SendRasterLine[%d][%d]: %v", c, i, err)
			}
		}
		isLast := c == copies-1
		if err := sess.EndPage(isLast); err != nil {
			t.Fatalf("EndPage[%d]: %v", c, err)
		}
	}
	conn.Close()

	<-mp.done

	// Last byte should be PrintEject.
	if mp.received[len(mp.received)-1] != 0x1A {
		t.Fatalf("last byte = 0x%02X, want 0x1A", mp.received[len(mp.received)-1])
	}

	// Count init sequences (100 zeros + ESC @) — should be 2.
	initCount := 0
	for i := 0; i <= len(mp.received)-102; i++ {
		if mp.received[i+100] == 0x1B && mp.received[i+101] == 0x40 {
			allZero := true
			for j := 0; j < 100; j++ {
				if mp.received[i+j] != 0x00 {
					allZero = false
					break
				}
			}
			if allZero {
				initCount++
			}
		}
	}
	if initCount != 2 {
		t.Errorf("found %d init sequences, want 2", initCount)
	}
}
