// Package main provides the embedded frontend build output.
// The embed directive must be at module root to reference web/dist/*
// without ".." (which Go's go:embed does not allow in patterns).
package main

import (
	"embed"
	"io/fs"
)

//go:embed web/dist/*
var embeddedFS embed.FS

// FS returns the embedded web/dist filesystem with the prefix stripped.
func FS() fs.FS {
	sub, err := fs.Sub(embeddedFS, "web/dist")
	if err != nil {
		panic("embedded assets: " + err.Error())
	}
	return sub
}
