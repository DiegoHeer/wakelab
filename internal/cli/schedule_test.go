package cli

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// cronRunner emulates crontab -l / crontab <file> plus ssh -G.
type cronRunner struct {
	fakeRunner
	crontab string
}

func newCronRunner(initial string) *cronRunner {
	c := &cronRunner{crontab: initial}
	c.respond = func(name string, args []string) (string, string, error) {
		if name == "crontab" && len(args) == 1 && args[0] == "-l" {
			if c.crontab == "" {
				return "", "no crontab for user", errors.New("exit 1")
			}
			return c.crontab, "", nil
		}
		return "", "", nil
	}
	return c
}

func (c *cronRunner) RunInput(_ context.Context, input, name string, args ...string) (string, string, error) {
	if name == "crontab" && len(args) == 1 && args[0] == "-" {
		c.crontab = input
		return "", "", nil
	}
	return "", "", nil
}

func scheduleEnv(t *testing.T, initialCrontab string) (*testEnv, *cronRunner) {
	env := newTestEnv(t, envHosts, envGroups)
	runner := newCronRunner(initialCrontab)
	env.app.Runner = runner
	env.app.LookPath = func(name string) (string, error) {
		if name == "crontab" {
			return "/usr/bin/crontab", nil
		}
		return "", errors.New("not found")
	}
	env.app.Executable = func() (string, error) { return "/usr/local/bin/wake", nil }
	return env, runner
}

func TestScheduleAddTime(t *testing.T) {
	env, runner := scheduleEnv(t, "")
	if err := env.run("schedule", "add", "server", "07:30"); err != nil {
		t.Fatalf("schedule add: %v", err)
	}
	mustContain(t, runner.crontab, "30 7 * * * /usr/local/bin/wake server  # wake-schedule target=server")
	mustContain(t, env.out.String(), "Scheduled: wake server  (30 7 * * *)")
}

func TestScheduleAddCronExpr(t *testing.T) {
	env, runner := scheduleEnv(t, "0 1 * * * backup\n")
	if err := env.run("schedule", "add", "lab", "0 7 * * 1-5"); err != nil {
		t.Fatalf("schedule add cron: %v", err)
	}
	// Existing entries survive.
	mustContain(t, runner.crontab, "0 1 * * * backup", "0 7 * * 1-5 /usr/local/bin/wake lab  # wake-schedule target=lab")
}

func TestScheduleAddInvalidTime(t *testing.T) {
	env, _ := scheduleEnv(t, "")
	if err := env.run("schedule", "add", "server", "25:00"); err == nil || !strings.Contains(err.Error(), "invalid time") {
		t.Errorf("err = %v", err)
	}
	if err := env.run("schedule", "add", "server", "sometimes"); err == nil || !strings.Contains(err.Error(), "HH:MM") {
		t.Errorf("err = %v", err)
	}
}

func TestScheduleAddUnknownTarget(t *testing.T) {
	env, _ := scheduleEnv(t, "")
	if err := env.run("schedule", "add", "ghost", "07:00"); err == nil || !strings.Contains(err.Error(), "unknown host or group") {
		t.Errorf("err = %v", err)
	}
}

func TestScheduleRequiresCrontab(t *testing.T) {
	env, _ := scheduleEnv(t, "")
	env.app.LookPath = func(string) (string, error) { return "", errors.New("nope") }
	if err := env.run("schedule", "list"); err == nil || !strings.Contains(err.Error(), "crontab not found") {
		t.Errorf("err = %v", err)
	}
}

func TestScheduleList(t *testing.T) {
	env, _ := scheduleEnv(t, "0 1 * * * backup\n30 7 * * * /usr/local/bin/wake server  # wake-schedule target=server\n0 9 * * 1 /usr/local/bin/wake lab  # wake-schedule target=lab\n")
	if err := env.run("schedule", "list"); err != nil {
		t.Fatalf("schedule list: %v", err)
	}
	out := env.out.String()
	mustContain(t, out, "ID", "WHEN (cron)", "TARGET", "30 7 * * *", "server", "0 9 * * 1", "lab")
	if strings.Contains(out, "backup") {
		t.Error("foreign cron lines must not be listed")
	}
}

func TestScheduleListEmpty(t *testing.T) {
	env, _ := scheduleEnv(t, "0 1 * * * backup\n")
	if err := env.run("schedule", "list"); err != nil {
		t.Fatalf("schedule list: %v", err)
	}
	mustContain(t, env.out.String(), "No scheduled wakes.")
}

func TestScheduleRm(t *testing.T) {
	env, runner := scheduleEnv(t, "0 1 * * * backup\n30 7 * * * /usr/local/bin/wake server  # wake-schedule target=server\n0 9 * * 1 /usr/local/bin/wake lab  # wake-schedule target=lab\n")
	if err := env.run("schedule", "rm", "1"); err != nil {
		t.Fatalf("schedule rm: %v", err)
	}
	if strings.Contains(runner.crontab, "target=server") {
		t.Errorf("schedule 1 not removed: %q", runner.crontab)
	}
	mustContain(t, runner.crontab, "backup", "target=lab")
	mustContain(t, env.out.String(), "Removed schedule 1")
}

func TestScheduleRmBadID(t *testing.T) {
	env, _ := scheduleEnv(t, "30 7 * * * /usr/local/bin/wake server  # wake-schedule target=server\n")
	if err := env.run("schedule", "rm", "9"); err == nil || !strings.Contains(err.Error(), "no schedule with id 9") {
		t.Errorf("err = %v", err)
	}
	if err := env.run("schedule", "rm", "x"); err == nil {
		t.Error("non-numeric id must error")
	}
}

func TestScheduleAddRejectsNewlineInCron(t *testing.T) {
	env, _ := scheduleEnv(t, "")
	if err := env.run("schedule", "add", "server", "0 7\n* * *"); err == nil || !strings.Contains(err.Error(), "HH:MM") {
		t.Errorf("err = %v", err)
	}
}
