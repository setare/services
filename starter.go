package services

import (
	"reflect"
	"sync"

	"context"

	"github.com/pkg/errors"
)

const (
	starterStateNone = iota
	starterStateStarting
	starterStateStopping
)

// Starter receives a list of `Service` (that should implement `Startable` or `StartableWithContext`) and is responsible
// for starting them when `Start` or `StartWithContext` is called.
type Starter struct {
	services        []Service
	startingMutex   sync.Mutex
	startingContext context.Context
	servicesStarted []Service
	state           int
	ctx             context.Context
	cancelFunc      context.CancelFunc
	startingCh      chan bool
	reporter        Reporter
}

// NewStarter receives a list of services and returns a new instance of `Starter` configured and ready to be started.
func NewStarter(services ...Service) *Starter {
	return &Starter{
		services:        services,
		servicesStarted: make([]Service, 0),
	}
}

// Start will initialize the starting process. Once it is finished, nil is returned in case of success. Otherwise,
// an error is returned.
//
// This method uses `StartWithContext`.
func (s *Starter) Start() error {
	s.ctx, s.cancelFunc = context.WithCancel(context.Background())
	defer s.cancelFunc()
	return s.startWithContext(s.ctx)
}

// startWithContext will start all process configured from top to bottom. If everything goes well, `nil` will be
// returned, otherwise, an error.
//
// If any error happens, all started processes will be stopped from bottom up.
func (s *Starter) startWithContext(ctx context.Context) (err error) {
	s.startingMutex.Lock()
	s.startingCh = make(chan bool)
	defer func() {
		s.state = starterStateNone
		close(s.startingCh)
		s.startingMutex.Unlock()
		s.ctx = nil
		s.cancelFunc = nil
	}()

	s.state = starterStateStarting

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
				return
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
	return
}

// Stop will go through all started services, in the opposite order they were started, stopping one by one. If any,
// failure is detected, the function will stop leaving some started services.
func (s *Starter) Stop() error {
	if s.cancelFunc != nil {
		s.cancelFunc() // Tries to cancel the starting proccess.
	}
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

// WithReporter sets the reporter for this Starter instance, returning it afterwards.
func (s *Starter) WithReporter(reporter Reporter) *Starter {
	s.reporter = reporter
	return s
}
