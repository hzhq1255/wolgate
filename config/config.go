// Package config handles configuration loading and merging.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Device represents a wake-on-LAN device.
type Device struct {
	Name  string `json:"name"`
	MAC   string `json:"mac"`
	IP    string `json:"ip,omitempty"`
	Group string `json:"group,omitempty"`
}

// ServerConfig holds server configuration.
type ServerConfig struct {
	Listen string `json:"listen" default:"127.0.0.1:9000"`
	Data   string `json:"data" default:"/data/wolgate.json"`
}

// WakeConfig holds Wake-on-LAN configuration.
type WakeConfig struct {
	Iface     string `json:"iface" default:""`
	Broadcast string `json:"broadcast" default:"255.255.255.255"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
	File       string `json:"file" default:"/tmp/wolgate.log"`
	Level      string `json:"level" default:"info"`
	MaxSize    int    `json:"max_size" default:"10"`
	MaxBackups int    `json:"max_backups" default:"3"`
	MaxAge     int    `json:"max_age" default:"7"`
}

// Config holds the complete configuration.
type Config struct {
	Server  ServerConfig `json:"server"`
	Wake    WakeConfig   `json:"wake"`
	Log     LogConfig    `json:"log"`
	Devices []Device     `json:"devices"`
}

// DefaultConfig returns a configuration with default values.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Listen: "127.0.0.1:9000",
			Data:   "/data/wolgate.json",
		},
		Wake: WakeConfig{
			Iface:     "",
			Broadcast: "255.255.255.255",
		},
		Log: LogConfig{
			File:       "/tmp/wolgate.log",
			Level:      "info",
			MaxSize:    10,
			MaxBackups: 3,
			MaxAge:     7,
		},
		Devices: []Device{},
	}
}

// Load loads configuration from a file.
// If the file doesn't exist, returns default configuration.
// If the file exists but is invalid, returns an error.
func Load(path string) (*Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// If path is empty, return defaults
	if path == "" {
		return cfg, nil
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist, return defaults
		return cfg, nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for any unset fields
	applyDefaults(cfg)

	return cfg, nil
}

// applyDefaults ensures all fields have valid default values.
func applyDefaults(cfg *Config) {
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = "127.0.0.1:9000"
	}
	if cfg.Server.Data == "" {
		cfg.Server.Data = "/data/wolgate.json"
	}

	if cfg.Wake.Broadcast == "" {
		cfg.Wake.Broadcast = "255.255.255.255"
	}

	if cfg.Log.File == "" {
		cfg.Log.File = "/tmp/wolgate.log"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Log.MaxSize == 0 {
		cfg.Log.MaxSize = 10
	}
	if cfg.Log.MaxBackups == 0 {
		cfg.Log.MaxBackups = 3
	}
	if cfg.Log.MaxAge == 0 {
		cfg.Log.MaxAge = 7
	}

	if cfg.Devices == nil {
		cfg.Devices = []Device{}
	}
}

// MergeFromEnv merges configuration from environment variables.
// Environment variables should be prefixed with "WOLGATE_" and use
// double underscore "__" to separate nested fields.
// Examples:
//   WOLGATE_SERVER__LISTEN=0.0.0.0:8080
//   WOLGATE_LOG__LEVEL=debug
func (c *Config) MergeFromEnv() *Config {
	// Server config
	if v := os.Getenv("WOLGATE_SERVER__LISTEN"); v != "" {
		c.Server.Listen = v
	}
	if v := os.Getenv("WOLGATE_SERVER__DATA"); v != "" {
		c.Server.Data = v
	}

	// Wake config
	if v := os.Getenv("WOLGATE_WAKE__IFACE"); v != "" {
		c.Wake.Iface = v
	}
	if v := os.Getenv("WOLGATE_WAKE__BROADCAST"); v != "" {
		c.Wake.Broadcast = v
	}

	// Log config
	if v := os.Getenv("WOLGATE_LOG__FILE"); v != "" {
		c.Log.File = v
	}
	if v := os.Getenv("WOLGATE_LOG__LEVEL"); v != "" {
		c.Log.Level = v
	}
	if v := os.Getenv("WOLGATE_LOG__MAX_SIZE"); v != "" {
		var size int
		if _, err := fmt.Sscanf(v, "%d", &size); err == nil && size > 0 {
			c.Log.MaxSize = size
		}
	}
	if v := os.Getenv("WOLGATE_LOG__MAX_BACKUPS"); v != "" {
		var backups int
		if _, err := fmt.Sscanf(v, "%d", &backups); err == nil && backups > 0 {
			c.Log.MaxBackups = backups
		}
	}
	if v := os.Getenv("WOLGATE_LOG__MAX_AGE"); v != "" {
		var age int
		if _, err := fmt.Sscanf(v, "%d", &age); err == nil && age >= 0 {
			c.Log.MaxAge = age
		}
	}

	return c
}

// MergeFromCLI merges configuration from command-line parameters.
// The params map uses dot notation to specify nested fields.
// Examples:
//   params["server.listen"] = "0.0.0.0:8080"
//   params["log.level"] = "debug"
func (c *Config) MergeFromCLI(params map[string]string) *Config {
	for key, value := range params {
		parts := strings.Split(key, ".")
		if len(parts) < 2 {
			continue
		}

		section := parts[0]
		field := parts[1]

		switch section {
		case "server":
			c.mergeServerField(field, value)
		case "wake":
			c.mergeWakeField(field, value)
		case "log":
			c.mergeLogField(field, value)
		}
	}

	return c
}

func (c *Config) mergeServerField(field, value string) {
	switch field {
	case "listen":
		c.Server.Listen = value
	case "data":
		c.Server.Data = value
	}
}

func (c *Config) mergeWakeField(field, value string) {
	switch field {
	case "iface":
		c.Wake.Iface = value
	case "broadcast":
		c.Wake.Broadcast = value
	}
}

func (c *Config) mergeLogField(field, value string) {
	switch field {
	case "file":
		c.Log.File = value
	case "level":
		c.Log.Level = value
	case "max_size":
		var size int
		if _, err := fmt.Sscanf(value, "%d", &size); err == nil && size > 0 {
			c.Log.MaxSize = size
		}
	case "max_backups":
		var backups int
		if _, err := fmt.Sscanf(value, "%d", &backups); err == nil && backups > 0 {
			c.Log.MaxBackups = backups
		}
	case "max_age":
		var age int
		if _, err := fmt.Sscanf(value, "%d", &age); err == nil && age >= 0 {
			c.Log.MaxAge = age
		}
	}
}

// Save saves the configuration to a file.
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
