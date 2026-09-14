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

// SourceKeyHandler serves the /sources/{id}/keys resource.
type SourceKeyHandler struct {
	keys services.SourceKeyService
}

// NewSourceKeyHandler constructs a SourceKeyHandler backed by keys.
func NewSourceKeyHandler(keys services.SourceKeyService) *SourceKeyHandler {
	return &SourceKeyHandler{keys: keys}
}

// RegisterRoutes mounts the handler's routes on r. It shares the "id" path
// parameter with SourceHandler's "/sources/{id}" route.
func (h *SourceKeyHandler) RegisterRoutes(r chi.Router) {
	r.Route("/sources/{id}/keys", func(r chi.Router) {
		r.Get("/", h.ListSourceKeys)
		r.Post("/", h.CreateSourceKey)
		r.Route("/{keyID}", func(r chi.Router) {
			r.Get("/", h.GetSourceKey)
			r.Delete("/", h.DeleteSourceKey)
			r.Post("/activate", h.ActivateSourceKey)
			r.Post("/deactivate", h.DeactivateSourceKey)
		})
	})
}

// sourceKeyResource is the JSON representation of a source key. It is an
// explicit projection of models.SourceKey: only the fields that make up the
// REST resource. The key's secret is never included here; it is returned
// only once, in the response to CreateSourceKey.
type sourceKeyResource struct {
	ID        string    `json:"id"`
	SourceID  string    `json:"source_id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newSourceKeyResource(k *models.SourceKey) sourceKeyResource {
	return sourceKeyResource{
		ID:        k.ID.String(),
		SourceID:  k.SourceID.String(),
		Name:      k.Name,
		IsActive:  k.IsActive,
		CreatedAt: k.CreatedAt,
		UpdatedAt: k.UpdatedAt,
	}
}

func newSourceKeyResourceList(keys []models.SourceKey) []sourceKeyResource {
	resources := make([]sourceKeyResource, 0, len(keys))
	for i := range keys {
		resources = append(resources, newSourceKeyResource(&keys[i]))
	}
	return resources
}

// createdSourceKeyResource is the JSON representation returned by
// CreateSourceKey: a sourceKeyResource plus the plaintext secret, which is
// never shown again after this response.
type createdSourceKeyResource struct {
	sourceKeyResource
	Secret string `json:"secret"`
}

// createSourceKeyRequest is the accepted body for POST /sources/{id}/keys.
type createSourceKeyRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListSourceKeys returns every key for a source as a JSON array.
func (h *SourceKeyHandler) ListSourceKeys(w http.ResponseWriter, r *http.Request) {
	sourceID, ok := urlUUID(w, r)
	if !ok {
		return
	}
	keys, err := h.keys.ListBySource(r.Context(), sourceID)
	if err != nil {
		renderSourceKeyError(w, r, err)
		return
	}
	render.JSON(w, r, newSourceKeyResourceList(keys))
}

// CreateSourceKey generates a new key for a source and returns it, along with
// its plaintext secret, with a 201. The secret is not retrievable again.
func (h *SourceKeyHandler) CreateSourceKey(w http.ResponseWriter, r *http.Request) {
	sourceID, ok := urlUUID(w, r)
	if !ok {
		return
	}

	var body createSourceKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		renderError(w, r, http.StatusBadRequest, "request body is not valid JSON")
		return
	}

	key, secret, err := h.keys.Create(r.Context(), sourceID, services.CreateSourceKeyInput{ID: body.ID, Name: body.Name})
	if err != nil {
		renderSourceKeyError(w, r, err)
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, createdSourceKeyResource{
		sourceKeyResource: newSourceKeyResource(key),
		Secret:            secret,
	})
}

// GetSourceKey returns a single key by ID.
func (h *SourceKeyHandler) GetSourceKey(w http.ResponseWriter, r *http.Request) {
	sourceID, ok := urlUUID(w, r)
	if !ok {
		return
	}
	keyID, ok := urlUUIDParam(w, r, "keyID")
	if !ok {
		return
	}
	key, err := h.keys.Get(r.Context(), sourceID, keyID)
	if err != nil {
		renderSourceKeyError(w, r, err)
		return
	}
	render.JSON(w, r, newSourceKeyResource(key))
}

// ActivateSourceKey marks a key as active.
func (h *SourceKeyHandler) ActivateSourceKey(w http.ResponseWriter, r *http.Request) {
	sourceID, ok := urlUUID(w, r)
	if !ok {
		return
	}
	keyID, ok := urlUUIDParam(w, r, "keyID")
	if !ok {
		return
	}
	key, err := h.keys.Activate(r.Context(), sourceID, keyID)
	if err != nil {
		renderSourceKeyError(w, r, err)
		return
	}
	render.JSON(w, r, newSourceKeyResource(key))
}

// DeactivateSourceKey marks a key as inactive.
func (h *SourceKeyHandler) DeactivateSourceKey(w http.ResponseWriter, r *http.Request) {
	sourceID, ok := urlUUID(w, r)
	if !ok {
		return
	}
	keyID, ok := urlUUIDParam(w, r, "keyID")
	if !ok {
		return
	}
	key, err := h.keys.Deactivate(r.Context(), sourceID, keyID)
	if err != nil {
		renderSourceKeyError(w, r, err)
		return
	}
	render.JSON(w, r, newSourceKeyResource(key))
}

// DeleteSourceKey removes a key by ID and returns 204.
func (h *SourceKeyHandler) DeleteSourceKey(w http.ResponseWriter, r *http.Request) {
	sourceID, ok := urlUUID(w, r)
	if !ok {
		return
	}
	keyID, ok := urlUUIDParam(w, r, "keyID")
	if !ok {
		return
	}
	if err := h.keys.Delete(r.Context(), sourceID, keyID); err != nil {
		renderSourceKeyError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// renderSourceKeyError maps the errors returned by services.SourceKeyService
// onto HTTP responses.
func renderSourceKeyError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, services.ErrSourceNotFound), errors.Is(err, services.ErrSourceKeyNotFound):
		renderError(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrSourceKeyIDTaken):
		renderError(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, services.ErrSourceKeyNameRequired),
		errors.Is(err, services.ErrSourceKeyNameTooLong),
		errors.Is(err, services.ErrSourceKeyIDInvalid):
		renderError(w, r, http.StatusUnprocessableEntity, err.Error())
	default:
		renderError(w, r, http.StatusInternalServerError, "internal server error")
	}
}
