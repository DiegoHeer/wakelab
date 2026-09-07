package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileCreatesEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts")
	text, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if text != "" {
		t.Errorf("Load = %q, want empty", text)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestSaveThenLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts")
	if err := Save(path, "Host a\n"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	text, err := Load(path)
	if err != nil || text != "Host a\n" {
		t.Errorf("Load = %q, %v", text, err)
	}
}

func TestDefaultPaths(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	if p := HostsPath(); p != "/home/test/.wol_hosts" {
		t.Errorf("HostsPath = %q", p)
	}
	if p := GroupsPath(); p != "/home/test/.wol_groups" {
		t.Errorf("GroupsPath = %q", p)
	}
}
