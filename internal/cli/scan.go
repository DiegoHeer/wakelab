package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

func newScanCmd(a *App) *cobra.Command {
	var (
		subnet  string
		jsonOut bool
	)
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "find devices on your LAN (marks which are new)",
		Long: `wake scan — discover devices on your local network

  wake scan                     scan your default subnet
  wake scan --subnet <CIDR>     scan a specific subnet (e.g. 192.168.1.0/24)
  wake scan --json              machine-readable output

Uses nmap if installed, otherwise a parallel ping-sweep. Shows each device's
IP, MAC, name (reverse-DNS, or your host name if known), and whether it is
already in ~/.wol_hosts. It only lists — add a new one yourself with:
  wake add <name> --mac <MAC>`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			ip4, prefix, err := a.scanSubnet(ctx, subnet)
			if err != nil {
				return err
			}
			base3 := ip4[:strings.LastIndex(ip4, ".")]
			if prefix < 24 && !jsonOut {
				fmt.Fprintf(a.Err, "note: /%d is large; scanning %s.0/24 only\n", prefix, base3)
			}
			cidr := base3 + ".0/24"

			method := "ping"
			if _, err := a.LookPath("nmap"); err == nil {
				method = "nmap"
			}
			if !jsonOut {
				fmt.Fprintf(a.Err, "Scanning %s (%s)...\n", cidr, method)
			}

			var live []string
			if method == "nmap" {
				stdout, _, _ := a.Runner.Run(ctx, "nmap", "-sn", "-n", cidr)
				for _, line := range strings.Split(stdout, "\n") {
					if strings.HasPrefix(line, "Nmap scan report for ") {
						fields := strings.Fields(line)
						live = append(live, fields[len(fields)-1])
					}
				}
				sort.Slice(live, func(i, j int) bool { return lastOctet(live[i]) < lastOctet(live[j]) })
			} else {
				var mu sync.Mutex
				var wg sync.WaitGroup
				sem := make(chan struct{}, 64) // cap like the Bash xargs -P64
				for i := 1; i <= 254; i++ {
					wg.Add(1)
					go func(ip string) {
						defer wg.Done()
						sem <- struct{}{}
						defer func() { <-sem }()
						if a.Prober.Ping(ip) {
							mu.Lock()
							live = append(live, ip)
							mu.Unlock()
						}
					}(fmt.Sprintf("%s.%d", base3, i))
				}
				wg.Wait()
				sort.Slice(live, func(i, j int) bool { return lastOctet(live[i]) < lastOctet(live[j]) })
			}

			// IP -> MAC from the neighbor table; known MAC -> our host name.
			neigh := map[string]string{}
			stdout, _, _ := a.Runner.Run(ctx, "ip", "neigh")
			for _, line := range strings.Split(stdout, "\n") {
				fields := strings.Fields(line)
				for i, f := range fields {
					if f == "lladdr" && i+1 < len(fields) {
						neigh[fields[0]] = strings.ToLower(fields[i+1])
					}
				}
			}
			_, hosts, err := a.loadHosts()
			if err != nil {
				return err
			}
			knownMac := map[string]string{}
			for _, h := range hosts {
				if h.Mac != "" {
					knownMac[strings.ToLower(h.Mac)] = h.Name
				}
			}

			type row struct {
				IP    string `json:"ip"`
				Mac   string `json:"mac"`
				Name  string `json:"name"`
				Known bool   `json:"known"`
			}
			var rows []row
			newCount := 0
			for _, ip := range live {
				mac := neigh[ip]
				r := row{IP: ip, Mac: mac}
				if mac != "" && knownMac[mac] != "" {
					r.Name = knownMac[mac]
					r.Known = true
				} else {
					newCount++
					r.Name = a.ReverseDNS(ip)
					if r.Name == "" {
						r.Name = "(unknown)"
					}
				}
				if r.Mac == "" {
					r.Mac = "?"
				}
				rows = append(rows, r)
			}

			if jsonOut {
				data, err := json.Marshal(rows)
				if err != nil {
					return err
				}
				if len(rows) == 0 {
					data = []byte("[]")
				}
				fmt.Fprintln(a.Out, string(data))
				return nil
			}
			if len(rows) == 0 {
				fmt.Fprintf(a.Out, "No devices found on %s.\n", cidr)
				return nil
			}
			fmt.Fprintf(a.Out, "%s%-15s %-18s %-16s %s%s\n", a.color(ansiBold), "IP", "MAC", "NAME", "KNOWN", a.color(ansiReset))
			for _, r := range rows {
				known := a.color(ansiYellow) + "NEW" + a.color(ansiReset)
				if r.Known {
					known = a.color(ansiGreen) + "yes" + a.color(ansiReset)
				}
				fmt.Fprintf(a.Out, "%-15s %-18s %-16s %s\n", r.IP, r.Mac, r.Name, known)
			}
			fmt.Fprintf(a.Out, "%d device(s), %d new. Add one with: wake add <name> --mac <MAC>\n", len(rows), newCount)
			return nil
		},
	}
	cmd.Flags().StringVar(&subnet, "subnet", "", "subnet to scan (CIDR, e.g. 192.168.1.0/24)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "machine-readable output")
	return cmd
}

// scanSubnet returns the base IPv4 and prefix to scan, from --subnet or the
// default route (parity with the Bash subnet autodetection).
func (a *App) scanSubnet(ctx context.Context, flag string) (string, int, error) {
	cidr := flag
	if cidr == "" {
		stdout, _, _ := a.Runner.Run(ctx, "ip", "-o", "-4", "route", "show", "to", "default")
		dev := ""
		fields := strings.Fields(stdout)
		for i, f := range fields {
			if f == "dev" && i+1 < len(fields) {
				dev = fields[i+1]
				break
			}
		}
		if dev == "" {
			return "", 0, fmt.Errorf("could not detect the default network device; pass --subnet <CIDR>")
		}
		stdout, _, _ = a.Runner.Run(ctx, "ip", "-o", "-4", "addr", "show", "dev", dev)
		for _, f := range strings.Fields(stdout) {
			if strings.Contains(f, "/") && strings.Count(f, ".") == 3 {
				cidr = f
				break
			}
		}
		if cidr == "" {
			return "", 0, fmt.Errorf("could not detect a subnet on '%s'; pass --subnet <CIDR>", dev)
		}
	}
	parts := strings.SplitN(cidr, "/", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid subnet '%s' (expected CIDR like 192.168.1.0/24)", cidr)
	}
	prefix, err := strconv.Atoi(parts[1])
	if err != nil || !strings.Contains(parts[0], ".") {
		return "", 0, fmt.Errorf("invalid subnet '%s' (expected CIDR like 192.168.1.0/24)", cidr)
	}
	return parts[0], prefix, nil
}

func lastOctet(ip string) int {
	n, _ := strconv.Atoi(ip[strings.LastIndex(ip, ".")+1:])
	return n
}
