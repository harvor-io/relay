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

// SubscriptionHandler serves the /destinations/{id}/subscriptions resource.
type SubscriptionHandler struct {
	subscriptions services.SubscriptionService
}

// NewSubscriptionHandler constructs a SubscriptionHandler backed by
// subscriptions.
func NewSubscriptionHandler(subscriptions services.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{subscriptions: subscriptions}
}

// RegisterRoutes mounts the handler's routes on r. It shares the "id" path
// parameter with DestinationHandler's "/destinations/{id}" route.
func (h *SubscriptionHandler) RegisterRoutes(r chi.Router) {
	r.Route("/destinations/{id}/subscriptions", func(r chi.Router) {
		r.Get("/", h.ListSubscriptions)
		r.Post("/", h.CreateSubscription)
		r.Route("/{subscriptionID}", func(r chi.Router) {
			r.Get("/", h.GetSubscription)
			r.Patch("/", h.UpdateSubscription)
			r.Delete("/", h.DeleteSubscription)
		})
	})
}

// subscriptionResource is the JSON representation of a subscription. It is
// an explicit projection of models.Subscription.
type subscriptionResource struct {
	ID            string    `json:"id"`
	DestinationID string    `json:"destination_id"`
	SourceID      string    `json:"source_id"`
	Topics        []string  `json:"topics"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func newSubscriptionResource(s *models.Subscription) subscriptionResource {
	topics := make([]string, 0, len(s.Topics))
	for _, topic := range s.Topics {
		topics = append(topics, topic.String())
	}
	return subscriptionResource{
		ID:            s.ID.String(),
		DestinationID: s.DestinationID.String(),
		SourceID:      s.SourceID.String(),
		Topics:        topics,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

func newSubscriptionResourceList(subscriptions []models.Subscription) []subscriptionResource {
	resources := make([]subscriptionResource, 0, len(subscriptions))
	for i := range subscriptions {
		resources = append(resources, newSubscriptionResource(&subscriptions[i]))
	}
	return resources
}

// createSubscriptionRequest is the accepted body for
// POST /destinations/{id}/subscriptions.
type createSubscriptionRequest struct {
	ID       string   `json:"id"`
	SourceID string   `json:"source_id"`
	Topics   []string `json:"topics"`
}

// updateSubscriptionRequest is the accepted body for
// PATCH /destinations/{id}/subscriptions/{subscriptionID}.
type updateSubscriptionRequest struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
}

// ListSubscriptions returns a page of a destination's subscriptions as a
// JSON array, with the other pages in the Link header.
func (h *SubscriptionHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	destinationID, ok := urlUUID(w, r)
	if !ok {
		return
	}
	selector, ok := cursorSelector(w, r)
	if !ok {
		return
	}
	page, err := h.subscriptions.List(r.Context(), destinationID, selector)
	if err != nil {
		renderSubscriptionError(w, r, err)
		return
	}
	setCursorLinks(w, r, page)
	render.JSON(w, r, newSubscriptionResourceList(page.Items))
}

// CreateSubscription subscribes a destination to a source's topics and
// returns the subscription with a 201.
func (h *SubscriptionHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	destinationID, ok := urlUUID(w, r)
	if !ok {
		return
	}

	var body createSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		renderError(w, r, http.StatusBadRequest, "request body is not valid JSON")
		return
	}

	subscription, err := h.subscriptions.Subscribe(r.Context(), destinationID, services.SubscribeInput{
		ID:       body.ID,
		SourceID: body.SourceID,
		Topics:   body.Topics,
	})
	if err != nil {
		renderSubscriptionError(w, r, err)
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, newSubscriptionResource(subscription))
}

// GetSubscription returns a single subscription by ID.
func (h *SubscriptionHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	destinationID, ok := urlUUID(w, r)
	if !ok {
		return
	}
	subscriptionID, ok := urlUUIDParam(w, r, "subscriptionID")
	if !ok {
		return
	}
	subscription, err := h.subscriptions.Get(r.Context(), destinationID, subscriptionID)
	if err != nil {
		renderSubscriptionError(w, r, err)
		return
	}
	render.JSON(w, r, newSubscriptionResource(subscription))
}

// UpdateSubscription adds topics to and removes topics from a
// subscription.
func (h *SubscriptionHandler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	destinationID, ok := urlUUID(w, r)
	if !ok {
		return
	}
	subscriptionID, ok := urlUUIDParam(w, r, "subscriptionID")
	if !ok {
		return
	}

	var body updateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		renderError(w, r, http.StatusBadRequest, "request body is not valid JSON")
		return
	}

	subscription, err := h.subscriptions.ChangeTopics(r.Context(), destinationID, subscriptionID, services.ChangeTopicsInput{
		Add:    body.Add,
		Remove: body.Remove,
	})
	if err != nil {
		renderSubscriptionError(w, r, err)
		return
	}
	render.JSON(w, r, newSubscriptionResource(subscription))
}

// DeleteSubscription unsubscribes a destination from a source and returns
// 204.
func (h *SubscriptionHandler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	destinationID, ok := urlUUID(w, r)
	if !ok {
		return
	}
	subscriptionID, ok := urlUUIDParam(w, r, "subscriptionID")
	if !ok {
		return
	}
	if err := h.subscriptions.Unsubscribe(r.Context(), destinationID, subscriptionID); err != nil {
		renderSubscriptionError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// renderSubscriptionError maps the errors returned by
// services.SubscriptionService onto HTTP responses.
func renderSubscriptionError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, services.ErrDestinationNotFound), errors.Is(err, services.ErrSubscriptionNotFound):
		renderError(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrSubscriptionExists), errors.Is(err, services.ErrSubscriptionIDTaken):
		renderError(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, services.ErrSubscriptionSourceInvalid),
		errors.Is(err, services.ErrSubscriptionIDInvalid),
		errors.Is(err, services.ErrSubscriptionTopicsRequired),
		errors.Is(err, services.ErrSubscriptionTopicInvalid),
		errors.Is(err, services.ErrSubscriptionLastTopic):
		renderError(w, r, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, services.ErrInvalidPage):
		renderError(w, r, http.StatusBadRequest, pageErrorMessage(err))
	default:
		renderInternalError(w, r, err)
	}
}
