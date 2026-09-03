// Package handlers contains the HTTP handlers exposed by the API.
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// HealthHandler serves the process health check.
type HealthHandler struct{}

// NewHealthHandler constructs a HealthHandler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// RegisterRoutes mounts the handler's routes on r.
func (h *HealthHandler) RegisterRoutes(r chi.Router) {
	r.Get("/healthz", h.GetHealth)
}

type getHealthResponse struct {
	Status string `json:"status"`
}

// GetHealth reports that the process can serve HTTP.
func (h *HealthHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, getHealthResponse{Status: http.StatusText(http.StatusOK)})
}
