package services

import "github.com/pkg/errors"

var (
	// ErrTimeout is returned when starting a service has timedout.
	ErrTimeout = errors.New("timeout")
	
	// ErrExhaustedAttempts is returned when the `Retrier` reached its retry limit for starting a service.
	ErrExhaustedAttempts = errors.New("exhausted attempts")
)