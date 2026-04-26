// Package server - Embedded static assets and templates.
// All assets embedded at build time using Go embed package.
// See AI.md PART 7 for single binary with embedded assets.
package server

import "embed"

// Static assets (CSS, JS, images)
//
//go:embed static
var StaticFS embed.FS

// HTML templates
//
//go:embed template
var TemplateFS embed.FS
