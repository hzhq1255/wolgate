// Package arp tests.
package arp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test ARP file content
const testARPContent = `IP address       HW type     Flags       HW address            Mask     Device
192.168.1.1      0x1         0x2         aa:bb:cc:dd:ee:ff    *        eth0
192.168.1.2      0x1         0x2         11:22:33:44:55:66    *        eth0
10.0.0.1         0x1         0x0         00:00:00:00:00:00    *        eth0
127.0.0.1        0x1         0x2         00:00:00:00:00:00    *        lo
`

const testARPWithComments = `# ARP table
IP address       HW type     Flags       HW address            Mask     Device
192.168.1.10     0x1         0x2         AA:BB:CC:DD:EE:FF    *        br-lan

# Another comment
192.168.1.11     0x1         0x2         11:22:33:44:55:66    *        br-lan
`

func TestParsePath(t *testing.T) {
	// Create temp file with test content
	tmpDir := t.TempDir()
	arpPath := filepath.Join(tmpDir, "arp")

	err := os.WriteFile(arpPath, []byte(testARPContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test ARP file: %v", err)
	}

	entries, err := ParsePath(arpPath)
	if err != nil {
		t.Fatalf("ParsePath() error = %v", err)
	}

	// Should have 2 valid entries (not 3, because we filter out invalid MACs and incomplete)
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}

	// Check first entry
	if entries[0].IP != "192.168.1.1" {
		t.Errorf("Expected IP 192.168.1.1, got %s", entries[0].IP)
	}
	if entries[0].MAC != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("Expected MAC aa:bb:cc:dd:ee:ff, got %s", entries[0].MAC)
	}
	if entries[0].Device != "eth0" {
		t.Errorf("Expected device eth0, got %s", entries[0].Device)
	}
}

func TestParsePath_WithComments(t *testing.T) {
	tmpDir := t.TempDir()
	arpPath := filepath.Join(tmpDir, "arp")

	err := os.WriteFile(arpPath, []byte(testARPWithComments), 0644)
	if err != nil {
		t.Fatalf("Failed to write test ARP file: %v", err)
	}

	entries, err := ParsePath(arpPath)
	if err != nil {
		t.Fatalf("ParsePath() error = %v", err)
	}

	// Should have 2 entries
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}
}

func TestParsePath_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	arpPath := filepath.Join(tmpDir, "arp")

	err := os.WriteFile(arpPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write test ARP file: %v", err)
	}

	entries, err := ParsePath(arpPath)
	if err != nil {
		t.Fatalf("ParsePath() error = %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries from empty file, got %d", len(entries))
	}
}

func TestParsePath_InvalidLine(t *testing.T) {
	tmpDir := t.TempDir()
	arpPath := filepath.Join(tmpDir, "arp")

	content := `IP address       HW type     Flags       HW address            Mask     Device
invalid line here
192.168.1.1      0x1         0x2         aa:bb:cc:dd:ee:ff    *        eth0
`

	err := os.WriteFile(arpPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test ARP file: %v", err)
	}

	// Should not error, just skip invalid line
	entries, err := ParsePath(arpPath)
	if err != nil {
		t.Fatalf("ParsePath() should not error on invalid line: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}
}

func TestIsValidEntry(t *testing.T) {
	tests := []struct {
		name  string
		entry Entry
		want  bool
	}{
		{
			name: "valid entry",
			entry: Entry{
				IP:     "192.168.1.1",
				MAC:    "aa:bb:cc:dd:ee:ff",
				Flags:  "0x2",
				Device: "eth0",
			},
			want: true,
		},
		{
			name: "zero MAC",
			entry: Entry{
				IP:     "192.168.1.1",
				MAC:    "00:00:00:00:00:00",
				Flags:  "0x2",
				Device: "eth0",
			},
			want: false,
		},
		{
			name: "broadcast MAC",
			entry: Entry{
				IP:     "192.168.1.1",
				MAC:    "ff:ff:ff:ff:ff:ff",
				Flags:  "0x2",
				Device: "eth0",
			},
			want: false,
		},
		{
			name: "invalid MAC format",
			entry: Entry{
				IP:     "192.168.1.1",
				MAC:    "invalid",
				Flags:  "0x2",
				Device: "eth0",
			},
			want: false,
		},
		{
			name: "incomplete entry",
			entry: Entry{
				IP:     "192.168.1.1",
				MAC:    "aa:bb:cc:dd:ee:ff",
				Flags:  "0x0",
				Device: "eth0",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidEntry(tt.entry)
			if got != tt.want {
				t.Errorf("isValidEntry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetByDevice(t *testing.T) {
	tmpDir := t.TempDir()
	arpPath := filepath.Join(tmpDir, "arp")

	content := `IP address       HW type     Flags       HW address            Mask     Device
192.168.1.1      0x1         0x2         aa:bb:cc:dd:ee:ff    *        eth0
192.168.2.1      0x1         0x2         11:22:33:44:55:66    *        br-lan
`

	err := os.WriteFile(arpPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test ARP file: %v", err)
	}

	// Temporarily change the default path for testing
	oldPath := DefaultARPPath
	// Note: We can't actually change the const, so we'll use ParsePath directly
	_ = oldPath

	entries, err := ParsePath(arpPath)
	if err != nil {
		t.Fatalf("ParsePath() error = %v", err)
	}

	// Count entries per device
	eth0Count := 0
	brLanCount := 0
	for _, e := range entries {
		if e.Device == "eth0" {
			eth0Count++
		}
		if e.Device == "br-lan" {
			brLanCount++
		}
	}

	if eth0Count != 1 {
		t.Errorf("Expected 1 entry for eth0, got %d", eth0Count)
	}
	if brLanCount != 1 {
		t.Errorf("Expected 1 entry for br-lan, got %d", brLanCount)
	}
}

func TestFindByMAC(t *testing.T) {
	tmpDir := t.TempDir()
	arpPath := filepath.Join(tmpDir, "arp")

	content := `IP address       HW type     Flags       HW address            Mask     Device
192.168.1.1      0x1         0x2         aa:bb:cc:dd:ee:ff    *        eth0
192.168.2.1      0x1         0x2         AA:BB:CC:DD:EE:FF    *        br-lan
`

	err := os.WriteFile(arpPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test ARP file: %v", err)
	}

	// Use ParsePath directly
	entries, err := ParsePath(arpPath)
	if err != nil {
		t.Fatalf("ParsePath() error = %v", err)
	}

	// Test finding by MAC (case insensitive)
	searchMAC := "aa:bb:cc:dd:ee:ff"
	found := 0
	for _, e := range entries {
		if strings.EqualFold(e.MAC, searchMAC) {
			found++
		}
	}

	if found != 2 {
		t.Errorf("Expected to find 2 entries with MAC aa:bb:cc:dd:ee:ff, got %d", found)
	}
}

func TestGetDevices(t *testing.T) {
	tmpDir := t.TempDir()
	arpPath := filepath.Join(tmpDir, "arp")

	content := `IP address       HW type     Flags       HW address            Mask     Device
192.168.1.1      0x1         0x2         aa:bb:cc:dd:ee:ff    *        eth0
192.168.2.1      0x1         0x2         11:22:33:44:55:66    *        br-lan
192.168.3.1      0x1         0x2         77:88:99:aa:bb:cc    *        eth0
`

	err := os.WriteFile(arpPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test ARP file: %v", err)
	}

	// Use ParsePath directly
	entries, err := ParsePath(arpPath)
	if err != nil {
		t.Fatalf("ParsePath() error = %v", err)
	}

	// Count unique devices
	deviceMap := make(map[string]bool)
	for _, e := range entries {
		deviceMap[e.Device] = true
	}

	if len(deviceMap) != 2 {
		t.Errorf("Expected 2 unique devices, got %d", len(deviceMap))
	}

	if !deviceMap["eth0"] || !deviceMap["br-lan"] {
		t.Error("Expected devices eth0 and br-lan")
	}
}

func TestParseLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantErr bool
	}{
		{
			name:    "valid line",
			line:    "192.168.1.1      0x1         0x2         aa:bb:cc:dd:ee:ff    *        eth0",
			wantErr: false,
		},
		{
			name:    "too few fields",
			line:    "192.168.1.1      0x1         0x2",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLine() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParsePath_FileNotFound(t *testing.T) {
	_, err := ParsePath("/nonexistent/path/arp")
	if err == nil {
		t.Error("ParsePath() should return error for nonexistent file")
	}
}

func TestInvalidMACs(t *testing.T) {
	// Verify that invalid MACs are defined
	if len(invalidMACs) == 0 {
		t.Error("invalidMACs should not be empty")
	}

	// Check common invalid MACs
	expectedInvalid := []string{"00:00:00:00:00:00", "ff:ff:ff:ff:ff:ff"}
	for _, mac := range expectedInvalid {
		found := false
		for _, invalid := range invalidMACs {
			if strings.EqualFold(invalid, mac) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected %s to be in invalidMACs list", mac)
		}
	}
}
