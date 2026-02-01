// Package main is the entry point for wolgate.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/hzhq1255/wolgate/config"
	"github.com/hzhq1255/wolgate/logger"
	"github.com/hzhq1255/wolgate/store"
	"github.com/hzhq1255/wolgate/web"
	"github.com/hzhq1255/wolgate/wol"
)

// Version is set by build using -ldflags
var Version = "dev"

const (
	// Default configuration values
	defaultListen    = "127.0.0.1:9000"
	defaultData      = "/data/wolgate.json"
	defaultLogFile   = "/tmp/wolgate.log"
	defaultLogLevel  = "info"
	defaultLogMaxSize  = 10  // MB
	defaultLogMaxBackups = 3
	defaultLogMaxAge = 7  // days
)

// Global flags (common to all subcommands)
var (
	configFile  string
	logFile     string
	logLevel    string
	logMaxSize  int
	logMaxBackups int
	logMaxAge   int
)

func main() {
	// Define usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "wolgate - Lightweight Wake-on-LAN gateway\n\n")
		fmt.Fprintf(os.Stderr, "Usage: wolgate <command> [options]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  server    Start web management service\n")
		fmt.Fprintf(os.Stderr, "  wake      Send WOL magic packet to a device\n")
		fmt.Fprintf(os.Stderr, "  version   Show version information\n")
		fmt.Fprintf(os.Stderr, "  help      Show this help message\n\n")
		fmt.Fprintf(os.Stderr, "Global Options:\n")
		fmt.Fprintf(os.Stderr, "  -config string\n")
		fmt.Fprintf(os.Stderr, "        Configuration file path (default: /etc/wolgate.json)\n")
		fmt.Fprintf(os.Stderr, "  -log string\n")
		fmt.Fprintf(os.Stderr, "        Log file path (default: %s)\n", defaultLogFile)
		fmt.Fprintf(os.Stderr, "  -log-level string\n")
		fmt.Fprintf(os.Stderr, "        Log level: debug, info, warn, error (default: %s)\n", defaultLogLevel)
		fmt.Fprintf(os.Stderr, "  -log-max-size int\n")
		fmt.Fprintf(os.Stderr, "        Maximum log file size in MB (default: %d)\n", defaultLogMaxSize)
		fmt.Fprintf(os.Stderr, "  -log-max-backups int\n")
		fmt.Fprintf(os.Stderr, "        Maximum number of backup log files (default: %d)\n", defaultLogMaxBackups)
		fmt.Fprintf(os.Stderr, "  -log-max-age int\n")
		fmt.Fprintf(os.Stderr, "        Maximum age of backup log files in days (default: %d)\n", defaultLogMaxAge)
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Parse global flags
	flag.StringVar(&configFile, "config", "/etc/wolgate.json", "Configuration file path")
	flag.StringVar(&logFile, "log", "", "Log file path")
	flag.StringVar(&logLevel, "log-level", "", "Log level")
	flag.IntVar(&logMaxSize, "log-max-size", 0, "Maximum log file size in MB")
	flag.IntVar(&logMaxBackups, "log-max-backups", 0, "Maximum number of backup log files")
	flag.IntVar(&logMaxAge, "log-max-age", 0, "Maximum age of backup log files in days")

	flag.Parse()

	// Get subcommand
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	command := args[0]

	// Execute subcommand
	switch command {
	case "server":
		runServer(args[1:])
	case "wake":
		runWake(args[1:])
	case "version":
		fmt.Printf("wolgate version %s\n", Version)
	case "help", "-h", "--help":
		flag.Usage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		flag.Usage()
		os.Exit(1)
	}
}

// runServer starts the web management service.
func runServer(args []string) {
	// Define server-specific flags
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	listen := fs.String("listen", "", "HTTP listen address")
	dataFile := fs.String("data", "", "Device data file path")
	iface := fs.String("iface", "", "Network interface for WOL")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Apply command-line overrides
	if *listen != "" {
		cfg.Server.Listen = *listen
	}
	if *dataFile != "" {
		cfg.Server.Data = *dataFile
	}
	if *iface != "" {
		cfg.Wake.Iface = *iface
	}

	// Initialize logger
	log, err := logger.New(logger.Config{
		File:       cfg.Log.File,
		Level:      cfg.Log.Level,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	log.Info("wolgate %s starting...", Version)

	// Initialize store
	st, err := store.NewStore(cfg.Server.Data)
	if err != nil {
		log.Error("Failed to initialize store: %v", err)
		os.Exit(1)
	}

	// Initialize WOL sender
	wolSender, err := wol.NewSender(cfg.Wake.Iface, cfg.Wake.Broadcast)
	if err != nil {
		log.Error("Failed to initialize WOL sender: %v", err)
		os.Exit(1)
	}

	// Initialize HTTP handler
	handler := web.NewHandler(st, wolSender)

	// Register routes
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Start HTTP server
	server := &http.Server{
		Addr:    cfg.Server.Listen,
		Handler: mux,
	}

	// Handle shutdown gracefully
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Info("Shutting down...")
		server.Shutdown(context.Background())
	}()

	log.Info("Server listening on %s", cfg.Server.Listen)
	log.Info("Data file: %s", cfg.Server.Data)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("Server error: %v", err)
		os.Exit(1)
	}

	log.Info("Server stopped")
}

// runWake sends a WOL magic packet.
func runWake(args []string) {
	// Define wake-specific flags
	fs := flag.NewFlagSet("wake", flag.ExitOnError)
	mac := fs.String("mac", "", "Target MAC address (required)")
	iface := fs.String("iface", "", "Network interface")
	bcast := fs.String("bcast", "", "Broadcast address")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Validate MAC
	if *mac == "" {
		fmt.Fprintf(os.Stderr, "Error: -mac is required\n")
		os.Exit(1)
	}

	// Load configuration for defaults
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		cfg = defaultConfig()
	}

	// Apply command-line overrides
	if *iface != "" {
		cfg.Wake.Iface = *iface
	}
	if *bcast != "" {
		cfg.Wake.Broadcast = *bcast
	}

	// Initialize logger (to stderr only if no log file specified)
	logCfg := logger.Config{
		File:  "", // Log to stderr
		Level: cfg.Log.Level,
	}
	if cfg.Log.File != "" {
		logCfg.File = cfg.Log.File
	}
	log, _ := logger.New(logCfg)
	defer log.Close()

	// Initialize WOL sender
	wolSender, err := wol.NewSender(cfg.Wake.Iface, cfg.Wake.Broadcast)
	if err != nil {
		log.Error("Failed to initialize WOL sender: %v", err)
		os.Exit(1)
	}

	// Send WOL packet
	log.Info("Sending WOL packet to %s", *mac)
	if cfg.Wake.Iface != "" {
		log.Info("Interface: %s", cfg.Wake.Iface)
	}
	log.Info("Broadcast: %s", cfg.Wake.Broadcast)

	if err := wolSender.SendRepeat(*mac, 3); err != nil {
		log.Error("Failed to send WOL packet: %v", err)
		os.Exit(1)
	}

	log.Info("WOL packet sent successfully to %s", *mac)
	fmt.Printf("✓ WOL packet sent to %s\n", *mac)
}

// loadConfig loads the configuration file.
func loadConfig() (*Config, error) {
	if configFile == "" {
		return defaultConfig(), nil
	}

	// Load config using the config package
	cfg, err := config.Load(configFile)
	if err != nil {
		return nil, err
	}

	// Convert config.Config to our Config
	return &Config{
		Server: ServerConfig{
			Listen: cfg.Server.Listen,
			Data:   cfg.Server.Data,
		},
		Wake: WakeConfig{
			Iface:     cfg.Wake.Iface,
			Broadcast: cfg.Wake.Broadcast,
		},
		Log: LogConfig{
			File:       cfg.Log.File,
			Level:      cfg.Log.Level,
			MaxSize:    cfg.Log.MaxSize,
			MaxBackups: cfg.Log.MaxBackups,
			MaxAge:     cfg.Log.MaxAge,
		},
	}, nil
}

// defaultConfig returns a default configuration.
func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Listen: defaultListen,
			Data:   defaultData,
		},
		Wake: WakeConfig{
			Iface:     "",
			Broadcast: "255.255.255.255",
		},
		Log: LogConfig{
			File:       defaultLogFile,
			Level:      defaultLogLevel,
			MaxSize:    defaultLogMaxSize,
			MaxBackups: defaultLogMaxBackups,
			MaxAge:     defaultLogMaxAge,
		},
	}
}

// Config represents the application configuration.
type Config struct {
	Server ServerConfig
	Wake   WakeConfig
	Log    LogConfig
}

// ServerConfig holds server configuration.
type ServerConfig struct {
	Listen string
	Data   string
}

// WakeConfig holds WOL configuration.
type WakeConfig struct {
	Iface     string
	Broadcast string
}

// LogConfig holds logging configuration.
type LogConfig struct {
	File       string
	Level      string
	MaxSize    int
	MaxBackups int
	MaxAge     int
}
