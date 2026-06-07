//go:build desktop

package main

// IsProduction is set to true when the desktop build tag is present,
// which Wails adds automatically for production builds (wails build).
// In dev mode (wails dev), this build tag is absent.
func init() {
	IsProduction = true
}
