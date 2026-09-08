package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// newCompletionCmd replaces Cobra's default completion command so that
// `wake completion install` exists next to the plain generators. It stays
// hidden: Homebrew and the release archives use the generators, and humans
// only ever need `install` (which the formula caveats point at).
func newCompletionCmd(a *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "completion [bash|zsh|fish|install]",
		Short:  "generate shell completions ('install' sets them up for your shell)",
		Hidden: true,
	}
	for _, shell := range []string{"bash", "zsh", "fish"} {
		cmd.AddCommand(&cobra.Command{
			Use:   shell,
			Short: "generate the " + shell + " completion script",
			Args:  cobra.NoArgs,
			RunE: func(c *cobra.Command, _ []string) error {
				return genCompletion(c.Root(), c.Name(), c.OutOrStdout())
			},
		})
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "install [bash|zsh|fish]",
		Short: "install completions for your shell (once; detects $SHELL)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			shell := ""
			if len(args) == 1 {
				shell = args[0]
			}
			return a.installCompletion(c.Root(), shell)
		},
	})
	return cmd
}

// genCompletion writes the named shell's completion script for root.
func genCompletion(root *cobra.Command, shell string, w io.Writer) error {
	switch shell {
	case "bash":
		return root.GenBashCompletionV2(w, true)
	case "zsh":
		return root.GenZshCompletion(w)
	case "fish":
		return root.GenFishCompletion(w, true)
	}
	return fmt.Errorf("unsupported shell %q (use: bash, zsh or fish)", shell)
}

// installCompletion writes the completion script into the user-level
// directory the shell loads automatically, so no rc edits are needed for
// bash (bash-completion's XDG dir) and fish. zsh has no such directory, so
// it additionally appends one guarded fpath line to ~/.zshrc.
func (a *App) installCompletion(root *cobra.Command, shell string) error {
	if shell == "" {
		shell = filepath.Base(a.Getenv("SHELL"))
	}
	home := a.Getenv("HOME")
	if home == "" {
		return errors.New("cannot locate your home directory ($HOME is unset)")
	}
	dataDir := a.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		dataDir = filepath.Join(home, ".local", "share")
	}
	confDir := a.Getenv("XDG_CONFIG_HOME")
	if confDir == "" {
		confDir = filepath.Join(home, ".config")
	}

	var path string
	switch shell {
	case "bash":
		path = filepath.Join(dataDir, "bash-completion", "completions", "wake")
	case "zsh":
		path = filepath.Join(dataDir, "zsh", "site-functions", "_wake")
	case "fish":
		path = filepath.Join(confDir, "fish", "completions", "wake.fish")
	default:
		return fmt.Errorf("unsupported shell %q (use: wake completion install bash|zsh|fish)", shell)
	}

	var buf bytes.Buffer
	if err := genCompletion(root, shell, &buf); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Wrote %s\n", path)

	if shell == "zsh" {
		if err := a.ensureZshFpath(home, filepath.Dir(path)); err != nil {
			return err
		}
	}
	fmt.Fprintln(a.Out, "Open a new shell to start using tab completion.")
	return nil
}

// ensureZshFpath appends a guarded fpath line to ~/.zshrc (once).
func (a *App) ensureZshFpath(home, fnDir string) error {
	zshrc := filepath.Join(home, ".zshrc")
	text := ""
	if b, err := os.ReadFile(zshrc); err == nil {
		text = string(b)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	marker := "# added by 'wake completion install'"
	if strings.Contains(text, marker) {
		return nil
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += fmt.Sprintf("fpath=(%s $fpath) %s\n", fnDir, marker)
	if err := os.WriteFile(zshrc, []byte(text), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Added an fpath line to %s (compinit must run after it).\n", zshrc)
	return nil
}
