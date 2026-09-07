// Package sshexec runs external commands (ssh, crontab, ping, ...) behind a
// small Runner interface so command logic is testable with fakes.
package sshexec

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
)

// Runner runs an external command. Production uses ExecRunner; tests use fakes.
type Runner interface {
	// Run executes the command and captures its output.
	Run(ctx context.Context, name string, args ...string) (stdout, stderr string, err error)
	// RunTTY executes the command wired to the user's terminal (for
	// interactive ssh -t sessions).
	RunTTY(ctx context.Context, name string, args ...string) error
	// RunInput executes the command with stdin fed from a string (for
	// `crontab -`).
	RunInput(ctx context.Context, stdin, name string, args ...string) (stdout, stderr string, err error)
}

// ExecRunner is the real Runner backed by os/exec.
type ExecRunner struct{}

// Run implements Runner.
func (ExecRunner) Run(ctx context.Context, name string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	return out.String(), errb.String(), err
}

// RunTTY implements Runner.
func (ExecRunner) RunTTY(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunInput implements Runner.
func (ExecRunner) RunInput(ctx context.Context, stdin, name string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	return out.String(), errb.String(), err
}

// SSHOption returns one key's value from `ssh -G <host>` output ("" if the
// lookup fails or the key is absent). Used for hostname and user.
func SSHOption(ctx context.Context, r Runner, hostName, key string) string {
	stdout, _, err := r.Run(ctx, "ssh", "-G", hostName)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == key {
			return fields[1]
		}
	}
	return ""
}
