package cli

import (
	"strings"
	"testing"
)

// complete runs Cobra's hidden __complete machinery and returns its output.
// The fixture hosts are server/desktop/pi and the fixture group is lab.
func complete(t *testing.T, args ...string) string {
	t.Helper()
	env := newTestEnv(t, envHosts, envGroups)
	if err := env.run(append([]string{"__complete"}, args...)...); err != nil {
		t.Fatalf("__complete %v: %v", args, err)
	}
	return env.out.String()
}

func mustNotContain(t *testing.T, s string, subs ...string) {
	t.Helper()
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			t.Errorf("output must not contain %q in:\n%s", sub, s)
		}
	}
}

func TestCompleteRootTarget(t *testing.T) {
	out := complete(t, "des")
	mustContain(t, out, "desktop", "server", "pi", "lab", "all")
	mustContain(t, out, ":4") // ShellCompDirectiveNoFileComp
}

func TestCompletePowerTargets(t *testing.T) {
	for _, cmd := range []string{"poweroff", "restart", "suspend", "status"} {
		out := complete(t, cmd, "")
		mustContain(t, out, "desktop", "lab", "all")
	}
}

func TestCompleteSecondTargetOffersNothing(t *testing.T) {
	out := complete(t, "poweroff", "desktop", "")
	mustNotContain(t, out, "server", "lab", "all")
	mustContain(t, out, ":4")
}

func TestCompleteHostOnlyCommands(t *testing.T) {
	for _, cmd := range []string{"edit", "rm"} {
		out := complete(t, cmd, "")
		mustContain(t, out, "desktop", "server", "pi")
		mustNotContain(t, out, "lab", "all")
	}
}

func TestCompleteScheduleAddTarget(t *testing.T) {
	out := complete(t, "schedule", "add", "")
	mustContain(t, out, "desktop", "lab", "all")
}

func TestCompleteGroupNames(t *testing.T) {
	for _, sub := range []string{"edit", "rm"} {
		out := complete(t, "group", sub, "")
		mustContain(t, out, "lab")
		mustNotContain(t, out, "desktop", "all")
	}
}

func TestCompleteGroupDeviceArgs(t *testing.T) {
	// After the group name, --devices/--add/--rm take host names.
	out := complete(t, "group", "edit", "lab", "")
	mustContain(t, out, "desktop", "server", "pi")
	mustNotContain(t, out, "all")

	out = complete(t, "group", "add", "newgroup", "")
	mustContain(t, out, "desktop")
}

func TestCompleteViaFlagOffersHosts(t *testing.T) {
	out := complete(t, "--via", "")
	mustContain(t, out, "desktop", "server", "pi")
	mustNotContain(t, out, "lab", "all")

	out = complete(t, "edit", "desktop", "--via", "")
	mustContain(t, out, "server")
}
