---
title: Icons
nav_order: 3
---

# Inline Icons

ptouch supports inline icons in text labels using the `:prefix-name:` shortcode syntax.
Icons are sourced from two open-source libraries and rendered as crisp vector graphics
at any font size.

## Icon Libraries

| Prefix | Library | License | Icons |
|--------|---------|---------|-------|
| `ti-` | [Tabler Icons](https://tabler.io/icons) | MIT | ~5000 outline icons |
| `bi-` | [Bootstrap Icons](https://icons.getbootstrap.com) | MIT | ~2000 icons (outline + fill variants) |

Browse the websites above to find icon names, then use them with the appropriate prefix.

## How It Works

1. Use `:ti-name:` or `:bi-name:` anywhere in a `--text` string
2. On first use, the icon SVG is downloaded from GitHub and cached locally
3. Subsequent uses load instantly from `~/.cache/ptouch/icons/`
4. Icons scale automatically to match the current font size
5. Unknown icon names are printed as literal text

## Examples

### Text with icon

```bash
ptouch print --text "I :ti-heart: labels"
```

![I heart labels](img/icons/example-heart.png)

### Multi-line with icons

```bash
ptouch print --text ":ti-alert-triangle: Caution" --text ":ti-bolt: High Voltage"
```

![Caution / High Voltage](img/icons/example-multiline.png)

### Bootstrap filled icons

```bash
ptouch print --text ":bi-telephone-fill: Call me"
```

![Call me](img/icons/example-phone.png)

```bash
ptouch print --text ":bi-check-circle-fill: Done"
```

![Done](img/icons/example-done.png)

### Mixed text and icons

```bash
ptouch print --text ":ti-star: Rating: 5/5"
```

![Rating 5/5](img/icons/example-rating.png)

## Tabler Icons (outline)

Use with the `ti-` prefix. These are clean outline-style icons.

| Icon | Shortcode | Preview |
|------|-----------|---------|
| Heart | `:ti-heart:` | ![ti-heart](img/icons/ti-heart.png){: width="40" } |
| Star | `:ti-star:` | ![ti-star](img/icons/ti-star.png){: width="40" } |
| Check | `:ti-check:` | ![ti-check](img/icons/ti-check.png){: width="40" } |
| Alert | `:ti-alert-triangle:` | ![ti-alert-triangle](img/icons/ti-alert-triangle.png){: width="40" } |
| Bolt | `:ti-bolt:` | ![ti-bolt](img/icons/ti-bolt.png){: width="40" } |
| Home | `:ti-home:` | ![ti-home](img/icons/ti-home.png){: width="40" } |
| Wifi | `:ti-wifi:` | ![ti-wifi](img/icons/ti-wifi.png){: width="40" } |
| Mail | `:ti-mail:` | ![ti-mail](img/icons/ti-mail.png){: width="40" } |
| Phone | `:ti-phone:` | ![ti-phone](img/icons/ti-phone.png){: width="40" } |
| Clock | `:ti-clock:` | ![ti-clock](img/icons/ti-clock.png){: width="40" } |
| Arrow right | `:ti-arrow-right:` | ![ti-arrow-right](img/icons/ti-arrow-right.png){: width="40" } |
| Arrow left | `:ti-arrow-left:` | ![ti-arrow-left](img/icons/ti-arrow-left.png){: width="40" } |
| Smile | `:ti-mood-smile:` | ![ti-mood-smile](img/icons/ti-mood-smile.png){: width="40" } |
| Flame | `:ti-flame:` | ![ti-flame](img/icons/ti-flame.png){: width="40" } |
| Settings | `:ti-settings:` | ![ti-settings](img/icons/ti-settings.png){: width="40" } |
| Lock | `:ti-lock:` | ![ti-lock](img/icons/ti-lock.png){: width="40" } |

[Browse all Tabler Icons](https://tabler.io/icons){: .btn }

## Bootstrap Icons (filled)

Use with the `bi-` prefix. Many icons have both outline and `-fill` variants.

| Icon | Shortcode | Preview |
|------|-----------|---------|
| Heart | `:bi-heart-fill:` | ![bi-heart-fill](img/icons/bi-heart-fill.png){: width="40" } |
| Star | `:bi-star-fill:` | ![bi-star-fill](img/icons/bi-star-fill.png){: width="40" } |
| Check | `:bi-check-circle-fill:` | ![bi-check-circle-fill](img/icons/bi-check-circle-fill.png){: width="40" } |
| Warning | `:bi-exclamation-triangle-fill:` | ![bi-exclamation-triangle-fill](img/icons/bi-exclamation-triangle-fill.png){: width="40" } |
| Lightning | `:bi-lightning-fill:` | ![bi-lightning-fill](img/icons/bi-lightning-fill.png){: width="40" } |
| House | `:bi-house-fill:` | ![bi-house-fill](img/icons/bi-house-fill.png){: width="40" } |
| Wifi | `:bi-wifi:` | ![bi-wifi](img/icons/bi-wifi.png){: width="40" } |
| Phone | `:bi-telephone-fill:` | ![bi-telephone-fill](img/icons/bi-telephone-fill.png){: width="40" } |
| Envelope | `:bi-envelope-fill:` | ![bi-envelope-fill](img/icons/bi-envelope-fill.png){: width="40" } |
| Clock | `:bi-clock-fill:` | ![bi-clock-fill](img/icons/bi-clock-fill.png){: width="40" } |
| Fire | `:bi-fire:` | ![bi-fire](img/icons/bi-fire.png){: width="40" } |
| Gear | `:bi-gear-fill:` | ![bi-gear-fill](img/icons/bi-gear-fill.png){: width="40" } |
| Lock | `:bi-lock-fill:` | ![bi-lock-fill](img/icons/bi-lock-fill.png){: width="40" } |
| Trash | `:bi-trash-fill:` | ![bi-trash-fill](img/icons/bi-trash-fill.png){: width="40" } |
| Eye | `:bi-eye-fill:` | ![bi-eye-fill](img/icons/bi-eye-fill.png){: width="40" } |
| Printer | `:bi-printer-fill:` | ![bi-printer-fill](img/icons/bi-printer-fill.png){: width="40" } |

[Browse all Bootstrap Icons](https://icons.getbootstrap.com){: .btn }

## Tips

- **Tabler** icons are outline-only and work great for clean, minimal labels
- **Bootstrap** icons have `-fill` variants that print bolder and more visible at small sizes
- You can mix both libraries in the same label: `:ti-star: :bi-heart-fill:`
- Icons work with all text features: multi-line, bold, alignment, fixed width
- Use `ptouch icons` on the command line for a quick reference

---

*[Tabler Icons](https://github.com/tabler/tabler-icons) and [Bootstrap Icons](https://github.com/twbs/icons) are open-source projects licensed under MIT.*
