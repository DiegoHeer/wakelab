package cli

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// scanRunner fakes ip route/addr/neigh and nmap.
func scanRunner(neighOut string) *fakeRunner {
	f := &fakeRunner{}
	f.respond = func(name string, args []string) (string, string, error) {
		switch name + " " + strings.Join(args, " ") {
		case "ip -o -4 route show to default":
			return "default via 192.168.1.1 dev eth0 proto dhcp metric 100\n", "", nil
		case "ip -o -4 addr show dev eth0":
			return "2: eth0    inet 192.168.1.5/24 brd 192.168.1.255 scope global\n", "", nil
		case "ip neigh":
			return neighOut, "", nil
		}
		return "", "", nil
	}
	return f
}

func TestScanPingSweep(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.app.Runner = scanRunner("192.168.1.10 dev eth0 lladdr 44:87:63:80:5E:F6 REACHABLE\n192.168.1.20 dev eth0 lladdr aa:bb:cc:dd:ee:99 STALE\n")
	env.app.Prober = fakeProber{pings: map[string]bool{"192.168.1.10": true, "192.168.1.20": true}}
	env.app.ReverseDNS = func(ip string) string {
		if ip == "192.168.1.20" {
			return "printer"
		}
		return ""
	}

	if err := env.run("scan"); err != nil {
		t.Fatalf("scan: %v", err)
	}
	out := env.out.String()
	// .10 has server's MAC -> known; .20 is new with a reverse-DNS name.
	mustContain(t, out, "IP", "MAC", "NAME", "KNOWN",
		"192.168.1.10", "server", "yes",
		"192.168.1.20", "aa:bb:cc:dd:ee:99", "printer", "NEW",
		"2 device(s), 1 new.")
}

func TestScanJSON(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.app.Runner = scanRunner("192.168.1.20 dev eth0 lladdr aa:bb:cc:dd:ee:99 STALE\n")
	env.app.Prober = fakeProber{pings: map[string]bool{"192.168.1.20": true}}
	env.app.ReverseDNS = func(string) string { return "" }

	if err := env.run("scan", "--subnet", "192.168.1.0/24", "--json"); err != nil {
		t.Fatalf("scan --json: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(env.out.Bytes(), &rows); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, env.out.String())
	}
	if len(rows) != 1 || rows[0]["ip"] != "192.168.1.20" || rows[0]["known"] != false || rows[0]["name"] != "(unknown)" {
		t.Errorf("rows = %v", rows)
	}
}

func TestScanNmap(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	runner := scanRunner("")
	inner := runner.respond
	runner.respond = func(name string, args []string) (string, string, error) {
		if name == "nmap" {
			return "Starting Nmap\nNmap scan report for 192.168.1.30\nHost is up.\nNmap done\n", "", nil
		}
		return inner(name, args)
	}
	env.app.Runner = runner
	env.app.LookPath = func(name string) (string, error) {
		if name == "nmap" {
			return "/usr/bin/nmap", nil
		}
		return "", errors.New("not found")
	}
	env.app.ReverseDNS = func(string) string { return "" }

	if err := env.run("scan", "--subnet", "192.168.1.0/24"); err != nil {
		t.Fatalf("scan nmap: %v", err)
	}
	mustContain(t, env.out.String(), "192.168.1.30", "1 device(s), 1 new.")
}

func TestScanNoDevices(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.app.Runner = scanRunner("")
	env.app.ReverseDNS = func(string) string { return "" }
	if err := env.run("scan", "--subnet", "10.9.8.0/24"); err != nil {
		t.Fatalf("scan: %v", err)
	}
	mustContain(t, env.out.String(), "No devices found on 10.9.8.0/24.")
}
