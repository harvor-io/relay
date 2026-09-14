package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/services"
)

// SourceHandler serves the /sources resource.
type SourceHandler struct {
	sources services.SourceService
}

// NewSourceHandler constructs a SourceHandler backed by sources.
func NewSourceHandler(sources services.SourceService) *SourceHandler {
	return &SourceHandler{sources: sources}
}

// RegisterRoutes mounts the handler's routes on r.
func (h *SourceHandler) RegisterRoutes(r chi.Router) {
	r.Route("/sources", func(r chi.Router) {
		r.Get("/", h.ListSources)
		r.Post("/", h.CreateSource)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetSource)
			r.Patch("/", h.UpdateSource)
			r.Delete("/", h.DeleteSource)
			r.Post("/activate", h.ActivateSource)
			r.Post("/deactivate", h.DeactivateSource)
		})
	})
}

// sourceResource is the JSON representation of a source. It is an explicit
// projection of models.Source: only the fields that make up the REST resource,
// with nothing about how the record is stored or counted.
type sourceResource struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description *string   `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newSourceResource(s *models.Source) sourceResource {
	return sourceResource{
		ID:          s.ID.String(),
		Name:        s.Name,
		Slug:        s.Slug,
		Description: s.Description,
		IsActive:    s.IsActive,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

func newSourceResourceList(sources []models.Source) []sourceResource {
	resources := make([]sourceResource, 0, len(sources))
	for i := range sources {
		resources = append(resources, newSourceResource(&sources[i]))
	}
	return resources
}

// createSourceRequest is the accepted body for POST /sources.
type createSourceRequest struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description"`
}

// updateSourceRequest is the accepted body for PATCH /sources/{id}. A field
// left out of the JSON body (or sent as null) is unchanged; to clear the
// description, send an empty string rather than null.
type updateSourceRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// ListSources returns every source as a JSON array.
func (h *SourceHandler) ListSources(w http.ResponseWriter, r *http.Request) {
	sources, err := h.sources.List(r.Context())
	if err != nil {
		renderSourceError(w, r, err)
		return
	}
	render.JSON(w, r, newSourceResourceList(sources))
}

// GetSource returns a single source by ID.
func (h *SourceHandler) GetSource(w http.ResponseWriter, r *http.Request) {
	id, ok := urlUUID(w, r)
	if !ok {
		return
	}
	source, err := h.sources.Get(r.Context(), id)
	if err != nil {
		renderSourceError(w, r, err)
		return
	}
	render.JSON(w, r, newSourceResource(source))
}

// CreateSource registers a new source and returns it with a 201.
func (h *SourceHandler) CreateSource(w http.ResponseWriter, r *http.Request) {
	var body createSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		renderError(w, r, http.StatusBadRequest, "request body is not valid JSON")
		return
	}

	source, err := h.sources.Create(r.Context(), services.CreateSourceInput{
		ID:          body.ID,
		Name:        body.Name,
		Slug:        body.Slug,
		Description: body.Description,
	})
	if err != nil {
		renderSourceError(w, r, err)
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, newSourceResource(source))
}

// UpdateSource changes the name and/or description of a source.
func (h *SourceHandler) UpdateSource(w http.ResponseWriter, r *http.Request) {
	id, ok := urlUUID(w, r)
	if !ok {
		return
	}

	var body updateSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		renderError(w, r, http.StatusBadRequest, "request body is not valid JSON")
		return
	}

	source, err := h.sources.Update(r.Context(), id, services.UpdateSourceInput{
		Name:        body.Name,
		Description: body.Description,
	})
	if err != nil {
		renderSourceError(w, r, err)
		return
	}
	render.JSON(w, r, newSourceResource(source))
}

// DeleteSource removes a source by ID and returns 204.
func (h *SourceHandler) DeleteSource(w http.ResponseWriter, r *http.Request) {
	id, ok := urlUUID(w, r)
	if !ok {
		return
	}
	if err := h.sources.Delete(r.Context(), id); err != nil {
		renderSourceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ActivateSource marks a source as active.
func (h *SourceHandler) ActivateSource(w http.ResponseWriter, r *http.Request) {
	id, ok := urlUUID(w, r)
	if !ok {
		return
	}
	source, err := h.sources.Activate(r.Context(), id)
	if err != nil {
		renderSourceError(w, r, err)
		return
	}
	render.JSON(w, r, newSourceResource(source))
}

// DeactivateSource marks a source as inactive.
func (h *SourceHandler) DeactivateSource(w http.ResponseWriter, r *http.Request) {
	id, ok := urlUUID(w, r)
	if !ok {
		return
	}
	source, err := h.sources.Deactivate(r.Context(), id)
	if err != nil {
		renderSourceError(w, r, err)
		return
	}
	render.JSON(w, r, newSourceResource(source))
}

// renderSourceError maps the errors returned by services.SourceService onto
// HTTP responses.
func renderSourceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, services.ErrSourceNotFound):
		renderError(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrSourceSlugTaken), errors.Is(err, services.ErrSourceIDTaken):
		renderError(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, services.ErrSourceNameRequired),
		errors.Is(err, services.ErrSourceNameTooLong),
		errors.Is(err, services.ErrSourceDescriptionTooLong),
		errors.Is(err, services.ErrSourceSlugInvalid),
		errors.Is(err, services.ErrSourceSlugUnderivable),
		errors.Is(err, services.ErrSourceIDInvalid):
		renderError(w, r, http.StatusUnprocessableEntity, err.Error())
	default:
		renderError(w, r, http.StatusInternalServerError, "internal server error")
	}
}
