// Package store handles device data persistence.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Device represents a wake-on-LAN device.
type Device struct {
	Name  string `json:"name"`
	MAC   string `json:"mac"`
	IP    string `json:"ip,omitempty"`
	Group string `json:"group,omitempty"`
}

// Store manages device persistence.
type Store struct {
	filePath string
	devices  []*Device
	mu       sync.RWMutex
}

// NewStore creates a new store instance.
func NewStore(filePath string) (*Store, error) {
	s := &Store{
		filePath: filePath,
		devices:  make([]*Device, 0),
	}

	// Load existing data if file exists
	if err := s.Load(); err != nil {
		// If file doesn't exist, that's ok - start with empty store
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load store: %w", err)
		}
	}

	return s, nil
}

// Load loads devices from the JSON file.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		s.devices = make([]*Device, 0)
		return nil
	}

	// Read file
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read store file: %w", err)
	}

	// Handle empty file
	if len(data) == 0 {
		s.devices = make([]*Device, 0)
		return nil
	}

	// Parse JSON
	var devices []*Device
	if err := json.Unmarshal(data, &devices); err != nil {
		return fmt.Errorf("failed to parse store file: %w", err)
	}

	s.devices = devices
	return nil
}

// Save saves devices to the JSON file.
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create directory if it doesn't exist
	dir := dirPath(s.filePath)
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(s.devices, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal devices: %w", err)
	}

	// Write to file
	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write store file: %w", err)
	}

	return nil
}

// dirPath returns the directory path from a file path.
func dirPath(path string) string {
	idx := len(path) - 1
	for idx > 0 && path[idx] != '/' {
		idx--
	}
	return path[:idx]
}

// List returns all devices.
func (s *Store) List() []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Device, len(s.devices))
	for i, d := range s.devices {
		result[i] = *d
	}
	return result
}

// Add adds a new device to the store.
func (s *Store) Add(device Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate MAC
	for _, d := range s.devices {
		if d.MAC == device.MAC {
			return fmt.Errorf("device with MAC %s already exists", device.MAC)
		}
	}

	// Add device
	s.devices = append(s.devices, &device)

	// Save to file
	if err := s.saveLocked(); err != nil {
		// Rollback on save error
		s.devices = s.devices[:len(s.devices)-1]
		return err
	}

	return nil
}

// Delete removes a device by MAC address.
func (s *Store) Delete(mac string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find and remove device
	found := false
	newDevices := make([]*Device, 0, len(s.devices))
	for _, d := range s.devices {
		if d.MAC != mac {
			newDevices = append(newDevices, d)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("device with MAC %s not found", mac)
	}

	oldDevices := s.devices
	s.devices = newDevices

	// Save to file
	if err := s.saveLocked(); err != nil {
		// Rollback on save error
		s.devices = oldDevices
		return err
	}

	return nil
}

// GetByMAC finds a device by MAC address.
func (s *Store) GetByMAC(mac string) (*Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, d := range s.devices {
		if d.MAC == mac {
			// Return a copy
			copy := *d
			return &copy, nil
		}
	}

	return nil, fmt.Errorf("device with MAC %s not found", mac)
}

// GetByGroup returns all devices in a group.
func (s *Store) GetByGroup(group string) []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Device, 0)
	for _, d := range s.devices {
		if d.Group == group {
			result = append(result, *d)
		}
	}
	return result
}

// Groups returns all unique group names.
func (s *Store) Groups() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	groupMap := make(map[string]bool)
	for _, d := range s.devices {
		if d.Group != "" {
			groupMap[d.Group] = true
		}
	}

	groups := make([]string, 0, len(groupMap))
	for g := range groupMap {
		groups = append(groups, g)
	}
	return groups
}

// Update updates an existing device.
func (s *Store) Update(mac string, updated Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find and update device
	found := false
	for i, d := range s.devices {
		if d.MAC == mac {
			// Keep the original MAC
			updated.MAC = mac
			s.devices[i] = &updated
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("device with MAC %s not found", mac)
	}

	// Save to file
	if err := s.saveLocked(); err != nil {
		return err
	}

	return nil
}

// Count returns the number of devices.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.devices)
}

// saveLocked saves devices to file (must be called with lock held).
func (s *Store) saveLocked() error {
	// Create directory if it doesn't exist
	dir := dirPath(s.filePath)
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(s.devices, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal devices: %w", err)
	}

	// Write to file
	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write store file: %w", err)
	}

	return nil
}
