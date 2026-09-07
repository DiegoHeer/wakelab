package config

import (
	"strings"
	"testing"

	"github.com/DiegoHeer/wakelab/internal/host"
)

func testHosts() []host.Host {
	return []host.Host{{Name: "server"}, {Name: "desktop"}, {Name: "pi"}}
}

func testGroups() []host.Group {
	return []host.Group{{Name: "lab", Members: []string{"server", "pi"}}}
}

func TestResolveHost(t *testing.T) {
	names, ok := Resolve("desktop", testHosts(), testGroups())
	if !ok || strings.Join(names, ",") != "desktop" {
		t.Errorf("Resolve(desktop) = %v, %v", names, ok)
	}
}

func TestResolveGroup(t *testing.T) {
	names, ok := Resolve("lab", testHosts(), testGroups())
	if !ok || strings.Join(names, ",") != "server,pi" {
		t.Errorf("Resolve(lab) = %v, %v", names, ok)
	}
}

func TestResolveAll(t *testing.T) {
	names, ok := Resolve("all", testHosts(), testGroups())
	if !ok || strings.Join(names, ",") != "server,desktop,pi" {
		t.Errorf("Resolve(all) = %v, %v", names, ok)
	}
}

func TestResolveUnknown(t *testing.T) {
	if names, ok := Resolve("ghost", testHosts(), testGroups()); ok {
		t.Errorf("Resolve(ghost) = %v, want not ok", names)
	}
}

func TestResolveGroupBeforeHost(t *testing.T) {
	// A group and host sharing a name: the group wins (parity with Bash).
	hosts := []host.Host{{Name: "x"}}
	groups := []host.Group{{Name: "x", Members: []string{"x"}}}
	names, ok := Resolve("x", hosts, groups)
	if !ok || strings.Join(names, ",") != "x" {
		t.Errorf("Resolve(x) = %v, %v", names, ok)
	}
}
