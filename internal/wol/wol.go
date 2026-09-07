// Package wol builds and sends Wake-on-LAN magic packets natively (no
// external wakeonlan dependency).
package wol

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"syscall"
)

// DefaultPort is the conventional Wake-on-LAN UDP port.
const DefaultPort = 9

// BuildMagicPacket returns the 102-byte magic packet for a MAC address
// (colon- or dash-separated): 6 x 0xFF followed by 16 repetitions of the MAC.
func BuildMagicPacket(mac string) ([]byte, error) {
	clean := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(mac, "-", ""), ":", ""))
	raw, err := hex.DecodeString(clean)
	if err != nil || len(raw) != 6 {
		return nil, fmt.Errorf("invalid MAC address %q", mac)
	}
	pkt := make([]byte, 0, 102)
	for i := 0; i < 6; i++ {
		pkt = append(pkt, 0xFF)
	}
	for i := 0; i < 16; i++ {
		pkt = append(pkt, raw...)
	}
	return pkt, nil
}

// Send emits the packet over UDP to addr:port with SO_BROADCAST set, so
// broadcast addresses like 255.255.255.255 work.
func Send(packet []byte, addr string, port int) error {
	ip := net.ParseIP(addr)
	if ip == nil {
		return fmt.Errorf("invalid broadcast address %q", addr)
	}
	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	rc, err := conn.SyscallConn()
	if err != nil {
		return err
	}
	var soErr error
	if err := rc.Control(func(fd uintptr) {
		soErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
	}); err != nil {
		return err
	}
	if soErr != nil {
		return soErr
	}
	_, err = conn.WriteToUDP(packet, &net.UDPAddr{IP: ip, Port: port})
	return err
}

// RelayScript returns a POSIX sh script that emits a magic packet on the
// machine it runs on, picking the best available sender: wake (this tool),
// else wakeonlan, else a python3 one-liner with SO_BROADCAST.
func RelayScript(bcast, mac string) string {
	hexMac := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(mac, "-", ""), ":", ""))
	packetHex := strings.Repeat("ff", 6) + strings.Repeat(hexMac, 16)
	py := fmt.Sprintf(`import socket;s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);s.setsockopt(socket.SOL_SOCKET,socket.SO_BROADCAST,1);s.sendto(bytes.fromhex("%s"),("%s",9))`, packetHex, bcast)
	return fmt.Sprintf(`if command -v wake >/dev/null 2>&1; then wake %s --broadcast %s
elif command -v wakeonlan >/dev/null 2>&1; then wakeonlan -i %s %s
elif command -v python3 >/dev/null 2>&1; then python3 -c '%s'
else echo "no WOL sender (wake/wakeonlan/python3) found" >&2; exit 1
fi`, mac, bcast, bcast, mac, py)
}
