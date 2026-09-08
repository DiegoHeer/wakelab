package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// completionEnv wires a testEnv whose App resolves env vars from a map.
func completionEnv(t *testing.T, vars map[string]string) *testEnv {
	t.Helper()
	env := newTestEnv(t, "", "")
	if _, ok := vars["HOME"]; !ok {
		vars["HOME"] = env.dir
	}
	env.app.Getenv = func(k string) string { return vars[k] }
	return env
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestCompletionInstallBashFromSHELL(t *testing.T) {
	env := completionEnv(t, map[string]string{"SHELL": "/bin/bash"})
	if err := env.run("completion", "install"); err != nil {
		t.Fatalf("install: %v", err)
	}
	path := filepath.Join(env.dir, ".local", "share", "bash-completion", "completions", "wake")
	mustContain(t, mustReadFile(t, path), "bash completion V2 for wake")
	mustContain(t, env.out.String(), path)
	mustContain(t, env.out.String(), "Open a new shell")
}

func TestCompletionInstallHonorsXDGDataHome(t *testing.T) {
	env := completionEnv(t, map[string]string{"SHELL": "/bin/bash"})
	xdg := filepath.Join(env.dir, "xdg-data")
	env.app.Getenv = func(k string) string {
		switch k {
		case "SHELL":
			return "/bin/bash"
		case "HOME":
			return env.dir
		case "XDG_DATA_HOME":
			return xdg
		}
		return ""
	}
	if err := env.run("completion", "install"); err != nil {
		t.Fatalf("install: %v", err)
	}
	path := filepath.Join(xdg, "bash-completion", "completions", "wake")
	mustContain(t, mustReadFile(t, path), "bash completion V2 for wake")
}

func TestCompletionInstallFishByArg(t *testing.T) {
	env := completionEnv(t, map[string]string{"SHELL": "/bin/bash"})
	if err := env.run("completion", "install", "fish"); err != nil {
		t.Fatalf("install fish: %v", err)
	}
	path := filepath.Join(env.dir, ".config", "fish", "completions", "wake.fish")
	mustContain(t, mustReadFile(t, path), "fish completion for wake")
}

func TestCompletionInstallZshWritesFpathOnce(t *testing.T) {
	env := completionEnv(t, map[string]string{"SHELL": "/usr/bin/zsh"})
	if err := env.run("completion", "install"); err != nil {
		t.Fatalf("install: %v", err)
	}
	fn := filepath.Join(env.dir, ".local", "share", "zsh", "site-functions", "_wake")
	mustContain(t, mustReadFile(t, fn), "#compdef wake")

	zshrc := mustReadFile(t, filepath.Join(env.dir, ".zshrc"))
	mustContain(t, zshrc, "fpath=(")
	mustContain(t, zshrc, "wake completion install")

	// A second run must not duplicate the .zshrc line.
	if err := env.run("completion", "install"); err != nil {
		t.Fatalf("second install: %v", err)
	}
	zshrc = mustReadFile(t, filepath.Join(env.dir, ".zshrc"))
	if n := strings.Count(zshrc, "fpath=("); n != 1 {
		t.Errorf("fpath line appears %d times, want 1:\n%s", n, zshrc)
	}
}

func TestCompletionInstallUnsupportedShell(t *testing.T) {
	env := completionEnv(t, map[string]string{"SHELL": "/bin/tcsh"})
	err := env.run("completion", "install")
	if err == nil || !strings.Contains(err.Error(), "unsupported shell") {
		t.Fatalf("want unsupported-shell error, got %v", err)
	}
}

func TestCompletionGeneratorsWriteToStdout(t *testing.T) {
	for shell, want := range map[string]string{
		"bash": "bash completion V2 for wake",
		"zsh":  "#compdef wake",
		"fish": "fish completion for wake",
	} {
		env := completionEnv(t, map[string]string{})
		if err := env.run("completion", shell); err != nil {
			t.Fatalf("completion %s: %v", shell, err)
		}
		mustContain(t, env.out.String(), want)
	}
}
