---
title: Usage Guide
nav_order: 2
---

# Usage Guide

## Installation

```bash
# From source
go build -o ptouch ./cmd/ptouch/

# Or install to $GOPATH/bin
go install github.com/jaykay/ptouch/cmd/ptouch@latest
```

## Commands

### `ptouch version`

Shows the installed version, commit hash, and build date.

```bash
ptouch version
```

Output:

```
ptouch v1.2.1 (commit da992a1, built 2026-03-05T15:17:49Z)
```

### `ptouch update`

Checks for a newer release on GitHub and updates the binary.
If Go is installed, it runs `go install ...@latest`. Otherwise it prints
the download URL for the GitHub release.

```bash
ptouch update
```

Output:

```
Updating v1.2.1 → v1.3.0 ...
Updated to v1.3.0
```

### Auto-update check

Every invocation checks for a newer version in the background (at most once
every 24 hours). If an update is available, a notice is printed to stderr
after the command finishes:

```
A new version of ptouch is available: v1.2.1 → v1.3.0
Run `ptouch update` to upgrade.
```

The check is non-blocking — it never slows down the command, and dev builds
are excluded. The cache is stored in `~/.cache/ptouch/`.

### `ptouch discover`

Scans the local subnet for devices with TCP port 9100 open and queries their HTTP web interface to identify them.

```bash
ptouch discover
```

Output:

```
Scanning 1 subnet(s) for port 9100...
  192.168.86.125/24

Found 2 device(s) with port 9100 open:
  192.168.86.130    Brother PT-P750W — READY
  192.168.86.129    MFC-L3740CDW series — unknown
```

| Flag | Description | Default |
|------|-------------|---------|
| `--discover-timeout` | Per-host TCP connect timeout | 500ms |

### `ptouch info`

Shows printer status, tape info, and model details by querying the printer's HTTP web interface. Displays actionable hints for common issues.

```bash
ptouch info --host 192.168.86.130
```

Output:

```
Printer:    Brother PT-P750W
Serial:     M4G469550
Firmware:   1.22
Status:     READY
Emulation:  Raster
Media:      12mm(0.47") (Not Empty)
Tape:       12mm (76 printable pixels, 2.0mm margin)
```

### `ptouch print`

Print text or image labels. Requires `--host` for printing, or `--preview` for offline rendering.

## Printing Text Labels

### Basic text

```bash
ptouch print --text "Hello World" --host 192.168.86.130
```

Font size is automatically chosen to fill the tape height.

### Multi-line

Repeat `--text` for multiple lines. Font size auto-scales so all lines fit:

```bash
ptouch print --text "Line 1" --text "Line 2" --text "Line 3" --host 192.168.86.130
```

### Fixed-width labels

Set a specific label length in mm with `--width`. Useful when the label must fit a specific space:

```bash
ptouch print --text "Asset Tag" --width 50 --host 192.168.86.130
```

If the text is wider than the given width, the font size is automatically reduced to fit.

### Alignment

Text is centered by default. Use `--align` to change:

```bash
# Left-aligned
ptouch print --text "Name:" --text "Value" --align left --host 192.168.86.130

# Right-aligned
ptouch print --text "Price" --text "€9.99" --align right --host 192.168.86.130
```

Alignment works with and without `--width`.

### Bold text

```bash
ptouch print --text "IMPORTANT" --bold --host 192.168.86.130
```

### Custom font

```bash
ptouch print --text "Custom" --font /path/to/font.ttf --host 192.168.86.130
```

Accepts any TTF or OTF font file. Without `--font`, the embedded Go Regular/Bold font is used.

### Fixed font size

Override auto-fit with a specific size in points:

```bash
ptouch print --text "Small" --fontsize 12 --host 192.168.86.130
```

## Printing Images

Print any PNG, JPEG, or GIF file. The image is scaled to fit the tape height, maintaining aspect ratio, and converted to 1-bit monochrome (black and white):

```bash
ptouch print --image logo.png --host 192.168.86.130
```

Images with transparency (PNG alpha channel) are composited onto a white background.

### Image margins

By default images go edge-to-edge. Add a margin in mm with `--margin`:

```bash
ptouch print --image logo.png --margin 2 --host 192.168.86.130
```

This adds 2mm padding on all four sides.

## Preview Mode

Render a label to a PNG file without connecting to a printer:

```bash
ptouch print --text "Hello" --preview label.png
ptouch print --image logo.png --preview logo_preview.png
```

The preview shows the label as it will appear on tape — readable, correct orientation, actual tape dimensions.

When combined with `--host`, the tape size is auto-detected from the printer. Otherwise the model default is used, or you can set it with `--tape`:

```bash
ptouch print --text "Hello" --tape 12 --preview label.png      # 12mm tape
ptouch print --text "Hello" --host 192.168.86.130 --preview label.png  # auto-detect
```

## Print Options

### Multiple copies

```bash
ptouch print --text "Asset Tag" --copies 5 --host 192.168.86.130
```

### Cut control

By default, the tape is cut after printing. To chain-print multiple labels without cutting between them:

```bash
# No cut after this label
ptouch print --text "Label 1" --no-cut --host 192.168.86.130
ptouch print --text "Label 2" --host 192.168.86.130

# Or use --chain (same effect)
ptouch print --text "Label" --chain --host 192.168.86.130
```

### Manual tape width

Tape width is normally auto-detected from the printer's web interface. Override with `--tape`:

```bash
ptouch print --text "Hello" --tape 12 --host 192.168.86.130
```

Supported: 4 (3.5mm), 6, 9, 12, 18, 24, 36.

### Model override

Default model is PT-P750W. Override with `--model`:

```bash
ptouch print --text "Hello" --model PT-P700 --host 192.168.86.130
```

### Verbose output

Add `-v` for debug output showing model resolution, tape detection, rendering parameters, and protocol details:

```bash
ptouch print --text "Hello" --host 192.168.86.130 -v
```

## Global Flags

These apply to all commands:

| Flag | Description | Default |
|------|-------------|---------|
| `--host` | Printer IP address or hostname | — |
| `--model` | Printer model name | PT-P750W |
| `--tape` | Tape width in mm (0 = auto-detect) | 0 |
| `--timeout` | TCP read/write timeout | 10s |
| `-v, --verbose` | Enable debug output | false |

## Print Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--text` | Text line (repeatable for multi-line) | — |
| `--image` | Image file path (PNG/JPEG/GIF) | — |
| `--font` | Custom TTF/OTF font file | embedded Go font |
| `--fontsize` | Font size in points (0 = auto-fit) | 0 |
| `--bold` | Use bold variant of embedded font | false |
| `--align` | Text alignment: `left`, `center`, `right` | center |
| `--width` | Label width in mm (0 = auto from content) | 0 |
| `--margin` | Image margin in mm | 0 |
| `--copies` | Number of copies to print | 1 |
| `--cut` | Cut tape after printing | true |
| `--no-cut` | Don't cut tape | false |
| `--chain` | Chain print (same as --no-cut) | false |
| `--preview` | Save preview PNG instead of printing | — |

Note: `--text` and `--image` are mutually exclusive.

---

*Brother and P-Touch are trademarks of Brother Industries, Ltd. This project is not affiliated with, endorsed by, or sponsored by Brother Industries.*
