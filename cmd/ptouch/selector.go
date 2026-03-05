package main

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// printerChoice represents a discovered printer for selection.
type printerChoice struct {
	Host  string
	Model string
}

func (p printerChoice) String() string {
	if p.Model != "" {
		return fmt.Sprintf("%s  %s", p.Host, p.Model)
	}
	return p.Host
}

// selectPrinter shows an interactive list and returns the chosen printer.
// Returns nil if the user cancelled (Ctrl+C / q / Esc).
func selectPrinter(choices []printerChoice) *printerChoice {
	if len(choices) == 0 {
		return nil
	}

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Fallback: non-interactive, pick first.
		return &choices[0]
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	cursor := 0
	buf := make([]byte, 3)

	render := func() {
		// Move to start and clear.
		fmt.Fprint(os.Stderr, "\r\033[J")
		fmt.Fprintln(os.Stderr, "Select a printer (arrow keys + Enter):\r")
		for i, c := range choices {
			if i == cursor {
				fmt.Fprintf(os.Stderr, "  \033[1m> %s\033[0m\r\n", c)
			} else {
				fmt.Fprintf(os.Stderr, "    %s\r\n", c)
			}
		}
	}

	render()

	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return nil
		}

		switch {
		case n == 1 && buf[0] == 13: // Enter
			// Clear the selector output.
			lines := len(choices) + 1
			for range lines {
				fmt.Fprint(os.Stderr, "\033[A\033[2K")
			}
			fmt.Fprint(os.Stderr, "\r")
			return &choices[cursor]

		case n == 1 && (buf[0] == 3 || buf[0] == 'q'): // Ctrl+C or q
			fmt.Fprint(os.Stderr, "\r\n")
			return nil

		case n == 1 && buf[0] == 27: // bare Esc (no sequence following)
			fmt.Fprint(os.Stderr, "\r\n")
			return nil

		case n == 3 && buf[0] == 27 && buf[1] == '[': // Arrow keys
			switch buf[2] {
			case 'A': // Up
				if cursor > 0 {
					cursor--
				}
			case 'B': // Down
				if cursor < len(choices)-1 {
					cursor++
				}
			}
		}

		// Rewrite list in place.
		lines := len(choices) + 1
		for range lines {
			fmt.Fprint(os.Stderr, "\033[A")
		}
		render()
	}
}
