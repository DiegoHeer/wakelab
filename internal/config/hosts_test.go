package config

import (
	"strings"
	"testing"

	"github.com/DiegoHeer/wakelab/internal/host"
)

const sampleHosts = `Host server
    Mac        44:87:63:80:5e:f6

Host desktop
    Mac        b0:41:6f:16:32:ee
    Broadcast  255.255.255.255
    Via        proxmox

# a comment between blocks
Host pi
    mac        02:00:96:71:f5:bb
    port       80
`

func TestParseHosts(t *testing.T) {
	hosts := ParseHosts(sampleHosts)
	if len(hosts) != 3 {
		t.Fatalf("got %d hosts, want 3", len(hosts))
	}
	want := []host.Host{
		{Name: "server", Mac: "44:87:63:80:5e:f6"},
		{Name: "desktop", Mac: "b0:41:6f:16:32:ee", Broadcast: "255.255.255.255", Via: "proxmox"},
		{Name: "pi", Mac: "02:00:96:71:f5:bb", Port: "80"},
	}
	for i, w := range want {
		if hosts[i] != w {
			t.Errorf("host %d = %+v, want %+v", i, hosts[i], w)
		}
	}
}

func TestParseHostsEmpty(t *testing.T) {
	if got := ParseHosts(""); len(got) != 0 {
		t.Errorf("ParseHosts(\"\") = %v, want empty", got)
	}
}

func TestParseHostsFirstKeyWins(t *testing.T) {
	text := "Host a\n    Mac 11:11:11:11:11:11\n    Mac 22:22:22:22:22:22\n"
	hosts := ParseHosts(text)
	if hosts[0].Mac != "11:11:11:11:11:11" {
		t.Errorf("Mac = %q, want the first occurrence", hosts[0].Mac)
	}
}

func TestFindHost(t *testing.T) {
	h, ok := FindHost(sampleHosts, "desktop")
	if !ok || h.Via != "proxmox" {
		t.Errorf("FindHost desktop = %+v, %v", h, ok)
	}
	if _, ok := FindHost(sampleHosts, "nope"); ok {
		t.Error("FindHost nope should not be found")
	}
}

func TestAppendHostRoundTrip(t *testing.T) {
	h := host.Host{Name: "new", Mac: "aa:bb:cc:dd:ee:ff", Broadcast: "10.0.0.255", Port: "80", Via: "relay"}
	out := AppendHost(sampleHosts, h)
	got, ok := FindHost(out, "new")
	if !ok || got != h {
		t.Errorf("round trip = %+v, %v; want %+v", got, ok, h)
	}
	// Existing hosts survive untouched.
	if len(ParseHosts(out)) != 4 {
		t.Errorf("got %d hosts, want 4", len(ParseHosts(out)))
	}
}

func TestAppendHostBlockFormat(t *testing.T) {
	h := host.Host{Name: "x", Mac: "aa:bb:cc:dd:ee:ff"}
	out := AppendHost("", h)
	want := "Host x\n    Mac        aa:bb:cc:dd:ee:ff\n\n"
	if out != want {
		t.Errorf("AppendHost = %q, want %q", out, want)
	}
}

func TestRemoveHostKeepsRest(t *testing.T) {
	out := RemoveHost(sampleHosts, "desktop")
	if _, ok := FindHost(out, "desktop"); ok {
		t.Error("desktop still present after RemoveHost")
	}
	if _, ok := FindHost(out, "server"); !ok {
		t.Error("server lost by RemoveHost")
	}
	if _, ok := FindHost(out, "pi"); !ok {
		t.Error("pi lost by RemoveHost")
	}
	// The comment line outside the removed block must survive.
	if !strings.Contains(out, "# a comment between blocks") {
		t.Error("comment line lost by RemoveHost")
	}
}

func TestAppendHostNoTrailingNewline(t *testing.T) {
	out := AppendHost("Host a\n    Mac        11:11:11:11:11:11", host.Host{Name: "b", Mac: "22:22:22:22:22:22"})
	hosts := ParseHosts(out)
	if len(hosts) != 2 || hosts[0].Name != "a" || hosts[1].Name != "b" {
		t.Errorf("hosts = %+v, want a and b intact", hosts)
	}
}

func TestRemoveHostLastBlockKeepsTrailingNewline(t *testing.T) {
	out := RemoveHost(sampleHosts, "pi")
	if out != "" && !strings.HasSuffix(out, "\n") {
		t.Errorf("RemoveHost result must end with a newline, got %q", out)
	}
	if _, ok := FindHost(out, "server"); !ok {
		t.Error("server lost")
	}
}

func TestBareHostLineTerminatesBlock(t *testing.T) {
	text := "Host a\n    Mac 11:11:11:11:11:11\nHost\n    Via relay\n"
	hosts := ParseHosts(text)
	if len(hosts) != 1 || hosts[0].Via != "" {
		t.Errorf("hosts = %+v; a bare Host line must end the block (Via must not bleed into a)", hosts)
	}
	out := RemoveHost(text, "a")
	if !strings.Contains(out, "Via relay") {
		t.Errorf("RemoveHost must stop at the bare Host line, got %q", out)
	}
}
