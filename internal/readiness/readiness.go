// Package readiness answers "is this machine up?" via TCP port checks with a
// ping fallback.
package readiness

import (
	"context"
	"net"
	"runtime"
	"time"

	"github.com/DiegoHeer/wakelab/internal/sshexec"
)

// Prober checks host reachability. Production uses NetProber; tests use fakes.
type Prober interface {
	// PortOpen reports whether TCP ip:port accepts a connection within ~1s.
	PortOpen(ip, port string) bool
	// Ping reports whether ip answers one ping within ~1s.
	Ping(ip string) bool
}

// NetProber is the real Prober: net.DialTimeout for ports, the system ping
// binary (via Runner) for ICMP, which needs no raw-socket privileges.
type NetProber struct {
	Runner sshexec.Runner
}

// PortOpen implements Prober.
func (NetProber) PortOpen(ip, port string) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, port), time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// Ping implements Prober.
func (p NetProber) Ping(ip string) bool {
	args := []string{"-c1", "-W1", ip}
	if runtime.GOOS == "darwin" {
		args = []string{"-c1", "-t1", ip} // macOS: -t is the timeout in seconds
	}
	_, _, err := p.Runner.Run(context.Background(), "ping", args...)
	return err == nil
}
