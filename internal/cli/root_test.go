package cli

import (
	"strings"
	"testing"
)

func TestNewRootCmd(t *testing.T) {
	env := newTestEnv(t, "", "")
	cmd := NewRootCmd(env.app)

	if cmd.Use != "wake [target]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "wake [target]")
	}
	if cmd.Short == "" {
		t.Error("Short must not be empty")
	}
	if cmd.Version == "" {
		t.Error("Version must not be empty")
	}
}

func TestRootCmdHelpDoesNotError(t *testing.T) {
	env := newTestEnv(t, "", "")
	if err := env.run("--help"); err != nil {
		t.Errorf("--help returned error: %v", err)
	}
}

func TestRootCmdHelpHidesCompletion(t *testing.T) {
	env := newTestEnv(t, "", "")
	if err := env.run("--help"); err != nil {
		t.Fatalf("--help returned error: %v", err)
	}
	if out := env.out.String(); strings.Contains(out, "completion") {
		t.Errorf("--help must not list the completion command, got:\n%s", out)
	}
}

func TestCompletionCmdStillWorks(t *testing.T) {
	env := newTestEnv(t, "", "")
	if err := env.run("completion", "bash"); err != nil {
		t.Fatalf("completion bash returned error: %v", err)
	}
	mustContain(t, env.out.String(), "bash completion")
}

func TestRootCmdNoArgsShowsHelp(t *testing.T) {
	env := newTestEnv(t, "", "")
	if err := env.run(); err != nil {
		t.Errorf("bare wake returned error: %v", err)
	}
	mustContain(t, env.out.String(), "Usage:")
}
