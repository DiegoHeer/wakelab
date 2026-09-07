// Package config reads and edits ~/.wol_hosts and ~/.wol_groups.
// Edits are text-preserving: they only append or remove whole blocks/lines,
// never rewrite unrelated content.
package config

import (
	"fmt"
	"strings"

	"github.com/DiegoHeer/wakelab/internal/host"
)

// ParseHosts parses ssh-config-style Host blocks. Keys are case-insensitive;
// within a block the first occurrence of a key wins.
func ParseHosts(text string) []host.Host {
	var hosts []host.Host
	cur := -1
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.ToLower(fields[0])
		if key == "host" {
			hosts = append(hosts, host.Host{Name: fields[1]})
			cur = len(hosts) - 1
			continue
		}
		if cur < 0 {
			continue
		}
		h := &hosts[cur]
		switch key {
		case "mac":
			if h.Mac == "" {
				h.Mac = fields[1]
			}
		case "broadcast":
			if h.Broadcast == "" {
				h.Broadcast = fields[1]
			}
		case "port":
			if h.Port == "" {
				h.Port = fields[1]
			}
		case "via":
			if h.Via == "" {
				h.Via = fields[1]
			}
		}
	}
	return hosts
}

// FindHost returns the first block named name.
func FindHost(text, name string) (host.Host, bool) {
	for _, h := range ParseHosts(text) {
		if h.Name == name {
			return h, true
		}
	}
	return host.Host{}, false
}

// AppendHost appends a Host block in the same format the Bash tool wrote.
func AppendHost(text string, h host.Host) string {
	var b strings.Builder
	b.WriteString(text)
	fmt.Fprintf(&b, "Host %s\n", h.Name)
	fmt.Fprintf(&b, "    %-11s%s\n", "Mac", h.Mac)
	if h.Broadcast != "" {
		fmt.Fprintf(&b, "    %-11s%s\n", "Broadcast", h.Broadcast)
	}
	if h.Port != "" {
		fmt.Fprintf(&b, "    %-11s%s\n", "Port", h.Port)
	}
	if h.Via != "" {
		fmt.Fprintf(&b, "    %-11s%s\n", "Via", h.Via)
	}
	b.WriteString("\n")
	return b.String()
}

// RemoveHost removes every block named name (the Host line, its keys, and
// blank lines inside the block) and keeps all other lines byte-for-byte.
func RemoveHost(text, name string) string {
	var out []string
	inBlock := false
	lines := strings.Split(text, "\n")
	// Split leaves one trailing empty element for text ending in \n; keep it
	// so Join restores the original trailing newline.
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.EqualFold(fields[0], "host") {
			inBlock = fields[1] == name
		}
		// Comment lines are never part of a block: removing a host must not
		// eat the user's comments (a small improvement over the Bash tool).
		isComment := len(fields) > 0 && strings.HasPrefix(fields[0], "#")
		if !inBlock || isComment {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
