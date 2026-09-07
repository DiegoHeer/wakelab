package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/DiegoHeer/wakelab/internal/config"
	"github.com/DiegoHeer/wakelab/internal/host"
	"github.com/DiegoHeer/wakelab/internal/sshexec"
	"github.com/spf13/cobra"
)

// checkNewName rejects a name that cannot become a new host.
func checkNewName(a *App, name string, hosts []host.Host, groups []host.Group) error {
	if name == "" {
		return fmt.Errorf("missing host name")
	}
	if strings.ContainsAny(name, " \t\n\r") {
		return fmt.Errorf("invalid host name '%s' (no whitespace allowed)", name)
	}
	if host.Reserved(name) {
		return fmt.Errorf("'%s' is a reserved word, pick another name", name)
	}
	if isGroup(groups, name) {
		return fmt.Errorf("'%s' is already a group name", name)
	}
	for _, h := range hosts {
		if h.Name == name {
			return fmt.Errorf("host '%s' already exists in %s", name, a.HostsPath)
		}
	}
	return nil
}

// macFromIP pings an IP (to fill the neighbor table), then reads its MAC from
// `ip neigh show` ("" when not found) — parity with the Bash mac_from_ip.
func (a *App) macFromIP(ctx context.Context, ip string) string {
	a.Prober.Ping(ip)
	stdout, _, err := a.Runner.Run(ctx, "ip", "neigh", "show", ip)
	if err != nil {
		return ""
	}
	fields := strings.Fields(stdout)
	for i, f := range fields {
		if f == "lladdr" && i+1 < len(fields) {
			return fields[i+1]
		}
	}
	return ""
}

func newAddCmd(a *App) *cobra.Command {
	var (
		viaSSH bool
		fromIP string
		rawMac string
		bcast  string
		port   string
		via    string
	)
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "add a host (--ssh | --ip <IP> | --mac <MAC>)",
		Long: `wake add — add a host to ~/.wol_hosts

  wake add <name> --ssh          get the MAC via the host's SSH-config IP
  wake add <name> --ip <IP>      get the MAC by pinging that IP
  wake add <name> --mac <MAC>    use a MAC address you provide

Optional extras (any source above):
  --broadcast <IP>   send the wake packet to this subnet broadcast
  --port <N>         readiness port for 'status --port' and 'wake --wait'
  --via <relay>      send the wake from this SSH host (a wired relay)

--ssh and --ip need the machine to be online right now.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := args[0]
			hostsText, hosts, err := a.loadHosts()
			if err != nil {
				return err
			}
			_, groups, err := a.loadGroups()
			if err != nil {
				return err
			}
			if err := checkNewName(a, name, hosts, groups); err != nil {
				return err
			}
			if port != "" && !host.ValidPort(port) {
				return fmt.Errorf("--port needs a number")
			}
			mac := rawMac
			switch {
			case viaSSH:
				ip := sshexec.SSHOption(ctx, a.Runner, name, "hostname")
				if ip == "" {
					return fmt.Errorf("no IP for '%s' in SSH config", name)
				}
				mac = a.macFromIP(ctx, ip)
				if mac == "" {
					return fmt.Errorf("could not read MAC for '%s' (%s). Is it online?", name, ip)
				}
			case fromIP != "":
				mac = a.macFromIP(ctx, fromIP)
				if mac == "" {
					return fmt.Errorf("could not read MAC from %s. Is it online?", fromIP)
				}
			}
			if mac == "" {
				return fmt.Errorf("usage: wake add <name> (--ssh | --ip <IP> | --mac <MAC>) [--broadcast <IP>] [--port <N>] [--via <relay>]")
			}
			mac = host.NormalizeMac(mac)
			if mac == "" {
				return fmt.Errorf("invalid MAC address")
			}
			if bcast != "" && !host.ValidIP(bcast) {
				return fmt.Errorf("invalid broadcast address '%s'", bcast)
			}
			h := host.Host{Name: name, Mac: mac, Broadcast: bcast, Port: port, Via: via}
			if err := config.Save(a.HostsPath, config.AppendHost(hostsText, h)); err != nil {
				return err
			}
			msg := fmt.Sprintf("Added '%s' -> %s", name, mac)
			if bcast != "" {
				msg += ", broadcast " + bcast
			}
			if port != "" {
				msg += ", port " + port
			}
			if via != "" {
				msg += ", via " + via
			}
			fmt.Fprintln(a.Out, msg)
			return nil
		},
	}
	cmd.Flags().BoolVar(&viaSSH, "ssh", false, "get the MAC via the host's SSH-config IP")
	cmd.Flags().StringVar(&fromIP, "ip", "", "get the MAC by pinging this IP")
	cmd.Flags().StringVar(&rawMac, "mac", "", "use this MAC address")
	cmd.Flags().StringVar(&bcast, "broadcast", "", "wake broadcast address")
	cmd.Flags().StringVar(&port, "port", "", "readiness TCP port")
	cmd.Flags().StringVar(&via, "via", "", "send the wake from this SSH relay")
	return cmd
}
