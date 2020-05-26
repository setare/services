package services

import (
	"context"
	"time"
)

// ServiceRetrier wraps a `Service` in order to provide functionality for retrying in case of its starting process
// fails.
type ServiceRetrier struct {
	service          Service
	tries            int
	timeout          time.Duration
	waitBetweenTries time.Duration
	reporter         RetrierReporter
}

// RetrierBuilder is the helper for building `ServiceRetrier`.
type RetrierBuilder struct {
	tries            int
	timeout          time.Duration
	waitBetweenTries time.Duration
	reporter         RetrierReporter
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
		reporter:         builder.reporter,
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
func (builder *RetrierBuilder) Tries(value int) *RetrierBuilder {
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

// Reporter set the reporter for the `Retrier`.
func (builder *RetrierBuilder) Reporter(value RetrierReporter) *RetrierBuilder {
	builder.reporter = value
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
	for i := 0; i < retrier.tries || retrier.tries == -1; i++ {
		select {
		case <-ctx.Done():
			if retrier.reporter != nil {
				retrier.reporter.AfterGiveUp(retrier.service, i, ctx.Err())
			}
			return ctx.Err()
		default:
			// This make the select not to block.
		}

		if i > 0 && retrier.reporter != nil {
			retrier.reporter.BeforeRetry(retrier.service, i)
		}

		if service, ok := retrier.service.(StartableWithContext); ok {
			err := service.StartWithContext(ctx)
			if err == nil {
				// If started ...
				return nil
			} else if ctx.Err() != nil {
				// If the starting process was cancelled...
				if retrier.reporter != nil {
					retrier.reporter.AfterGiveUp(retrier.service, i, ctx.Err())
				}
				return ctx.Err()
			}
		} else if service, ok := retrier.service.(Startable); ok {
			err := service.Start()
			if err == nil {
				return nil
			}
			if retrier.reporter != nil {
				retrier.reporter.AfterStart(retrier.service, err)
			}
		}
		if retrier.waitBetweenTries > 0 {
			time.Sleep(retrier.waitBetweenTries)
		}
	}
	err := ErrExhaustedAttempts
	retrier.reporter.AfterGiveUp(retrier.service, retrier.tries, err)
	return err
}
