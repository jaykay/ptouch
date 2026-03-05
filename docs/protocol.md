---
title: Protocol
nav_order: 5
---

# Brother ESC/P Raster Protocol

This document describes the binary protocol used by Brother P-Touch printers over TCP port 9100 (and USB). Based on the [C reference implementation](https://git.familie-radermacher.ch/linux/ptouch-print.git).

## Transport

- **Port**: TCP 9100 (raw socket, no HTTP or IPP wrapper)
- **Direction**: Write-only over TCP. Printers do not respond to commands on this port. (USB is bidirectional.)
- **Byte order**: Little-endian for multi-byte values.

## Command Sequence

A complete print job follows this sequence:

```
1. Init              100× 0x00, then ESC @
2. Start Raster      ESC i R 0x01  (or ESC i a 0x01 for P700 series)
3. Media Info         ESC i z [11 bytes]
4. Compression        M [mode]
5. Precut (optional)  ESC i M [flag]
6. Raster Lines       G [lenLo] [lenHi] [data...]  (×N)
7. End Page           0x1A (cut) or 0x0C (no cut)
```

## Commands

### Init

```
[100 × 0x00] [0x1B] [0x40]
```

102 bytes total. The 100 zero bytes flush any previous partial command. `ESC @` (0x1B 0x40) resets the printer.

### Start Raster Mode

Standard:
```
[0x1B] [0x69] [0x52] [0x01]     ESC i R 0x01
```

P700 series (FlagP700Init):
```
[0x1B] [0x69] [0x61] [0x01]     ESC i a 0x01
```

### Media Info

```
[0x1B] [0x69] [0x7A]            ESC i z
[0x86]                           validity flags (width + length + raster count)
[mediaType]                      0x00 = auto
[widthMM]                        tape width in mm (e.g., 24)
[lengthMM]                       0 = continuous tape
[rasterLines: uint32 LE]         total number of raster rows
[0x00]                           starting page
[0x00]                           reserved
```

13 bytes total.

### Compression Mode

```
[0x4D] [mode]
```

- `0x00` — no compression
- `0x02` — PackBits compression

### Precut

```
[0x1B] [0x69] [0x4D] [flag]
```

- `0x00` — precut disabled
- `0x40` — precut enabled

Only sent if the model has `FlagHasPrecut`.

### Raster Line

Uncompressed:
```
[0x47] [lenLo] [lenHi] [pixel data...]
```

PackBits compressed:
```
[0x47] [lenLo] [lenHi] [compressed data...]
```

`len` is a 16-bit little-endian value giving the byte count of the data that follows.

Each raster line represents one vertical slice across the tape width. Pixel data is packed MSB-first: bit 7 of byte 0 is the top edge of the tape.

The data must be padded to `MaxPixels / 8` bytes (e.g., 16 bytes for a 128-pixel printhead), regardless of actual tape width.

### Empty Raster Line

```
[0x5A]
```

Advances the tape by one row without printing.

### End Page

Cut (print and eject):
```
[0x1A]
```

No cut (form feed / chain print):
```
[0x0C]
```

## PackBits Compression

PackBits is a simple run-length encoding scheme used to compress raster lines.

The compressed data is a sequence of packets:

- **Run packet**: `[count] [byte]` where count is `-(n-1)` as a signed byte (0x00 to 0x7F maps to -0 to -127). Repeat `byte` n times (2–128).
- **Literal packet**: `[count] [byte1] [byte2] ...` where count is `n-1` (0x00 to 0x7F). Copy the following n bytes literally (1–128).

Example: `[0xFF] [0xAA]` means repeat 0xAA twice. `[0x02] [0x11] [0x22] [0x33]` means copy three literal bytes.

## Status Packet (USB only)

32 bytes, returned in response to `ESC i S` (status request). Not available over TCP.

| Offset | Field | Description |
|--------|-------|-------------|
| 0 | PrintHeadMark | Always 0x80 |
| 1 | Size | Always 0x20 (32) |
| 2 | BrotherCode | Manufacturer code |
| 3 | SeriesCode | Series identifier |
| 4 | Model | Model code |
| 5 | Country | Country code |
| 8-9 | Error | Error flags (16-bit LE) |
| 10 | MediaWidth | Tape width in mm |
| 11 | MediaType | Media type byte |
| 17 | MediaLength | Length in mm (0 = continuous) |
| 18 | StatusType | 0x00=reply, 0x01=done, 0x02=error |
| 24 | TapeColor | Background color |
| 25 | TextColor | Text/ink color |

### Error Flags

| Bit | Flag | Description |
|-----|------|-------------|
| 0 | ErrNoMedia | No tape cassette |
| 2 | ErrCutterJam | Cutter mechanism jammed |
| 3 | ErrLowBattery | Low battery |
| 4 | ErrInUse | Printer in use |
| 8 | ErrCoverOpen | Tape compartment open |
| 9 | ErrOverheat | Printhead overheated |
| 10 | ErrTapeNotLoaded | Tape not loaded properly |

### Media Types

| Value | Type |
|-------|------|
| 0x00 | None |
| 0x01 | Laminated |
| 0x03 | Non-laminated |
| 0x04 | Fabric |
| 0x11 | Heat-shrink 2:1 |
| 0x17 | Heat-shrink 3:1 |
| 0x13 | Flexible ID |
| 0xFF | Incompatible |

## Web Interface (TCP Alternative)

Since status is unavailable over TCP, printer info is scraped from the HTTP web interface:

- **Status page**: `http://<host>/general/status.html`
  - Model name, device status, emulation, media status, media type
- **Info page**: `http://<host>/general/information.html?kind=item`
  - Serial number, firmware version

Tape width is parsed from the media type string (e.g., "12mm(0.47\")").

## References

- [ptouch-print C reference](https://git.familie-radermacher.ch/linux/ptouch-print.git) — original C implementation
- Brother PT-P750W technical reference (from reverse engineering)

---

*Brother and P-Touch are trademarks of Brother Industries, Ltd. This project is not affiliated with, endorsed by, or sponsored by Brother Industries.*
