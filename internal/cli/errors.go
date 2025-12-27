package cli

import "errors"

// ErrTestsFailed is returned when one or more tests fail or error.
var ErrTestsFailed = errors.New("one or more tests failed")
