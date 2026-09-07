package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/DiegoHeer/wakelab/internal/sshexec"
	"github.com/spf13/cobra"
)

// newPowerCmd builds poweroff, restart, or suspend — they differ only in the
// remote command and wording (parity with the Bash tool).
func newPowerCmd(a *App, use, remoteCmd, pretty, extra string) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   use + " <target>",
		Short: pretty + " a host, group, or 'all' over SSH",
		Long: fmt.Sprintf(`wake %s — %s a host, group, or 'all' over SSH

  wake %s <target> [-y|--yes]
%s
Every targeted host must be in ~/.ssh/config. Asks for confirmation
unless you pass -y. Uses sudo automatically when your SSH user is not root.`,
			use, strings.ToLower(pretty), use, extra),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			target := args[0]
			_, hosts, err := a.loadHosts()
			if err != nil {
				return err
			}
			_, groups, err := a.loadGroups()
			if err != nil {
				return err
			}
			names, ok := resolveNames(target, hosts, groups)
			if !ok {
				return fmt.Errorf("unknown host or group '%s'", target)
			}
			if len(names) == 0 {
				return fmt.Errorf("target '%s' has no hosts", target)
			}
			var missing []string
			for _, n := range names {
				if !a.inSSHConfig(n) {
					missing = append(missing, n)
				}
			}
			if len(missing) > 0 {
				return fmt.Errorf("not in ~/.ssh/config: %s — %s needs SSH. Add them to ~/.ssh/config first.",
					strings.Join(missing, " "), pretty)
			}
			if !yes {
				fmt.Fprintf(a.Out, "%s these host(s): %s ? [y/N] ", pretty, strings.Join(names, " "))
				reply, _ := bufio.NewReader(a.Stdin).ReadString('\n')
				reply = strings.TrimSpace(reply)
				if reply != "y" && reply != "Y" {
					fmt.Fprintln(a.Out, "Cancelled.")
					return nil
				}
			}
			for _, n := range names {
				remote := remoteCmd
				if sshexec.SSHOption(ctx, a.Runner, n, "user") != "root" {
					remote = "sudo " + remoteCmd
				}
				fmt.Fprintf(a.Out, "%s '%s'...\n", pretty, n)
				if err := a.Runner.RunTTY(ctx, "ssh", "-t", n, remote); err != nil {
					fmt.Fprintf(a.Out, "%s  (failed on %s)%s\n", a.color(ansiRed), n, a.color(ansiReset))
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation prompt")
	return cmd
}

func newPoweroffCmd(a *App) *cobra.Command {
	return newPowerCmd(a, "poweroff", "poweroff", "Power off", "")
}

func newRestartCmd(a *App) *cobra.Command {
	return newPowerCmd(a, "restart", "reboot", "Restart", "")
}

func newSuspendCmd(a *App) *cobra.Command {
	return newPowerCmd(a, "suspend", "systemctl suspend", "Suspend",
		"\nRuns 'systemctl suspend' on each host. Wake it again later with 'wake <target>'.\n")
}
