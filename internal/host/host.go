// Package host defines the Host and Group model shared by all commands.
package host

import (
	"regexp"
	"strconv"
	"strings"
)

// Host is one machine from ~/.wol_hosts. Empty optional fields mean "unset";
// use BroadcastAddr and ReadyPort for the effective values.
type Host struct {
	Name      string
	Mac       string
	Broadcast string // empty = default 255.255.255.255
	Port      string // empty = default 22 (readiness port)
	Via       string // empty = send the wake locally
}

// BroadcastAddr returns the wake broadcast address, defaulted.
func (h Host) BroadcastAddr() string {
	if h.Broadcast == "" {
		return "255.255.255.255"
	}
	return h.Broadcast
}

// ReadyPort returns the readiness TCP port, defaulted.
func (h Host) ReadyPort() string {
	if h.Port == "" {
		return "22"
	}
	return h.Port
}

// Group is one named set of hosts from ~/.wol_groups.
type Group struct {
	Name    string
	Members []string
}

var reserved = map[string]bool{
	"ls": true, "status": true, "add": true, "rm": true, "import-ssh": true,
	"poweroff": true, "restart": true, "suspend": true, "group": true,
	"groups": true, "edit": true, "doctor": true, "schedule": true,
	"install-wol": true, "scan": true, "all": true, "help": true,
}

// Reserved reports whether name is a command word and cannot name a host or group.
func Reserved(name string) bool { return reserved[name] }

var macRe = regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`)

// NormalizeMac lower-cases a MAC and converts dashes to colons.
// It returns "" if the input is not a valid MAC address.
func NormalizeMac(s string) string {
	m := strings.ReplaceAll(s, "-", ":")
	if !macRe.MatchString(m) {
		return ""
	}
	return strings.ToLower(m)
}

// ValidIP reports whether s looks like a dotted-quad IPv4 address.
func ValidIP(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if p == "" || len(p) > 3 || strings.Trim(p, "0123456789") != "" {
			return false
		}
		n, _ := strconv.Atoi(p)
		if n > 255 {
			return false
		}
	}
	return true
}
