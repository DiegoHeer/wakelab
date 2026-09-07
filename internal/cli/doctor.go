package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/DiegoHeer/wakelab/internal/host"
	"github.com/spf13/cobra"
)

// relayProbe checks (on the relay, over ssh) whether any WOL sender exists.
const relayProbe = `if command -v wake >/dev/null 2>&1; then echo WOL_OK; ` +
	`elif command -v wakeonlan >/dev/null 2>&1; then echo WOL_OK; ` +
	`elif command -v python3 >/dev/null 2>&1; then echo WOL_OK; ` +
	`else echo WOL_NO; fi`

func newDoctorCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "check the config for problems",
		Long: `wake doctor — check the config for problems

Checks: duplicate names, invalid MACs/broadcast/port, hosts missing from
~/.ssh/config, group members that aren't known hosts, and name clashes.
Also SSHes each relay (Via) to check it is reachable and has a WOL sender
(wake, wakeonlan, or python3 — this step does network I/O and may wait a
few seconds if a relay is down). Exits non-zero if it finds an error.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			problems, warnings := 0, 0
			ok := func(format string, args ...any) {
				fmt.Fprintf(a.Out, "%sok  %s %s\n", a.color(ansiGreen), a.color(ansiReset), fmt.Sprintf(format, args...))
			}
			warn := func(format string, args ...any) {
				fmt.Fprintf(a.Out, "%swarn%s %s\n", a.color(ansiYellow), a.color(ansiReset), fmt.Sprintf(format, args...))
				warnings++
			}
			errf := func(format string, args ...any) {
				fmt.Fprintf(a.Out, "%sERR %s %s\n", a.color(ansiRed), a.color(ansiReset), fmt.Sprintf(format, args...))
				problems++
			}

			_, hosts, err := a.loadHosts()
			if err != nil {
				return err
			}
			groupsText, groups, err := a.loadGroups()
			if err != nil {
				return err
			}

			ok("Wake-on-LAN is native (no wakeonlan needed on this machine)")

			seen := map[string]int{}
			for _, h := range hosts {
				seen[h.Name]++
			}
			var dups []string
			for name, n := range seen {
				if n > 1 {
					dups = append(dups, name)
				}
			}
			sort.Strings(dups)
			if len(dups) == 0 {
				ok("no duplicate host names")
			} else {
				errf("duplicate host names: %s", strings.Join(dups, " "))
			}

			badValues := false
			for _, h := range hosts {
				if host.NormalizeMac(h.Mac) == "" {
					errf("invalid or missing MAC for '%s': %s", h.Name, h.Mac)
					badValues = true
				}
				if h.Broadcast != "" && !host.ValidIP(h.Broadcast) {
					errf("invalid broadcast for '%s': %s", h.Name, h.Broadcast)
					badValues = true
				}
				if h.Port != "" && !host.ValidPort(h.Port) {
					errf("invalid port for '%s': %s", h.Name, h.Port)
					badValues = true
				}
			}
			if !badValues {
				ok("all MAC/broadcast/port values look valid")
			}

			for _, h := range hosts {
				if !a.inSSHConfig(h.Name) {
					warn("'%s' is not in ~/.ssh/config (status/poweroff/restart won't work)", h.Name)
				}
				if h.Via != "" && !a.inSSHConfig(h.Via) {
					warn("'%s' relays via '%s', which is not in ~/.ssh/config (can't SSH to it)", h.Name, h.Via)
				}
			}

			for _, g := range groups {
				if hostExists(hosts, g.Name) {
					errf("group '%s' clashes with a host of the same name", g.Name)
				}
				for _, m := range g.Members {
					if !hostExists(hosts, m) {
						errf("group '%s' has unknown member '%s'", g.Name, m)
					}
				}
			}
			if strings.TrimSpace(groupsText) != "" {
				ok("groups checked")
			}

			// Probe each unique relay over SSH: reachable? has a WOL sender?
			relaySet := map[string]bool{}
			var relays []string
			for _, h := range hosts {
				if h.Via != "" && !relaySet[h.Via] {
					relaySet[h.Via] = true
					relays = append(relays, h.Via)
				}
			}
			sort.Strings(relays)
			for _, r := range relays {
				stdout, _, err := a.Runner.Run(ctx, "ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", r, relayProbe)
				switch {
				case err != nil:
					warn("relay '%s' not reachable over SSH now (is it on? key known? in ~/.ssh/config?)", r)
				case strings.Contains(stdout, "WOL_OK"):
					ok("relay '%s' reachable, has a WOL sender", r)
				default:
					errf("relay '%s' reachable but has no WOL sender (wake/wakeonlan/python3)", r)
				}
			}

			fmt.Fprintln(a.Out)
			if problems > 0 {
				fmt.Fprintf(a.Out, "%s%d error(s)%s, %d warning(s).\n", a.color(ansiRed), problems, a.color(ansiReset), warnings)
				return errSilent
			}
			fmt.Fprintf(a.Out, "%sNo errors%s, %d warning(s).\n", a.color(ansiGreen), a.color(ansiReset), warnings)
			return nil
		},
	}
}
