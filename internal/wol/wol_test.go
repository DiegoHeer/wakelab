package wol

import (
	"bytes"
	"net"
	"strings"
	"testing"
	"time"
)

func TestBuildMagicPacket(t *testing.T) {
	pkt, err := BuildMagicPacket("44:87:63:80:5e:f6")
	if err != nil {
		t.Fatalf("BuildMagicPacket: %v", err)
	}
	if len(pkt) != 102 {
		t.Fatalf("packet length = %d, want 102", len(pkt))
	}
	if !bytes.Equal(pkt[:6], bytes.Repeat([]byte{0xFF}, 6)) {
		t.Errorf("header = % x, want 6 x ff", pkt[:6])
	}
	mac := []byte{0x44, 0x87, 0x63, 0x80, 0x5e, 0xf6}
	for i := 0; i < 16; i++ {
		got := pkt[6+i*6 : 12+i*6]
		if !bytes.Equal(got, mac) {
			t.Fatalf("repetition %d = % x, want % x", i, got, mac)
		}
	}
}

func TestBuildMagicPacketAcceptsDashes(t *testing.T) {
	a, _ := BuildMagicPacket("44-87-63-80-5E-F6")
	b, _ := BuildMagicPacket("44:87:63:80:5e:f6")
	if !bytes.Equal(a, b) {
		t.Error("dash and colon forms must build the same packet")
	}
}

func TestBuildMagicPacketInvalid(t *testing.T) {
	for _, mac := range []string{"", "not-a-mac", "44:87:63:80:5e", "gg:87:63:80:5e:f6"} {
		if _, err := BuildMagicPacket(mac); err == nil {
			t.Errorf("BuildMagicPacket(%q) should fail", mac)
		}
	}
}

func TestSendLoopback(t *testing.T) {
	// A real UDP socket, but fully local: listen on 127.0.0.1 and receive.
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer conn.Close()
	port := conn.LocalAddr().(*net.UDPAddr).Port

	pkt, _ := BuildMagicPacket("aa:bb:cc:dd:ee:ff")
	if err := Send(pkt, "127.0.0.1", port); err != nil {
		t.Fatalf("Send: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 200)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(buf[:n], pkt) {
		t.Errorf("received %d bytes, not the sent packet", n)
	}
}

func TestRelayScript(t *testing.T) {
	s, err := RelayScript("192.168.1.255", "AA-BB-CC-DD-EE-FF")
	if err != nil {
		t.Fatalf("RelayScript: %v", err)
	}
	// Inputs are canonicalized: lower-case colon MAC, dotted-quad broadcast.
	for _, want := range []string{"wake", "wakeonlan", "python3", "192.168.1.255", "aa:bb:cc:dd:ee:ff"} {
		if !strings.Contains(s, want) {
			t.Errorf("RelayScript missing %q:\n%s", want, s)
		}
	}
	// The python fallback embeds the packet as hex: 6 x ff + 16 x mac.
	if !strings.Contains(s, "ffffffffffff") || !strings.Contains(s, "aabbccddeeff") {
		t.Errorf("RelayScript python fallback lacks packet hex:\n%s", s)
	}
}

func TestRelayScriptRejectsBadInput(t *testing.T) {
	if _, err := RelayScript("192.168.1.255", "aa:bb; rm -rf /"); err == nil {
		t.Error("RelayScript must reject an invalid MAC")
	}
	if _, err := RelayScript("999.1.1.1'; reboot;'", "aa:bb:cc:dd:ee:ff"); err == nil {
		t.Error("RelayScript must reject an invalid broadcast")
	}
	if _, err := RelayScript("ff02::1", "aa:bb:cc:dd:ee:ff"); err == nil {
		t.Error("RelayScript must reject a non-IPv4 broadcast")
	}
}

func TestSendRejectsNonIPv4(t *testing.T) {
	pkt, _ := BuildMagicPacket("aa:bb:cc:dd:ee:ff")
	for _, addr := range []string{"ff02::1", "not-an-ip", ""} {
		if err := Send(pkt, addr, 9); err == nil {
			t.Errorf("Send(%q) should fail", addr)
		}
	}
}
