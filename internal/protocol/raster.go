package protocol

import "fmt"

// RasterLine encodes a single row of pixel data as a standard (uncompressed)
// raster command: 0x47 [lenLo] [lenHi] [pixelData].
func RasterLine(pixelData []byte) []byte {
	n := len(pixelData)
	buf := make([]byte, 3+n)
	buf[0] = 0x47
	buf[1] = byte(n)
	buf[2] = byte(n >> 8)
	copy(buf[3:], pixelData)
	return buf
}

// RasterLinePackBits encodes a single row of pixel data using PackBits compression.
// Format is the same as standard: 0x47 [lenLo] [lenHi] [compressedData].
func RasterLinePackBits(pixelData []byte) []byte {
	compressed := PackBits(pixelData)
	n := len(compressed)
	buf := make([]byte, 3+n)
	buf[0] = 0x47
	buf[1] = byte(n)
	buf[2] = byte(n >> 8)
	copy(buf[3:], compressed)
	return buf
}

// PackBits compresses data using the PackBits algorithm (TIFF/Apple style).
//
// Encoding rules:
//   - Run of 2+ identical bytes: emit (257-count) then the byte. Max count: 128.
//   - Literal sequence: emit (count-1) then the bytes. Max count: 128.
func PackBits(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}

	var out []byte
	i := 0

	for i < len(data) {
		// Check for a run of identical bytes.
		runLen := 1
		for i+runLen < len(data) && runLen < 128 && data[i+runLen] == data[i] {
			runLen++
		}

		if runLen >= 2 {
			// Emit run: (257 - runLen) as a byte, then the repeated byte.
			out = append(out, byte(257-runLen), data[i])
			i += runLen
			continue
		}

		// Collect literals until we hit a run of 2+ or reach 128.
		litStart := i
		litLen := 0
		for i+litLen < len(data) && litLen < 128 {
			// Look ahead: if next 2 bytes are identical, stop literals here.
			if i+litLen+1 < len(data) && data[i+litLen] == data[i+litLen+1] {
				break
			}
			litLen++
		}
		if litLen == 0 {
			// Edge case: the very next byte starts a run, but runLen was 1.
			// This happens when next two bytes are the same but we didn't catch it
			// above because runLen check was for current position.
			// Emit single literal.
			litLen = 1
		}
		out = append(out, byte(litLen-1))
		out = append(out, data[litStart:litStart+litLen]...)
		i += litLen
	}

	return out
}

// UnpackBits decompresses PackBits-encoded data.
func UnpackBits(data []byte) ([]byte, error) {
	var out []byte
	i := 0

	for i < len(data) {
		control := data[i]
		i++

		if control < 128 {
			// Literal run: (control + 1) bytes follow.
			n := int(control) + 1
			if i+n > len(data) {
				return nil, fmt.Errorf("packbits: literal overflow at offset %d", i-1)
			}
			out = append(out, data[i:i+n]...)
			i += n
		} else if control > 128 {
			// Repeated run: (257 - control) copies of next byte.
			if i >= len(data) {
				return nil, fmt.Errorf("packbits: run byte missing at offset %d", i-1)
			}
			n := 257 - int(control)
			for j := 0; j < n; j++ {
				out = append(out, data[i])
			}
			i++
		}
		// control == 128 is a no-op (padding), skip it.
	}

	return out, nil
}
