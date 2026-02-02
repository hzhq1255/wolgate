// Package wol implements Wake-on-LAN magic packet sending.
package wol

import (
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

// macRegex validates MAC address format (XX:XX:XX:XX:XX:XX)
var macRegex = regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`)

// WOLSender sends Wake-on-LAN magic packets.
type WOLSender struct {
	iface     string
	broadcast string
}

// NewSender creates a new WOL sender.
// If iface is empty, uses the default network interface.
// If broadcast is empty, uses "255.255.255.255".
func NewSender(iface, broadcast string) (*WOLSender, error) {
	// Set default broadcast address
	if broadcast == "" {
		broadcast = "255.255.255.255"
	}

	// Validate broadcast address
	if net.ParseIP(broadcast) == nil {
		return nil, fmt.Errorf("invalid broadcast address: %s", broadcast)
	}

	return &WOLSender{
		iface:     iface,
		broadcast: broadcast,
	}, nil
}

// Send sends a Wake-on-LAN magic packet to the specified MAC address.
func (w *WOLSender) Send(mac string) error {
	return w.SendRepeat(mac, 1)
}

// SendRepeat sends multiple Wake-on-LAN magic packets for reliability.
func (w *WOLSender) SendRepeat(mac string, count int) error {
	// Validate and normalize MAC address
	macBytes, err := parseMAC(mac)
	if err != nil {
		return err
	}

	// Construct magic packet
	magicPacket := constructMagicPacket(macBytes)

	// Send the packet
	for i := 0; i < count; i++ {
		if err := w.sendPacket(magicPacket); err != nil {
			return err
		}
		// Small delay between packets
		if i < count-1 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	return nil
}

// parseMAC validates and parses a MAC address string.
func parseMAC(mac string) ([]byte, error) {
	// Remove any separators
	mac = strings.TrimSpace(mac)

	// Handle Cisco-style dot format (AABB.CCDD.EEFF)
	if strings.Contains(mac, ".") {
		mac = strings.ReplaceAll(mac, ".", "")
		mac = insertColons(mac, 2)
	} else {
		// Handle dash and colon formats
		mac = strings.ReplaceAll(mac, "-", ":")
	}

	// Validate format
	if !macRegex.MatchString(mac) {
		return nil, fmt.Errorf("invalid MAC address format: %s", mac)
	}

	// Parse hex bytes
	mac = strings.ReplaceAll(mac, ":", "")
	macBytes, err := hex.DecodeString(mac)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MAC address: %w", err)
	}

	return macBytes, nil
}

// insertColons inserts colons every n characters.
func insertColons(s string, n int) string {
	var result string
	for i, r := range s {
		if i > 0 && i%n == 0 {
			result += ":"
		}
		result += string(r)
	}
	return result
}

// constructMagicPacket creates a Wake-on-LAN magic packet.
// The packet consists of:
// - 6 bytes of 0xFF (synchronization stream)
// - 16 repetitions of the target MAC address (96 bytes)
// Total: 102 bytes
func constructMagicPacket(mac []byte) []byte {
	packet := make([]byte, 6+16*6)

	// First 6 bytes: 0xFF synchronization stream
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}

	// Next 96 bytes: 16 repetitions of the MAC address
	for i := 0; i < 16; i++ {
		copy(packet[6+i*6:], mac)
	}

	return packet
}

// sendPacket sends a magic packet via UDP broadcast.
func (w *WOLSender) sendPacket(packet []byte) error {
	// Create UDP connection
	var conn *net.UDPConn
	var err error

	if w.iface != "" {
		// Set the interface to use for sending
		iface, err := net.InterfaceByName(w.iface)
		if err != nil {
			return fmt.Errorf("interface %s not found: %w", w.iface, err)
		}

		// Get the interface addresses
		addrs, err := iface.Addrs()
		if err != nil {
			return fmt.Errorf("failed to get interface addresses: %w", err)
		}

		// Find a suitable IPv4 address to bind to
		var localIP net.IP
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() {
				localIP = ipnet.IP
				break
			}
		}

		if localIP == nil {
			return fmt.Errorf("no suitable IPv4 address found on interface %s", w.iface)
		}

		// Bind to specific interface IP to control outgoing interface
		addr := &net.UDPAddr{
			IP:   localIP,
			Port: 0,
		}

		conn, err = net.ListenUDP("udp4", addr)
		if err != nil {
			return fmt.Errorf("failed to create UDP socket on %s: %w", w.iface, err)
		}
		defer conn.Close()
	} else {
		// Bind to any available interface
		addr := &net.UDPAddr{
			IP:   net.IPv4zero,
			Port: 0,
		}

		conn, err = net.ListenUDP("udp4", addr)
		if err != nil {
			return fmt.Errorf("failed to create UDP socket: %w", err)
		}
		defer conn.Close()
	}

	// Set broadcast permission
	// In Go, this is handled automatically when sending to a broadcast address

	// Send to broadcast address
	destAddr := &net.UDPAddr{
		IP:   net.ParseIP(w.broadcast),
		Port: 9, // Standard WOL port (alternatively 7)
	}

	_, err = conn.WriteToUDP(packet, destAddr)
	if err != nil {
		return fmt.Errorf("failed to send magic packet: %w", err)
	}

	return nil
}

// ValidateMAC validates a MAC address string format.
func ValidateMAC(mac string) error {
	mac = strings.TrimSpace(mac)

	// Handle Cisco-style dot format (AABB.CCDD.EEFF)
	if strings.Contains(mac, ".") {
		mac = strings.ReplaceAll(mac, ".", "")
		mac = insertColons(mac, 2)
	} else {
		// Handle dash and colon formats
		mac = strings.ReplaceAll(mac, "-", ":")
	}

	if !macRegex.MatchString(mac) {
		return fmt.Errorf("invalid MAC address format: %s", mac)
	}
	return nil
}

// NormalizeMAC normalizes a MAC address to XX:XX:XX:XX:XX:XX format.
func NormalizeMAC(mac string) (string, error) {
	bytes, err := parseMAC(mac)
	if err != nil {
		return "", err
	}

	parts := make([]string, 6)
	for i, b := range bytes {
		parts[i] = fmt.Sprintf("%02X", b)
	}
	return strings.Join(parts, ":"), nil
}
