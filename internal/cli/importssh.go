package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/DiegoHeer/wakelab/internal/config"
	"github.com/DiegoHeer/wakelab/internal/host"
	"github.com/DiegoHeer/wakelab/internal/sshexec"
	"github.com/spf13/cobra"
)

func newImportSSHCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "import-ssh",
		Short: "import all online hosts from ~/.ssh/config",
		Long: `wake import-ssh — import hosts from ~/.ssh/config

Adds every SSH host that is online and not already listed.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			data, err := os.ReadFile(a.SSHConfigPath)
			if err != nil {
				return fmt.Errorf("no SSH config at ~/.ssh/config")
			}
			hostsText, hosts, err := a.loadHosts()
			if err != nil {
				return err
			}
			added, skipped := 0, 0
			for _, line := range strings.Split(string(data), "\n") {
				fields := strings.Fields(line)
				if len(fields) < 2 || fields[0] != "Host" {
					continue
				}
				for _, name := range fields[1:] {
					if strings.ContainsAny(name, "*?") {
						continue
					}
					if hostExists(hosts, name) {
						fmt.Fprintf(a.Out, "  %s= %s (already listed)%s\n", a.color(ansiDim), name, a.color(ansiReset))
						continue
					}
					ip := sshexec.SSHOption(ctx, a.Runner, name, "hostname")
					mac := host.NormalizeMac(a.macFromIP(ctx, ip))
					if mac == "" {
						fmt.Fprintf(a.Out, "  %s- %s (offline / no MAC)%s\n", a.color(ansiRed), name, a.color(ansiReset))
						skipped++
						continue
					}
					h := host.Host{Name: name, Mac: mac}
					hostsText = config.AppendHost(hostsText, h)
					hosts = append(hosts, h)
					fmt.Fprintf(a.Out, "  %s+ %s -> %s%s\n", a.color(ansiGreen), name, mac, a.color(ansiReset))
					added++
				}
			}
			if err := config.Save(a.HostsPath, hostsText); err != nil {
				return err
			}
			fmt.Fprintf(a.Out, "Done: %d added, %d skipped.\n", added, skipped)
			return nil
		},
	}
}
