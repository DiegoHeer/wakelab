package config

import (
	"os"
	"path/filepath"
)

// HostsPath returns the hosts config path (~/.wol_hosts).
func HostsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".wol_hosts")
}

// GroupsPath returns the groups config path (~/.wol_groups).
func GroupsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".wol_groups")
}

// Load reads a config file, creating it empty when missing (touch semantics,
// parity with the Bash tool).
func Load(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if werr := os.WriteFile(path, nil, 0o644); werr != nil {
			return "", werr
		}
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Save writes a config file.
func Save(path, text string) error {
	return os.WriteFile(path, []byte(text), 0o644)
}
