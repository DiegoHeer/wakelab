package host

import "testing"

func TestBroadcastAddrDefault(t *testing.T) {
	h := Host{Name: "a", Mac: "aa:bb:cc:dd:ee:ff"}
	if got := h.BroadcastAddr(); got != "255.255.255.255" {
		t.Errorf("BroadcastAddr() = %q, want 255.255.255.255", got)
	}
	h.Broadcast = "192.168.1.255"
	if got := h.BroadcastAddr(); got != "192.168.1.255" {
		t.Errorf("BroadcastAddr() = %q, want 192.168.1.255", got)
	}
}

func TestReadyPortDefault(t *testing.T) {
	h := Host{Name: "a"}
	if got := h.ReadyPort(); got != "22" {
		t.Errorf("ReadyPort() = %q, want 22", got)
	}
	h.Port = "8080"
	if got := h.ReadyPort(); got != "8080" {
		t.Errorf("ReadyPort() = %q, want 8080", got)
	}
}

func TestNormalizeMac(t *testing.T) {
	tests := []struct{ in, want string }{
		{"44:87:63:80:5E:F6", "44:87:63:80:5e:f6"},
		{"44-87-63-80-5e-f6", "44:87:63:80:5e:f6"},
		{"AA:BB:CC:DD:EE:FF", "aa:bb:cc:dd:ee:ff"},
		{"not-a-mac", ""},
		{"44:87:63:80:5e", ""},
		{"44:87:63:80:5e:f6:00", ""},
		{"gg:87:63:80:5e:f6", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := NormalizeMac(tt.in); got != tt.want {
			t.Errorf("NormalizeMac(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestValidIP(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"192.168.1.255", true},
		{"255.255.255.255", true},
		{"0.0.0.0", true},
		{"256.1.1.1", false},
		{"1.2.3", false},
		{"1.2.3.4.5", false},
		{"a.b.c.d", false},
		{"", false},
		{"1.2.3.04", true}, // bash accepts leading zeros
	}
	for _, tt := range tests {
		if got := ValidIP(tt.in); got != tt.want {
			t.Errorf("ValidIP(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestReserved(t *testing.T) {
	for _, name := range []string{"ls", "status", "all", "help", "install-wol", "scan"} {
		if !Reserved(name) {
			t.Errorf("Reserved(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"server", "desktop", ""} {
		if Reserved(name) {
			t.Errorf("Reserved(%q) = true, want false", name)
		}
	}
}

func TestReservedIncludesCobraBuiltins(t *testing.T) {
	for _, name := range []string{"completion", "version"} {
		if !Reserved(name) {
			t.Errorf("Reserved(%q) = false, want true", name)
		}
	}
}

func TestValidPort(t *testing.T) {
	valid := []string{"22", "8080", "1"}
	invalid := []string{"", "+22", "-1", "2 2", "port", "22.5"}
	for _, p := range valid {
		if !ValidPort(p) {
			t.Errorf("ValidPort(%q) = false, want true", p)
		}
	}
	for _, p := range invalid {
		if ValidPort(p) {
			t.Errorf("ValidPort(%q) = true, want false", p)
		}
	}
}
