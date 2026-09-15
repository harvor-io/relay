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

// DestinationHandler serves the /destinations resource.
type DestinationHandler struct {
	destinations services.DestinationService
}

// NewDestinationHandler constructs a DestinationHandler backed by
// destinations.
func NewDestinationHandler(destinations services.DestinationService) *DestinationHandler {
	return &DestinationHandler{destinations: destinations}
}

// RegisterRoutes mounts the handler's routes on r.
func (h *DestinationHandler) RegisterRoutes(r chi.Router) {
	r.Route("/destinations", func(r chi.Router) {
		r.Get("/", h.ListDestinations)
		r.Post("/", h.CreateDestination)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetDestination)
			r.Patch("/", h.UpdateDestination)
			r.Delete("/", h.DeleteDestination)
			r.Post("/activate", h.ActivateDestination)
			r.Post("/deactivate", h.DeactivateDestination)
		})
	})
}

// destinationResource is the JSON representation of a destination. It is an
// explicit projection of models.Destination: only the fields that make up
// the REST resource, with nothing about how the record is stored or
// counted.
type destinationResource struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	Config        json.RawMessage `json:"config"`
	Description   *string         `json:"description"`
	Subscriptions []string        `json:"subscriptions"`
	IsActive      bool            `json:"is_active"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

func newDestinationResource(d *models.Destination) destinationResource {
	return destinationResource{
		ID:            d.ID.String(),
		Name:          d.Name,
		Type:          string(d.Type),
		Config:        d.Config,
		Description:   d.Description,
		Subscriptions: destinationSubscriptionStrings(d.Subscriptions),
		IsActive:      d.IsActive,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}

// destinationSubscriptionStrings converts a destination's subscriptions to
// their plain string form for JSON responses.
func destinationSubscriptionStrings(subscriptions []models.Subscription) []string {
	result := make([]string, len(subscriptions))
	for i, s := range subscriptions {
		result[i] = s.String()
	}
	return result
}

func newDestinationResourceList(destinations []models.Destination) []destinationResource {
	resources := make([]destinationResource, 0, len(destinations))
	for i := range destinations {
		resources = append(resources, newDestinationResource(&destinations[i]))
	}
	return resources
}

// createDestinationRequest is the accepted body for POST /destinations.
type createDestinationRequest struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	Config        json.RawMessage `json:"config"`
	Description   *string         `json:"description"`
	Subscriptions []string        `json:"subscriptions"`
}

// updateDestinationRequest is the accepted body for PATCH /destinations/{id}.
// A field left out of the JSON body (or sent as null) is unchanged; to clear
// the description, send an empty string rather than null; to clear
// subscriptions, send an empty array rather than null.
type updateDestinationRequest struct {
	Name          *string   `json:"name"`
	Description   *string   `json:"description"`
	Subscriptions *[]string `json:"subscriptions"`
}

// ListDestinations returns every destination as a JSON array.
func (h *DestinationHandler) ListDestinations(w http.ResponseWriter, r *http.Request) {
	destinations, err := h.destinations.List(r.Context())
	if err != nil {
		renderDestinationError(w, r, err)
		return
	}
	render.JSON(w, r, newDestinationResourceList(destinations))
}

// GetDestination returns a single destination by ID.
func (h *DestinationHandler) GetDestination(w http.ResponseWriter, r *http.Request) {
	id, ok := urlUUID(w, r)
	if !ok {
		return
	}
	destination, err := h.destinations.Get(r.Context(), id)
	if err != nil {
		renderDestinationError(w, r, err)
		return
	}
	render.JSON(w, r, newDestinationResource(destination))
}

// CreateDestination registers a new destination and returns it with a 201.
func (h *DestinationHandler) CreateDestination(w http.ResponseWriter, r *http.Request) {
	var body createDestinationRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		renderError(w, r, http.StatusBadRequest, "request body is not valid JSON")
		return
	}

	destination, err := h.destinations.Create(r.Context(), services.CreateDestinationInput{
		ID:            body.ID,
		Name:          body.Name,
		Type:          body.Type,
		Config:        body.Config,
		Description:   body.Description,
		Subscriptions: body.Subscriptions,
	})
	if err != nil {
		renderDestinationError(w, r, err)
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, newDestinationResource(destination))
}

// UpdateDestination changes the name and/or description of a destination.
func (h *DestinationHandler) UpdateDestination(w http.ResponseWriter, r *http.Request) {
	id, ok := urlUUID(w, r)
	if !ok {
		return
	}

	var body updateDestinationRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		renderError(w, r, http.StatusBadRequest, "request body is not valid JSON")
		return
	}

	destination, err := h.destinations.Update(r.Context(), id, services.UpdateDestinationInput{
		Name:          body.Name,
		Description:   body.Description,
		Subscriptions: body.Subscriptions,
	})
	if err != nil {
		renderDestinationError(w, r, err)
		return
	}
	render.JSON(w, r, newDestinationResource(destination))
}

// DeleteDestination removes a destination by ID and returns 204.
func (h *DestinationHandler) DeleteDestination(w http.ResponseWriter, r *http.Request) {
	id, ok := urlUUID(w, r)
	if !ok {
		return
	}
	if err := h.destinations.Delete(r.Context(), id); err != nil {
		renderDestinationError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ActivateDestination marks a destination as active.
func (h *DestinationHandler) ActivateDestination(w http.ResponseWriter, r *http.Request) {
	id, ok := urlUUID(w, r)
	if !ok {
		return
	}
	destination, err := h.destinations.Activate(r.Context(), id)
	if err != nil {
		renderDestinationError(w, r, err)
		return
	}
	render.JSON(w, r, newDestinationResource(destination))
}

// DeactivateDestination marks a destination as inactive.
func (h *DestinationHandler) DeactivateDestination(w http.ResponseWriter, r *http.Request) {
	id, ok := urlUUID(w, r)
	if !ok {
		return
	}
	destination, err := h.destinations.Deactivate(r.Context(), id)
	if err != nil {
		renderDestinationError(w, r, err)
		return
	}
	render.JSON(w, r, newDestinationResource(destination))
}

// renderDestinationError maps the errors returned by
// services.DestinationService onto HTTP responses.
func renderDestinationError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, services.ErrDestinationNotFound):
		renderError(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrDestinationIDTaken):
		renderError(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, services.ErrDestinationNameRequired),
		errors.Is(err, services.ErrDestinationNameTooLong),
		errors.Is(err, services.ErrDestinationDescriptionTooLong),
		errors.Is(err, services.ErrDestinationTypeInvalid),
		errors.Is(err, services.ErrDestinationConfigRequired),
		errors.Is(err, services.ErrDestinationConfigInvalid),
		errors.Is(err, services.ErrDestinationConfigURLRequired),
		errors.Is(err, services.ErrDestinationAuthTypeInvalid),
		errors.Is(err, services.ErrDestinationHMACSecretRequired),
		errors.Is(err, services.ErrDestinationHMACConfigInvalid),
		errors.Is(err, services.ErrDestinationSubscriptionInvalid),
		errors.Is(err, services.ErrDestinationSubscriptionTooLong),
		errors.Is(err, services.ErrDestinationIDInvalid):
		renderError(w, r, http.StatusUnprocessableEntity, err.Error())
	default:
		renderError(w, r, http.StatusInternalServerError, "internal server error")
	}
}
