package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DiegoHeer/wakelab/internal/config"
	"github.com/DiegoHeer/wakelab/internal/host"
	"github.com/DiegoHeer/wakelab/internal/wol"
)

// resolveNames expands a target to host names (see config.Resolve).
func resolveNames(target string, hosts []host.Host, groups []host.Group) ([]string, bool) {
	return config.Resolve(target, hosts, groups)
}

// wakeOptions carries the root command's wake flags.
type wakeOptions struct {
	wait      bool
	timeout   int
	port      string
	viaSet    bool
	via       string
	broadcast string
}

// runWake is `wake <target>`: send WOL to a host, group, 'all', or a raw MAC.
func (a *App) runWake(ctx context.Context, target string, opts wakeOptions) error {
	if opts.port != "" && !host.ValidPort(opts.port) {
		return fmt.Errorf("--port needs a number")
	}

	// A raw MAC wakes directly — this is what relays run: wake <mac> --broadcast <ip>.
	if mac := host.NormalizeMac(target); mac != "" {
		if opts.wait || opts.port != "" || opts.viaSet {
			return fmt.Errorf("a raw MAC target supports only --broadcast (no --wait, --port, or --via)")
		}
		bcast := opts.broadcast
		if bcast == "" {
			bcast = "255.255.255.255"
		}
		fmt.Fprintf(a.Out, "Waking %s via %s...\n", mac, bcast)
		return a.SendWOL(mac, bcast)
	}
	if opts.broadcast != "" {
		return fmt.Errorf("--broadcast only applies to a raw MAC target")
	}

	_, hosts, err := a.loadHosts()
	if err != nil {
		return err
	}
	_, groups, err := a.loadGroups()
	if err != nil {
		return err
	}
	names, ok := resolveNames(target, hosts, groups)
	if !ok {
		return fmt.Errorf("unknown host or group '%s' (see: wake ls / wake groups)", target)
	}
	if len(names) == 0 {
		return fmt.Errorf("target '%s' has no hosts", target)
	}

	var woken []host.Host
	for _, name := range names {
		h := findHost(hosts, name)
		if h.Mac == "" {
			fmt.Fprintf(a.Out, "%sskip %s (no MAC)%s\n", a.color(ansiYellow), name, a.color(ansiReset))
			continue
		}
		relay := h.Via
		if opts.viaSet {
			relay = opts.via // empty forces a local send
		}
		if relay != "" {
			fmt.Fprintf(a.Out, "Waking '%s' (%s) via %s, sent from '%s'...\n", h.Name, h.Mac, h.BroadcastAddr(), relay)
			script, err := wol.RelayScript(h.BroadcastAddr(), h.Mac)
			if err != nil {
				fmt.Fprintf(a.Out, "%s  (bad config for %s: %v)%s\n", a.color(ansiRed), h.Name, err, a.color(ansiReset))
				continue
			}
			if _, _, err := a.Runner.Run(ctx, "ssh", relay, script); err != nil {
				fmt.Fprintf(a.Out, "%s  (relay '%s' failed for %s — is it reachable?)%s\n",
					a.color(ansiRed), relay, h.Name, a.color(ansiReset))
			}
		} else {
			fmt.Fprintf(a.Out, "Waking '%s' (%s) via %s...\n", h.Name, h.Mac, h.BroadcastAddr())
			if err := a.SendWOL(h.Mac, h.BroadcastAddr()); err != nil {
				return err
			}
		}
		woken = append(woken, h)
	}
	if !opts.wait {
		return nil
	}
	if len(woken) == 0 {
		fmt.Fprintf(a.Out, "%sNothing was woken, nothing to wait for.%s\n", a.color(ansiRed), a.color(ansiReset))
		return errSilent
	}
	return a.waitForHosts(ctx, woken, opts.port, opts.timeout)
}

// waitForHosts polls every 2s until every host is online or timeout passes.
func (a *App) waitForHosts(ctx context.Context, hosts []host.Host, waitPort string, timeout int) error {
	portMsg := "each host's port"
	if waitPort != "" {
		portMsg = "port " + waitPort
	}
	names := make([]string, len(hosts))
	for i, h := range hosts {
		names[i] = h.Name
	}
	fmt.Fprintf(a.Out, "Waiting up to %ds for: %s (%s open)\n", timeout, strings.Join(names, " "), portMsg)
	for waited := 0; waited < timeout; waited += 2 {
		allUp := true
		for _, st := range a.checkHosts(ctx, hosts, waitPort, waitPort == "") {
			if st != "online" {
				allUp = false
			}
		}
		if allUp {
			fmt.Fprintf(a.Out, "%sAll online.%s\n", a.color(ansiGreen), a.color(ansiReset))
			return nil
		}
		a.Sleep(2 * time.Second)
	}
	fmt.Fprintf(a.Out, "%sTimeout: not all hosts came online.%s\n", a.color(ansiRed), a.color(ansiReset))
	return errSilent
}
