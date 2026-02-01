// Package web provides API implementations for wolgate.
package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hzhq1255/wolgate/arp"
	"github.com/hzhq1255/wolgate/store"
)

// ARPEntry represents an ARP entry for import.
type ARPEntry struct {
	IP     string `json:"ip"`
	MAC    string `json:"mac"`
	Device string `json:"device"`
}

// importHandler returns a list of ARP entries for device discovery.
func (h *Handler) importHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		h.handleImportGet(w)
	} else if r.Method == http.MethodPost {
		h.handleImportPost(w, r)
	} else {
		h.respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleImportGet returns the list of devices from ARP table.
func (h *Handler) handleImportGet(w http.ResponseWriter) {
	// Get local ARP entries
	entries, err := arp.GetLocalEntries()
	if err != nil {
		h.respondError(w, "Failed to read ARP table: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to API format
	result := make([]ARPEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, ARPEntry{
			IP:     entry.IP,
			MAC:    entry.MAC,
			Device: entry.Device,
		})
	}

	h.respondSuccess(w, result)
}

// handleImportPost imports selected ARP entries into the device store.
func (h *Handler) handleImportPost(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Devices []struct {
			Name string `json:"name"`
			MAC  string `json:"mac"`
			IP   string `json:"ip,omitempty"`
		} `json:"devices"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	successCount := 0
	skippedCount := 0
	for _, device := range req.Devices {
		// Validate MAC
		if device.MAC == "" {
			continue
		}

		// Skip if already exists
		if _, err := h.store.GetByMAC(device.MAC); err == nil {
			skippedCount++
			continue
		}

		// Use provided name or generate one
		name := device.Name
		if name == "" {
			name = "Device-" + device.MAC[:8]
		}

		newDevice := store.Device{
			Name: name,
			MAC:  device.MAC,
			IP:   device.IP,
		}

		// Add device
		if err := h.store.Add(newDevice); err == nil {
			successCount++
		}
	}

	message := fmt.Sprintf("Imported %d devices", successCount)
	if skippedCount > 0 {
		message += fmt.Sprintf(", skipped %d (already exist)", skippedCount)
	}

	h.respond(w, Response{
		Success: successCount > 0,
		Message: message,
	})
}
