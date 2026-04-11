package mongo

import "errors"

// ErrUninitialized is returned when a DB handle or wrapper was not constructed
// (e.g. nil Database or Collection with no underlying driver value).
var ErrUninitialized = errors.New("mongo: uninitialized handle")
