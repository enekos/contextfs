//go:build !ui_assets

package ui

import (
	"io/fs"
	"testing/fstest"
)

// FS provides a minimal fallback filesystem for builds that do not embed UI assets.
var FS fs.FS = fstest.MapFS{
	"dist/index.html": &fstest.MapFile{
		Data: []byte("<!doctype html><html><body><h1>Mairu UI not embedded</h1><p>Build UI assets and rebuild with -tags ui_assets.</p></body></html>"),
	},
}
