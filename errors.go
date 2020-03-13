package services

import "github.com/pkg/errors"

var (
	// ErrNotStartable is returned when a `Service` that does not implement neither `Startable` or `StartableWithContext`
	// is passed to a method that expects it.
	ErrNotStartable = errors.New("service is not Startable")

	// ErrTimeout is returned when starting a service has timedout.
	ErrTimeout = errors.New("timeout")
	
	// ErrExhaustedAttempts is returned when the `Retrier` reached its retry limit for starting a service.
	ErrExhaustedAttempts = errors.New("exhausted attempts")
)