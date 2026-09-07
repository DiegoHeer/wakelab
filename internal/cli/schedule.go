package cli

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const schedTag = "# wake-schedule"

const scheduleHelp = `wake schedule — wake a target automatically on a timer (via cron)

  wake schedule add <target> <HH:MM>        wake daily at that time
  wake schedule add <target> "<cron expr>"  wake on a 5-field cron schedule
  wake schedule list                        show scheduled wakes (with IDs)
  wake schedule rm <id>                     remove one by its ID

Examples:
  wake schedule add server 07:00
  wake schedule add desktop "0 7 * * 1-5"   # 07:00 on weekdays

Needs cron installed and running (Fedora: sudo dnf install cronie,
then sudo systemctl enable --now crond). Only touches its own lines.`

// currentCrontab returns the user's crontab ("" when there is none).
func (a *App) currentCrontab(ctx context.Context) string {
	stdout, _, err := a.Runner.Run(ctx, "crontab", "-l")
	if err != nil {
		return ""
	}
	return stdout
}

// writeCrontab replaces the user's crontab.
func (a *App) writeCrontab(ctx context.Context, content string) error {
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if _, stderr, err := a.Runner.RunInput(ctx, content, "crontab", "-"); err != nil {
		return fmt.Errorf("crontab write failed: %s", strings.TrimSpace(stderr))
	}
	return nil
}

func newScheduleCmd(a *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "wake a target on a timer (add/list/rm, uses cron)",
		Long:  scheduleHelp,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if _, err := a.LookPath("crontab"); err != nil {
				//nolint:revive,staticcheck // parity with the Bash tool's exact message
				return fmt.Errorf("crontab not found. Install cron (Fedora: sudo dnf install cronie && sudo systemctl enable --now crond).")
			}
			return nil
		},
	}
	cmd.AddCommand(newScheduleAddCmd(a), newScheduleListCmd(a), newScheduleRmCmd(a))
	return cmd
}

var hhmmRe = regexp.MustCompile(`^([0-9]{1,2}):([0-9]{2})$`)

func newScheduleAddCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "add <target> <HH:MM | \"cron expr\">",
		Short: "schedule a daily or cron-timed wake",
		Long:  scheduleHelp,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			target, when := args[0], args[1]
			_, hosts, err := a.loadHosts()
			if err != nil {
				return err
			}
			_, groups, err := a.loadGroups()
			if err != nil {
				return err
			}
			if _, ok := resolveNames(target, hosts, groups); !ok {
				return fmt.Errorf("unknown host or group '%s'", target)
			}
			var cron string
			if m := hhmmRe.FindStringSubmatch(when); m != nil {
				hh, _ := strconv.Atoi(m[1])
				mm, _ := strconv.Atoi(m[2])
				if hh >= 24 || mm >= 60 {
					return fmt.Errorf("invalid time '%s' (HH:MM, 00:00-23:59)", when)
				}
				cron = fmt.Sprintf("%d %d * * *", mm, hh)
			} else {
				if strings.ContainsAny(when, "\n\r") || len(strings.Fields(when)) != 5 {
					return fmt.Errorf("time must be HH:MM or a 5-field cron expression in quotes")
				}
				cron = when
			}
			wakeBin, err := a.LookPath("wake")
			if err != nil {
				if wakeBin, err = a.Executable(); err != nil {
					return err
				}
			}
			cur := a.currentCrontab(ctx)
			line := fmt.Sprintf("%s %s %s  %s target=%s", cron, wakeBin, target, schedTag, target)
			if cur != "" && !strings.HasSuffix(cur, "\n") {
				cur += "\n"
			}
			if err := a.writeCrontab(ctx, cur+line+"\n"); err != nil {
				return err
			}
			fmt.Fprintf(a.Out, "Scheduled: wake %s  (%s)\n", target, cron)
			if _, err := a.LookPath("systemctl"); err == nil {
				if _, _, err := a.Runner.Run(ctx, "systemctl", "is-active", "--quiet", "crond"); err != nil {
					fmt.Fprintf(a.Err, "%sNote: cron service is not active. Enable it: sudo systemctl enable --now crond%s\n",
						a.color(ansiYellow), a.color(ansiReset))
				}
			}
			return nil
		},
	}
}

// schedLines returns the tagged schedule lines from the crontab.
func schedLines(crontab string) []string {
	var out []string
	for _, line := range strings.Split(crontab, "\n") {
		if strings.Contains(line, schedTag) {
			out = append(out, line)
		}
	}
	return out
}

var targetRe = regexp.MustCompile(`target=(\S+)`)

func newScheduleListCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "show scheduled wakes (with IDs)",
		Long:    scheduleHelp,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			lines := schedLines(a.currentCrontab(cmd.Context()))
			if len(lines) == 0 {
				fmt.Fprintln(a.Out, "No scheduled wakes. Add one with: wake schedule add <target> <HH:MM>")
				return nil
			}
			fmt.Fprintf(a.Out, "%s%-4s %-18s %s%s\n", a.color(ansiBold), "ID", "WHEN (cron)", "TARGET", a.color(ansiReset))
			for i, l := range lines {
				fields := strings.Fields(l)
				cron := ""
				if len(fields) >= 5 {
					cron = strings.Join(fields[:5], " ")
				}
				target := ""
				if m := targetRe.FindStringSubmatch(l); m != nil {
					target = m[1]
				}
				fmt.Fprintf(a.Out, "%-4d %-18s %s\n", i+1, cron, target)
			}
			return nil
		},
	}
}

func newScheduleRmCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "rm <id>",
		Short: "remove a scheduled wake by its ID",
		Long:  scheduleHelp,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			id, err := strconv.Atoi(args[0])
			if err != nil || id < 1 || strings.ContainsAny(args[0], "+-") {
				return fmt.Errorf("usage: wake schedule rm <id> (see: wake schedule list)")
			}
			cur := a.currentCrontab(ctx)
			n := 0
			removed := false
			var kept []string
			lines := strings.Split(strings.TrimSuffix(cur, "\n"), "\n")
			for _, l := range lines {
				if strings.Contains(l, schedTag) {
					n++
					if n == id {
						removed = true
						continue
					}
				}
				kept = append(kept, l)
			}
			if !removed {
				return fmt.Errorf("no schedule with id %d (see: wake schedule list)", id)
			}
			if err := a.writeCrontab(ctx, strings.Join(kept, "\n")); err != nil {
				return err
			}
			fmt.Fprintf(a.Out, "Removed schedule %d\n", id)
			return nil
		},
	}
}
