package cli

import "testing"

func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd()

	if cmd.Use != "wake" {
		t.Errorf("Use = %q, want %q", cmd.Use, "wake")
	}
	if cmd.Short == "" {
		t.Error("Short must not be empty")
	}
	if cmd.Version == "" {
		t.Error("Version must not be empty")
	}
}

func TestRootCmdHelpDoesNotError(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("--help returned error: %v", err)
	}
}
