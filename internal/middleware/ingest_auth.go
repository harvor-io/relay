package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/services"
)

// SignatureHeader is the HTTP header carrying the hex-encoded HMAC-SHA256
// signature of the request body, computed with the secret of one of the
// ingesting Source's keys.
const SignatureHeader = "X-Relay-Signature"

// sourceContextKey is the context key under which the Source resolved by
// IngestAuth is stored.
type sourceContextKey struct{}

// WithSource attaches source to ctx.
func WithSource(ctx context.Context, source *models.Source) context.Context {
	return context.WithValue(ctx, sourceContextKey{}, source)
}

// SourceFromContext returns the Source attached by IngestAuth, or ok == false
// if none was attached.
func SourceFromContext(ctx context.Context) (source *models.Source, ok bool) {
	source, ok = ctx.Value(sourceContextKey{}).(*models.Source)
	return source, ok
}

// IngestAuth authenticates requests to the ingest endpoint. It resolves the
// "source" URL parameter — a Source's ID or slug — via sources, writing a
// 404 if neither matches, then verifies the request body against
// SignatureHeader using every active key belonging to that Source via keys,
// writing a 401 if none match. On success it attaches the resolved Source to
// the request context, retrievable by the next handler with
// SourceFromContext, and rewinds the body so the handler can read it again.
func IngestAuth(sources services.SourceService, keys services.SourceKeyService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			source, err := sources.GetByIDOrSlug(r.Context(), chi.URLParam(r, "source"))
			if err != nil {
				if errors.Is(err, services.ErrSourceNotFound) {
					writeIngestAuthError(w, http.StatusNotFound, err.Error())
					return
				}
				writeIngestAuthError(w, http.StatusInternalServerError, "internal server error")
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeIngestAuthError(w, http.StatusBadRequest, "failed to read request body")
				return
			}
			_ = r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))

			signature := r.Header.Get(SignatureHeader)
			if signature == "" {
				writeIngestAuthError(w, http.StatusUnauthorized, "missing "+SignatureHeader+" header")
				return
			}

			ok, err := keys.VerifySignature(r.Context(), source.ID, body, signature)
			if err != nil {
				writeIngestAuthError(w, http.StatusInternalServerError, "internal server error")
				return
			}
			if !ok {
				writeIngestAuthError(w, http.StatusUnauthorized, "invalid signature")
				return
			}

			next.ServeHTTP(w, r.WithContext(WithSource(r.Context(), source)))
		})
	}
}

// writeIngestAuthError writes status with a JSON body matching the API's
// Error schema ({"error": "message"}). It is a self-contained duplicate of
// that shape rather than a dependency on the handlers package, keeping
// middleware independent of the HTTP layer that mounts it.
func writeIngestAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{Error: message})
}
