// Package repositories defines the persistence boundary for the domain models:
// one interface per aggregate, describing how records are stored and retrieved
// without committing to a particular database. Concrete implementations live in
// sub-packages (currently only sqlite).
package repositories

import "errors"

// ErrNotFound is returned by a repository when a lookup, update, or delete
// targets a record that does not exist.
var ErrNotFound = errors.New("repositories: record not found")
