package protocol

import "fmt"

// ProtocolError represents a protocol-level error (malformed packet, unexpected response).
type ProtocolError struct {
	Op  string
	Msg string
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("protocol: %s: %s", e.Op, e.Msg)
}

// PrinterError represents an error reported by the printer in its status packet.
type PrinterError struct {
	Flags ErrorFlags
}

func (e *PrinterError) Error() string {
	return fmt.Sprintf("printer error: %s", e.Flags)
}
