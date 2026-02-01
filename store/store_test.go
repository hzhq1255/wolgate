// Package store tests.
package store

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestNewStore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.json")

	store, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if store == nil {
		t.Fatal("NewStore() returned nil")
	}
	if store.Count() != 0 {
		t.Errorf("Expected empty store, got %d devices", store.Count())
	}
}

func TestNewStore_WithExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.json")

	// Create a store and add a device
	store1, _ := NewStore(storePath)
	store1.Add(Device{Name: "Test", MAC: "AA:BB:CC:DD:EE:FF"})

	// Create a new store instance
	store2, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if store2.Count() != 1 {
		t.Errorf("Expected 1 device from existing file, got %d", store2.Count())
	}
}

func TestStore_Add(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.json")

	store, _ := NewStore(storePath)

	device := Device{
		Name: "Test Device",
		MAC:  "AA:BB:CC:DD:EE:FF",
		IP:   "192.168.1.100",
		Group: "Office",
	}

	err := store.Add(device)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Verify device was added
	if store.Count() != 1 {
		t.Errorf("Expected 1 device, got %d", store.Count())
	}

	// Verify device was persisted
	store2, _ := NewStore(storePath)
	if store2.Count() != 1 {
		t.Error("Device was not persisted to file")
	}
}

func TestStore_Add_DuplicateMAC(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.json")

	store, _ := NewStore(storePath)

	device1 := Device{Name: "Device 1", MAC: "AA:BB:CC:DD:EE:FF"}
	device2 := Device{Name: "Device 2", MAC: "AA:BB:CC:DD:EE:FF"}

	store.Add(device1)
	err := store.Add(device2)

	if err == nil {
		t.Error("Add() should return error for duplicate MAC")
	}
}

func TestStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.json")

	store, _ := NewStore(storePath)

	device := Device{Name: "Test", MAC: "AA:BB:CC:DD:EE:FF"}
	store.Add(device)

	// Delete the device
	err := store.Delete("AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if store.Count() != 0 {
		t.Errorf("Expected 0 devices after delete, got %d", store.Count())
	}
}

func TestStore_Delete_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.json")

	store, _ := NewStore(storePath)

	err := store.Delete("AA:BB:CC:DD:EE:FF")
	if err == nil {
		t.Error("Delete() should return error for non-existent device")
	}
}

func TestStore_List(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.json")

	store, _ := NewStore(storePath)

	store.Add(Device{Name: "Device 1", MAC: "AA:BB:CC:DD:EE:01"})
	store.Add(Device{Name: "Device 2", MAC: "AA:BB:CC:DD:EE:02"})

	devices := store.List()
	if len(devices) != 2 {
		t.Errorf("Expected 2 devices, got %d", len(devices))
	}
}

func TestStore_GetByMAC(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.json")

	store, _ := NewStore(storePath)

	device := Device{Name: "Test", MAC: "AA:BB:CC:DD:EE:FF", IP: "192.168.1.100"}
	store.Add(device)

	found, err := store.GetByMAC("AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatalf("GetByMAC() error = %v", err)
	}
	if found.Name != "Test" {
		t.Errorf("Expected name 'Test', got %s", found.Name)
	}
	if found.IP != "192.168.1.100" {
		t.Errorf("Expected IP '192.168.1.100', got %s", found.IP)
	}
}

func TestStore_GetByMAC_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.json")

	store, _ := NewStore(storePath)

	_, err := store.GetByMAC("AA:BB:CC:DD:EE:FF")
	if err == nil {
		t.Error("GetByMAC() should return error for non-existent device")
	}
}

func TestStore_GetByGroup(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.json")

	store, _ := NewStore(storePath)

	store.Add(Device{Name: "D1", MAC: "AA:BB:CC:DD:EE:01", Group: "Office"})
	store.Add(Device{Name: "D2", MAC: "AA:BB:CC:DD:EE:02", Group: "Office"})
	store.Add(Device{Name: "D3", MAC: "AA:BB:CC:DD:EE:03", Group: "Home"})

	devices := store.GetByGroup("Office")
	if len(devices) != 2 {
		t.Errorf("Expected 2 devices in Office group, got %d", len(devices))
	}

	devices = store.GetByGroup("Home")
	if len(devices) != 1 {
		t.Errorf("Expected 1 device in Home group, got %d", len(devices))
	}
}

func TestStore_Groups(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.json")

	store, _ := NewStore(storePath)

	store.Add(Device{Name: "D1", MAC: "AA:BB:CC:DD:EE:01", Group: "Office"})
	store.Add(Device{Name: "D2", MAC: "AA:BB:CC:DD:EE:02", Group: "Office"})
	store.Add(Device{Name: "D3", MAC: "AA:BB:CC:DD:EE:03", Group: "Home"})
	store.Add(Device{Name: "D4", MAC: "AA:BB:CC:DD:EE:04", Group: ""}) // No group

	groups := store.Groups()
	if len(groups) != 2 {
		t.Errorf("Expected 2 unique groups, got %d", len(groups))
	}
}

func TestStore_Update(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.json")

	store, _ := NewStore(storePath)

	device := Device{Name: "Old Name", MAC: "AA:BB:CC:DD:EE:FF", Group: "Old Group"}
	store.Add(device)

	// Update the device
	updated := Device{Name: "New Name", MAC: "AA:BB:CC:DD:EE:FF", Group: "New Group"}
	err := store.Update("AA:BB:CC:DD:EE:FF", updated)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	found, _ := store.GetByMAC("AA:BB:CC:DD:EE:FF")
	if found.Name != "New Name" {
		t.Errorf("Expected updated name 'New Name', got %s", found.Name)
	}
	if found.Group != "New Group" {
		t.Errorf("Expected updated group 'New Group', got %s", found.Group)
	}
}

func TestStore_Update_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.json")

	store, _ := NewStore(storePath)

	updated := Device{Name: "New Name", MAC: "AA:BB:CC:DD:EE:FF"}
	err := store.Update("AA:BB:CC:DD:EE:FF", updated)
	if err == nil {
		t.Error("Update() should return error for non-existent device")
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.json")

	store, _ := NewStore(storePath)

	// Launch multiple goroutines
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			mac := "AA:BB:CC:DD:EE:" + string(rune('0'+n))
			store.Add(Device{Name: "Device", MAC: mac})
			store.List()
			store.GetByGroup("test")
		}(i)
	}
	wg.Wait()

	// Should not panic or deadlock
	if store.Count() > 0 {
		t.Logf("Successfully handled concurrent access, %d devices added", store.Count())
	}
}

func TestStore_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.json")

	// Create empty file
	os.WriteFile(storePath, []byte{}, 0644)

	store, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore() with empty file error = %v", err)
	}
	if store.Count() != 0 {
		t.Errorf("Expected empty store from empty file, got %d devices", store.Count())
	}
}

func TestStore_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.json")

	// Create file with invalid JSON
	os.WriteFile(storePath, []byte("{ invalid json"), 0644)

	_, err := NewStore(storePath)
	if err == nil {
		t.Error("NewStore() should return error for invalid JSON")
	}
}
