package services

import (
	"context"
	"time"
)

// ServiceRetrier wraps a `Service` in order to provide functionality for retrying in case of its starting process
// fails.
type ServiceRetrier struct {
	service          Service
	tries            uint
	timeout          time.Duration
	waitBetweenTries time.Duration
}

// RetrierBuilder is the helper for building `ServiceRetrier`.
type RetrierBuilder struct {
	tries            uint
	timeout          time.Duration
	waitBetweenTries time.Duration
}

// Retrier returns a new `RetrierBuilder` instance.
func Retrier() *RetrierBuilder {
	return &RetrierBuilder{
		tries: 3,
	}
}

// Build creates a new `ServiceRetrier` with
func (builder *RetrierBuilder) Build(service Service) Service {
	sr := &ServiceRetrier{
		service:          service,
		timeout:          builder.timeout,
		tries:            builder.tries,
		waitBetweenTries: builder.waitBetweenTries,
	}
	if configurable, ok := service.(Configurable); ok {
		return struct {
			*ServiceRetrier
			Configurable
		}{
			sr,
			configurable,
		}
	}
	return sr
}

// Tries sets the tries for the `Retrier`.
func (builder *RetrierBuilder) Tries(value uint) *RetrierBuilder {
	builder.tries = value
	return builder
}

// Timeout set the timeout for the `Retrier`.
func (builder *RetrierBuilder) Timeout(value time.Duration) *RetrierBuilder {
	builder.timeout = value
	return builder
}

// WaitBetweenTries set the timeout for the `Retrier`.
func (builder *RetrierBuilder) WaitBetweenTries(value time.Duration) *RetrierBuilder {
	builder.waitBetweenTries = value
	return builder
}

// Name will return a human identifiable name for this service. Ex: Postgresql Connection.
func (retrier *ServiceRetrier) Name() string {
	return retrier.service.Name()
}

// Stop will stop this service.
//
// For most implementations it will be blocking and should return only when the service finishes stopping.
//
// If the service is successfully stopped, `nil` should be returned. Otherwise, an error must be returned.
func (retrier *ServiceRetrier) Stop() error {
	return retrier.service.Stop()
}

// StartWithContext implements the logic of starting a service. If it fails, it should use the configuration to retry.
func (retrier *ServiceRetrier) StartWithContext(ctx context.Context) error {
	if retrier.timeout > 0 {
		ctx2, cancel := context.WithTimeout(ctx, retrier.timeout)
		ctx = ctx2
		defer cancel()
	}
	for i := uint(0); i < retrier.tries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// This make the select not to block.
		}

		if service, ok := retrier.service.(StartableWithContext); ok {
			err := service.StartWithContext(ctx)
			if err == nil {
				// If started ...
				return nil
			} else if ctx.Err() != nil {
				// If the starting process was cancelled...
				return ctx.Err()
			}
		} else if service, ok := retrier.service.(Startable); ok {
			err := service.Start()
			if err == nil {
				return nil
			}
		}
		if retrier.waitBetweenTries > 0 {
			time.Sleep(retrier.waitBetweenTries)
		}
	}
	return ErrExhaustedAttempts
}
