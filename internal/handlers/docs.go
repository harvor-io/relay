package handlers

import (
	"bytes"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/harvor-io/relay/static"
)

// DocsHandler serves the OpenAPI documentation and its supporting static
// files, rendered with Stoplight Elements.
type DocsHandler struct {
	fileServer http.Handler
}

// NewDocsHandler constructs a DocsHandler backed by the embedded OpenAPI
// spec in static/docs.
func NewDocsHandler() (*DocsHandler, error) {
	sub, err := fs.Sub(static.Docs, "docs")
	if err != nil {
		return nil, err
	}
	return &DocsHandler{
		fileServer: http.StripPrefix("/docs/", http.FileServer(http.FS(sub))),
	}, nil
}

// RegisterRoutes mounts the handler's routes on r.
func (h *DocsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/docs/api", h.GetDocs)
	r.Get("/docs/*", h.fileServer.ServeHTTP)
}

var docsTemplate = template.Must(template.New("docs").Parse(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Relay API Documentation</title>
    <script src="https://unpkg.com/@stoplight/elements/web-components.min.js"></script>
    <link rel="stylesheet" href="https://unpkg.com/@stoplight/elements/styles.min.css">
  </head>
  <body>
    <elements-api
      apiDescriptionUrl="{{.}}"
      router="hash"
    ></elements-api>
  </body>
</html>`))

// GetDocs renders the Stoplight Elements HTML page for the OpenAPI spec.
func (h *DocsHandler) GetDocs(w http.ResponseWriter, r *http.Request) {
	protocol := "http"
	if r.TLS != nil {
		protocol = "https"
	}
	if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		protocol = forwardedProto
	}

	host := r.Host
	if forwardedHost := r.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
		host = forwardedHost
	}

	port := ""
	if forwardedPort := r.Header.Get("X-Forwarded-Port"); forwardedPort != "" {
		port = ":" + forwardedPort
	}

	// Extract the route prefix so links keep working behind a reverse proxy.
	var routePrefix string
	if docsIndex := strings.LastIndex(r.URL.Path, "/docs/api"); docsIndex != -1 {
		routePrefix = r.URL.Path[:docsIndex]
	}

	specPath := routePrefix + "/docs/api.yaml"
	if forwardedPrefix := r.Header.Get("X-Forwarded-Path-Prefix"); forwardedPrefix != "" {
		specPath = forwardedPrefix + specPath
	}

	apiDescriptionURL := protocol + "://" + host + port + specPath

	// Render to a buffer first so a template failure doesn't leave a partial
	// 200 response on the wire.
	var buf bytes.Buffer
	if err := docsTemplate.Execute(&buf, apiDescriptionURL); err != nil {
		log.Printf("docs: failed to render documentation template: %v", err)
		http.Error(w, "Failed to render documentation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write(buf.Bytes())
}
