// Package logger provides logging utilities.
package logger

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// backupInfo holds information about a backup file.
type backupInfo struct {
	Path     string
	ModTime  time.Time
	FileInfo os.FileInfo
}

// findBackupFiles finds all backup files for the given log file.
func findBackupFiles(logPath string) ([]backupInfo, error) {
	dir := filepath.Dir(logPath)
	baseName := filepath.Base(logPath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var backups []backupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Check if this is a backup file (starts with base name + ".")
		if strings.HasPrefix(name, baseName+".") {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			fullPath := filepath.Join(dir, name)
			backups = append(backups, backupInfo{
				Path:     fullPath,
				ModTime:  info.ModTime(),
				FileInfo: info,
			})
		}
	}

	return backups, nil
}

// sortBackups sorts backup files by modification time (newest first).
func sortBackups(backups []backupInfo) {
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})
}
