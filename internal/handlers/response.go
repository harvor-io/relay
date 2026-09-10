package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/gofrs/uuid/v5"
)

// errorResponse is the body returned for every non-2xx API response. It matches
// the Error schema in the OpenAPI spec: a single human-readable message and
// nothing else.
type errorResponse struct {
	Error string `json:"error"`
}

// renderError writes status with an errorResponse body carrying message.
func renderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	render.Status(r, status)
	render.JSON(w, r, errorResponse{Error: message})
}

// urlUUID parses the {id} path parameter as a UUID. On failure it writes a 400
// response and returns ok == false, so the caller can simply return.
func urlUUID(w http.ResponseWriter, r *http.Request) (id uuid.UUID, ok bool) {
	id, err := uuid.FromString(chi.URLParam(r, "id"))
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "invalid id: must be a UUID")
		return uuid.Nil, false
	}
	return id, true
}
