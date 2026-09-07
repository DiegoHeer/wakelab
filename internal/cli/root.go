// Package cli wires up the wake command tree.
package cli

import "github.com/spf13/cobra"

// version is the build-time version, overridden via -ldflags.
var version = "dev"

// NewRootCmd builds the top-level `wake` command.
func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "wake",
		Short:        "Wake and sleep your whole homelab from the CLI",
		Version:      version,
		SilenceUsage: true,
	}
}

// Execute builds and runs the root command.
func Execute() error {
	return NewRootCmd().Execute()
}
