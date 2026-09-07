package config

import (
	"strings"
	"testing"
)

const sampleGroups = `# groups file
minirack cluster1 cluster2 cluster3 proxmox
lab server pi
`

func TestParseGroups(t *testing.T) {
	groups := ParseGroups(sampleGroups)
	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(groups))
	}
	if groups[0].Name != "minirack" || len(groups[0].Members) != 4 {
		t.Errorf("group 0 = %+v", groups[0])
	}
	if groups[1].Name != "lab" || strings.Join(groups[1].Members, ",") != "server,pi" {
		t.Errorf("group 1 = %+v", groups[1])
	}
}

func TestFindGroup(t *testing.T) {
	g, ok := FindGroup(sampleGroups, "lab")
	if !ok || len(g.Members) != 2 {
		t.Errorf("FindGroup lab = %+v, %v", g, ok)
	}
	if _, ok := FindGroup(sampleGroups, "nope"); ok {
		t.Error("FindGroup nope should not be found")
	}
}

func TestAppendGroup(t *testing.T) {
	out := AppendGroup("", "g1", []string{"a", "b"})
	if out != "g1 a b\n" {
		t.Errorf("AppendGroup = %q", out)
	}
	out = AppendGroup("x y\n", "g2", []string{"c"})
	if out != "x y\ng2 c\n" {
		t.Errorf("AppendGroup = %q", out)
	}
	// Text without a trailing newline still yields valid lines.
	out = AppendGroup("x y", "g3", []string{"d"})
	if out != "x y\ng3 d\n" {
		t.Errorf("AppendGroup = %q", out)
	}
}

func TestRemoveGroup(t *testing.T) {
	out := RemoveGroup(sampleGroups, "lab")
	if _, ok := FindGroup(out, "lab"); ok {
		t.Error("lab still present after RemoveGroup")
	}
	if _, ok := FindGroup(out, "minirack"); !ok {
		t.Error("minirack lost by RemoveGroup")
	}
	if !strings.Contains(out, "# groups file") {
		t.Error("comment lost by RemoveGroup")
	}
}

func TestRenameMember(t *testing.T) {
	out := RenameMember(sampleGroups, "proxmox", "pve")
	g, _ := FindGroup(out, "minirack")
	if strings.Join(g.Members, ",") != "cluster1,cluster2,cluster3,pve" {
		t.Errorf("members = %v", g.Members)
	}
	// Group names in the first column are not renamed.
	out = RenameMember(sampleGroups, "lab", "labs")
	if _, ok := FindGroup(out, "lab"); !ok {
		t.Error("group name must not be renamed by RenameMember")
	}
	// Untouched lines keep their exact text.
	out = RenameMember(sampleGroups, "unknown-host", "x")
	if out != sampleGroups {
		t.Errorf("text changed with no matching member:\n%q", out)
	}
}
