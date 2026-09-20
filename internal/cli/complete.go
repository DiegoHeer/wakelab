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

// completeRelays offers relay targets for --via: known hosts plus concrete
// ~/.ssh/config Host entries (a relay only needs to be reachable over SSH, so
// it may live only in ~/.ssh/config), deduped.
func (a *App) completeRelays(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	seen := map[string]bool{}
	var names []string
	for _, n := range append(a.hostNames(), a.sshConfigHosts()...) {
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
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
