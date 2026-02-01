// Package config tests.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.Listen != "127.0.0.1:9000" {
		t.Errorf("Expected default listen 127.0.0.1:9000, got %s", cfg.Server.Listen)
	}
	if cfg.Server.Data != "/data/wolgate.json" {
		t.Errorf("Expected default data /data/wolgate.json, got %s", cfg.Server.Data)
	}
	if cfg.Wake.Broadcast != "255.255.255.255" {
		t.Errorf("Expected default broadcast 255.255.255.255, got %s", cfg.Wake.Broadcast)
	}
	if cfg.Log.File != "/tmp/wolgate.log" {
		t.Errorf("Expected default log file /tmp/wolgate.log, got %s", cfg.Log.File)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Expected default log level info, got %s", cfg.Log.Level)
	}
	if cfg.Log.MaxSize != 10 {
		t.Errorf("Expected default max size 10, got %d", cfg.Log.MaxSize)
	}
	if cfg.Devices == nil {
		t.Error("Expected devices to be initialized, got nil")
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	cfg, err := Load("/non/existent/path.json")
	if err != nil {
		t.Fatalf("Load() should return default for non-existent file, got error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
	// Should have default values
	if cfg.Server.Listen != "127.0.0.1:9000" {
		t.Error("Non-existent file should return default config")
	}
}

func TestLoad_EmptyPath(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() should return default for empty path, got error: %v", err)
	}
	if cfg.Server.Listen != "127.0.0.1:9000" {
		t.Error("Empty path should return default config")
	}
}

func TestLoad_ValidFile(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.json")

	// Write test config
	testCfg := Config{
		Server: ServerConfig{
			Listen: "0.0.0.0:8080",
			Data:   "/tmp/test.json",
		},
		Wake: WakeConfig{
			Broadcast: "192.168.1.255",
		},
		Log: LogConfig{
			Level: "debug",
		},
	}

	data, _ := json.Marshal(testCfg)
	os.WriteFile(cfgPath, data, 0644)

	// Load config
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Listen != "0.0.0.0:8080" {
		t.Errorf("Expected listen 0.0.0.0:8080, got %s", cfg.Server.Listen)
	}
	if cfg.Wake.Broadcast != "192.168.1.255" {
		t.Errorf("Expected broadcast 192.168.1.255, got %s", cfg.Wake.Broadcast)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Expected level debug, got %s", cfg.Log.Level)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	// Create temp file with invalid JSON
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "invalid.json")
	os.WriteFile(cfgPath, []byte("{ invalid json"), 0644)

	_, err := Load(cfgPath)
	if err == nil {
		t.Error("Load() should return error for invalid JSON")
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}

	applyDefaults(cfg)

	if cfg.Server.Listen != "127.0.0.1:9000" {
		t.Error("applyDefaults should set default listen")
	}
	if cfg.Server.Data != "/data/wolgate.json" {
		t.Error("applyDefaults should set default data")
	}
	if cfg.Wake.Broadcast != "255.255.255.255" {
		t.Error("applyDefaults should set default broadcast")
	}
	if cfg.Log.File != "/tmp/wolgate.log" {
		t.Error("applyDefaults should set default log file")
	}
	if cfg.Log.Level != "info" {
		t.Error("applyDefaults should set default log level")
	}
}

func TestMergeFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("WOLGATE_SERVER__LISTEN", "0.0.0.0:9000")
	os.Setenv("WOLGATE_LOG__LEVEL", "debug")
	os.Setenv("WOLGATE_WAKE__BROADCAST", "192.168.1.255")
	defer func() {
		os.Unsetenv("WOLGATE_SERVER__LISTEN")
		os.Unsetenv("WOLGATE_LOG__LEVEL")
		os.Unsetenv("WOLGATE_WAKE__BROADCAST")
	}()

	cfg := DefaultConfig()
	cfg.MergeFromEnv()

	if cfg.Server.Listen != "0.0.0.0:9000" {
		t.Errorf("Expected listen from env, got %s", cfg.Server.Listen)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Expected log level from env, got %s", cfg.Log.Level)
	}
	if cfg.Wake.Broadcast != "192.168.1.255" {
		t.Errorf("Expected broadcast from env, got %s", cfg.Wake.Broadcast)
	}
}

func TestMergeFromCLI(t *testing.T) {
	cfg := DefaultConfig()

	params := map[string]string{
		"server.listen":   "0.0.0.0:8080",
		"log.level":       "debug",
		"wake.broadcast":  "192.168.1.255",
		"log.max_size":    "20",
		"log.max_backups": "5",
		"log.max_age":     "14",
	}

	cfg.MergeFromCLI(params)

	if cfg.Server.Listen != "0.0.0.0:8080" {
		t.Errorf("Expected listen from CLI, got %s", cfg.Server.Listen)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Expected log level from CLI, got %s", cfg.Log.Level)
	}
	if cfg.Wake.Broadcast != "192.168.1.255" {
		t.Errorf("Expected broadcast from CLI, got %s", cfg.Wake.Broadcast)
	}
	if cfg.Log.MaxSize != 20 {
		t.Errorf("Expected max size 20 from CLI, got %d", cfg.Log.MaxSize)
	}
	if cfg.Log.MaxBackups != 5 {
		t.Errorf("Expected max backups 5 from CLI, got %d", cfg.Log.MaxBackups)
	}
	if cfg.Log.MaxAge != 14 {
		t.Errorf("Expected max age 14 from CLI, got %d", cfg.Log.MaxAge)
	}
}

func TestMergeFromCLI_InvalidValues(t *testing.T) {
	cfg := DefaultConfig()

	params := map[string]string{
		"log.max_size":    "invalid",
		"log.max_backups": "-5",
		"log.max_age":     "invalid",
	}

	// Should not panic, just ignore invalid values
	cfg.MergeFromCLI(params)

	// Values should remain at defaults
	if cfg.Log.MaxSize != 10 {
		t.Errorf("Expected default max size for invalid input, got %d", cfg.Log.MaxSize)
	}
	if cfg.Log.MaxBackups != 3 {
		t.Errorf("Expected default max backups for negative input, got %d", cfg.Log.MaxBackups)
	}
	if cfg.Log.MaxAge != 7 {
		t.Errorf("Expected default max age for invalid input, got %d", cfg.Log.MaxAge)
	}
}

func TestConfigPriority(t *testing.T) {
	// Create config file
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.json")

	testCfg := Config{
		Server: ServerConfig{
			Listen: "127.0.0.1:9000", // From file
		},
		Log: LogConfig{
			Level: "info", // From file
		},
	}

	data, _ := json.Marshal(testCfg)
	os.WriteFile(cfgPath, data, 0644)

	// Load from file
	cfg, _ := Load(cfgPath)

	// Merge from env
	os.Setenv("WOLGATE_LOG__LEVEL", "warn")
	defer os.Unsetenv("WOLGATE_LOG__LEVEL")
	cfg.MergeFromEnv()

	// Merge from CLI (highest priority)
	params := map[string]string{
		"log.level": "error",
	}
	cfg.MergeFromCLI(params)

	// CLI should win
	if cfg.Log.Level != "error" {
		t.Errorf("Expected CLI level to override, got %s", cfg.Log.Level)
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "save-test.json")

	cfg := DefaultConfig()
	cfg.Server.Listen = "0.0.0.0:8080"

	err := cfg.Save(cfgPath)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Error("Save() did not create file")
	}

	// Load and verify
	loaded, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loaded.Server.Listen != "0.0.0.0:8080" {
		t.Errorf("Saved config not persisted correctly, got %s", loaded.Server.Listen)
	}
}
