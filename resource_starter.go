package services

import (
	"context"
	"sync"
)

type ResourceStarter struct {
	resourceServices []ResourceService

	startStopMutex sync.Mutex

	listenCh        chan bool
	listenChMutex   sync.Mutex
	listenWaitGroup sync.WaitGroup

	reporter Reporter

	cancelFuncMutex sync.Mutex
	cancelFunc      func()
}

type ManagerOption = func(*ResourceStarter)

// NewManager creates a new instance of ResourceStarter.
//
// If a listener is not defined, it will create one based on DefaultSignals.
func NewManager(opts ...ManagerOption) *ResourceStarter {
	manager := &ResourceStarter{
		resourceServices: make([]ResourceService, 0),
	}
	for _, opt := range opts {
		opt(manager)
	}

	return manager
}

// WithReporter is a ManagerOption that will set the signal listener instance of a ResourceStarter.
func WithReporter(reporter Reporter) ManagerOption {
	return func(manager *ResourceStarter) {
		manager.reporter = reporter
	}
}

// Start will initialize the starting process. Once it is finished, nil is returned in case of success. Otherwise,
// an error is returned.
func (starter *ResourceStarter) Start(ctx context.Context, services ...ResourceService) (errResult error) {
	defer func() {
		r := recover()

		// If the return was an error or anything panicked, we should stop the resourceServices in the reverse order.
		if errResult != nil || r != nil {
			starter.Stop(ctx)
		}
		if r != nil {
			panic(r) // re-panic ...
		}
	}()

	// Starting resourceServices is exclusive.
	starter.startStopMutex.Lock()

	ctx, cancelFunc := context.WithCancel(ctx)
	starter.cancelFuncMutex.Lock()
	starter.cancelFunc = cancelFunc
	starter.cancelFuncMutex.Unlock()

	defer func() {
		cancelFunc()
		starter.startStopMutex.Unlock()
	}()

	// Go through all resourceServices starting one by one.
	for _, service := range services {

		// Check if the starting process was cancelled.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Not cancelled ...
		}

		// If the service is configurable
		if srv, ok := service.(Configurable); ok {
			if starter.reporter != nil {
				starter.reporter.BeforeLoad(ctx, srv)
			}
			errResult = srv.Load(ctx)
			if starter.reporter != nil {
				starter.reporter.AfterLoad(ctx, srv, errResult)
			}
			if errResult != nil {
				return
			}
		}

		// Loading configuration can take a long time. Then, check if the starting process was cancelled again.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Not cancelled ...
		}

		// PRIORITY 2: If the service is a ResourceService, use it.
		if starter.reporter != nil {
			starter.reporter.BeforeStart(ctx, service)
		}
		errResult = service.Start(ctx)
		if starter.reporter != nil {
			starter.reporter.AfterStart(ctx, service, errResult)
		}
		if errResult != nil {
			return
		}
		starter.resourceServices = append(starter.resourceServices, service)
	}
	return nil
}

// Stop will go through all started resourceServices, in the opposite order they were started, stopping one by one. If any,
// failure is detected, the function will stop leaving some started resourceServices.
func (starter *ResourceStarter) Stop(ctx context.Context) (errResult error) {
	starter.cancelFuncMutex.Lock()
	if starter.cancelFunc != nil {
		starter.cancelFunc()
	}
	starter.cancelFuncMutex.Unlock()

	starter.listenChMutex.Lock()
	if starter.listenCh != nil {
		// Wait the listen to be finished.
		<-starter.listenCh
	}
	starter.listenChMutex.Unlock()

	// stopping the resourceServices is exclusive.
	starter.startStopMutex.Lock()
	defer func() {
		starter.startStopMutex.Unlock()
	}()

	ctx, cancelFunc := context.WithCancel(ctx)

	defer func() {
		cancelFunc()
	}()

	for i := len(starter.resourceServices) - 1; i >= 0; i-- {
		service := starter.resourceServices[i]
		if starter.reporter != nil {
			starter.reporter.BeforeStop(ctx, service)
		}
		err := service.Stop(ctx)
		if starter.reporter != nil {
			starter.reporter.AfterStop(ctx, service, err)
		}
		if err != nil {
			return err
		}
		starter.resourceServices = starter.resourceServices[:len(starter.resourceServices)-1]
	}
	return nil
}

// WithReporter sets the reporter for this ResourceStarter instance, returning it afterwards.
func (starter *ResourceStarter) WithReporter(reporter Reporter) *ResourceStarter {
	starter.reporter = reporter
	return starter
}
