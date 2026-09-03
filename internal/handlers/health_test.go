package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/harvor-io/relay/internal/handlers"
)

func TestGetHealth(t *testing.T) {
	r := chi.NewRouter()
	handlers.NewHealthHandler().RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != http.StatusText(http.StatusOK) {
		t.Fatalf("status = %q, want %q", body.Status, http.StatusText(http.StatusOK))
	}
}
