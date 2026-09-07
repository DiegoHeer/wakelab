package main

import (
	"fmt"
	"os"

	"github.com/DiegoHeer/wakelab/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
