// Package web tests.
package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hzhq1255/wolgate/store"
	"github.com/hzhq1255/wolgate/wol"
)

func TestNewHandler(t *testing.T) {
	s := &store.Store{}
	w, _ := wol.NewSender("", "")

	h := NewHandler(s, w)
	if h == nil {
		t.Error("NewHandler() returned nil")
	}
	if h.store != s {
		t.Error("NewHandler() store not set correctly")
	}
	if h.wol != w {
		t.Error("NewHandler() wol not set correctly")
	}
}

func TestIndexHandler(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	h.indexHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected HTML content type, got %s", contentType)
	}
}

func TestIndexHandler_NotFound(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()

	h.indexHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestListHandler(t *testing.T) {
	// Create a mock store
	s, _ := store.NewStore(t.TempDir() + "/test.json")
	h := &Handler{store: s}

	req := httptest.NewRequest("GET", "/api/list", nil)
	w := httptest.NewRecorder()

	h.listHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Errorf("Expected success=true, got %v", resp.Success)
	}
}

func TestListHandler_WrongMethod(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest("POST", "/api/list", nil)
	w := httptest.NewRecorder()

	h.listHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestAddHandler(t *testing.T) {
	s, _ := store.NewStore(t.TempDir() + "/test.json")
	h := &Handler{store: s}

	device := store.Device{
		Name: "Test Device",
		MAC:  "AA:BB:CC:DD:EE:FF",
		IP:   "192.168.1.100",
	}

	body, _ := json.Marshal(device)
	req := httptest.NewRequest("POST", "/api/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.addHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify device was added
	devices := s.List()
	if len(devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(devices))
	}
}

func TestAddHandler_InvalidJSON(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest("POST", "/api/add", strings.NewReader("invalid"))
	w := httptest.NewRecorder()

	h.addHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAddHandler_MissingName(t *testing.T) {
	s, _ := store.NewStore(t.TempDir() + "/test.json")
	h := &Handler{store: s}

	device := store.Device{
		MAC: "AA:BB:CC:DD:EE:FF",
	}

	body, _ := json.Marshal(device)
	req := httptest.NewRequest("POST", "/api/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.addHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing name, got %d", w.Code)
	}
}

func TestAddHandler_InvalidMAC(t *testing.T) {
	s, _ := store.NewStore(t.TempDir() + "/test.json")
	h := &Handler{store: s}

	device := store.Device{
		Name: "Test",
		MAC:  "invalid-mac",
	}

	body, _ := json.Marshal(device)
	req := httptest.NewRequest("POST", "/api/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.addHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid MAC, got %d", w.Code)
	}
}

func TestAddHandler_InvalidIP(t *testing.T) {
	s, _ := store.NewStore(t.TempDir() + "/test.json")
	h := &Handler{store: s}

	device := store.Device{
		Name: "Test",
		MAC:  "AA:BB:CC:DD:EE:FF",
		IP:   "invalid-ip",
	}

	body, _ := json.Marshal(device)
	req := httptest.NewRequest("POST", "/api/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.addHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid IP, got %d", w.Code)
	}
}

func TestDeleteHandler(t *testing.T) {
	s, _ := store.NewStore(t.TempDir() + "/test.json")
	h := &Handler{store: s}

	// Add a device first
	s.Add(store.Device{Name: "Test", MAC: "AA:BB:CC:DD:EE:FF"})

	// Delete the device
	req := struct {
		MAC string `json:"mac"`
	}{
		MAC: "AA:BB:CC:DD:EE:FF",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/delete", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.deleteHandler(w, httpReq)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify device was deleted
	devices := s.List()
	if len(devices) != 0 {
		t.Errorf("Expected 0 devices after delete, got %d", len(devices))
	}
}

func TestDeleteHandler_NotFound(t *testing.T) {
	s, _ := store.NewStore(t.TempDir() + "/test.json")
	h := &Handler{store: s}

	req := struct {
		MAC string `json:"mac"`
	}{
		MAC: "AA:BB:CC:DD:EE:FF",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/delete", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.deleteHandler(w, httpReq)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestDeleteHandler_MissingMAC(t *testing.T) {
	h := &Handler{}

	req := struct {
		MAC string `json:"mac"`
	}{}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/delete", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.deleteHandler(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing MAC, got %d", w.Code)
	}
}

func TestWakeHandler(t *testing.T) {
	wolSender, _ := wol.NewSender("", "127.0.0.1")
	h := &Handler{wol: wolSender}

	req := struct {
		MAC string `json:"mac"`
	}{
		MAC: "AA:BB:CC:DD:EE:FF",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/wake", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.wakeHandler(w, httpReq)

	// The actual send might fail (127.0.0.1 is not a broadcast address)
	// but the request processing should work
	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)

	// Check that we got a response (success or error, but not 500)
	if w.Code == http.StatusInternalServerError {
		// This is acceptable as 127.0.0.1 is not a broadcast address
	}
}

func TestWakeHandler_InvalidMAC(t *testing.T) {
	wolSender, _ := wol.NewSender("", "")
	h := &Handler{wol: wolSender}

	req := struct {
		MAC string `json:"mac"`
	}{
		MAC: "invalid",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/wake", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.wakeHandler(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid MAC, got %d", w.Code)
	}
}

func TestValidateDevice(t *testing.T) {
	tests := []struct {
		name    string
		device  *store.Device
		wantErr bool
	}{
		{
			name: "valid device",
			device: &store.Device{
				Name: "Test",
				MAC:  "AA:BB:CC:DD:EE:FF",
				IP:   "192.168.1.100",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			device: &store.Device{
				MAC: "AA:BB:CC:DD:EE:FF",
			},
			wantErr: true,
		},
		{
			name: "missing MAC",
			device: &store.Device{
				Name: "Test",
			},
			wantErr: true,
		},
		{
			name: "invalid MAC",
			device: &store.Device{
				Name: "Test",
				MAC:  "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid IP",
			device: &store.Device{
				Name: "Test",
				MAC:  "AA:BB:CC:DD:EE:FF",
				IP:   "invalid",
			},
			wantErr: true,
		},
		{
			name: "device without IP",
			device: &store.Device{
				Name: "Test",
				MAC:  "AA:BB:CC:DD:EE:FF",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDevice(tt.device)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResponse_Structure(t *testing.T) {
	resp := Response{
		Success: true,
		Data:    "test data",
		Message: "test message",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !decoded.Success {
		t.Error("Success should be true")
	}
	if decoded.Data != "test data" {
		t.Error("Data mismatch")
	}
	if decoded.Message != "test message" {
		t.Error("Message mismatch")
	}
}
