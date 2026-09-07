package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/DiegoHeer/wakelab/internal/host"
	"github.com/spf13/cobra"
)

func newStatusCmd(a *App) *cobra.Command {
	var (
		jsonOut   bool
		watch     bool
		interval  int
		checkPort string
	)
	cmd := &cobra.Command{
		Use:   "status [target]",
		Short: "online/offline status (table); one host prints a line",
		Long: `wake status — show online/offline status

  wake status                 table of every host
  wake status <target>        table of a host, group, or 'all'
  wake status <host>          one host: prints a line, exit 0 online / 1 not
  wake status ... --json      machine-readable JSON (no colors)
  wake status ... --watch     refresh the table until you press Ctrl-C
  wake status ... --interval N   seconds between refreshes (default 2)
  wake status ... --port N    'online' = TCP port N is open (e.g. 22 for SSH)`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if checkPort != "" && !host.ValidPort(checkPort) {
				return fmt.Errorf("--port needs a number")
			}
			ctx := cmd.Context()
			_, hosts, err := a.loadHosts()
			if err != nil {
				return err
			}
			_, groups, err := a.loadGroups()
			if err != nil {
				return err
			}

			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			var names []string
			if target != "" {
				var ok bool
				names, ok = resolveNames(target, hosts, groups)
				if !ok {
					return fmt.Errorf("unknown host or group '%s'", target)
				}
			} else {
				names, _ = resolveNames("all", hosts, groups)
			}
			list := make([]host.Host, len(names))
			for i, n := range names {
				list[i] = findHost(hosts, n)
			}

			if jsonOut {
				return a.statusJSON(ctx, list, checkPort)
			}
			if watch {
				for {
					fmt.Fprint(a.Out, "\x1b[H\x1b[2J")
					fmt.Fprintf(a.Out, "wake status — every %ds (Ctrl-C to stop)\n", interval)
					a.statusTable(ctx, list, checkPort)
					a.Sleep(time.Duration(interval) * time.Second)
				}
			}
			// A single explicit host prints one line with a status exit code.
			if target != "" && target != "all" && !isGroup(groups, target) {
				st := a.statusOf(ctx, list[0], checkPort, false)
				fmt.Fprintf(a.Out, "%s is %s\n", target, a.paint(st))
				if st != "online" {
					return errSilent
				}
				return nil
			}
			a.statusTable(ctx, list, checkPort)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "machine-readable JSON output")
	cmd.Flags().BoolVar(&watch, "watch", false, "refresh the table until Ctrl-C")
	cmd.Flags().IntVar(&interval, "interval", 2, "seconds between --watch refreshes")
	cmd.Flags().StringVar(&checkPort, "port", "", "'online' = TCP port N is open")
	return cmd
}

func isGroup(groups []host.Group, name string) bool {
	for _, g := range groups {
		if g.Name == name {
			return true
		}
	}
	return false
}

func (a *App) statusTable(ctx context.Context, hosts []host.Host, checkPort string) {
	statuses := a.checkHosts(ctx, hosts, checkPort, false)
	fmt.Fprintf(a.Out, "%s%-10s %-15s %s%s\n", a.color(ansiBold), "HOST", "IP", "STATUS", a.color(ansiReset))
	for i, h := range hosts {
		fmt.Fprintf(a.Out, "%-10s %-15s %s\n", h.Name, a.hostIP(ctx, h.Name), a.paint(statuses[i]))
	}
}

// statusRow matches the Bash tool's JSON schema, key for key.
type statusRow struct {
	Host      string `json:"host"`
	Mac       string `json:"mac"`
	IP        string `json:"ip"`
	Broadcast string `json:"broadcast"`
	Port      any    `json:"port"`
	Via       string `json:"via"`
	Status    string `json:"status"`
	SSH       bool   `json:"ssh"`
}

func (a *App) statusJSON(ctx context.Context, hosts []host.Host, checkPort string) error {
	statuses := a.checkHosts(ctx, hosts, checkPort, false)
	rows := make([]statusRow, len(hosts))
	for i, h := range hosts {
		var port any = h.ReadyPort()
		if n, err := strconv.Atoi(h.ReadyPort()); err == nil {
			port = n
		}
		rows[i] = statusRow{
			Host: h.Name, Mac: h.Mac, IP: a.hostIP(ctx, h.Name),
			Broadcast: h.BroadcastAddr(), Port: port, Via: h.Via,
			Status: statuses[i], SSH: a.inSSHConfig(h.Name),
		}
	}
	data, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	fmt.Fprintln(a.Out, string(data))
	return nil
}
