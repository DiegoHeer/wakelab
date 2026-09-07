package config

import (
	"strings"

	"github.com/DiegoHeer/wakelab/internal/host"
)

// ParseGroups parses "name member..." lines; '#' starts a comment line.
func ParseGroups(text string) []host.Group {
	var groups []host.Group
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		groups = append(groups, host.Group{Name: fields[0], Members: fields[1:]})
	}
	return groups
}

// FindGroup returns the first group named name.
func FindGroup(text, name string) (host.Group, bool) {
	for _, g := range ParseGroups(text) {
		if g.Name == name {
			return g, true
		}
	}
	return host.Group{}, false
}

// AppendGroup appends one "name member..." line.
func AppendGroup(text, name string, members []string) string {
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return text + name + " " + strings.Join(members, " ") + "\n"
}

// RemoveGroup drops every line whose first field is name; other lines are kept
// byte-for-byte.
func RemoveGroup(text, name string) string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == name {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// RenameMember replaces oldName with newName in the member columns (2..N) of every
// group line. Lines without a match keep their exact text.
func RenameMember(text, oldName, newName string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		changed := false
		for j := 1; j < len(fields); j++ {
			if fields[j] == oldName {
				fields[j] = newName
				changed = true
			}
		}
		if changed {
			lines[i] = strings.Join(fields, " ")
		}
	}
	return strings.Join(lines, "\n")
}
