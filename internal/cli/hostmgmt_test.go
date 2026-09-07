package cli

import (
	"os"
	"strings"
	"testing"
)

func (e *testEnv) hostsText(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(e.app.HostsPath)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func (e *testEnv) groupsText(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(e.app.GroupsPath)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// neighRunner answers `ip neigh show <ip>` with a MAC, and ssh -G lookups.
func neighRunner(ips map[string]string, neigh map[string]string) *fakeRunner {
	f := sshGRunner(ips, nil)
	inner := f.respond
	f.respond = func(name string, args []string) (string, string, error) {
		if name == "ip" && len(args) == 3 && args[0] == "neigh" && args[1] == "show" {
			mac := neigh[args[2]]
			if mac == "" {
				return "", "", nil
			}
			return args[2] + " dev eth0 lladdr " + mac + " REACHABLE\n", "", nil
		}
		return inner(name, args)
	}
	return f
}

func TestAddWithMac(t *testing.T) {
	env := newTestEnv(t, "", "")
	if err := env.run("add", "nas", "--mac", "AA-BB-CC-DD-EE-FF", "--broadcast", "10.0.0.255", "--port", "445", "--via", "proxmox"); err != nil {
		t.Fatalf("add: %v", err)
	}
	text := env.hostsText(t)
	mustContain(t, text, "Host nas", "aa:bb:cc:dd:ee:ff", "10.0.0.255", "445", "proxmox")
	mustContain(t, env.out.String(), "Added 'nas' -> aa:bb:cc:dd:ee:ff", "broadcast 10.0.0.255", "port 445", "via proxmox")
}

func TestAddInvalidMac(t *testing.T) {
	env := newTestEnv(t, "", "")
	if err := env.run("add", "nas", "--mac", "nope"); err == nil || !strings.Contains(err.Error(), "invalid MAC") {
		t.Errorf("err = %v", err)
	}
}

func TestAddReservedName(t *testing.T) {
	env := newTestEnv(t, "", "")
	if err := env.run("add", "status", "--mac", "aa:bb:cc:dd:ee:ff"); err == nil || !strings.Contains(err.Error(), "reserved word") {
		t.Errorf("err = %v", err)
	}
}

func TestAddDuplicateHost(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	if err := env.run("add", "server", "--mac", "aa:bb:cc:dd:ee:ff"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("err = %v", err)
	}
}

func TestAddViaSSH(t *testing.T) {
	env := newTestEnv(t, "", "")
	env.app.Runner = neighRunner(
		map[string]string{"nas": "192.168.1.50"},
		map[string]string{"192.168.1.50": "AA:BB:CC:DD:EE:11"},
	)
	if err := env.run("add", "nas", "--ssh"); err != nil {
		t.Fatalf("add --ssh: %v", err)
	}
	mustContain(t, env.hostsText(t), "Host nas", "aa:bb:cc:dd:ee:11")
}

func TestAddViaIPOffline(t *testing.T) {
	env := newTestEnv(t, "", "")
	env.app.Runner = neighRunner(nil, nil)
	err := env.run("add", "nas", "--ip", "192.168.1.99")
	if err == nil || !strings.Contains(err.Error(), "Is it online?") {
		t.Errorf("err = %v", err)
	}
}

func TestEditMacAndPort(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	if err := env.run("edit", "server", "--mac", "11:22:33:44:55:66", "--port", "8080"); err != nil {
		t.Fatalf("edit: %v", err)
	}
	text := env.hostsText(t)
	mustContain(t, text, "11:22:33:44:55:66", "8080")
	if strings.Contains(text, "44:87:63:80:5e:f6") {
		t.Error("old MAC still present")
	}
	mustContain(t, env.out.String(), "Updated 'server'")
}

func TestEditRenameUpdatesGroups(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	if err := env.run("edit", "server", "--name", "bigserver"); err != nil {
		t.Fatalf("edit --name: %v", err)
	}
	mustContain(t, env.hostsText(t), "Host bigserver")
	mustContain(t, env.groupsText(t), "bigserver")
	if strings.Contains(env.groupsText(t), " server") {
		t.Errorf("groups still reference old name: %q", env.groupsText(t))
	}
}

func TestEditClearVia(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	if err := env.run("edit", "desktop", "--via", ""); err != nil {
		t.Fatalf("edit --via '': %v", err)
	}
	if strings.Contains(env.hostsText(t), "proxmox") {
		t.Errorf("via not cleared: %q", env.hostsText(t))
	}
}

func TestEditNothingToChange(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	if err := env.run("edit", "server"); err == nil || !strings.Contains(err.Error(), "nothing to change") {
		t.Errorf("err = %v", err)
	}
}

func TestRm(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	if err := env.run("rm", "pi"); err != nil {
		t.Fatalf("rm: %v", err)
	}
	if strings.Contains(env.hostsText(t), "Host pi") {
		t.Error("pi still in hosts file")
	}
	mustContain(t, env.out.String(), "Removed 'pi'")
}

func TestRmUnknown(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	if err := env.run("rm", "ghost"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("err = %v", err)
	}
}

func TestImportSSH(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.writeSSHConfig(t, "Host server newbox star*\nHost offline\n")
	env.app.Runner = neighRunner(
		map[string]string{"newbox": "192.168.1.60", "offline": "192.168.1.61", "server": "192.168.1.10"},
		map[string]string{"192.168.1.60": "aa:bb:cc:dd:ee:22"},
	)
	if err := env.run("import-ssh"); err != nil {
		t.Fatalf("import-ssh: %v", err)
	}
	mustContain(t, env.hostsText(t), "Host newbox", "aa:bb:cc:dd:ee:22")
	out := env.out.String()
	mustContain(t, out, "= server (already listed)", "+ newbox -> aa:bb:cc:dd:ee:22", "- offline (offline / no MAC)", "Done: 1 added, 1 skipped.")
	if strings.Contains(env.hostsText(t), "star") {
		t.Error("wildcard host must be skipped")
	}
}

func TestGroupAdd(t *testing.T) {
	env := newTestEnv(t, envHosts, "")
	if err := env.run("group", "add", "mini", "--devices", "server", "pi"); err != nil {
		t.Fatalf("group add: %v", err)
	}
	mustContain(t, env.groupsText(t), "mini server pi")
	mustContain(t, env.out.String(), "Group 'mini' = server pi")
}

func TestGroupAddUnknownMember(t *testing.T) {
	env := newTestEnv(t, envHosts, "")
	if err := env.run("group", "add", "mini", "--devices", "ghost"); err == nil || !strings.Contains(err.Error(), "not a known host") {
		t.Errorf("err = %v", err)
	}
}

func TestGroupEditAddRmRename(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	if err := env.run("group", "edit", "lab", "--add", "desktop", "--rm", "pi", "--name", "homelab"); err != nil {
		t.Fatalf("group edit: %v", err)
	}
	text := env.groupsText(t)
	mustContain(t, text, "homelab server desktop")
	if strings.Contains(text, "lab server pi") {
		t.Errorf("old group line still present: %q", text)
	}
}

func TestGroupEditWouldEmpty(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	err := env.run("group", "edit", "lab", "--rm", "server", "pi")
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("err = %v", err)
	}
}

func TestGroupRm(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	if err := env.run("group", "rm", "lab"); err != nil {
		t.Fatalf("group rm: %v", err)
	}
	if strings.Contains(env.groupsText(t), "lab") {
		t.Error("lab still present")
	}
	mustContain(t, env.out.String(), "Removed group 'lab'")
}

func TestGroupsList(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	if err := env.run("groups"); err != nil {
		t.Fatalf("groups: %v", err)
	}
	mustContain(t, env.out.String(), "GROUP", "MEMBERS", "lab", "server pi")
}

func TestGroupsEmpty(t *testing.T) {
	env := newTestEnv(t, envHosts, "")
	if err := env.run("groups"); err != nil {
		t.Fatalf("groups: %v", err)
	}
	mustContain(t, env.out.String(), "No groups defined.")
}

func TestAddRejectsSignedPort(t *testing.T) {
	env := newTestEnv(t, "", "")
	if err := env.run("add", "nas", "--mac", "aa:bb:cc:dd:ee:ff", "--port", "+22"); err == nil {
		t.Error("signed --port must be rejected")
	}
}

func TestGroupAddRejectsWhitespaceName(t *testing.T) {
	env := newTestEnv(t, envHosts, "")
	if err := env.run("group", "add", "my group", "--devices", "server"); err == nil {
		t.Error("group name with whitespace must be rejected")
	}
}
