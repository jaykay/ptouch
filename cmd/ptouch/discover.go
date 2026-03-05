package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jaykay/ptouch/internal/network"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flagDiscoverTimeout time.Duration

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Find Brother printers on the local network",
	Long: `Scans the local subnet for devices with TCP port 9100 open,
then queries each device's HTTP interface to identify Brother printers.

If printers are found, an interactive selector lets you choose one.
The selected printer is saved to the config file so you don't need
to pass --host on every command.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		out := cmd.OutOrStdout()

		subnets, err := network.LocalSubnets()
		if err != nil {
			return fmt.Errorf("detect subnets: %w", err)
		}
		if len(subnets) == 0 {
			return fmt.Errorf("no active network interfaces found")
		}

		fmt.Fprintf(out, "Scanning %d subnet(s) for port 9100...\n", len(subnets))
		for _, s := range subnets {
			fmt.Fprintf(out, "  %s\n", s)
		}

		var allHits []string
		for _, subnet := range subnets {
			hits, err := network.ScanPort(ctx, subnet, 9100, flagDiscoverTimeout)
			if err != nil {
				fmt.Fprintf(out, "  warning: scan %s: %v\n", subnet, err)
				continue
			}
			for _, ip := range hits {
				allHits = append(allHits, ip.String())
			}
		}

		if len(allHits) == 0 {
			fmt.Fprintln(out, "No devices with port 9100 found.")
			return nil
		}

		fmt.Fprintf(out, "\nFound %d device(s) with port 9100 open:\n", len(allHits))

		var choices []printerChoice
		for _, host := range allHits {
			ws, err := fetchWebStatus(host)
			if err != nil {
				fmt.Fprintf(out, "  %-16s  (port 9100 open, HTTP not available)\n", host)
				choices = append(choices, printerChoice{Host: host})
				continue
			}
			fmt.Fprintf(out, "  %-16s  %s — %s\n", host, ws.ModelName, ws.DeviceStatus)
			choices = append(choices, printerChoice{Host: host, Model: ws.ModelName})
		}

		fmt.Fprintln(out)

		selected := selectPrinter(choices)
		if selected == nil {
			return nil
		}

		viper.Set("host", selected.Host)
		if selected.Model != "" {
			// The web interface returns "Brother PT-P750W" but the device
			// database uses just "PT-P750W". Strip the vendor prefix.
			model := strings.TrimPrefix(selected.Model, "Brother ")
			viper.Set("model", model)
		}

		if err := saveConfig(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Fprintf(out, "Saved %s as default printer in %s\n", selected.Host, configFilePath())
		return nil
	},
}

func init() {
	discoverCmd.Flags().DurationVar(&flagDiscoverTimeout, "discover-timeout", 500*time.Millisecond, "per-host TCP connect timeout")
	rootCmd.AddCommand(discoverCmd)
}
