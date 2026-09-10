package cli

import (
	"fmt"

	"github.com/DiegoHeer/wakelab/internal/config"
	"github.com/spf13/cobra"
)

func newRmCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:               "rm <name>",
		Short:             "remove a host",
		Long:              "wake rm — remove a host from ~/.wol_hosts",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: a.completeHosts,
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			hostsText, hosts, err := a.loadHosts()
			if err != nil {
				return err
			}
			if !hostExists(hosts, name) {
				return fmt.Errorf("host '%s' not found in %s", name, a.HostsPath)
			}
			if err := config.Save(a.HostsPath, config.RemoveHost(hostsText, name)); err != nil {
				return err
			}
			fmt.Fprintf(a.Out, "Removed '%s'\n", name)
			return nil
		},
	}
}
