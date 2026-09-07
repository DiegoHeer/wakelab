package cli

import (
	"strings"
	"testing"
)

func TestWakeLocalHost(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	if err := env.run("server"); err != nil {
		t.Fatalf("wake server: %v", err)
	}
	if len(env.sent) != 1 || env.sent[0] != "44:87:63:80:5e:f6@255.255.255.255" {
		t.Errorf("sent = %v", env.sent)
	}
	mustContain(t, env.out.String(), "Waking 'server' (44:87:63:80:5e:f6) via 255.255.255.255...")
}

func TestWakeHostCustomBroadcast(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	if err := env.run("pi"); err != nil {
		t.Fatalf("wake pi: %v", err)
	}
	if len(env.sent) != 1 || env.sent[0] != "02:00:96:71:f5:bb@192.168.1.255" {
		t.Errorf("sent = %v", env.sent)
	}
}

func TestWakeViaRelay(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	runner := &fakeRunner{}
	env.app.Runner = runner

	if err := env.run("desktop"); err != nil {
		t.Fatalf("wake desktop: %v", err)
	}
	if len(env.sent) != 0 {
		t.Errorf("relay wake must not send locally, sent = %v", env.sent)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("calls = %v, want one ssh call", runner.calls)
	}
	call := runner.calls[0]
	if call[0] != "ssh" || call[1] != "proxmox" {
		t.Errorf("call = %v, want ssh proxmox <script>", call)
	}
	script := strings.Join(call[2:], " ")
	mustContain(t, script, "wakeonlan", "python3", "b0:41:6f:16:32:ee")
	mustContain(t, env.out.String(), "sent from 'proxmox'")
}

func TestWakeViaOverrideForcesLocal(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	runner := &fakeRunner{}
	env.app.Runner = runner

	if err := env.run("desktop", "--via", ""); err != nil {
		t.Fatalf("wake desktop --via '': %v", err)
	}
	if len(runner.calls) != 0 {
		t.Errorf("forced-local wake must not ssh, calls = %v", runner.calls)
	}
	if len(env.sent) != 1 {
		t.Errorf("sent = %v", env.sent)
	}
}

func TestWakeGroup(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	if err := env.run("lab"); err != nil {
		t.Fatalf("wake lab: %v", err)
	}
	if len(env.sent) != 2 {
		t.Errorf("sent = %v, want server and pi", env.sent)
	}
}

func TestWakeUnknownTarget(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	err := env.run("ghost")
	if err == nil || !strings.Contains(err.Error(), "unknown host or group") {
		t.Errorf("err = %v", err)
	}
}

func TestWakeSkipsHostWithoutMac(t *testing.T) {
	env := newTestEnv(t, "Host nomac\n", "")
	if err := env.run("nomac"); err != nil {
		t.Fatalf("wake nomac: %v", err)
	}
	if len(env.sent) != 0 {
		t.Errorf("sent = %v, want none", env.sent)
	}
	mustContain(t, env.out.String(), "skip nomac (no MAC)")
}

func TestWakeRawMac(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	if err := env.run("AA-BB-CC-DD-EE-FF", "--broadcast", "10.0.0.255"); err != nil {
		t.Fatalf("wake raw mac: %v", err)
	}
	if len(env.sent) != 1 || env.sent[0] != "aa:bb:cc:dd:ee:ff@10.0.0.255" {
		t.Errorf("sent = %v", env.sent)
	}
}

func TestWakeWaitTimesOut(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.app.Runner = sshGRunner(map[string]string{"server": "192.168.1.10"}, nil)
	env.app.Prober = fakeProber{} // never comes up
	err := env.run("server", "--wait", "--timeout", "6")
	if err == nil {
		t.Fatal("wait must fail when the host never comes up")
	}
	mustContain(t, env.out.String(), "Timeout: not all hosts came online.")
}

func TestWakeWaitSucceeds(t *testing.T) {
	env := newTestEnv(t, envHosts, envGroups)
	env.app.Runner = sshGRunner(map[string]string{"server": "192.168.1.10"}, nil)
	env.app.Prober = fakeProber{ports: map[string]bool{"192.168.1.10:22": true}}

	if err := env.run("server", "--wait", "--timeout", "6"); err != nil {
		t.Fatalf("wake --wait: %v", err)
	}
	mustContain(t, env.out.String(), "All online.")
}
