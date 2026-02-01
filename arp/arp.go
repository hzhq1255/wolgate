// Package arp provides ARP table parsing functionality.
package arp

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Entry represents an ARP table entry.
type Entry struct {
	IP     string // IP address
	MAC    string // MAC address
	Device string // Network interface name
	Flags  string // ARP flags (0x0= incomplete, 0x2= complete, etc.)
}

// DefaultARPPath is the default path to the ARP table.
const DefaultARPPath = "/proc/net/arp"

// macRegex matches MAC addresses in various formats
var macRegex = regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`)

// invalidMACs contains MAC addresses that should be filtered out
var invalidMACs = []string{
	"00:00:00:00:00:00",
	"ff:ff:ff:ff:ff:ff",
}

// Parse parses the ARP table at the default path.
func Parse() ([]Entry, error) {
	return ParsePath(DefaultARPPath)
}

// ParsePath parses the ARP table from a specific file path.
func ParsePath(path string) ([]Entry, error) {
	// Open the ARP table file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open ARP table: %w", err)
	}
	defer file.Close()

	var entries []Entry
	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Skip header line
	if scanner.Scan() {
		lineNum++
	}

	// Parse each line
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		entry, err := parseLine(line)
		if err != nil {
			// Log error but continue parsing other lines
			// fmt.Fprintf(os.Stderr, "Warning: line %d: %v\n", lineNum, err)
			continue
		}

		// Filter out invalid entries
		if isValidEntry(entry) {
			entries = append(entries, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading ARP table: %w", err)
	}

	return entries, nil
}

// parseLine parses a single line from the ARP table.
// Format: IP address HW type Flags HW address Mask Device
func parseLine(line string) (Entry, error) {
	fields := strings.Fields(line)
	if len(fields) < 6 {
		return Entry{}, fmt.Errorf("invalid ARP entry format: %s", line)
	}

	entry := Entry{
		IP:     fields[0],
		Flags:  fields[2],
		MAC:    fields[3],
		Device: fields[5],
	}

	return entry, nil
}

// isValidEntry checks if an ARP entry is valid for use.
func isValidEntry(entry Entry) bool {
	// Check MAC address format
	if !macRegex.MatchString(entry.MAC) {
		return false
	}

	// Check for invalid MAC addresses
	for _, invalid := range invalidMACs {
		if strings.EqualFold(entry.MAC, invalid) {
			return false
		}
	}

	// Check for incomplete entries (flags 0x0)
	if entry.Flags == "0x0" {
		return false
	}

	return true
}

// GetLocalEntries returns ARP entries for local network devices.
// This filters out entries for the loopback interface and incomplete entries.
func GetLocalEntries() ([]Entry, error) {
	entries, err := Parse()
	if err != nil {
		return nil, err
	}

	// Filter for local interfaces only
	var localEntries []Entry
	for _, entry := range entries {
		// Skip loopback
		if entry.Device == "lo" {
			continue
		}

		// Skip entries with no device
		if entry.Device == "" {
			continue
		}

		localEntries = append(localEntries, entry)
	}

	return localEntries, nil
}

// GetByDevice returns ARP entries for a specific device.
func GetByDevice(device string) ([]Entry, error) {
	entries, err := Parse()
	if err != nil {
		return nil, err
	}

	var deviceEntries []Entry
	for _, entry := range entries {
		if entry.Device == device {
			deviceEntries = append(deviceEntries, entry)
		}
	}

	return deviceEntries, nil
}

// GetByIP returns an ARP entry for a specific IP address.
func GetByIP(ip string) (*Entry, error) {
	entries, err := Parse()
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IP == ip {
			return &entry, nil
		}
	}

	return nil, fmt.Errorf("IP %s not found in ARP table", ip)
}

// FindByMAC finds ARP entries that match a MAC address.
// The mac parameter can be in any format (it will be normalized).
func FindByMAC(mac string) ([]Entry, error) {
	entries, err := Parse()
	if err != nil {
		return nil, err
	}

	// Normalize the search MAC
	mac = strings.ToLower(strings.ReplaceAll(mac, ":", ""))
	mac = strings.ToLower(strings.ReplaceAll(mac, "-", ""))
	mac = strings.ToLower(strings.ReplaceAll(mac, ".", ""))

	var results []Entry
	for _, entry := range entries {
		// Normalize entry MAC
		entryMAC := strings.ToLower(strings.ReplaceAll(entry.MAC, ":", ""))

		if entryMAC == mac {
			results = append(results, entry)
		}
	}

	return results, nil
}

// GetDevices returns a list of unique device names from the ARP table.
func GetDevices() ([]string, error) {
	entries, err := Parse()
	if err != nil {
		return nil, err
	}

	deviceMap := make(map[string]bool)
	for _, entry := range entries {
		if entry.Device != "" {
			deviceMap[entry.Device] = true
		}
	}

	devices := make([]string, 0, len(deviceMap))
	for device := range deviceMap {
		devices = append(devices, device)
	}

	return devices, nil
}
