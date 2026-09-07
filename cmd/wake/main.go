// Command wake is the wakelab CLI entrypoint.
package main

import (
	"os"

	"github.com/DiegoHeer/wakelab/internal/cli"
)

func main() {
	// Execute prints its own errors; main only sets the exit code.
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
