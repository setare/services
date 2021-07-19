package services

import "github.com/setare/go-errors"

const (
	// ErrStartCancelledBySignal is returned when Runner.Run receives a shutdown signal while starting the list of
	// Resource and Server.
	ErrStartCancelledBySignal = errors.Error("start cancelled by signal")
)
