package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jaykay/ptouch/internal/device"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show printer status and tape info",
	Long: `Query the printer's web interface for status information.

Note: P-Touch printers on the network do not support bidirectional
communication over port 9100. Status is read from the HTTP interface.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagHost == "" {
			return fmt.Errorf("--host is required")
		}

		out := cmd.OutOrStdout()

		status, err := fetchWebStatus(flagHost)
		if err != nil {
			return fmt.Errorf("query printer: %w\n\nHint: make sure the printer is turned on and reachable at %s", err, flagHost)
		}

		fmt.Fprintf(out, "Printer:    %s\n", status.ModelName)
		if status.Serial != "" {
			fmt.Fprintf(out, "Serial:     %s\n", status.Serial)
		}
		if status.Firmware != "" {
			fmt.Fprintf(out, "Firmware:   %s\n", status.Firmware)
		}
		fmt.Fprintf(out, "Status:     %s\n", status.DeviceStatus)
		fmt.Fprintf(out, "Emulation:  %s\n", status.Emulation)
		fmt.Fprintf(out, "Media:      %s (%s)\n", status.MediaType, status.MediaStatus)

		// Look up model in database.
		if m := device.LookupByName(status.ModelName); m != nil {
			fmt.Fprintf(out, "Printhead:  %d px @ %d DPI\n", m.MaxPixels, m.DPI)
		}

		// Look up tape if we can parse the width.
		if status.TapeWidthMM > 0 {
			if t := device.LookupTape(status.TapeWidthMM); t != nil {
				fmt.Fprintf(out, "Tape:       %dmm (%d printable pixels, %.1fmm margin)\n",
					t.WidthMM, t.Pixels, t.MarginMM)
			}
		}

		// Actionable hints for common issues.
		printStatusHints(out, status)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

// webStatus holds information scraped from the printer's web interface.
type webStatus struct {
	ModelName    string
	Serial       string
	Firmware     string
	DeviceStatus string
	Emulation    string
	MediaStatus  string
	MediaType    string
	TapeWidthMM  int
}

// fetchWebStatus queries the printer's HTTP interface for status information.
func fetchWebStatus(host string) (*webStatus, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	// Fetch the status page.
	statusHTML, err := httpGet(client, fmt.Sprintf("http://%s/general/status.html", host))
	if err != nil {
		return nil, fmt.Errorf("status page: %w", err)
	}

	// Fetch the info page.
	infoHTML, err := httpGet(client, fmt.Sprintf("http://%s/general/information.html?kind=item", host))
	if err != nil {
		// Info page is optional (might need auth).
		infoHTML = ""
	}

	ws := &webStatus{}

	// Parse status page.
	ws.ModelName = extractDT(statusHTML, "Model Name")
	if ws.ModelName == "" {
		// Try title tag: <title>Brother PT-P750W</title>
		if m := regexp.MustCompile(`<title>(?:Brother\s+)?([^<]+)</title>`).FindStringSubmatch(statusHTML); m != nil {
			ws.ModelName = strings.TrimSpace(m[1])
		}
	}
	ws.DeviceStatus = extractMoniStatus(statusHTML)
	ws.Emulation = extractDT(statusHTML, "Emulation")
	ws.MediaStatus = extractDT(statusHTML, "Media Status")
	ws.MediaType = extractDT(statusHTML, "Media Type")

	// Parse tape width from media type like "24mm(0.94\")"
	if m := regexp.MustCompile(`(\d+)mm`).FindStringSubmatch(ws.MediaType); m != nil {
		fmt.Sscanf(m[1], "%d", &ws.TapeWidthMM)
	}

	// Parse info page.
	if infoHTML != "" {
		if v := extractDT(infoHTML, "Model Name"); v != "" {
			ws.ModelName = v
		}
		ws.Serial = extractDT(infoHTML, "Serial no.")
		ws.Firmware = extractDT(infoHTML, "Firmware Version")
	}

	if ws.ModelName == "" {
		ws.ModelName = "unknown"
	}

	return ws, nil
}

func httpGet(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// extractDT finds the <dd> value after a <dt> containing the given label.
var dtDDPattern = regexp.MustCompile(`<dt[^>]*>([^<]*(?:&#\d+;[^<]*)*)</dt>\s*<dd[^>]*>(.*?)</dd>`)

func extractDT(html, label string) string {
	matches := dtDDPattern.FindAllStringSubmatch(html, -1)
	for _, m := range matches {
		dtText := decodeHTMLEntities(m[1])
		if strings.EqualFold(strings.TrimSpace(dtText), label) {
			// Strip tags from dd value.
			dd := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(m[2], "")
			return strings.TrimSpace(decodeHTMLEntities(dd))
		}
	}
	return ""
}

// extractMoniStatus finds the device status from the moni span.
func extractMoniStatus(html string) string {
	re := regexp.MustCompile(`class="moni\s+moni\w+"[^>]*>([^<]*)<`)
	if m := re.FindStringSubmatch(html); m != nil {
		return strings.TrimSpace(m[1])
	}
	return "unknown"
}

// printStatusHints prints actionable hints for known printer issues.
func printStatusHints(w io.Writer, ws *webStatus) {
	ds := strings.ToUpper(ws.DeviceStatus)
	ms := strings.ToUpper(ws.MediaStatus)

	switch {
	case strings.Contains(ds, "COVER OPEN"), strings.Contains(ds, "COVER IS OPEN"):
		fmt.Fprintln(w, "\nWarning: cover is open — close the tape compartment cover before printing.")
	case (strings.Contains(ms, "EMPTY") && !strings.Contains(ms, "NOT EMPTY")), strings.Contains(ms, "NO MEDIA"):
		fmt.Fprintln(w, "\nWarning: no tape loaded — insert a tape cassette.")
	case strings.Contains(ds, "CUTTER"), strings.Contains(ds, "JAM"):
		fmt.Fprintln(w, "\nWarning: cutter jam — open the cover, remove jammed tape, and close again.")
	case strings.Contains(ds, "COOLING"), strings.Contains(ds, "OVERHEAT"):
		fmt.Fprintln(w, "\nWarning: printer is overheated — wait for it to cool down before printing.")
	case strings.Contains(ds, "ERROR"):
		fmt.Fprintln(w, "\nWarning: printer reports an error — try turning it off and on again.")
	}
}

func decodeHTMLEntities(s string) string {
	s = strings.ReplaceAll(s, "&#32;", " ")
	s = strings.ReplaceAll(s, "&#38;", "&")
	s = strings.ReplaceAll(s, "&#60;", "<")
	s = strings.ReplaceAll(s, "&#62;", ">")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	return s
}
