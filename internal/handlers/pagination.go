package handlers

import (
	"errors"
	"fmt"
	"net/http"

	turnhttp "github.com/exalynt/turn/http"
	"github.com/exalynt/turn/paginator"
	"github.com/exalynt/turn/paginator/cursor"
)

// cursorQuery reads a listing's page selection from the after, before, and
// size query parameters.
var cursorQuery turnhttp.CursorQuery

// cursorSelector parses the request's page selection. When it's invalid, it
// writes a 400 and returns ok == false, so the caller can simply return.
func cursorSelector(w http.ResponseWriter, r *http.Request) (selector cursor.Selector, ok bool) {
	selector, err := cursorQuery.Parse(r.URL.Query())
	if err != nil {
		renderError(w, r, http.StatusBadRequest, pageErrorMessage(err))
		return cursor.Selector{}, false
	}
	return selector, true
}

// setCursorLinks sets the Link header to the page's first, prev, next, and
// last pages, as far as each exists, keeping the request's other query
// parameters.
func setCursorLinks[T any](w http.ResponseWriter, r *http.Request, page cursor.Page[T]) {
	if links := turnhttp.CursorLinks(page, cursorQuery.URL(r.URL)); len(links) > 0 {
		w.Header().Set("Link", turnhttp.FormatLinks(links...))
	}
}

// pageErrorMessage says what's wrong with a page selection, without echoing
// the cursor back.
func pageErrorMessage(err error) string {
	var sizeErr *paginator.SizeError
	switch {
	case errors.As(err, &sizeErr):
		return fmt.Sprintf("size must be between 1 and %d", sizeErr.MaxSize)
	case errors.Is(err, paginator.ErrInvalidSize):
		return "size must be an integer"
	case errors.Is(err, paginator.ErrInvalidDirection):
		return "after and before can't be used together"
	default:
		return "invalid cursor: follow the Link header's links rather than building cursors"
	}
}
