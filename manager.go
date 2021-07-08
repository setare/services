package services

import (
	"context"
	"os"
	"sync"

	signals "github.com/setare/go-os-signals"
)

var (
	// DefaultSignals is the list of signals that the Manager will listen if no listener is specified.
	DefaultSignals = []os.Signal{os.Interrupt}
)

// Manager
type Manager struct {
	resourceServices []ResourceService
	signalListener   signals.Listener
	startStopMutex   sync.Mutex
	cancelFuncMutex  sync.Mutex
	cancelFunc       func()
	reporter         Reporter
}

type ManagerOption = func(*Manager)

// NewManager creates a new instance of Manager.
//
// If a listener is not defined, it will create one based on DefaultSignals.
func NewManager(opts ...ManagerOption) *Manager {
	manager := &Manager{
		resourceServices: make([]ResourceService, 0),
	}
	for _, opt := range opts {
		opt(manager)
	}
	if manager.signalListener == nil {
		manager.signalListener = signals.NewListener(DefaultSignals...)
	}

	go manager.startListener()

	return manager
}

// WithListener is a ManagerOption that will set the signal listener instance of a Manager.
func WithListener(listener signals.Listener) ManagerOption {
	return func(manager *Manager) {
		manager.signalListener = listener
	}
}

// WithReporter is a ManagerOption that will set the signal listener instance of a Manager.
func WithReporter(reporter Reporter) ManagerOption {
	return func(manager *Manager) {
		manager.reporter = reporter
	}
}

// Start will initialize the starting process. Once it is finished, nil is returned in case of success. Otherwise,
// an error is returned.
func (manager *Manager) Start(ctx context.Context, services ...ResourceService) (errResult error) {
	defer func() {
		r := recover()

		// If the return was an error or anything panicked, we should stop the resourceServices in the reverse order.
		if errResult != nil || r != nil {
			manager.Stop(ctx)
		}
		if r != nil {
			panic(r) // re-panic ...
		}
	}()

	// Starting resourceServices is exclusive.
	manager.startStopMutex.Lock()

	ctx, cancelFunc := context.WithCancel(ctx)
	manager.cancelFuncMutex.Lock()
	manager.cancelFunc = cancelFunc
	manager.cancelFuncMutex.Unlock()

	defer func() {
		cancelFunc()
		manager.startStopMutex.Unlock()
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
			if manager.reporter != nil {
				manager.reporter.BeforeLoad(ctx, srv)
			}
			errResult = srv.Load(ctx)
			if manager.reporter != nil {
				manager.reporter.AfterLoad(ctx, srv, errResult)
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
		if manager.reporter != nil {
			manager.reporter.BeforeStart(ctx, service)
		}
		errResult = service.Start(ctx)
		if manager.reporter != nil {
			manager.reporter.AfterStart(ctx, service, errResult)
		}
		if errResult != nil {
			return
		}
		manager.resourceServices = append(manager.resourceServices, service)
	}
	return nil
}

// Stop will go through all started resourceServices, in the opposite order they were started, stopping one by one. If any,
// failure is detected, the function will stop leaving some started resourceServices.
func (manager *Manager) Stop(ctx context.Context) (errResult error) {
	manager.cancelFuncMutex.Lock()
	if manager.cancelFunc != nil {
		manager.cancelFunc()
	}
	manager.cancelFuncMutex.Unlock()

	// stopping the resourceServices is exclusive.
	manager.startStopMutex.Lock()
	defer func() {
		manager.startStopMutex.Unlock()
	}()

	ctx, cancelFunc := context.WithCancel(ctx)

	defer func() {
		cancelFunc()
	}()

	for i := len(manager.resourceServices) - 1; i >= 0; i-- {
		service := manager.resourceServices[i]
		if manager.reporter != nil {
			manager.reporter.BeforeStop(ctx, service)
		}
		err := service.Stop(ctx)
		if manager.reporter != nil {
			manager.reporter.AfterStop(ctx, service, err)
		}
		if err != nil {
			return err
		}
		manager.resourceServices = manager.resourceServices[:len(manager.resourceServices)-1]
	}
	return nil
}

func (manager *Manager) startListener() error {
	select {
	case osSig, ok := <-manager.signalListener.Receive():
		if !ok {
			// That was closed, so stop was already called.
			return nil
		}
		manager.signalListener.Stop()
		if manager.reporter != nil {
			manager.reporter.SignalReceived(osSig)
		}
		return manager.Stop(context.Background())
	}
}

// WithReporter sets the reporter for this Manager instance, returning it afterwards.
func (manager *Manager) WithReporter(reporter Reporter) *Manager {
	manager.reporter = reporter
	return manager
}
