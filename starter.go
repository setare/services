package services

import (
	"context"
	"os"
	"reflect"
	"sync"

	"github.com/pkg/errors"
	signals "github.com/setare/go-os-signals"
)

const (
	starterStateNone = iota
	starterStateStarting
	starterStateStopping
)

// Starter receives a list of `Service` (that should implement `Startable` or `StartableWithContext`) and is responsible
// for starting them when `Start` or `StartWithContext` is called.
type Starter struct {
	services             []Service
	signalListener       signals.Listener
	ctxMutex             sync.Mutex
	startingMutex        sync.Mutex
	stoppingMutex        sync.Mutex
	startingContext      context.Context
	servicesStarted      []Service
	ctx                  context.Context
	cancelFunc           context.CancelFunc
	startingCh           chan bool
	reporter             Reporter
	givenStartingChannel chan bool
}

// NewStarter receives a list of services and returns a new instance of `Starter` configured and ready to be started.
func NewStarter(services ...Service) *Starter {
	return &Starter{
		services:        services,
		servicesStarted: make([]Service, 0),
		signalListener:  signals.NewListener(os.Interrupt),
	}
}

// Start will initialize the starting process. Once it is finished, nil is returned in case of success. Otherwise,
// an error is returned.
//
// This method uses `StartWithContext`.
func (s *Starter) Start() error {
	s.ctxMutex.Lock()
	s.ctx, s.cancelFunc = context.WithCancel(context.Background())
	s.ctxMutex.Unlock()
	return s.startWithContext(s.ctx)
}

// startWithContext will start all process configured from top to bottom. If everything goes well, `nil` will be
// returned, otherwise, an error.
//
// If any error happens, all started processes will be stopped from bottom up.
func (s *Starter) startWithContext(ctx context.Context) (err error) {
	s.startingMutex.Lock()
	go func() {
		// In case the sigterm comes before finishing starting.
		_, ok := <-s.signalListener.Receive()
		if ok {
			s.cancelFunc() // Cancel the initialization.
		}
	}()

	s.startingCh = make(chan bool)
	defer func() {
		// Stops listening for interrupt signals
		s.signalListener.Stop()

		// Signals the start has been finished.
		close(s.startingCh)

		// Unlocks
		s.startingMutex.Unlock()

		if s.givenStartingChannel != nil {
			close(s.givenStartingChannel)
			s.givenStartingChannel = nil
		}
	}()

	defer func() {
		r := recover()
		// If the return was an error or anything panicked, we should stop the
		// services in the reverse order.
		if err != nil || r != nil {
			for i := len(s.servicesStarted) - 1; i >= 0; i-- {
				service := s.servicesStarted[i]
				if s.reporter != nil {
					s.reporter.BeforeStop(service)
				}
				err := s.servicesStarted[i].Stop()
				if s.reporter != nil {
					s.reporter.AfterStop(service, err)
				}
				s.servicesStarted = s.servicesStarted[:i]
			}
		}
		if r != nil {
			panic(err)
		}
	}()

	// Go through all services starting one by one.
	for _, service := range s.services {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
			// This make the select not to block.
		}

		if srv, ok := service.(Configurable); ok {
			if s.reporter != nil {
				s.reporter.BeforeLoad(srv)
			}
			err = srv.Load()
			if s.reporter != nil {
				s.reporter.AfterLoad(srv, err)
			}
			if err != nil {
				return err
			}
		}

		if srv, ok := service.(StartableWithContext); ok {
			// PRIORITY 1: If the service is a StartableWithContext, use it.
			if s.reporter != nil {
				s.reporter.BeforeStart(service)
			}
			err = srv.StartWithContext(ctx)
			if s.reporter != nil {
				s.reporter.AfterStart(service, err)
			}
			if err != nil {
				return
			}
			s.servicesStarted = append(s.servicesStarted, service)
			continue
		} else if srv, ok := service.(Startable); ok {
			// PRIORITY 2: If the service is a Startable, use it.
			if s.reporter != nil {
				s.reporter.BeforeStart(service)
			}
			err = srv.Start()
			if s.reporter != nil {
				s.reporter.AfterStart(service, err)
			}
			if err != nil {
				return
			}
			s.servicesStarted = append(s.servicesStarted, service)
			continue
		}
		// The service does not implement neither Startable or StartablWithContext.
		err = errors.Wrap(ErrNotStartable, reflect.TypeOf(service).String())
		return
	}
	return nil
}

// Stop will go through all started services, in the opposite order they were started, stopping one by one. If any,
// failure is detected, the function will stop leaving some started services.
func (s *Starter) Stop() error {
	// Prevents two stops running at the sabe time
	s.stoppingMutex.Lock()
	defer s.stoppingMutex.Unlock()

	s.ctxMutex.Lock()
	if s.cancelFunc == nil {
		// Already stopped
		s.ctxMutex.Unlock()
		return nil
	}

	s.cancelFunc() // Tries to cancel the starting proccess.
	s.ctxMutex.Unlock()

	if s.reporter != nil {
		s.reporter.BeforeStop(nil)
	}

	defer func() {
		s.ctxMutex.Lock()
		s.ctx = nil
		s.cancelFunc = nil
		s.ctxMutex.Unlock()

		if s.reporter != nil {
			s.reporter.AfterStop(nil, nil)
		}
	}()

	<-s.startingCh // Wait the starting process to be finished.
	for i := len(s.servicesStarted) - 1; i >= 0; i-- {
		service := s.servicesStarted[i]
		if s.reporter != nil {
			s.reporter.BeforeStop(service)
		}
		err := service.Stop()
		if s.reporter != nil {
			s.reporter.AfterStop(service, err)
		}
		if err != nil {
			return err
		}
		s.servicesStarted = s.servicesStarted[:len(s.servicesStarted)-1]
	}
	return nil
}

func (s *Starter) ListenSignals() error {
	listener := signals.NewListener(os.Interrupt)
	select {
	case osSig := <-listener.Receive():
		listener.Stop()
		if s.reporter != nil {
			s.reporter.SignalReceived(osSig)
		}
		return s.Stop()
	case <-s.ctx.Done():
		listener.Stop()
		errCtx := s.ctx.Err()
		err := s.Stop()
		if err != nil {
			return err
		}
		return errCtx
	}
}

// WithReporter sets the reporter for this Starter instance, returning it afterwards.
func (s *Starter) WithReporter(reporter Reporter) *Starter {
	s.reporter = reporter
	return s
}

// WithStartingChannel sets the channel that the Starter will close after
// finishing the starting process.
func (s *Starter) WithStartingChannel(ch chan bool) *Starter {
	s.givenStartingChannel = ch
	return s
}
