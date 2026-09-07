package cli

import (
	"strings"
	"testing"
)

func TestPoweroffRequiresSSHConfig(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.writeSSHConfig(t, "Host server\n")
	err := env.run("poweroff", "lab", "-y")
	if err == nil || !strings.Contains(err.Error(), "not in ~/.ssh/config: pi") {
		t.Errorf("err = %v", err)
	}
}

func TestPoweroffSudoForNonRoot(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.writeSSHConfig(t, "Host server\n")
	runner := sshGRunner(map[string]string{"server": "192.168.1.10"}, map[string]string{"server": "diego"})
	env.app.Runner = runner

	if err := env.run("poweroff", "server", "-y"); err != nil {
		t.Fatalf("poweroff: %v", err)
	}
	var tty []string
	for _, c := range runner.calls {
		if c[0] == "ssh" && c[1] == "-t" {
			tty = c
		}
	}
	if len(tty) != 4 || tty[2] != "server" || tty[3] != "sudo poweroff" {
		t.Errorf("ssh -t call = %v, want sudo poweroff", tty)
	}
	mustContain(t, env.out.String(), "Power off 'server'...")
}

func TestRestartNoSudoForRoot(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.writeSSHConfig(t, "Host server\n")
	runner := sshGRunner(map[string]string{"server": "192.168.1.10"}, map[string]string{"server": "root"})
	env.app.Runner = runner

	if err := env.run("restart", "server", "-y"); err != nil {
		t.Fatalf("restart: %v", err)
	}
	found := false
	for _, c := range runner.calls {
		if c[0] == "ssh" && len(c) == 4 && c[1] == "-t" && c[3] == "reboot" {
			found = true
		}
	}
	if !found {
		t.Errorf("calls = %v, want ssh -t server reboot", runner.calls)
	}
}

func TestSuspendCommand(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.writeSSHConfig(t, "Host server\n")
	runner := sshGRunner(map[string]string{"server": "192.168.1.10"}, map[string]string{"server": "diego"})
	env.app.Runner = runner

	if err := env.run("suspend", "server", "--yes"); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	found := false
	for _, c := range runner.calls {
		if c[0] == "ssh" && len(c) == 4 && c[3] == "sudo systemctl suspend" {
			found = true
		}
	}
	if !found {
		t.Errorf("calls = %v, want sudo systemctl suspend", runner.calls)
	}
}

func TestPoweroffConfirmationDeclined(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.writeSSHConfig(t, "Host server\n")
	runner := sshGRunner(map[string]string{"server": "192.168.1.10"}, nil)
	env.app.Runner = runner
	env.app.Stdin = strings.NewReader("n\n")

	if err := env.run("poweroff", "server"); err != nil {
		t.Fatalf("declined poweroff must exit 0, got %v", err)
	}
	mustContain(t, env.out.String(), "Cancelled.")
	for _, c := range runner.calls {
		if c[0] == "ssh" && c[1] == "-t" {
			t.Errorf("declined poweroff must not ssh -t: %v", c)
		}
	}
}

func TestPoweroffConfirmationAccepted(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.writeSSHConfig(t, "Host server\n")
	runner := sshGRunner(map[string]string{"server": "192.168.1.10"}, nil)
	env.app.Runner = runner
	env.app.Stdin = strings.NewReader("y\n")

	if err := env.run("poweroff", "server"); err != nil {
		t.Fatalf("poweroff: %v", err)
	}
	mustContain(t, env.out.String(), "Power off these host(s): server ? [y/N]")
}
