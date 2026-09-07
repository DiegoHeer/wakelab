package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLsCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "quick list of hosts (name, MAC, in-SSH-config)",
		Long:  "wake ls — quick list of hosts (name, MAC, and whether it's in ~/.ssh/config)",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			_, hosts, err := a.loadHosts()
			if err != nil {
				return err
			}
			fmt.Fprintf(a.Out, "%s%-10s %-18s %-6s %-17s %-9s %s%s\n",
				a.color(ansiBold), "HOST", "MAC", "PORT", "BROADCAST", "VIA", "SSH", a.color(ansiReset))
			for _, h := range hosts {
				ssh := a.color(ansiDim) + "no" + a.color(ansiReset)
				if a.inSSHConfig(h.Name) {
					ssh = a.color(ansiGreen) + "yes" + a.color(ansiReset)
				}
				via := h.Via
				if via == "" {
					via = "-"
				}
				fmt.Fprintf(a.Out, "%-10s %-18s %-6s %-17s %-9s %s\n",
					h.Name, h.Mac, h.ReadyPort(), h.BroadcastAddr(), via, ssh)
			}
			return nil
		},
	}
}
