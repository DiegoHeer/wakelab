package cli

import (
	"github.com/DiegoHeer/wakelab/internal/config"
	"github.com/spf13/cobra"
)

// Shell-completion helpers. They read only the local config files (never the
// network) and swallow every error: a broken config must degrade to "no
// suggestions", not break the user's shell.

// hostNames returns every host name in ~/.wol_hosts (nil on error).
func (a *App) hostNames() []string {
	text, err := config.Load(a.HostsPath)
	if err != nil {
		return nil
	}
	var names []string
	for _, h := range config.ParseHosts(text) {
		names = append(names, h.Name)
	}
	return names
}

// groupNames returns every group name in ~/.wol_groups (nil on error).
func (a *App) groupNames() []string {
	text, err := config.Load(a.GroupsPath)
	if err != nil {
		return nil
	}
	var names []string
	for _, g := range config.ParseGroups(text) {
		names = append(names, g.Name)
	}
	return names
}

// completeTargets offers hosts, groups, and 'all' for a first argument.
func (a *App) completeTargets(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names := append(a.hostNames(), a.groupNames()...)
	return append(names, "all"), cobra.ShellCompDirectiveNoFileComp
}

// completeHosts offers host names for a first argument.
func (a *App) completeHosts(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return a.hostNames(), cobra.ShellCompDirectiveNoFileComp
}

// completeHostsAnywhere offers host names regardless of position (flag
// values, and device lists of the flag-parsing-disabled group commands).
func (a *App) completeHostsAnywhere(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return a.hostNames(), cobra.ShellCompDirectiveNoFileComp
}

// completeGroups offers group names for a first argument.
func (a *App) completeGroups(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return a.groupNames(), cobra.ShellCompDirectiveNoFileComp
}

// completeGroupThenHosts offers group names first, then host names for the
// member arguments (group edit <name> --add <host...>).
func (a *App) completeGroupThenHosts(cmd *cobra.Command, args []string, tc string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return a.completeGroups(cmd, args, tc)
	}
	return a.hostNames(), cobra.ShellCompDirectiveNoFileComp
}

// completeNewNameThenHosts offers nothing for the (new) name, then host
// names for the member arguments (group add <name> --devices <host...>).
func (a *App) completeNewNameThenHosts(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return a.hostNames(), cobra.ShellCompDirectiveNoFileComp
}
