package main

import "fmt"

// debugf prints a message to stderr if verbose mode is enabled.
func debugf(format string, args ...any) {
	if flagVerbose {
		fmt.Fprintf(rootCmd.ErrOrStderr(), "[debug] "+format+"\n", args...)
	}
}
