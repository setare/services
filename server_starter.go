package services

import (
	"context"
	"sync"

	signals "github.com/jamillosantos/go-os-signals"
)

type errPair struct {
	idx int
	err error
}

// ServerStarter is capable of starting multiple ServerService in one call
type ServerStarter struct {
	listener  signals.Listener
	stopMutex sync.Mutex
	stopCh    chan struct{}
}

// NewServerStarter returns a new instance of ServerStarter with a listener set.
func NewServerStarter(listener signals.Listener) ServerStarter {
	return ServerStarter{
		listener: listener,
	}
}

// Listen will start all services by calling its Listen method.
func (starter *ServerStarter) Listen(ctx context.Context, services ...ServerService) []error {
	if starter.listener == nil {
		starter.listener = signals.NewListener(DefaultSignals...)
	}

	starter.stopMutex.Lock()
	starter.stopCh = make(chan struct{})
	starter.stopMutex.Unlock()

	var wg sync.WaitGroup
	errs := make(chan errPair, len(services))

	wg.Add(len(services))
	for idx, service := range services {
		go func(idx int, service ServerService) {
			ctx, cancelFunc := context.WithCancel(ctx)
			defer cancelFunc()
			defer wg.Done()

			// Tries to load the file if needed.
			if c, ok := service.(Configurable); ok {
				err := c.Load(ctx)
				if err != nil {
					errs <- errPair{idx, err}
					return
				}
			}

			err := service.Listen(ctx)
			errs <- errPair{idx, err}
		}(idx, service)
	}
	errResult := make([]error, len(services))
	hasError := false

	select {
	case <-starter.stopCh:
		// Close was called...
	case <-starter.listener.Receive():
		// Received a signal
	case ep := <-errs:
		// A server errored
		hasError = ep.err != nil
		errResult[ep.idx] = ep.err
	}

	for i := len(services) - 1; i >= 0; i-- {
		services[i].Close(ctx)
	}

	wg.Wait() // Wait the services to stop.

	close(errs) // All errs are done and the channel can be closed...

	for ep := range errs {
		//
		if ep.err != nil {
			hasError = true
			errResult[ep.idx] = ep.err
		}
	}
	if hasError {
		return errResult
	}
	return nil
}

func (starter *ServerStarter) Close(_ context.Context) error {
	starter.stopMutex.Lock()
	if starter.stopCh != nil {
		close(starter.stopCh)
	}
	starter.stopMutex.Unlock()

	return nil
}
