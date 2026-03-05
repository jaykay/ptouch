---
title: Architecture
nav_order: 4
---

# Architecture

## Overview

```
ptouch
├── cmd/ptouch/           CLI layer (cobra)
│   ├── main.go           Root command, global flags
│   ├── print.go          Print command — rendering + print pipeline
│   ├── info.go           Info command — web interface scraping
│   ├── discover.go       Discover command — subnet scanning
│   └── log.go            Debug logging helper
│
├── internal/protocol/    ESC/P protocol encoding & decoding
│   ├── commands.go       Command builders (Init, MediaInfo, Compression, etc.)
│   ├── raster.go         Raster line encoding + PackBits compression
│   ├── session.go        High-level session wrapping io.ReadWriter
│   ├── status.go         32-byte status packet parser
│   └── errors.go         Protocol and printer error types
│
├── internal/network/     TCP transport & discovery
│   ├── conn.go           Connection with per-op timeouts, Dial with health check
│   ├── scan.go           Subnet port scanning, local interface detection
│   └── discover.go       mDNS/Bonjour discovery (fallback)
│
├── internal/device/      Printer model database & tape specs
│   ├── models.go         27 printer models with capabilities
│   ├── tape.go           7 tape sizes with pixel widths and margins
│   └── detect.go         Auto-detect model + tape from status packet
│
└── internal/raster/      Text/image rendering
    ├── bitmap.go         1-bit bitmap (MSB-first), transpose, padding
    ├── text.go           Text rendering, auto-fit, alignment, multi-line
    └── image.go          Image loading, scaling, monochrome conversion
```

## Package Dependencies

```
cmd/ptouch
  ├── internal/protocol
  ├── internal/network   → internal/protocol
  ├── internal/device    → internal/protocol
  └── internal/raster    (standalone)
```

The protocol package is the foundation. Network depends on protocol for health checks. Device depends on protocol for capability flags. Raster is fully independent — it produces bitmaps and raster rows without knowing about printers.

## Data Flow

### Print Pipeline

```
User Input (--text / --image)
    │
    ▼
┌──────────────┐
│ Render       │  raster.RenderText() or raster.LoadImage()
│              │  → RGBA canvas → 1-bit bitmap → transpose → pad center
└──────┬───────┘
       │ RenderResult { Bitmap, Preview, RasterRows, WidthPx, HeightPx }
       ▼
┌──────────────┐
│ Connect      │  network.Dial(host, WithoutHealthCheck())
└──────┬───────┘
       │ Connection (io.ReadWriter)
       ▼
┌──────────────┐
│ Session      │  protocol.NewSession(conn, model.Flags)
│              │  Init → StartRaster → SetMediaInfo → SetCompression
│              │  → SendRasterLine (×N) → EndPage
└──────────────┘
```

### Preview Pipeline

Same rendering step, but saves the `Preview` bitmap (pre-transpose, human-readable) as PNG instead of connecting to the printer.

### Text Rendering Detail

```
TextConfig { Lines, FontSize, Bold, Align, MaxWidthPx }
    │
    ├── FontSize == 0? → autoFitSize() binary search (20 iterations)
    │   Constrains on both tape height AND MaxWidthPx
    │
    ├── Create RGBA canvas (canvasWidth × tapePixels)
    │   └── White background
    │
    ├── Draw each line with horizontal alignment + padding
    │   └── Vertically centered on tape
    │
    ├── FromImage() → 1-bit Bitmap (luminance threshold 127)
    │
    ├── Transpose() → (x,y) → (y,x) for tape orientation
    │
    └── PadCenter(maxPixels) → center on printhead
```

### Image Rendering Detail

```
image.Decode(file)
    │
    ├── Scale to tape height (minus margins), preserving aspect ratio
    │
    ├── FromImage() → 1-bit Bitmap
    │   └── Alpha compositing onto white background
    │
    ├── Place on canvas with margin padding
    │
    ├── Transpose()
    │
    └── PadCenter(maxPixels)
```

## Bitmap Orientation

The printer receives raster data one row at a time as the tape feeds through. Each row represents a vertical slice across the tape width.

The canvas is drawn in natural reading orientation (x = label width, y = tape height). `Transpose()` maps `(x, y) → (y, x)` so that:

- Each column of the canvas becomes a raster row
- Top of canvas (y=0) maps to MSB of byte 0 (top edge of tape)
- Left of canvas (x=0) maps to the first raster row sent

For tapes narrower than the printhead (e.g., 12mm tape on 128px printhead), `PadCenter()` offsets the content within each raster row so it aligns with the tape's physical position.

## Tape Detection

Three-tier fallback:

1. `--tape` flag (explicit)
2. Printer's HTTP web interface (`/general/status.html` → parse tape width)
3. Model default (largest tape that fits the printhead)

## Session Protocol

The printer communicates via ESC-prefixed binary commands over raw TCP on port 9100. This is the same protocol as USB, with no wrapper.

P-Touch printers are **write-only over TCP** — they don't respond to status requests on port 9100. Bidirectional communication only works over USB. Printer status is obtained from the HTTP web interface instead.

## Supported Models

The model database (`internal/device/models.go`) contains 27 printers from the C reference implementation. Each model has:

- **Name** — e.g., "PT-P750W"
- **ProductID** — USB PID for auto-detection via status packet
- **MaxPixels** — printhead width (128 for most models, 384 for PT-9200DX)
- **DPI** — 180 for most, 360 for PT-9200DX
- **Flags** — capability bitfield:
  - `FlagRasterPackBits` — supports PackBits compression
  - `FlagP700Init` — uses P700-style raster start command
  - `FlagHasPrecut` — supports precut command
  - `FlagPLite` — P-Lite mode (filtered from lookups)
  - `FlagUseInfoCmd` — uses info command
  - `FlagD460BTMagic` — D460BT-style initialization

## Tape Sizes

| Width | Pixels (180 DPI) | Margin |
|-------|-------------------|--------|
| 3.5mm | 24 | 0.5mm |
| 6mm   | 32 | 1.0mm |
| 9mm   | 52 | 1.0mm |
| 12mm  | 76 | 2.0mm |
| 18mm  | 120 | 3.0mm |
| 24mm  | 128 | 3.0mm |
| 36mm  | 192 | 4.5mm |

---

*Brother and P-Touch are trademarks of Brother Industries, Ltd. This project is not affiliated with, endorsed by, or sponsored by Brother Industries.*
