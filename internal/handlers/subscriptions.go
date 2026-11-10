package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/services"
)

// SubscriptionHandler serves subscriptions, the links between a destination
// and the topics it wants: all of them at /destinations/{id}/subscriptions,
// and each at /destinations/{id}/subscriptions/{topic}.
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
		r.Put("/", h.ReplaceSubscriptions)
		r.Route("/{topic}", func(r chi.Router) {
			r.Get("/", h.GetSubscription)
			r.Put("/", h.PutSubscription)
			r.Delete("/", h.DeleteSubscription)
		})
	})
}

// subscriptionResource is the JSON representation of a subscription. It is
// an explicit projection of models.Subscription.
type subscriptionResource struct {
	DestinationID string    `json:"destination_id"`
	Topic         string    `json:"topic"`
	CreatedAt     time.Time `json:"created_at"`
}

func newSubscriptionResource(s *models.Subscription) subscriptionResource {
	return subscriptionResource{
		DestinationID: s.DestinationID.String(),
		Topic:         s.Topic.String(),
		CreatedAt:     s.CreatedAt,
	}
}

func newSubscriptionResourceList(subscriptions []models.Subscription) []subscriptionResource {
	resources := make([]subscriptionResource, 0, len(subscriptions))
	for i := range subscriptions {
		resources = append(resources, newSubscriptionResource(&subscriptions[i]))
	}
	return resources
}

// replaceSubscriptionsRequest is the accepted body for
// PUT /destinations/{id}/subscriptions.
type replaceSubscriptionsRequest struct {
	Topics []string `json:"topics"`
}

// urlTopic returns the {topic} path parameter, decoded, since a client may
// percent-encode a pattern's "*".
func urlTopic(w http.ResponseWriter, r *http.Request) (string, bool) {
	topic, err := url.PathUnescape(chi.URLParam(r, "topic"))
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "topic is not a valid path segment")
		return "", false
	}
	return topic, true
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

// ReplaceSubscriptions makes a destination's subscriptions exactly the
// topics in the body and returns 204. Sending the same body again changes
// nothing.
func (h *SubscriptionHandler) ReplaceSubscriptions(w http.ResponseWriter, r *http.Request) {
	destinationID, ok := urlUUID(w, r)
	if !ok {
		return
	}

	var body replaceSubscriptionsRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		renderError(w, r, http.StatusBadRequest, "request body is not valid JSON")
		return
	}
	if body.Topics == nil {
		renderError(w, r, http.StatusUnprocessableEntity, "topics is required; send an empty list to unsubscribe from every topic")
		return
	}

	if err := h.subscriptions.Replace(r.Context(), destinationID, body.Topics); err != nil {
		renderSubscriptionError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetSubscription returns a destination's subscription to a topic.
func (h *SubscriptionHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	destinationID, ok := urlUUID(w, r)
	if !ok {
		return
	}
	topic, ok := urlTopic(w, r)
	if !ok {
		return
	}
	subscription, err := h.subscriptions.Get(r.Context(), destinationID, topic)
	if err != nil {
		renderSubscriptionError(w, r, err)
		return
	}
	render.JSON(w, r, newSubscriptionResource(subscription))
}

// PutSubscription subscribes a destination to a topic. It returns 201 when
// the subscription is new and 200 when it already existed, so a retry is
// safe.
func (h *SubscriptionHandler) PutSubscription(w http.ResponseWriter, r *http.Request) {
	destinationID, ok := urlUUID(w, r)
	if !ok {
		return
	}
	topic, ok := urlTopic(w, r)
	if !ok {
		return
	}
	subscription, created, err := h.subscriptions.Subscribe(r.Context(), destinationID, topic)
	if err != nil {
		renderSubscriptionError(w, r, err)
		return
	}
	if created {
		render.Status(r, http.StatusCreated)
	}
	render.JSON(w, r, newSubscriptionResource(subscription))
}

// DeleteSubscription unsubscribes a destination from a topic and returns
// 204, whether or not it was subscribed, so a retry is safe.
func (h *SubscriptionHandler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	destinationID, ok := urlUUID(w, r)
	if !ok {
		return
	}
	topic, ok := urlTopic(w, r)
	if !ok {
		return
	}
	if err := h.subscriptions.Unsubscribe(r.Context(), destinationID, topic); err != nil {
		renderSubscriptionError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// renderSubscriptionError maps the errors returned by
// services.SubscriptionService onto HTTP responses.
func renderSubscriptionError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, services.ErrDestinationNotFound),
		errors.Is(err, services.ErrSubscriptionNotFound):
		renderError(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrSubscriptionTopicInvalid):
		renderError(w, r, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, services.ErrInvalidPage):
		renderError(w, r, http.StatusBadRequest, pageErrorMessage(err))
	default:
		renderInternalError(w, r, err)
	}
}
