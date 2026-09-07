package cli

import "testing"

func TestLs(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.writeSSHConfig(t, "Host server proxmox\n    User root\n")

	if err := env.run("ls"); err != nil {
		t.Fatalf("ls: %v", err)
	}
	out := env.out.String()
	mustContain(t, out,
		"HOST", "MAC", "PORT", "BROADCAST", "VIA", "SSH",
		"server", "44:87:63:80:5e:f6", "yes",
		"desktop", "proxmox",
		"pi", "192.168.1.255", "80",
	)
	// desktop is not in the ssh config.
	lines := splitLines(out)
	for _, l := range lines {
		if len(l) > 7 && l[:7] == "desktop" && !contains(l, "no") {
			t.Errorf("desktop should show ssh=no: %q", l)
		}
	}
	// Defaults are rendered: server has port 22, broadcast 255.255.255.255, via "-".
	for _, l := range lines {
		if len(l) > 6 && l[:6] == "server" {
			mustContain(t, l, "22", "255.255.255.255", "-")
		}
	}
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return out
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
