package services

import (
	"errors"
	"fmt"

	"github.com/exalynt/turn/paginator"
)

// ErrInvalidPage means a listing's page selection is invalid: a size out of
// range, both after and before, or a cursor that doesn't belong to the
// listing. It wraps turn's error, which says which.
var ErrInvalidPage = errors.New("invalid page")

// listError translates a repository's listing error: a page selection turn
// rejected becomes ErrInvalidPage, and anything else is wrapped with op.
func listError(op string, err error) error {
	if paginator.IsSelectorError(err) {
		return fmt.Errorf("%w: %w", ErrInvalidPage, err)
	}
	return fmt.Errorf("services: %s: %w", op, err)
}
