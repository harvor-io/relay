// Package middleware contains the HTTP middleware used by the API.
package middleware

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid/v5"
)

// requestIDKey is the context key under which the per-request ID is stored.
type requestIDKey struct{}

// RequestIDHeader is the response header carrying the request ID.
const RequestIDHeader = "X-Request-Id"

// RequestID assigns every request a UUIDv7 identifier, echoes it back on the
// response, and stashes it on the request context for handlers and logging.
// An inbound X-Request-Id is trusted and reused so a single ID can follow a
// request across services.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			if v, err := uuid.NewV7(); err == nil {
				id = v.String()
			}
		}

		w.Header().Set(RequestIDHeader, id)
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext returns the request ID assigned by RequestID, or an
// empty string when the middleware did not run.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
