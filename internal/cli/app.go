package cli

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/DiegoHeer/wakelab/internal/config"
	"github.com/DiegoHeer/wakelab/internal/host"
	"github.com/DiegoHeer/wakelab/internal/readiness"
	"github.com/DiegoHeer/wakelab/internal/sshexec"
	"github.com/DiegoHeer/wakelab/internal/wol"
)

// App carries every side-effect dependency of the commands. Production wiring
// comes from NewApp; tests substitute fakes.
type App struct {
	HostsPath     string
	GroupsPath    string
	SSHConfigPath string

	Runner   sshexec.Runner
	Prober   readiness.Prober
	SendWOL  func(mac, bcast string) error
	Out, Err io.Writer
	IsTTY    bool
	Sleep    func(time.Duration)
	LookPath func(string) (string, error)
}

// NewApp wires the real world.
func NewApp() *App {
	home, _ := os.UserHomeDir()
	runner := sshexec.ExecRunner{}
	fi, _ := os.Stdout.Stat()
	return &App{
		HostsPath:     config.HostsPath(),
		GroupsPath:    config.GroupsPath(),
		SSHConfigPath: filepath.Join(home, ".ssh", "config"),
		Runner:        runner,
		Prober:        readiness.NetProber{Runner: runner},
		SendWOL: func(mac, bcast string) error {
			pkt, err := wol.BuildMagicPacket(mac)
			if err != nil {
				return err
			}
			return wol.Send(pkt, bcast, wol.DefaultPort)
		},
		Out:      os.Stdout,
		Err:      os.Stderr,
		IsTTY:    fi != nil && fi.Mode()&os.ModeCharDevice != 0,
		Sleep:    time.Sleep,
		LookPath: exec.LookPath,
	}
}

// loadHosts returns the hosts file text and its parsed hosts.
func (a *App) loadHosts() (string, []host.Host, error) {
	text, err := config.Load(a.HostsPath)
	if err != nil {
		return "", nil, err
	}
	return text, config.ParseHosts(text), nil
}

// loadGroups returns the groups file text and its parsed groups.
func (a *App) loadGroups() (string, []host.Group, error) {
	text, err := config.Load(a.GroupsPath)
	if err != nil {
		return "", nil, err
	}
	return text, config.ParseGroups(text), nil
}

// findHost returns the named host, or a zero Host carrying just the name
// (a group can list members that have no ~/.wol_hosts block).
func findHost(hosts []host.Host, name string) host.Host {
	for _, h := range hosts {
		if h.Name == name {
			return h
		}
	}
	return host.Host{Name: name}
}

// hostIP resolves a host's IP through `ssh -G` ("" when unresolvable).
func (a *App) hostIP(ctx context.Context, name string) string {
	return sshexec.SSHOption(ctx, a.Runner, name, "hostname")
}

// ANSI colors, empty when stdout is not a terminal (parity with the Bash tool).
const (
	ansiGreen  = "\x1b[32m"
	ansiRed    = "\x1b[31m"
	ansiYellow = "\x1b[33m"
	ansiDim    = "\x1b[2m"
	ansiBold   = "\x1b[1m"
	ansiReset  = "\x1b[0m"
)

func (a *App) color(code string) string {
	if a.IsTTY {
		return code
	}
	return ""
}

// paint renders a status word in its conventional color.
func (a *App) paint(status string) string {
	switch status {
	case "online":
		return a.color(ansiGreen) + "online" + a.color(ansiReset)
	case "offline":
		return a.color(ansiRed) + "offline" + a.color(ansiReset)
	default:
		return a.color(ansiDim) + "unknown" + a.color(ansiReset)
	}
}

// statusOf reports one host's status: "online", "offline", or "unknown".
// checkPort (if set) overrides the probe port for every host; useHostPort
// probes each host's own readiness port; with neither, a plain ping decides.
func (a *App) statusOf(ctx context.Context, h host.Host, checkPort string, useHostPort bool) string {
	ip := a.hostIP(ctx, h.Name)
	if ip == "" {
		return "unknown"
	}
	port := ""
	if checkPort != "" {
		port = checkPort
	} else if useHostPort {
		port = h.ReadyPort()
	}
	up := false
	if port != "" {
		up = a.Prober.PortOpen(ip, port)
	} else {
		up = a.Prober.Ping(ip)
	}
	if up {
		return "online"
	}
	return "offline"
}

// checkHosts probes a list of hosts in parallel, preserving order.
func (a *App) checkHosts(ctx context.Context, hosts []host.Host, checkPort string, useHostPort bool) []string {
	statuses := make([]string, len(hosts))
	var wg sync.WaitGroup
	for i, h := range hosts {
		wg.Add(1)
		go func(i int, h host.Host) {
			defer wg.Done()
			statuses[i] = a.statusOf(ctx, h, checkPort, useHostPort)
		}(i, h)
	}
	wg.Wait()
	return statuses
}
