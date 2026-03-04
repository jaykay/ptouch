# Troubleshooting

## Discovery

### `ptouch discover` finds no printers

The discovery scans the local subnet for TCP port 9100. If nothing is found:

1. **Check network**: Make sure your computer and printer are on the same subnet.
2. **Check printer**: The printer must be powered on and connected to Wi-Fi/Ethernet.
3. **Manual scan**: Try `nmap -p 9100 --open 192.168.x.0/24` (replace with your subnet).
4. **Firewall**: Some networks block port scanning between hosts.

### Discovery finds the printer but shows "HTTP not available"

The printer has port 9100 open but doesn't have a web interface. This happens with some non-Brother devices. Use `--host` directly if you know it's your printer.

## Connection

### `connect: dial tcp ...: connection refused`

The printer is not accepting connections on port 9100.

- Check if the printer is powered on.
- Verify the IP address: `ping 192.168.x.x`
- Check if port 9100 is open: `nc -zv 192.168.x.x 9100`

### `connect: dial tcp ...: i/o timeout`

The printer is unreachable.

- Wrong IP address or subnet.
- Printer is powered off or disconnected from the network.
- Try increasing the timeout: `--timeout 30s`

## Printing

### Label comes out blank

The print data was sent but no content was rendered. Check:

- Are you providing `--text` or `--image`?
- Try `--preview /tmp/test.png` to see what would be printed.
- The tape might be exhausted — check the cartridge.

### Text appears garbled or barcode-like

This was a known issue with PackBits compression encoding (fixed in current version). Make sure you're running the latest build:

```bash
go build -o ptouch ./cmd/ptouch/
```

### Text is not centered on the tape (shifted up/down)

This happens when the tape size doesn't match what the printer detects. The content must be offset to the tape's physical position on the printhead.

- Check `ptouch info --host ...` to see the detected tape size.
- Use `--tape` to set the correct tape width manually.
- Use `-v` to see the detected tape and printhead configuration.

### Image is all black

The image likely has a transparent background (PNG with alpha channel). Transparent pixels are composited onto a white background. If the image is still all black:

- The image might genuinely be all-dark pixels. Check the preview: `--preview test.png`
- Try adjusting the image to have a white background before printing.

### Image is inverted (white on black instead of black on white)

The monochrome conversion uses a luminance threshold of 127. Pixels darker than the threshold become black. If your image has a dark background with light content, it will appear inverted.

Solution: invert the image colors before printing, or edit it to have a white background.

### Font looks wrong or characters are missing

The embedded Go font covers basic Latin characters. For special characters or a specific look:

```bash
ptouch print --text "Spëcial" --font /path/to/font.ttf --host ...
```

Use a TTF/OTF font file that contains the characters you need.

## Tape Detection

### Wrong tape size detected

The tape width is read from the printer's HTTP web interface. If it reports the wrong size:

```bash
# Check what the printer reports
ptouch info --host 192.168.86.130

# Override manually
ptouch print --text "Hello" --tape 12 --host 192.168.86.130
```

### "could not determine tape width"

The printer's web interface is not available and no `--tape` flag was given.

```bash
# Set tape manually
ptouch print --text "Hello" --tape 24 --host 192.168.86.130
```

## Preview

### Preview image looks different from printed output

The preview shows the label in reading orientation (as you'd see it on the tape). The printed output goes through transpose and printhead alignment. Minor differences in pixel alignment are normal.

### Preview shows wrong tape height

Without `--host`, preview uses the model default tape size (24mm for PT-P750W). To get accurate dimensions:

```bash
# Auto-detect from printer
ptouch print --text "Hello" --host 192.168.86.130 --preview label.png

# Or set manually
ptouch print --text "Hello" --tape 12 --preview label.png
```

## Verbose Mode

When something isn't working, add `-v` for detailed debug output:

```bash
ptouch print --text "Hello" --host 192.168.86.130 -v
```

This shows:

- Model resolution (name, pixels, DPI, flags)
- Printer status from web interface
- Tape detection source (web interface, flag, or default)
- Rendering dimensions and raster row count
- Compression and connection details

## Common Error Messages

| Message | Cause | Fix |
|---------|-------|-----|
| `provide --text or --image` | No content specified | Add `--text "..."` or `--image file.png` |
| `--text and --image are mutually exclusive` | Both specified | Use one or the other |
| `--host is required` | No printer address | Add `--host IP` or use `--preview` |
| `unknown model "..."` | Invalid `--model` value | Use a supported model name (e.g., PT-P750W) |
| `unsupported tape width Nmm` | Invalid `--tape` value | Use 4, 6, 9, 12, 18, 24, or 36 |
| `printer not ready: cover is open` | Tape compartment open | Close the cover |
| `printer not ready: no tape loaded` | No tape cassette | Insert a tape cassette |
| `printer not ready: cutter jam` | Cutter stuck | Open cover, remove jammed tape |
| `printer not ready: printer is overheated` | Thermal protection | Wait a few minutes |
