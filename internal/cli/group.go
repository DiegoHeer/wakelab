package cli

import (
	"fmt"
	"strings"

	"github.com/DiegoHeer/wakelab/internal/config"
	"github.com/DiegoHeer/wakelab/internal/host"
	"github.com/spf13/cobra"
)

const groupHelp = `wake group / wake groups — manage host groups (~/.wol_groups)

  wake group add <name> --devices <host...>   create a group
  wake group edit <name> --add <host...>      add hosts to a group
  wake group edit <name> --rm <host...>       remove hosts from a group
  wake group edit <name> --name <new>         rename a group
  wake group rm <name>                        delete a group
  wake groups                                 list all groups

You can combine --add, --rm and --name in one 'edit'.
A group name can be used anywhere a host can: wake, status, poweroff, restart.`

func newGroupCmd(a *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "manage host groups (add/edit/rm)",
		Long:  groupHelp,
	}
	cmd.AddCommand(newGroupAddCmd(a), newGroupEditCmd(a), newGroupRmCmd(a))
	return cmd
}

// wantsHelp reports whether manually-parsed args ask for help.
func wantsHelp(args []string) bool {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			return true
		}
	}
	return false
}

// checkNewGroupName rejects a name that cannot become a new group.
func checkNewGroupName(name string, hosts []host.Host, groups []host.Group) error {
	if name == "" || strings.ContainsAny(name, " \t\n\r") {
		return fmt.Errorf("invalid group name '%s' (no whitespace allowed)", name)
	}
	if host.Reserved(name) {
		return fmt.Errorf("'%s' is a reserved word", name)
	}
	if hostExists(hosts, name) {
		return fmt.Errorf("'%s' is already a host name", name)
	}
	if isGroup(groups, name) {
		return fmt.Errorf("group '%s' already exists", name)
	}
	return nil
}

// group add/edit take space-separated host lists after --devices/--add/--rm
// (parity with the Bash CLI), so they parse their own args.
func newGroupAddCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:                "add <name> --devices <host...>",
		Short:              "create a group",
		Long:               groupHelp,
		DisableFlagParsing: true,
		ValidArgsFunction:  a.completeNewNameThenHosts,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			if len(args) < 1 {
				return fmt.Errorf("usage: wake group add <name> --devices <host...>")
			}
			name := args[0]
			_, hosts, err := a.loadHosts()
			if err != nil {
				return err
			}
			groupsText, groups, err := a.loadGroups()
			if err != nil {
				return err
			}
			if err := checkNewGroupName(name, hosts, groups); err != nil {
				if isGroup(groups, name) {
					return fmt.Errorf("group '%s' already exists (change it with: wake group edit %s)", name, name)
				}
				return err
			}
			var members []string
			rest := args[1:]
			for len(rest) > 0 {
				switch {
				case rest[0] == "--devices":
					rest = rest[1:]
					for len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
						members = append(members, rest[0])
						rest = rest[1:]
					}
				case strings.HasPrefix(rest[0], "-"):
					return fmt.Errorf("unknown option '%s' (see: wake group --help)", rest[0])
				default:
					return fmt.Errorf("unexpected argument '%s' (list hosts after --devices)", rest[0])
				}
			}
			if len(members) == 0 {
				return fmt.Errorf("usage: wake group add <name> --devices <host...>")
			}
			for _, m := range members {
				if !hostExists(hosts, m) {
					return fmt.Errorf("member '%s' is not a known host (add it first)", m)
				}
			}
			if err := config.Save(a.GroupsPath, config.AppendGroup(groupsText, name, members)); err != nil {
				return err
			}
			fmt.Fprintf(a.Out, "Group '%s' = %s\n", name, strings.Join(members, " "))
			return nil
		},
	}
}

func newGroupEditCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:                "edit <name> [--add <host...>] [--rm <host...>] [--name <new>]",
		Short:              "change a group",
		Long:               groupHelp,
		DisableFlagParsing: true,
		ValidArgsFunction:  a.completeGroupThenHosts,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			if len(args) < 1 {
				return fmt.Errorf("usage: wake group edit <name> [--add <host...>] [--rm <host...>] [--name <new>]")
			}
			name := args[0]
			_, hosts, err := a.loadHosts()
			if err != nil {
				return err
			}
			groupsText, groups, err := a.loadGroups()
			if err != nil {
				return err
			}
			if !isGroup(groups, name) {
				return fmt.Errorf("group '%s' not found (create it with: wake group add)", name)
			}
			var addList, rmList []string
			newName := ""
			rest := args[1:]
			for len(rest) > 0 {
				switch {
				case rest[0] == "--add" || rest[0] == "--rm":
					flag := rest[0]
					rest = rest[1:]
					for len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
						if flag == "--add" {
							addList = append(addList, rest[0])
						} else {
							rmList = append(rmList, rest[0])
						}
						rest = rest[1:]
					}
				case rest[0] == "--name":
					if len(rest) < 2 || rest[1] == "" {
						return fmt.Errorf("--name needs a value")
					}
					newName = rest[1]
					rest = rest[2:]
				case strings.HasPrefix(rest[0], "-"):
					return fmt.Errorf("unknown option '%s' (see: wake group --help)", rest[0])
				default:
					return fmt.Errorf("unexpected argument '%s'", rest[0])
				}
			}
			if len(addList) == 0 && len(rmList) == 0 && newName == "" {
				return fmt.Errorf("nothing to change (use --add, --rm, and/or --name)")
			}
			for _, m := range addList {
				if !hostExists(hosts, m) {
					return fmt.Errorf("'%s' is not a known host (add it first)", m)
				}
			}

			// New member list: keep current order, drop --rm, append new --add.
			var cur []string
			for _, g := range groups {
				if g.Name == name {
					cur = g.Members
					break
				}
			}
			have := map[string]bool{}
			for _, m := range cur {
				have[m] = true
			}
			for _, m := range rmList {
				delete(have, m)
			}
			for _, m := range addList {
				have[m] = true
			}
			var members []string
			for _, m := range append(append([]string{}, cur...), addList...) {
				if have[m] {
					members = append(members, m)
					delete(have, m)
				}
			}
			if len(members) == 0 {
				return fmt.Errorf("that would leave '%s' empty — delete it with: wake group rm %s", name, name)
			}

			finalName := name
			if newName != "" && newName != name {
				if err := checkNewGroupName(newName, hosts, groups); err != nil {
					return err
				}
				finalName = newName
			}
			out := config.AppendGroup(config.RemoveGroup(groupsText, name), finalName, members)
			if err := config.Save(a.GroupsPath, out); err != nil {
				return err
			}
			fmt.Fprintf(a.Out, "Group '%s' = %s\n", finalName, strings.Join(members, " "))
			return nil
		},
	}
}

func newGroupRmCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:               "rm <name>",
		Short:             "delete a group",
		Long:              groupHelp,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: a.completeGroups,
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			groupsText, groups, err := a.loadGroups()
			if err != nil {
				return err
			}
			if !isGroup(groups, name) {
				return fmt.Errorf("group '%s' not found", name)
			}
			if err := config.Save(a.GroupsPath, config.RemoveGroup(groupsText, name)); err != nil {
				return err
			}
			fmt.Fprintf(a.Out, "Removed group '%s'\n", name)
			return nil
		},
	}
}

func newGroupsCmd(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "groups",
		Short: "list groups",
		Long:  groupHelp,
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			_, groups, err := a.loadGroups()
			if err != nil {
				return err
			}
			if len(groups) == 0 {
				fmt.Fprintln(a.Out, "No groups defined. Create one with: wake group add <name> --devices <host...>")
				return nil
			}
			fmt.Fprintf(a.Out, "%s%-10s %s%s\n", a.color(ansiBold), "GROUP", "MEMBERS", a.color(ansiReset))
			for _, g := range groups {
				fmt.Fprintf(a.Out, "%-10s %s\n", g.Name, strings.Join(g.Members, " "))
			}
			return nil
		},
	}
}
