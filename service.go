//go:generate go run github.com/golang/mock/mockgen -destination=mocks_test.go -package services_test . ResourceService,Reporter,Configurable,RetrierReporter

package services

import (
	"context"
	"io"
)

// Service is the abstraction of what minimum signature a service must have.
//
// ## Why does start is not in the `Service` 
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

// ServerService is the interface the abstracts a blocking service.
type ServerService interface {
	Service
	io.Closer

	Serve() error
}