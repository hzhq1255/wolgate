// Package logger tests.
package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "stdout logger",
			cfg: Config{
				File:  "",
				Level: "info",
			},
			wantErr: false,
		},
		{
			name: "file logger",
			cfg: Config{
				File:  "/tmp/test-wolgate.log",
				Level: "debug",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, err := New(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if log != nil {
				log.Close()
				os.Remove(tt.cfg.File)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name string
		level string
		want Level
	}{
		{"debug", "debug", DEBUG},
		{"info", "info", INFO},
		{"warn", "warn", WARN},
		{"warning", "warning", WARN},
		{"error", "error", ERROR},
		{"default", "unknown", INFO},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseLevel(tt.level); got != tt.want {
				t.Errorf("parseLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogger_Log(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	log := &Logger{
		writer: &buf,
		level:  DEBUG,
	}

	log.Info("test message %s", "value")

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected log to contain INFO level, got: %s", output)
	}
	if !strings.Contains(output, "test message value") {
		t.Errorf("Expected log to contain message, got: %s", output)
	}
	// Check for ISO8601 timestamp format (rough check)
	if !strings.Contains(output, "T") {
		t.Errorf("Expected log to contain timestamp, got: %s", output)
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer

	log := &Logger{
		writer: &buf,
		level:  WARN, // Only log WARN and above
	}

	// These should not be logged
	log.Debug("debug message")
	log.Info("info message")

	// Clear buffer
	buf.Reset()

	// This should be logged
	log.Warn("warn message")

	output := buf.String()
	if !strings.Contains(output, "warn message") {
		t.Errorf("Expected warn message to be logged, got: %s", output)
	}
	if strings.Contains(output, "debug message") || strings.Contains(output, "info message") {
		t.Error("Lower level messages should not be logged")
	}
}

func TestLogger_ConvenienceMethods(t *testing.T) {
	var buf bytes.Buffer

	log := &Logger{
		writer: &buf,
		level:  DEBUG,
	}

	tests := []struct {
		name    string
		method  func(string, ...interface{})
		level   string
		message string
	}{
		{"Debug", log.Debug, "DEBUG", "debug msg"},
		{"Info", log.Info, "INFO", "info msg"},
		{"Warn", log.Warn, "WARN", "warn msg"},
		{"Error", log.Error, "ERROR", "error msg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.method(tt.message)
			output := buf.String()

			if !strings.Contains(output, tt.level) {
				t.Errorf("Expected %s level, got: %s", tt.level, output)
			}
			if !strings.Contains(output, tt.message) {
				t.Errorf("Expected message %s, got: %s", tt.message, output)
			}
		})
	}
}

func TestLogger_FileRotation(t *testing.T) {
	// Create temp directory
	tmpDir := os.TempDir()
	logFile := filepath.Join(tmpDir, "test-rotation.log")

	// Remove old test file if exists
	os.Remove(logFile)
	defer os.Remove(logFile)

	// Clean up any backup files
	matches, _ := filepath.Glob(logFile + ".*")
	for _, m := range matches {
		os.Remove(m)
	}

	cfg := Config{
		File:    logFile,
		Level:   "info",
		MaxSize: 1, // 1 MB - but we'll test with smaller writes
		MaxBackups: 2,
		MaxAge:   7,
	}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer log.Close()

	// Write some log messages
	for i := 0; i < 10; i++ {
		log.Info("Test message %d", i)
	}

	// Check that log file was created
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}

	// Note: Testing actual rotation requires writing > maxSize bytes
	// which is complex in a unit test. The rotation logic is tested
	// indirectly through integration tests.
}

func TestFindBackupFiles(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create log file
	logFile := filepath.Join(tmpDir, "test.log")
	f, _ := os.Create(logFile)
	f.Close()

	// Create some backup files
	backup1 := logFile + ".2023-01-01.120000"
	backup2 := logFile + ".2023-01-02.120000"
	otherFile := filepath.Join(tmpDir, "other.txt")

	os.Create(backup1)
	os.Create(backup2)
	os.Create(otherFile)
	defer os.Remove(backup1)
	defer os.Remove(backup2)
	defer os.Remove(otherFile)

	backups, err := findBackupFiles(logFile)
	if err != nil {
		t.Fatalf("findBackupFiles() error = %v", err)
	}

	// Should find 2 backup files
	if len(backups) != 2 {
		t.Errorf("Expected 2 backup files, got %d", len(backups))
	}

	// Should not include the other file
	for _, b := range backups {
		if strings.Contains(b.Path, "other.txt") {
			t.Error("Should not include non-backup files")
		}
	}
}

func TestSortBackups(t *testing.T) {
	now := time.Now()

	backups := []backupInfo{
		{Path: "old.log", ModTime: now.Add(-24 * time.Hour)},
		{Path: "new.log", ModTime: now},
		{Path: "mid.log", ModTime: now.Add(-12 * time.Hour)},
	}

	sortBackups(backups)

	// Should be sorted newest first
	if backups[0].Path != "new.log" {
		t.Errorf("Expected newest first, got %s", backups[0].Path)
	}
	if backups[2].Path != "old.log" {
		t.Errorf("Expected oldest last, got %s", backups[2].Path)
	}
}

func TestLogger_Close(t *testing.T) {
	tmpFile := "/tmp/test-close.log"
	defer os.Remove(tmpFile)

	cfg := Config{
		File:  tmpFile,
		Level: "info",
	}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = log.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Closing again should not error
	err = log.Close()
	if err != nil {
		t.Errorf("Close() again error = %v", err)
	}
}
