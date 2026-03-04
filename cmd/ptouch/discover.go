package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jaykay/ptouch/internal/network"
	"github.com/spf13/cobra"
)

var flagDiscoverTimeout time.Duration

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Find Brother printers on the local network",
	Long: `Scans the local subnet for devices with TCP port 9100 open,
then queries each device's HTTP interface to identify Brother printers.`,
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
		for _, host := range allHits {
			ws, err := fetchWebStatus(host)
			if err != nil {
				fmt.Fprintf(out, "  %-16s  (port 9100 open, HTTP not available)\n", host)
				continue
			}
			fmt.Fprintf(out, "  %-16s  %s — %s\n", host, ws.ModelName, ws.DeviceStatus)
		}
		return nil
	},
}

func init() {
	discoverCmd.Flags().DurationVar(&flagDiscoverTimeout, "discover-timeout", 500*time.Millisecond, "per-host TCP connect timeout")
	rootCmd.AddCommand(discoverCmd)
}
