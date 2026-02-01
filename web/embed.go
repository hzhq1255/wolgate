// Package web provides the web interface for wolgate.
package web

import (
	"embed"
	"io/fs"
)

//go:embed index.html
var staticFiles embed.FS

// IndexHTML returns the embedded index.html content.
func IndexHTML() ([]byte, error) {
	content, err := staticFiles.ReadFile("index.html")
	return content, err
}

// StaticFS returns the embedded file system for static files.
func StaticFS() (fs.FS, error) {
	return fs.Sub(staticFiles, ".")
}
