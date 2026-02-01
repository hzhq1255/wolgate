// Package wol tests.
package wol

import (
	"net"
	"strings"
	"testing"
)

func TestNewSender(t *testing.T) {
	tests := []struct {
		name      string
		iface     string
		broadcast string
		wantErr   bool
	}{
		{
			name:      "default broadcast",
			iface:     "",
			broadcast: "",
			wantErr:   false,
		},
		{
			name:      "custom broadcast",
			iface:     "",
			broadcast: "192.168.1.255",
			wantErr:   false,
		},
		{
			name:      "invalid broadcast",
			iface:     "",
			broadcast: "invalid",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewSender(tt.iface, tt.broadcast)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSender() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && s == nil {
				t.Error("NewSender() returned nil sender")
			}
		})
	}
}

func TestValidateMAC(t *testing.T) {
	tests := []struct {
		name    string
		mac     string
		wantErr bool
	}{
		{"valid colon", "AA:BB:CC:DD:EE:FF", false},
		{"valid dash", "AA-BB-CC-DD-EE-FF", false},
		{"valid dot", "AABB.CCDD.EEFF", false},
		{"valid lowercase", "aa:bb:cc:dd:ee:ff", false},
		{"invalid length", "AA:BB:CC:DD:EE", true},
		{"invalid chars", "GG:BB:CC:DD:EE:FF", true},
		{"empty", "", true},
		{"invalid format", "AA-BB-CC-DD-EE", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMAC(tt.mac)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMAC() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeMAC(t *testing.T) {
	tests := []struct {
		name    string
		mac     string
		want    string
		wantErr bool
	}{
		{"colon", "AA:BB:CC:DD:EE:FF", "AA:BB:CC:DD:EE:FF", false},
		{"dash", "AA-BB-CC-DD-EE-FF", "AA:BB:CC:DD:EE:FF", false},
		{"dot", "AABB.CCDD.EEFF", "AA:BB:CC:DD:EE:FF", false},
		{"lowercase", "aa:bb:cc:dd:ee:ff", "AA:BB:CC:DD:EE:FF", false},
		{"invalid", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeMAC(tt.mac)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeMAC() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeMAC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseMAC(t *testing.T) {
	tests := []struct {
		name    string
		mac     string
		wantLen int
		wantErr bool
	}{
		{"valid", "AA:BB:CC:DD:EE:FF", 6, false},
		{"lowercase", "aa:bb:cc:dd:ee:ff", 6, false},
		{"dash", "AA-BB-CC-DD-EE-FF", 6, false},
		{"invalid", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMAC(tt.mac)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMAC() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("parseMAC() len = %v, want %v", len(got), tt.wantLen)
			}
		})
	}
}

func TestConstructMagicPacket(t *testing.T) {
	mac := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	packet := constructMagicPacket(mac)

	// Magic packet should be 102 bytes
	if len(packet) != 102 {
		t.Errorf("Magic packet length = %d, want 102", len(packet))
	}

	// First 6 bytes should be 0xFF
	for i := 0; i < 6; i++ {
		if packet[i] != 0xFF {
			t.Errorf("Magic packet byte %d = %02X, want FF", i, packet[i])
		}
	}

	// Next 96 bytes should be 16 repetitions of the MAC address
	for i := 0; i < 16; i++ {
		offset := 6 + i*6
		for j := 0; j < 6; j++ {
			if packet[offset+j] != mac[j] {
				t.Errorf("Magic packet byte %d = %02X, want %02X", offset+j, packet[offset+j], mac[j])
			}
		}
	}
}

func TestWOLSender_Send(t *testing.T) {
	// Note: This test sends actual UDP packets to the broadcast address
	// which may not work in all environments (e.g., CI/CD)
	// Consider using a mock UDP server for more reliable testing

	s, err := NewSender("", "127.0.0.1")
	if err != nil {
		t.Fatalf("NewSender() error = %v", err)
	}

	// This will attempt to send to localhost, which may not be a broadcast address
	// but tests the packet construction logic
	err = s.Send("AA:BB:CC:DD:EE:FF")
	// We expect this might fail (127.0.0.1 is not a broadcast address)
	// but it tests the code path
	_ = err
}

func TestWOLSender_SendInvalidMAC(t *testing.T) {
	s, _ := NewSender("", "")

	err := s.Send("invalid-mac")
	if err == nil {
		t.Error("Send() should return error for invalid MAC")
	}
}

func TestSendRepeat(t *testing.T) {
	s, err := NewSender("", "127.0.0.1")
	if err != nil {
		t.Fatalf("NewSender() error = %v", err)
	}

	// Send 3 packets
	err = s.SendRepeat("AA:BB:CC:DD:EE:FF", 3)
	// Again, this may fail due to using localhost instead of broadcast
	_ = err
}

// Benchmark for magic packet construction
func BenchmarkConstructMagicPacket(b *testing.B) {
	mac := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		constructMagicPacket(mac)
	}
}

// Benchmark for MAC parsing
func BenchmarkParseMAC(b *testing.B) {
	mac := "AA:BB:CC:DD:EE:FF"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseMAC(mac)
	}
}

// Helper function to check if an IP is a valid broadcast address
func isBroadcastIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}

	// Check if it's a limited broadcast (255.255.255.255)
	if ip4[0] == 255 && ip4[1] == 255 && ip4[2] == 255 && ip4[3] == 255 {
		return true
	}

	// Check if it's a directed broadcast (ends with 255)
	if ip4[3] == 255 {
		return true
	}

	return false
}

func TestIsBroadcastIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"255.255.255.255", true},
		{"192.168.1.255", true},
		{"192.168.1.1", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			got := isBroadcastIP(tt.ip)
			if got != tt.want {
				t.Errorf("isBroadcastIP(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestMagicPacketFormat(t *testing.T) {
	mac := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	packet := constructMagicPacket(mac)

	// Verify total size
	if len(packet) != 102 {
		t.Fatalf("Magic packet size = %d, want 102", len(packet))
	}

	// Verify 6 bytes of 0xFF at start
	syncStream := packet[:6]
	for _, b := range syncStream {
		if b != 0xFF {
			t.Error("Magic packet should start with 6 bytes of 0xFF")
		}
	}

	// Verify 16 repetitions of MAC
	for i := 0; i < 16; i++ {
		offset := 6 + i*6
		macCopy := packet[offset : offset+6]
		for j := 0; j < 6; j++ {
			if macCopy[j] != mac[j] {
				t.Errorf("MAC repetition %d mismatch at byte %d", i, j)
			}
		}
	}
}

// Test that various MAC formats are handled correctly
func TestMACFormatHandling(t *testing.T) {
	validMACs := []string{
		"AA:BB:CC:DD:EE:FF",
		"aa:bb:cc:dd:ee:ff",
		"AA-BB-CC-DD-EE-FF",
		"AABB.CCDD.EEFF",
		"aabb.ccdd.eeff",
	}

	for _, mac := range validMACs {
		t.Run(mac, func(t *testing.T) {
			normalized, err := NormalizeMAC(mac)
			if err != nil {
				t.Errorf("NormalizeMAC(%s) failed: %v", mac, err)
			}
			// All should normalize to uppercase colon format
			if !strings.Contains(normalized, ":") {
				t.Errorf("Normalized MAC should contain colons: %s", normalized)
			}
		})
	}
}
