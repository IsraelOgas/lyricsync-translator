//go:build !desktop

package main

// IsProduction is false in dev mode (wails dev). Wails does not add
// the desktop build tag, so the Vite dev server handles the frontend
// with full HMR support.
func init() {
	IsProduction = false
}
