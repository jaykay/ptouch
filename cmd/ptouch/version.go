package main

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func init() {
	if info, ok := debug.ReadBuildInfo(); ok && version == "dev" {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				if len(s.Value) >= 7 {
					commit = s.Value[:7]
				}
			case "vcs.time":
				date = s.Value
			}
		}
	}

	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of ptouch",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ptouch %s (commit %s, built %s)\n", version, commit, date)
	},
}
