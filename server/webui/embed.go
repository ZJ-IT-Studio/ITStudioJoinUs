package webui

import "embed"

// Dist contains the compiled Vue application.
//
//go:embed dist
var Dist embed.FS
