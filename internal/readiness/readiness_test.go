package readiness

import (
	"context"
	"errors"
	"net"
	"strconv"
	"testing"
)

func TestPortOpen(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)

	p := NetProber{}
	if !p.PortOpen("127.0.0.1", port) {
		t.Error("PortOpen = false for a listening port")
	}
	ln.Close()
	if p.PortOpen("127.0.0.1", port) {
		t.Error("PortOpen = true for a closed port")
	}
}

type pingRunner struct{ fail bool }

func (p pingRunner) Run(_ context.Context, _ string, _ ...string) (string, string, error) {
	if p.fail {
		return "", "", errors.New("host unreachable")
	}
	return "", "", nil
}
func (p pingRunner) RunTTY(_ context.Context, _ string, _ ...string) error { return nil }

func TestPing(t *testing.T) {
	if !(NetProber{Runner: pingRunner{fail: false}}).Ping("1.2.3.4") {
		t.Error("Ping = false when ping succeeds")
	}
	if (NetProber{Runner: pingRunner{fail: true}}).Ping("1.2.3.4") {
		t.Error("Ping = true when ping fails")
	}
}
