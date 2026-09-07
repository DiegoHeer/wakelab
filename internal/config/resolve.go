package config

import "github.com/DiegoHeer/wakelab/internal/host"

// Resolve expands a target (host name | group name | "all") to host names.
// It returns ok=false for an unknown target. Order of precedence matches the
// Bash tool: all, then groups, then hosts.
func Resolve(target string, hosts []host.Host, groups []host.Group) ([]string, bool) {
	if target == "all" {
		names := make([]string, len(hosts))
		for i, h := range hosts {
			names[i] = h.Name
		}
		return names, true
	}
	for _, g := range groups {
		if g.Name == target {
			return g.Members, true
		}
	}
	for _, h := range hosts {
		if h.Name == target {
			return []string{target}, true
		}
	}
	return nil, false
}
