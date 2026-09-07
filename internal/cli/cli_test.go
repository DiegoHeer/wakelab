package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeRunner records every call and replies via the respond func. Commands
// probe hosts in parallel, so the call log is mutex-guarded.
type fakeRunner struct {
	mu      sync.Mutex
	calls   [][]string
	respond func(name string, args []string) (string, string, error)
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, string, error) {
	f.mu.Lock()
	f.calls = append(f.calls, append([]string{name}, args...))
	f.mu.Unlock()
	if f.respond != nil {
		return f.respond(name, args)
	}
	return "", "", nil
}

func (f *fakeRunner) RunTTY(ctx context.Context, name string, args ...string) error {
	_, _, err := f.Run(ctx, name, args...)
	return err
}

// sshGRunner answers `ssh -G <host>` with a fixed hostname/user per host and
// records all other calls.
func sshGRunner(ips map[string]string, users map[string]string) *fakeRunner {
	f := &fakeRunner{}
	f.respond = func(name string, args []string) (string, string, error) {
		if name == "ssh" && len(args) == 2 && args[0] == "-G" {
			ip, ok := ips[args[1]]
			if !ok {
				return "", "", errors.New("no such host")
			}
			user := users[args[1]]
			if user == "" {
				user = "diego"
			}
			return fmt.Sprintf("user %s\nhostname %s\n", user, ip), "", nil
		}
		return "", "", nil
	}
	return f
}

// fakeProber answers port/ping checks from maps.
type fakeProber struct {
	ports map[string]bool // key "ip:port"
	pings map[string]bool // key ip
}

func (f fakeProber) PortOpen(ip, port string) bool { return f.ports[ip+":"+port] }
func (f fakeProber) Ping(ip string) bool           { return f.pings[ip] }

// testApp builds an App wired to fakes and temp config files.
type testEnv struct {
	app  *App
	out  *bytes.Buffer
	errb *bytes.Buffer
	sent []string // "mac@bcast" for every SendWOL call
	dir  string
}

func newTestEnv(t *testing.T, hostsText, groupsText string) *testEnv {
	t.Helper()
	dir := t.TempDir()
	hostsPath := filepath.Join(dir, "wol_hosts")
	groupsPath := filepath.Join(dir, "wol_groups")
	sshPath := filepath.Join(dir, "ssh_config")
	if err := os.WriteFile(hostsPath, []byte(hostsText), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(groupsPath, []byte(groupsText), 0o644); err != nil {
		t.Fatal(err)
	}
	env := &testEnv{out: &bytes.Buffer{}, errb: &bytes.Buffer{}, dir: dir}
	env.app = &App{
		HostsPath:     hostsPath,
		GroupsPath:    groupsPath,
		SSHConfigPath: sshPath,
		Runner:        &fakeRunner{},
		Prober:        fakeProber{},
		SendWOL: func(mac, bcast string) error {
			env.sent = append(env.sent, mac+"@"+bcast)
			return nil
		},
		Out:      env.out,
		Err:      env.errb,
		IsTTY:    false,
		Sleep:    func(time.Duration) {},
		LookPath: func(string) (string, error) { return "", errors.New("not found") },
	}
	return env
}

func (e *testEnv) writeSSHConfig(t *testing.T, text string) {
	t.Helper()
	if err := os.WriteFile(e.app.SSHConfigPath, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// run executes the wake CLI with args and returns the error.
func (e *testEnv) run(args ...string) error {
	cmd := NewRootCmd(e.app)
	cmd.SetOut(e.out)
	cmd.SetErr(e.errb)
	cmd.SetArgs(args)
	return cmd.Execute()
}

const envHosts = `Host server
    Mac        44:87:63:80:5e:f6

Host desktop
    Mac        b0:41:6f:16:32:ee
    Via        proxmox

Host pi
    Mac        02:00:96:71:f5:bb
    Broadcast  192.168.1.255
    Port       80
`

const envGroups = "lab server pi\n"

func mustContain(t *testing.T, s string, subs ...string) {
	t.Helper()
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			t.Errorf("output missing %q in:\n%s", sub, s)
		}
	}
}
