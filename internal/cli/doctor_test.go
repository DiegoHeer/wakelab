package cli

import (
	"errors"
	"strings"
	"testing"
)

func TestDoctorCleanConfig(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.writeSSHConfig(t, "Host server desktop pi proxmox\n")
	// desktop relays via proxmox; the relay probe reports a sender present.
	runner := &fakeRunner{}
	runner.respond = func(name string, args []string) (string, string, error) {
		if name == "ssh" && len(args) > 0 && args[len(args)-2] == "proxmox" {
			return "WOL_OK\n", "", nil
		}
		return "", "", nil
	}
	env.app.Runner = runner

	if err := env.run("doctor"); err != nil {
		t.Fatalf("doctor: %v", err)
	}
	out := env.out.String()
	mustContain(t, out,
		"no duplicate host names",
		"all MAC/broadcast/port values look valid",
		"groups checked",
		"relay 'proxmox' reachable",
		"No errors")
}

func TestDoctorFindsProblems(t *testing.T) {
	badHosts := `Host a
    Mac        not-a-mac

Host a
    Mac        11:22:33:44:55:66

Host b
    Mac        11:22:33:44:55:67
    Broadcast  999.1.1.1
    Port       x7
`
	env := newTestEnv(t, badHosts, "g1 a ghost\nb 11\n")
	env.writeSSHConfig(t, "Host a\n")

	err := env.run("doctor")
	if err == nil {
		t.Fatal("doctor must exit non-zero on errors")
	}
	if err.Error() != "" {
		t.Errorf("doctor exit must be silent, got %q", err.Error())
	}
	out := env.out.String()
	mustContain(t, out,
		"duplicate host names: a",
		"invalid or missing MAC for 'a'",
		"invalid broadcast for 'b': 999.1.1.1",
		"invalid port for 'b': x7",
		"'b' is not in ~/.ssh/config",
		"group 'g1' has unknown member 'ghost'",
		"group 'b' clashes with a host of the same name",
		"error(s)")
}

func TestDoctorRelayMissingSender(t *testing.T) {
	env := newTestEnv(t, envHosts, "")
	env.writeSSHConfig(t, "Host server desktop pi proxmox\n")
	runner := &fakeRunner{}
	runner.respond = func(name string, _ []string) (string, string, error) {
		if name == "ssh" {
			return "WOL_NO\n", "", nil
		}
		return "", "", nil
	}
	env.app.Runner = runner

	if err := env.run("doctor"); err == nil {
		t.Fatal("missing relay sender must be an error")
	}
	mustContain(t, env.out.String(), "relay 'proxmox' reachable but has no WOL sender")
}

func TestDoctorRelayUnreachableIsWarning(t *testing.T) {
	env := newTestEnv(t, envHosts, "")
	env.writeSSHConfig(t, "Host server desktop pi proxmox\n")
	runner := &fakeRunner{}
	runner.respond = func(name string, _ []string) (string, string, error) {
		if name == "ssh" {
			return "", "", errors.New("connection refused")
		}
		return "", "", nil
	}
	env.app.Runner = runner

	if err := env.run("doctor"); err != nil {
		t.Fatalf("unreachable relay is only a warning: %v", err)
	}
	out := env.out.String()
	mustContain(t, out, "relay 'proxmox' not reachable over SSH now")
	if !strings.Contains(out, "warning(s)") {
		t.Errorf("summary missing warnings: %s", out)
	}
}
