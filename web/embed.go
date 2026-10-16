// Package web holds the embedded page.
package web

import "embed"

// FS holds the page files.
//
//go:embed index.html style.css js vendor
var FS embed.FS
