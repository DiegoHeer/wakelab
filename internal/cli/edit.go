package cli

import (
	"fmt"

	"github.com/DiegoHeer/wakelab/internal/config"
	"github.com/DiegoHeer/wakelab/internal/host"
	"github.com/spf13/cobra"
)

func newEditCmd(a *App) *cobra.Command {
	var (
		newName  string
		newMac   string
		newBcast string
		newPort  string
		newVia   string
	)
	cmd := &cobra.Command{
		Use:   "edit <host>",
		Short: "change a host (--name, --mac, --broadcast, --port, --via)",
		Long: `wake edit — change an existing host

  wake edit <host> --name <new>       rename the host
  wake edit <host> --mac <MAC>        change its MAC
  wake edit <host> --broadcast <IP>   change its wake broadcast address
  wake edit <host> --port <N>         change its readiness port
  wake edit <host> --via <relay>      send its wake from this SSH host
  wake edit <host> --via ""           clear the relay (send locally again)
  (combine any of these in one command)

Renaming also updates the host inside any groups.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			hostsText, hosts, err := a.loadHosts()
			if err != nil {
				return err
			}
			groupsText, groups, err := a.loadGroups()
			if err != nil {
				return err
			}
			cur := findHost(hosts, name)
			if cur.Mac == "" && !hostExists(hosts, name) {
				return fmt.Errorf("host '%s' not found in %s", name, a.HostsPath)
			}
			viaSet := cmd.Flags().Changed("via")
			if newName == "" && newMac == "" && newBcast == "" && newPort == "" && !viaSet {
				return fmt.Errorf("nothing to change (use --name, --mac, --broadcast, --port and/or --via)")
			}

			final := cur
			if newName != "" && newName != name {
				if err := checkNewName(a, newName, hosts, groups); err != nil {
					return err
				}
				final.Name = newName
			}
			if newMac != "" {
				final.Mac = host.NormalizeMac(newMac)
				if final.Mac == "" {
					return fmt.Errorf("invalid MAC address")
				}
			}
			if newBcast != "" {
				if !host.ValidIP(newBcast) {
					return fmt.Errorf("invalid broadcast address '%s'", newBcast)
				}
				final.Broadcast = newBcast
			}
			if newPort != "" {
				if !host.ValidPort(newPort) {
					return fmt.Errorf("--port needs a number")
				}
				final.Port = newPort
			}
			if viaSet {
				final.Via = newVia
			}

			out := config.AppendHost(config.RemoveHost(hostsText, name), final)
			if err := config.Save(a.HostsPath, out); err != nil {
				return err
			}
			if final.Name != name {
				if err := config.Save(a.GroupsPath, config.RenameMember(groupsText, name, final.Name)); err != nil {
					return err
				}
			}
			msg := fmt.Sprintf("Updated '%s' -> name=%s mac=%s", name, final.Name, final.Mac)
			if final.Broadcast != "" {
				msg += " broadcast=" + final.Broadcast
			}
			if final.Port != "" {
				msg += " port=" + final.Port
			}
			if final.Via != "" {
				msg += " via=" + final.Via
			}
			fmt.Fprintln(a.Out, msg)
			return nil
		},
	}
	cmd.Flags().StringVar(&newName, "name", "", "new host name")
	cmd.Flags().StringVar(&newMac, "mac", "", "new MAC address")
	cmd.Flags().StringVar(&newBcast, "broadcast", "", "new wake broadcast address")
	cmd.Flags().StringVar(&newPort, "port", "", "new readiness TCP port")
	cmd.Flags().StringVar(&newVia, "via", "", "new SSH relay ('' clears it)")
	return cmd
}

func hostExists(hosts []host.Host, name string) bool {
	for _, h := range hosts {
		if h.Name == name {
			return true
		}
	}
	return false
}
