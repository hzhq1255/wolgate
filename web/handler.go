// Package web provides HTTP handlers for wolgate.
package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/hzhq1255/wolgate/store"
	"github.com/hzhq1255/wolgate/wol"
)

// Handler handles HTTP requests.
type Handler struct {
	store *store.Store
	wol   *wol.WOLSender
}

// NewHandler creates a new HTTP handler.
func NewHandler(store *store.Store, wol *wol.WOLSender) *Handler {
	return &Handler{
		store: store,
		wol:   wol,
	}
}

// Response represents a standard API response.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// RegisterRoutes registers all HTTP routes.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Serve index page
	mux.HandleFunc("/", h.indexHandler)

	// API routes
	mux.HandleFunc("/api/list", h.listHandler)
	mux.HandleFunc("/api/add", h.addHandler)
	mux.HandleFunc("/api/delete", h.deleteHandler)
	mux.HandleFunc("/api/wake", h.wakeHandler)
	mux.HandleFunc("/api/import", h.importHandler)
}

// indexHandler serves the main HTML page.
func (h *Handler) indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	content, err := IndexHTML()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

// listHandler returns the list of all devices.
func (h *Handler) listHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	devices := h.store.List()
	h.respondSuccess(w, devices)
}

// addHandler adds a new device.
func (h *Handler) addHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var device store.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		h.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate device
	if err := validateDevice(&device); err != nil {
		h.respondError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if device exists (for updates)
	existing, _ := h.store.GetByMAC(device.MAC)
	if existing != nil {
		// Update existing device
		if err := h.store.Update(device.MAC, device); err != nil {
			h.respondError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Add new device
		if err := h.store.Add(device); err != nil {
			h.respondError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	h.respondSuccess(w, nil)
}

// deleteHandler deletes a device.
func (h *Handler) deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MAC string `json:"mac"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.MAC == "" {
		h.respondError(w, "MAC address is required", http.StatusBadRequest)
		return
	}

	if err := h.store.Delete(req.MAC); err != nil {
		h.respondError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.respondSuccess(w, nil)
}

// wakeHandler sends a WOL magic packet.
func (h *Handler) wakeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MAC string `json:"mac"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate MAC
	if err := wol.ValidateMAC(req.MAC); err != nil {
		h.respondError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Send magic packet (3 times for reliability)
	if err := h.wol.SendRepeat(req.MAC, 3); err != nil {
		h.respondError(w, fmt.Sprintf("Failed to send WOL packet: %v", err), http.StatusInternalServerError)
		return
	}

	h.respond(w, Response{
		Success: true,
		Message: fmt.Sprintf("WOL packet sent to %s", req.MAC),
	})
}

// validateDevice validates a device before adding/updating.
func validateDevice(device *store.Device) error {
	if device.Name == "" {
		return fmt.Errorf("device name is required")
	}

	if device.MAC == "" {
		return fmt.Errorf("MAC address is required")
	}

	if err := wol.ValidateMAC(device.MAC); err != nil {
		return fmt.Errorf("invalid MAC address: %w", err)
	}

	// Validate IP if provided
	if device.IP != "" {
		ipRegex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
		if !ipRegex.MatchString(device.IP) {
			return fmt.Errorf("invalid IP address format")
		}
	}

	return nil
}

// respondSuccess sends a success response.
func (h *Handler) respondSuccess(w http.ResponseWriter, data interface{}) {
	h.respond(w, Response{
		Success: true,
		Data:    data,
	})
}

// respondError sends an error response.
func (h *Handler) respondError(w http.ResponseWriter, message string, status int) {
	h.respondWithStatus(w, Response{
		Success: false,
		Error:   message,
	}, status)
}

// respond sends a response with default status code.
func (h *Handler) respond(w http.ResponseWriter, data Response) {
	w.Header().Set("Content-Type", "application/json")

	// Determine status code based on success
	status := http.StatusOK
	if !data.Success {
		status = http.StatusBadRequest
	}

	// Write status and JSON
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondWithStatus sends a response with a specific status code.
func (h *Handler) respondWithStatus(w http.ResponseWriter, data Response, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
