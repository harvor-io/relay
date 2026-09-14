package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/harvor-io/relay/internal/middleware"
	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/services"
)

// IngestHandler serves the /ingest/{source} resource: the public entry point
// through which Sources deliver events into relay.
type IngestHandler struct {
	ingest services.IngestService
	auth   func(http.Handler) http.Handler
}

// NewIngestHandler constructs an IngestHandler backed by ingest, which
// authenticates requests with auth (see middleware.IngestAuth) before
// resolving them to a Source.
func NewIngestHandler(ingest services.IngestService, auth func(http.Handler) http.Handler) *IngestHandler {
	return &IngestHandler{ingest: ingest, auth: auth}
}

// RegisterRoutes mounts the handler's route on r.
func (h *IngestHandler) RegisterRoutes(r chi.Router) {
	r.With(h.auth).Post("/ingest/{source}", h.Ingest)
}

// ingestRequest is the accepted body for POST /ingest/{source}.
type ingestRequest struct {
	Type       string          `json:"type"`
	Data       json.RawMessage `json:"data"`
	EnvelopeID string          `json:"envelope_id"`
}

// envelopeResource is the JSON representation of an envelope. It is an
// explicit projection of models.Envelope: only the fields that make up the
// REST resource.
type envelopeResource struct {
	ID        string          `json:"id"`
	SourceID  string          `json:"source_id"`
	Source    string          `json:"source"`
	Type      string          `json:"type"`
	Topic     string          `json:"topic"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"created_at"`
}

func newEnvelopeResource(e *models.Envelope) envelopeResource {
	return envelopeResource{
		ID:        e.ID.String(),
		SourceID:  e.SourceID.String(),
		Source:    e.Source,
		Type:      e.Type,
		Topic:     e.Topic().String(),
		Data:      e.Data,
		CreatedAt: e.CreatedAt,
	}
}

// Ingest accepts an event from an already-authenticated Source (attached to
// the request context by middleware.IngestAuth) and persists it as an
// Envelope, returning it with a 201.
func (h *IngestHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	source, ok := middleware.SourceFromContext(r.Context())
	if !ok {
		renderError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	var body ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		renderError(w, r, http.StatusBadRequest, "request body is not valid JSON")
		return
	}

	envelope, err := h.ingest.Ingest(r.Context(), source, services.IngestInput{
		Type:       body.Type,
		Data:       body.Data,
		EnvelopeID: body.EnvelopeID,
	})
	if err != nil {
		renderIngestError(w, r, err)
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, newEnvelopeResource(envelope))
}

// renderIngestError maps the errors returned by services.IngestService onto
// HTTP responses.
func renderIngestError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, services.ErrIngestEnvelopeIDTaken):
		renderError(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, services.ErrIngestTypeRequired),
		errors.Is(err, services.ErrIngestTypeInvalid),
		errors.Is(err, services.ErrIngestTypeTooLong),
		errors.Is(err, services.ErrIngestDataRequired),
		errors.Is(err, services.ErrIngestEnvelopeIDInvalid):
		renderError(w, r, http.StatusUnprocessableEntity, err.Error())
	default:
		renderError(w, r, http.StatusInternalServerError, "internal server error")
	}
}
