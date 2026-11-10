package handlers

import (
	"bytes"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/harvor-io/relay/static"
)

// DocsHandler serves the OpenAPI documentation, rendered with Scalar, along
// with the spec it is rendered from: bundled into one document, and as its
// source files.
type DocsHandler struct {
	fileServer http.Handler
	bundle     []byte
	stability  docsStability
}

// NewDocsHandler constructs a DocsHandler backed by the embedded OpenAPI
// spec in static/docs. It bundles the spec up front, so a broken reference
// or an operation without a stability level fails here rather than in a
// reader's browser.
func NewDocsHandler() (*DocsHandler, error) {
	sub, err := fs.Sub(static.Docs, "docs")
	if err != nil {
		return nil, err
	}
	bundle, stability, err := bundleSpec(sub, "api.yaml")
	if err != nil {
		return nil, err
	}
	return &DocsHandler{
		fileServer: http.StripPrefix("/docs/", http.FileServer(http.FS(sub))),
		bundle:     bundle,
		stability:  stability,
	}, nil
}

// RegisterRoutes mounts the handler's routes on r.
func (h *DocsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/docs/api", h.GetDocs)
	r.Get("/docs/openapi.yaml", h.GetBundledSpec)
	r.Get("/docs/*", h.fileServer.ServeHTTP)
}

// GetBundledSpec returns the OpenAPI spec as a single YAML document.
func (h *DocsHandler) GetBundledSpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(h.bundle)
}

var docsTemplate = template.Must(template.New("docs").Parse(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Relay API Documentation</title>
    <link rel="stylesheet" href="stability-marks.css">
  </head>
  <body>
    <div id="app"></div>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
    <script>
      Scalar.createApiReference("#app", {
        url: {{.SpecURL}},
        // We don't offer Scalar's AI chat or generated client libraries.
        agent: { disabled: true },
        hiddenClients: true,
      });
    </script>
    <script>
      window.EXALYNT_STABILITY = {{.Stability}};
    </script>
    <script src="stability-marks.js"></script>
  </body>
</html>`))

// GetDocs renders the Scalar HTML page for the bundled OpenAPI spec.
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

	specPath := routePrefix + "/docs/openapi.yaml"
	if forwardedPrefix := r.Header.Get("X-Forwarded-Path-Prefix"); forwardedPrefix != "" {
		specPath = forwardedPrefix + specPath
	}

	apiDescriptionURL := protocol + "://" + host + port + specPath

	// Render to a buffer first so a template failure doesn't leave a partial
	// 200 response on the wire.
	var buf bytes.Buffer
	data := struct {
		SpecURL   string
		Stability docsStability
	}{apiDescriptionURL, h.stability}
	if err := docsTemplate.Execute(&buf, data); err != nil {
		slog.ErrorContext(r.Context(), "docs: render documentation template", "error", err)
		http.Error(w, "Failed to render documentation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write(buf.Bytes())
}
