// Package services holds the application's use-case layer: the business rules
// that sit between the HTTP handlers and the persistence interfaces. A service
// validates and normalises input, coordinates one or more repositories, and
// returns domain models or one of its own well-defined errors.
//
// Services never speak HTTP. They translate lower-level failures (such as
// repositories.ErrNotFound) into errors that describe the use case in domain
// terms, so that each caller — an HTTP handler today, something else tomorrow —
// can decide how to present them.
package services
