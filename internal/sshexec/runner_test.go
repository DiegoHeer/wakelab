package sshexec

import (
	"context"
	"strings"
	"testing"
)

func TestExecRunnerRun(t *testing.T) {
	var r ExecRunner
	stdout, stderr, err := r.Run(context.Background(), "sh", "-c", "echo out; echo err >&2")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.TrimSpace(stdout) != "out" || strings.TrimSpace(stderr) != "err" {
		t.Errorf("stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestExecRunnerRunError(t *testing.T) {
	var r ExecRunner
	if _, _, err := r.Run(context.Background(), "sh", "-c", "exit 3"); err == nil {
		t.Error("Run should return the command's error")
	}
}

// fakeRunner returns canned output for ssh -G.
type fakeRunner struct{ out string }

func (f fakeRunner) Run(_ context.Context, _ string, _ ...string) (string, string, error) {
	return f.out, "", nil
}
func (f fakeRunner) RunTTY(_ context.Context, _ string, _ ...string) error { return nil }

func TestSSHOption(t *testing.T) {
	r := fakeRunner{out: "user root\nhostname 192.168.1.10\nport 22\n"}
	if got := SSHOption(context.Background(), r, "server", "hostname"); got != "192.168.1.10" {
		t.Errorf("hostname = %q", got)
	}
	if got := SSHOption(context.Background(), r, "server", "user"); got != "root" {
		t.Errorf("user = %q", got)
	}
	if got := SSHOption(context.Background(), r, "server", "nope"); got != "" {
		t.Errorf("missing key = %q, want empty", got)
	}
}
