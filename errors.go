package services

import "github.com/pkg/errors"

var (
	// ErrAlreadyListening is returned by the ServerStarter when it tries to listen a service that is already
	// listeining.
	ErrAlreadyListening = errors.New("service already listening")
)
