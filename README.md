# ptouch

CLI tool to print labels on Brother P-Touch printers via network (TCP port 9100).

Reimplements the Brother ESC/P raster protocol in Go — no USB required, no drivers needed.

## Quick Start

```bash
go install github.com/jaykay/ptouch/cmd/ptouch@latest

ptouch discover                                          # find & select printer (saved to config)
ptouch info                                              # show printer status
ptouch print --text "Hello World"                        # print a label
ptouch print --text "Hello" --preview label.png          # preview without printing
ptouch update                                            # self-update to latest
```

## Features

- **Text labels** with auto-fit font sizing, multi-line, bold, custom fonts, [inline icons](#inline-icons)
- **Image labels** from PNG, JPEG, GIF — scaled and converted to monochrome
- **Fixed-width labels** with alignment (left/center/right) and auto-shrink
- **Preview mode** — render to PNG without a printer
- **Printer discovery** — scan local network, select interactively, saved to config
- **Tape auto-detection** from the printer's web interface
- **Multiple copies**, chain printing, cut control

## Supported Hardware

Tested with PT-P750W. Built-in database covers 27 Brother PT models.
Supports tape widths: 3.5mm, 6mm, 9mm, 12mm, 18mm, 24mm, 36mm.

## Documentation

See [`docs/`](docs/) for the full documentation:

- [Usage Guide](docs/usage.md) — commands, flags, and examples
- [Architecture](docs/architecture.md) — how the code is structured
- [Protocol](docs/protocol.md) — Brother ESC/P raster protocol details
- [Troubleshooting](docs/troubleshooting.md) — common issues and fixes

## Build

```bash
go build -o ptouch ./cmd/ptouch/
go test ./...    # 106 tests
```

## Acknowledgments

The protocol implementation is based on [ptouch-print](https://git.familie-radermacher.ch/linux/ptouch-print.git) by Dominic Radermacher — a C reference implementation that was invaluable for understanding the Brother ESC/P raster protocol.

## Disclaimer

Brother and P-Touch are trademarks of Brother Industries, Ltd. This project is not affiliated with, endorsed by, or sponsored by Brother Industries. All trademarks belong to their respective owners.

## License

MIT — see [LICENSE](LICENSE).
