package services

import "context"

// Service is the abstraction of what minimum signature a service must have.
//
// ## Why does start is not in the `Service` 
type Service interface {
	// Name will return a human identifiable name for this service. Ex: Postgresql Connection.
	Name() string

	// Stop will stop this service.
	//
	// For most implementations it will be blocking and should return only when the service finishes stopping.
	//
	// If the service is successfully stopped, `nil` should be returned. Otherwise, an error must be returned.
	Stop() error
}

// Startable is the interface that must be implemented for services that its start is NOT cancellable.
type Startable interface {
	// Start will start the service in a blocking way.
	//
	// If the service is successfully started, `nil` should be returned. Otherwise, an error must be returned.
	Start() error
}

// StartableWithContext is the interface that must be implemented for services that its start is cancellable.
type StartableWithContext interface {
	// StartWithContext start the service in a blocking way. This is cancellable, so the context received can be
	// cancelled at any moment. If your start implementation is not cancellable, you should implement `Startable`
	// instead.
	//
	// If the service is successfully started, `nil` should be returned. Otherwise, an error must be returned.
	StartWithContext(context.Context) error
}
