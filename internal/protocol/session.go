package protocol

import (
	"fmt"
	"io"
)

// Session manages communication with a P-Touch printer over any transport.
type Session struct {
	rw    io.ReadWriter
	flags Flag
}

// NewSession creates a Session with the given transport and capability flags.
func NewSession(rw io.ReadWriter, flags Flag) *Session {
	return &Session{rw: rw, flags: flags}
}

// Flags returns the capability flags for this session.
func (s *Session) Flags() Flag { return s.flags }

// Init sends the initialization sequence (100 zero bytes + ESC @).
func (s *Session) Init() error {
	return s.send(Init())
}

// RequestStatus sends a status request and reads the 32-byte response.
func (s *Session) RequestStatus() (Status, error) {
	if err := s.send(StatusRequest()); err != nil {
		return Status{}, fmt.Errorf("protocol: request status: %w", err)
	}
	return s.readStatus()
}

// StartRaster sends the raster-mode start command.
// Uses P700 variant if FlagP700Init is set.
func (s *Session) StartRaster() error {
	return s.send(RasterStart(s.flags.Has(FlagP700Init)))
}

// SetMediaInfo sends the media info command.
func (s *Session) SetMediaInfo(mediaType byte, width, length byte, rasterLines uint32) error {
	return s.send(MediaInfo(mediaType, width, length, rasterLines))
}

// SetCompression sends the compression mode command.
// Enables PackBits if enable is true, otherwise disables compression.
func (s *Session) SetCompression(enable bool) error {
	mode := CompressionNone
	if enable {
		mode = CompressionPackBits
	}
	return s.send(Compression(mode))
}

// SetMargin sends the feed margin command.
// dots is the margin in printer dots on each side of the label.
func (s *Session) SetMargin(dots int) error {
	return s.send(Margin(dots))
}

// SetPrecut sends the precut command. No-op if the printer lacks FlagHasPrecut.
func (s *Session) SetPrecut(enable bool) error {
	if !s.flags.Has(FlagHasPrecut) {
		return nil
	}
	return s.send(Precut(enable))
}

// SendRasterLine sends a single raster row.
// Uses PackBits encoding if FlagRasterPackBits is set.
func (s *Session) SendRasterLine(pixelData []byte) error {
	if s.flags.Has(FlagRasterPackBits) {
		return s.send(RasterLinePackBits(pixelData))
	}
	return s.send(RasterLine(pixelData))
}

// SendEmptyLine sends an empty raster line.
func (s *Session) SendEmptyLine() error {
	return s.send(EmptyLine())
}

// EndPage sends the finalize command.
// If eject is true, sends PrintEject (cut tape). Otherwise sends FormFeed (chain print).
func (s *Session) EndPage(eject bool) error {
	if eject {
		return s.send(PrintEject())
	}
	return s.send(FormFeed())
}

func (s *Session) send(data []byte) error {
	_, err := s.rw.Write(data)
	if err != nil {
		return fmt.Errorf("protocol: write: %w", err)
	}
	return nil
}

func (s *Session) readStatus() (Status, error) {
	buf := make([]byte, StatusPacketSize)
	if _, err := io.ReadFull(s.rw, buf); err != nil {
		return Status{}, fmt.Errorf("protocol: read status: %w", err)
	}
	return ParseStatus(buf)
}
