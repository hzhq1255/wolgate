// Package logger provides a logging system with file rotation support.
package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level represents the log level.
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

// String returns the string representation of the log level.
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Config holds the logger configuration.
type Config struct {
	File        string // Log file path
	Level       string // Log level (debug/info/warn/error)
	MaxSize     int    // Maximum file size in MB before rotation
	MaxBackups  int    // Maximum number of backup files to keep
	MaxAge      int    // Maximum number of days to retain old log files
}

// Logger provides leveled logging with file rotation.
type Logger struct {
	file       *os.File
	filePath   string
	maxSize    int64  // in bytes
	maxBackups int
	maxAge     int    // in days
	level      Level
	mu         sync.Mutex
	writer     io.Writer
	closed     bool
}

// New creates a new logger instance.
// If the file path is empty, logs will be written to stdout.
func New(cfg Config) (*Logger, error) {
	l := &Logger{
		filePath:   cfg.File,
		maxSize:    int64(cfg.MaxSize) * 1024 * 1024, // Convert MB to bytes
		maxBackups: cfg.MaxBackups,
		maxAge:     cfg.MaxAge,
		level:      parseLevel(cfg.Level),
	}

	if cfg.File == "" {
		l.writer = os.Stdout
		return l, nil
	}

	// Open or create log file
	file, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	l.file = file
	l.writer = file

	return l, nil
}

// parseLevel parses the log level string.
func parseLevel(level string) Level {
	switch level {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}

// shouldLog returns true if the given level should be logged.
func (l *Logger) shouldLog(level Level) bool {
	return level >= l.level
}

// Log writes a log message at the specified level.
func (l *Logger) Log(level Level, format string, args ...interface{}) {
	if !l.shouldLog(level) {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Check file size before writing
	if l.file != nil {
		if err := l.checkRotation(); err != nil {
			// If rotation fails, still try to log the error
			fmt.Fprintf(os.Stderr, "log rotation error: %v\n", err)
		}
	}

	// Format: 2006-01-02T15:04:05.000Z07:00 LEVEL message
	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("%s %s %s\n", timestamp, level, message)

	if _, err := io.WriteString(l.writer, logLine); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write log: %v\n", err)
	}
}

// Debug logs a debug message.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.Log(DEBUG, format, args...)
}

// Info logs an info message.
func (l *Logger) Info(format string, args ...interface{}) {
	l.Log(INFO, format, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...interface{}) {
	l.Log(WARN, format, args...)
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...interface{}) {
	l.Log(ERROR, format, args...)
}

// checkRotation checks if the log file needs rotation.
func (l *Logger) checkRotation() error {
	if l.file == nil || l.maxSize <= 0 {
		return nil
	}

	info, err := l.file.Stat()
	if err != nil {
		return err
	}

	// Check if file size exceeds max size
	if info.Size() >= l.maxSize {
		return l.rotate()
	}

	return nil
}

// rotate performs log file rotation.
func (l *Logger) rotate() error {
	if l.file == nil {
		return nil
	}

	// Close current file
	if err := l.file.Close(); err != nil {
		return err
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("2006-01-02.150405")
	backupPath := l.filePath + "." + timestamp

	// Rename current file to backup
	if err := os.Rename(l.filePath, backupPath); err != nil {
		return err
	}

	// Clean up old backups
	go l.cleanupOldBackups()

	// Open new log file
	file, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	l.file = file
	l.writer = file

	return nil
}

// cleanupOldBackups removes old backup files based on maxBackups and maxAge.
func (l *Logger) cleanupOldBackups() {
	if l.maxBackups <= 0 && l.maxAge <= 0 {
		return
	}

	// Find all backup files
	matches, err := findBackupFiles(l.filePath)
	if err != nil {
		return
	}

	// Sort by modification time (newest first)
	sortBackups(matches)

	// Remove excess backups
	if l.maxBackups > 0 && len(matches) > l.maxBackups {
		for _, f := range matches[l.maxBackups:] {
			os.Remove(f.Path)
		}
		matches = matches[:l.maxBackups]
	}

	// Remove old files based on maxAge
	if l.maxAge > 0 {
		cutoff := time.Now().AddDate(0, 0, -l.maxAge)
		for _, f := range matches {
			if f.ModTime.Before(cutoff) {
				os.Remove(f.Path)
			}
		}
	}
}

// Close closes the logger and releases resources.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	if l.file != nil {
		l.closed = true
		return l.file.Close()
	}

	l.closed = true
	return nil
}
