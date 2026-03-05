package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var iconsCmd = &cobra.Command{
	Use:   "icons",
	Short: "Show how to use icons in --text",
	Long: `Icons from Tabler Icons and Bootstrap Icons can be used inline in text labels.
They are downloaded on first use and cached locally.

Use the :prefix-name: syntax in --text to insert an icon:
  ti-  for Tabler Icons     (https://tabler.io/icons)
  bi-  for Bootstrap Icons  (https://icons.getbootstrap.com)

Examples:
  ptouch print --text "I :ti-heart: labels"
  ptouch print --text ":bi-check-circle: Done"`,
	Run: func(cmd *cobra.Command, args []string) {
		out := cmd.OutOrStdout()
		fmt.Fprintln(out, "Icons are loaded from two open-source libraries:")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  Tabler Icons      :ti-<name>:    https://tabler.io/icons")
		fmt.Fprintln(out, "  Bootstrap Icons    :bi-<name>:    https://icons.getbootstrap.com")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Icons are downloaded on first use and cached in ~/.cache/ptouch/icons/.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Examples:")
		fmt.Fprintln(out, `  ptouch print --text "I :ti-heart: labels"`)
		fmt.Fprintln(out, `  ptouch print --text ":ti-star: Rating"`)
		fmt.Fprintln(out, `  ptouch print --text ":bi-check-circle: Done"`)
		fmt.Fprintln(out, `  ptouch print --text ":bi-exclamation-triangle: Warning"`)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Browse the full icon sets on the websites above to find icon names.")
	},
}

func init() {
	rootCmd.AddCommand(iconsCmd)
}
