// Package static embeds the OpenAPI documentation source served by the docs
// handler, so the API binary can serve it without any files on disk at
// runtime.
package static

import "embed"

//go:embed all:docs
var Docs embed.FS
