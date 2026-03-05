package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	flagHost    string
	flagModel   string
	flagTape    int
	flagTimeout time.Duration
	flagVerbose bool
)

var rootCmd = &cobra.Command{
	Use:   "ptouch",
	Short: "Brother P-Touch network label printer CLI",
	Long:  "Print labels on Brother P-Touch printers via TCP/IP (port 9100).",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagHost, "host", "", "printer IP address or hostname")
	rootCmd.PersistentFlags().StringVar(&flagModel, "model", "PT-P750W", "printer model (e.g. PT-P750W)")
	rootCmd.PersistentFlags().IntVar(&flagTape, "tape", 0, "tape width in mm (0 = auto from model/web interface)")
	rootCmd.PersistentFlags().DurationVar(&flagTimeout, "timeout", 10*time.Second, "read/write timeout")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "verbose debug output")
}

func main() {
	initConfig()
	startUpdateCheck()
	err := rootCmd.Execute()
	if cmd, _, _ := rootCmd.Find(os.Args[1:]); cmd != updateCmd && cmd != versionCmd {
		printUpdateNotice()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
