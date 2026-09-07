package cli

import (
	"encoding/json"
	"testing"
)

func TestStatusTableAllHosts(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.app.Runner = sshGRunner(map[string]string{
		"server": "192.168.1.10", "desktop": "192.168.1.11",
	}, nil)
	env.app.Prober = fakeProber{pings: map[string]bool{"192.168.1.10": true}}

	if err := env.run("status"); err != nil {
		t.Fatalf("status: %v", err)
	}
	out := env.out.String()
	mustContain(t, out, "HOST", "IP", "STATUS", "192.168.1.10", "online", "offline")
	// pi has no ssh -G entry -> unknown.
	mustContain(t, out, "unknown")
}

func TestStatusSingleHostOnlineExitsZero(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.app.Runner = sshGRunner(map[string]string{"server": "192.168.1.10"}, nil)
	env.app.Prober = fakeProber{pings: map[string]bool{"192.168.1.10": true}}

	if err := env.run("status", "server"); err != nil {
		t.Fatalf("status server: %v", err)
	}
	mustContain(t, env.out.String(), "server is online")
}

func TestStatusSingleHostOfflineExitsNonZero(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.app.Runner = sshGRunner(map[string]string{"server": "192.168.1.10"}, nil)
	env.app.Prober = fakeProber{}

	err := env.run("status", "server")
	if err == nil {
		t.Fatal("offline host must exit non-zero")
	}
	if err.Error() != "" {
		t.Errorf("offline exit must be silent, got %q", err.Error())
	}
	mustContain(t, env.out.String(), "server is offline")
}

func TestStatusUnknownTarget(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	if err := env.run("status", "ghost"); err == nil {
		t.Fatal("unknown target must error")
	}
}

func TestStatusJSON(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.app.Runner = sshGRunner(map[string]string{"server": "192.168.1.10"}, nil)
	env.app.Prober = fakeProber{pings: map[string]bool{"192.168.1.10": true}}
	env.writeSSHConfig(t, "Host server\n")

	if err := env.run("status", "server", "--json"); err != nil {
		t.Fatalf("status --json: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(env.out.Bytes(), &rows); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, env.out.String())
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	row := rows[0]
	want := map[string]any{
		"host": "server", "mac": "44:87:63:80:5e:f6", "ip": "192.168.1.10",
		"broadcast": "255.255.255.255", "via": "", "status": "online",
	}
	for k, v := range want {
		if row[k] != v {
			t.Errorf("%s = %v, want %v", k, row[k], v)
		}
	}
	if row["port"] != float64(22) {
		t.Errorf("port = %v (%T), want number 22", row["port"], row["port"])
	}
	if row["ssh"] != true {
		t.Errorf("ssh = %v, want true", row["ssh"])
	}
}

func TestStatusPortOverride(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.app.Runner = sshGRunner(map[string]string{"server": "192.168.1.10"}, nil)
	// Ping says down, but port 8080 is open: --port must win.
	env.app.Prober = fakeProber{ports: map[string]bool{"192.168.1.10:8080": true}}

	if err := env.run("status", "server", "--port", "8080"); err != nil {
		t.Fatalf("status --port: %v", err)
	}
	mustContain(t, env.out.String(), "server is online")
}
