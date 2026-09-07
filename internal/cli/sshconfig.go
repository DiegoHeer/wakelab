package cli

import (
	"os"
	"strings"
)

// inSSHConfig reports whether name appears as an exact Host entry in the
// user's ~/.ssh/config (parity with the Bash in_ssh_config).
func (a *App) inSSHConfig(name string) bool {
	data, err := os.ReadFile(a.SSHConfigPath)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "Host" {
			continue
		}
		for _, f := range fields[1:] {
			if f == name {
				return true
			}
		}
	}
	return false
}
