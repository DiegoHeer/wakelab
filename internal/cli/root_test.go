package cli

import "testing"

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

func TestRootCmdNoArgsShowsHelp(t *testing.T) {
	env := newTestEnv(t, "", "")
	if err := env.run(); err != nil {
		t.Errorf("bare wake returned error: %v", err)
	}
	mustContain(t, env.out.String(), "Usage:")
}
