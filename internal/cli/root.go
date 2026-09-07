// Package cli wires up the wake command tree.
package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// version is the build-time version, overridden via -ldflags.
var version = "dev"

// errSilent signals a non-zero exit with no error message (e.g. `wake status
// <host>` reporting offline).
var errSilent = errors.New("")

// NewRootCmd builds the top-level `wake` command.
func NewRootCmd(a *App) *cobra.Command {
	opts := wakeOptions{}
	root := &cobra.Command{
		Use:   "wake [target]",
		Short: "Wake and sleep your whole homelab from the CLI",
		Long: `wake — wake your machines over the network (Wake-on-LAN) and control them over SSH

A <target> is a host name, a group name, a raw MAC address, or the word 'all'.
Hosts live in ssh-config-style blocks in ~/.wol_hosts; groups in ~/.wol_groups.`,
		Example: `  wake server                 wake one host
  wake minirack --wait        wake a group and wait until every host is up
  wake status --json          machine-readable status of every host`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if len(args) > 1 {
				return fmt.Errorf("unexpected argument '%s'", args[1])
			}
			opts.viaSet = cmd.Flags().Changed("via")
			return a.runWake(cmd.Context(), args[0], opts)
		},
	}
	root.Flags().BoolVar(&opts.wait, "wait", false, "poll until the target is up")
	root.Flags().IntVar(&opts.timeout, "timeout", 60, "--wait timeout in seconds")
	root.Flags().StringVar(&opts.port, "port", "", "--wait probes this TCP port on every host")
	root.Flags().StringVar(&opts.via, "via", "", "send the wake from this SSH relay ('' forces local)")
	root.Flags().StringVar(&opts.broadcast, "broadcast", "", "broadcast address for a raw MAC target")

	root.AddCommand(newLsCmd(a))
	root.AddCommand(newStatusCmd(a))
	root.AddCommand(newAddCmd(a))
	root.AddCommand(newEditCmd(a))
	root.AddCommand(newRmCmd(a))
	root.AddCommand(newImportSSHCmd(a))
	root.AddCommand(newGroupCmd(a))
	root.AddCommand(newGroupsCmd(a))
	root.AddCommand(newPoweroffCmd(a))
	root.AddCommand(newRestartCmd(a))
	root.AddCommand(newSuspendCmd(a))
	root.AddCommand(newScheduleCmd(a))
	root.AddCommand(newScanCmd(a))
	root.AddCommand(newDoctorCmd(a))
	return root
}

// Execute builds and runs the root command, printing non-silent errors as
// "Error: ..." (parity with the Bash tool).
func Execute() error {
	a := NewApp()
	cmd := NewRootCmd(a)
	err := cmd.Execute()
	if err != nil && err.Error() != "" {
		fmt.Fprintf(a.Err, "Error: %v\n", err)
	}
	return err
}
