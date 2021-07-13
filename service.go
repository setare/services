package services

import (
	"context"
	"os"
)

var (
	// DefaultSignals is the list of signals that the ResourceStarter will listen if no listener is specified.
	DefaultSignals = []os.Signal{os.Interrupt}
)

// Service is the abstraction of what minimum signature a service must have.
type Service interface {
	// Name will return a human identifiable name for this service. Ex: Postgresql Connection.
	Name() string
}

// ResourceService is the interface that must be implemented for resourceServices that its start is NOT cancellable.
type ResourceService interface {
	Service

	// Start will start the service in a blocking way.
	//
	// If the service is successfully started, `nil` should be returned. Otherwise, an error must be returned.
	Start(ctx context.Context) error

	// Stop will stop this service.
	//
	// For most implementations it will be blocking and should return only when the service finishes stopping.
	//
	// If the service is successfully stopped, `nil` should be returned. Otherwise, an error must be returned.
	Stop(ctx context.Context) error
}

// ServerService is the interface that must be implemented for resourceServices that its start is NOT cancellable.
type ServerService interface {
	Service

	// Listen will start the server service and will block until the service is closed.
	//
	// If the services is already listining, this should return an error ErrAlreadyListening.
	Listen(ctx context.Context) error

	// Close will stop this service.
	//
	// If the services has not started, or is already stopped, this should do nothing and just return nil.
	Close(ctx context.Context) error
}
