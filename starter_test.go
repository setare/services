package services

import (
	"context"
	"os"
	"time"

	"github.com/pkg/errors"
	signals "github.com/setare/go-os-signals"

	. "github.com/onsi/gomega"
)

type serviceOnly struct {
}

func (*serviceOnly) Name() string {
	return "Not Startable"
}

func (*serviceOnly) Stop() error {
	return nil
}

type serviceStart struct {
	name           string
	startCallCount int
	startErr       []error
	startDelay     time.Duration
	started        bool
	stopErr        []error
	stopped        bool
}

func (s *serviceStart) Name() string {
	return "Service " + s.name
}

func (s *serviceStart) Stop() error {
	if len(s.stopErr) == 0 {
		s.stopped = true
		return nil
	}
	err := s.stopErr[0]
	s.stopErr = s.stopErr[1:]
	s.stopped = err == nil
	return err
}

func (s *serviceStart) Start() error {
	s.startCallCount++
	time.Sleep(s.startDelay)
	if len(s.startErr) == 0 {
		s.started = true
		return nil
	}
	err := s.startErr[0]
	s.startErr = s.startErr[1:]
	s.started = err == nil
	return err
}

type serviceStartConfigurable struct {
	serviceStart
	loaded  bool
	errLoad error
}

func (s *serviceStartConfigurable) Load() error {
	s.loaded = s.errLoad == nil
	return s.errLoad
}

type serviceStartWithContext struct {
	name           string
	startErr       []error
	startCallCount int
	startDelay     time.Duration
	started        bool
	stopErr        []error
	stopped        bool
}

func (s *serviceStartWithContext) Name() string {
	return "Service With Context " + s.name
}

func (s *serviceStartWithContext) Stop() error {
	s.stopped = true
	if len(s.stopErr) == 0 {
		return nil
	}
	err := s.stopErr[0]
	s.stopErr = s.stopErr[1:]
	return err
}

func (s *serviceStartWithContext) StartWithContext(ctx context.Context) error {
	s.startCallCount++
	select {
	case <-time.After(s.startDelay):
		// Nothing
	case <-ctx.Done():
		return ctx.Err()
	}
	if len(s.startErr) == 0 {
		s.started = true
		return nil
	}
	err := s.startErr[0]
	s.startErr = s.startErr[1:]
	s.started = err == nil
	return err
}

var _ = Describe("Starter", func() {
	Describe("Start", func() {
		It("should start a service", func() {
			// 1. Create 3 services
			serviceA := &serviceStart{
				name: "A",
			}
			serviceB := &serviceStart{
				name: "B",
			}
			serviceC := &serviceStart{
				name: "C",
			}

			// 2. Create and Start the Starter
			starter := NewStarter(serviceA, serviceB, serviceC)
			Expect(starter.Start()).To(Succeed())

			// 3. Ensure the services were started.
			Expect(serviceA.started).To(BeTrue())
			Expect(serviceB.started).To(BeTrue())
			Expect(serviceC.started).To(BeTrue())

			// 4. Ensure the services were started at the right order.
			Expect(starter.servicesStarted).To(Equal([]Service{serviceA, serviceB, serviceC}))
		})

		It("should interrupt starting services", func() {
			// 1. Create 3 services
			serviceA := &serviceStart{
				name:       "A",
				startDelay: time.Millisecond * 50,
			}
			serviceB := &serviceStart{
				name:       "B",
				startDelay: time.Millisecond * 50,
			}
			serviceC := &serviceStart{
				name:       "C",
				startDelay: time.Millisecond * 100,
			}

			// 2. Create and Start the Starter
			starter := NewStarter(serviceA, serviceB, serviceC)
			go func() {
				// 3. Triggers the goroutine to interrupt the starting process before serviceC have chance
				// of finishing
				time.Sleep(time.Millisecond * 75)
				starter.Stop()
			}()

			// 4. Checks if the start was cancelled.
			err := starter.Start()
			Expect(err).To(Equal(context.Canceled))

			// 5. Ensure the services were started.
			Expect(serviceA.started).To(BeTrue())
			Expect(serviceB.started).To(BeTrue())
			Expect(serviceC.started).To(BeFalse()) // Canceled before actually calling the StartWithContext

			// 6. Ensure the services already started were stopped.
			Expect(serviceA.stopped).To(BeTrue())
			Expect(serviceB.stopped).To(BeTrue())

			// 7. Ensure the services were started at the right order.
			Expect(starter.servicesStarted).To(BeEmpty())
		})

		It("should interrupt starting services with a os.Interrupt", func() {
			// 1. Create 3 services
			serviceA := &serviceStart{
				name:       "A",
				startDelay: time.Millisecond * 50,
			}
			serviceB := &serviceStart{
				name:       "B",
				startDelay: time.Millisecond * 50,
			}
			serviceC := &serviceStart{
				name:       "C",
				startDelay: time.Millisecond * 100,
			}

			// 2. Create and Start the Starter
			starter := NewStarter(serviceA, serviceB, serviceC)
			// Override the default signalListener by a mocked one.
			mockListener := signals.NewMockListener(os.Interrupt)
			starter.signalListener = mockListener
			go func() {
				// 3. Triggers the goroutine to send a os.Interrupt signal before serviceC have chance
				// of finishing
				time.Sleep(time.Millisecond * 75)
				mockListener.Send(os.Interrupt)
			}()

			// 4. Checks if the start was cancelled.
			err := starter.Start()
			Expect(err).To(Equal(context.Canceled))

			// 5. Ensure the services were started.
			Expect(serviceA.started).To(BeTrue())
			Expect(serviceB.started).To(BeTrue())
			Expect(serviceC.started).To(BeFalse()) // Canceled before actually calling the StartWithContext

			// 6. Ensure the services already started were stopped.
			Expect(serviceA.stopped).To(BeTrue())
			Expect(serviceB.stopped).To(BeTrue())

			// 7. Ensure the services were started at the right order.
			Expect(starter.servicesStarted).To(BeEmpty())
		})

		It("should interrupt starting services when load fails", func() {
			// 1. Create 3 services
			serviceA := &serviceStart{
				name: "A",
			}
			errB := errors.New("error")
			serviceB := &serviceStartConfigurable{
				serviceStart: serviceStart{name: "B"},
				errLoad:      errB,
			}
			serviceC := &serviceStart{
				name:       "C",
				startDelay: time.Millisecond * 100,
			}

			// 2. Create and Start the Starter
			starter := NewStarter(serviceA, serviceB, serviceC)

			// 4. Checks if the start was cancelled.
			err := starter.Start()
			Expect(err).To(Equal(errB))

			// 5. Ensure the services were started.
			Expect(serviceA.started).To(BeTrue())
			Expect(serviceB.started).To(BeFalse())
			Expect(serviceB.loaded).To(BeFalse())
			Expect(serviceC.started).To(BeFalse()) // Canceled before actually calling the StartWithContext

			// 6. Ensure the services already started were stopped.
			Expect(serviceA.stopped).To(BeTrue())
			Expect(serviceB.stopped).To(BeFalse())

			// 7. Ensure the services were started at the right order.
			Expect(starter.servicesStarted).To(BeEmpty())
		})

		It("should interrupt starting services with context", func() {
			// 1. Create 3 services
			serviceA := &serviceStartWithContext{
				name:       "A",
				startDelay: time.Millisecond * 50,
			}
			serviceB := &serviceStartWithContext{
				name:       "B",
				startDelay: time.Millisecond * 50,
			}
			serviceC := &serviceStartWithContext{
				name:       "C",
				startDelay: time.Millisecond * 100,
			}

			// 2. Create and Start the Starter
			starter := NewStarter(serviceA, serviceB, serviceC)
			go func() {
				// 3. Triggers the goroutine to interrupt the starting process before serviceC have chance
				// of finishing
				time.Sleep(time.Millisecond * 125)
				starter.Stop()
			}()

			// 4. Checks if the start was cancelled.
			err := starter.Start()
			Expect(err).To(Equal(context.Canceled))

			// 5. Ensure the services were started.
			Expect(serviceA.started).To(BeTrue())
			Expect(serviceB.started).To(BeTrue())
			Expect(serviceC.started).To(BeFalse()) // Canceled before actually calling the StartWithContext

			// 6. Ensure the services already started were stopped.
			Expect(serviceA.stopped).To(BeTrue())
			Expect(serviceB.stopped).To(BeTrue())

			// 7. Ensure the services were started at the right order.
			Expect(starter.servicesStarted).To(BeEmpty())
		})

		It("should fallback stopping all started services after then last failed", func() {
			// 1. Create 3 services, 2 health and 1 broken (the last one).
			serviceA := &serviceStart{
				name: "A",
			}
			serviceB := &serviceStart{
				name: "B",
			}
			errC := errors.New("any error")
			serviceC := &serviceStart{
				name:     "C",
				startErr: []error{errC},
			}

			// 2. Setup Starter and run Start
			starter := NewStarter(serviceA, serviceB, serviceC)
			Expect(starter.Start()).To(Equal(errC))

			// 3. Ensure the health services were started...
			Expect(serviceA.started).To(BeTrue())
			Expect(serviceB.started).To(BeTrue())

			// 4. Ensure the broken services is not started.
			Expect(serviceC.started).To(BeFalse())

			// 5. Ensure the health services were stopped after the third service failed.
			Expect(serviceA.stopped).To(BeTrue())
			Expect(serviceB.stopped).To(BeTrue())

			// 6. Ensure the health services were stopped after the third service failed.
			Expect(starter.servicesStarted).To(BeEmpty())
		})

		It("should fallback stopping all started services with context after then last failed", func() {
			// 1. Create 3 StartableWithContext services
			serviceA := &serviceStartWithContext{
				name: "A",
			}
			serviceB := &serviceStartWithContext{
				name: "B",
			}

			// 2. The last with error.
			errC := errors.New("any error")
			serviceC := &serviceStartWithContext{
				name:     "C",
				startErr: []error{errC},
			}

			// 3. Intiialize the starter and Start it.
			starter := NewStarter(serviceA, serviceB, serviceC)
			Expect(starter.Start()).To(Equal(errC))

			// 4. Check if the first 2 are started and the broken is not.
			Expect(serviceA.started).To(BeTrue())
			Expect(serviceB.started).To(BeTrue())
			Expect(serviceC.started).To(BeFalse())
			// 5. Check if, after the broken service crash the other services wre stopped
			Expect(serviceA.stopped).To(BeTrue())
			Expect(serviceB.stopped).To(BeTrue())

			// 6. Ensure the servicesStarted list is empty
			Expect(starter.servicesStarted).To(BeEmpty())
		})

		It("should fail starting a service that is not Startable", func() {
			// 1. Initialize 2 StartableWithContext and one that does not implement neither `Startable` or
			// `StartableWithContext`.
			serviceA := &serviceStartWithContext{
				name: "A",
			}
			serviceB := &serviceStartWithContext{
				name: "B",
			}
			serviceC := &serviceOnly{}

			// 2. Intialize and Start the `Starter`.
			starter := NewStarter(serviceA, serviceB, serviceC)
			err := starter.Start()

			// 3. The Start failed.
			Expect(err).To(HaveOccurred())

			// 4. Check for the expected error.
			Expect(errors.Is(err, ErrNotStartable)).To(BeTrue())

			// 5. Check if the first 2 services were started.
			Expect(serviceA.started).To(BeTrue())
			Expect(serviceB.started).To(BeTrue())

			// 6. Check if the first 2 services were stopped after the 3rd failed.
			Expect(serviceA.stopped).To(BeTrue())
			Expect(serviceB.stopped).To(BeTrue())

			// 7. Ensure the services started list is empty.
			Expect(starter.servicesStarted).To(BeEmpty())
		})
	})

	Describe("Stop", func() {
		It("should stop all services", func() {
			// 1. Create 3 health services.
			serviceA := &serviceStart{
				name: "A",
			}
			serviceB := &serviceStart{
				name: "B",
			}
			serviceC := &serviceStart{
				name: "C",
			}

			// 2. Triggers the starter
			starter := NewStarter(serviceA, serviceB, serviceC)
			Expect(starter.Start()).To(Succeed())

			// 3. Ensure services were started.
			Expect(serviceA.started).To(BeTrue())
			Expect(serviceB.started).To(BeTrue())
			Expect(serviceC.started).To(BeTrue())

			// 4. Ensure the order the the services were started.
			Expect(starter.servicesStarted).To(Equal([]Service{serviceA, serviceB, serviceC}))

			// 5. Stop the services.
			Expect(starter.Stop()).To(Succeed())

			// 6. Ensure the services were stopped.
			Expect(serviceA.stopped).To(BeTrue())
			Expect(serviceB.stopped).To(BeTrue())
			Expect(serviceC.stopped).To(BeTrue())

			// 7. Ensure the started services list is empty.
			Expect(starter.servicesStarted).To(BeEmpty())
		})

		It("should stop all services", func() {
			// 1. Create 3 services but the first (last to be stopped) fails to stop.
			errA := errors.New("any error")
			serviceA := &serviceStart{
				name:    "A",
				stopErr: []error{errA},
			}
			serviceB := &serviceStart{
				name: "B",
			}
			serviceC := &serviceStart{
				name: "C",
			}

			// 2. Triggers the Starter
			starter := NewStarter(serviceA, serviceB, serviceC)
			Expect(starter.Start()).To(Succeed())

			// 3. Ensure all services are started.
			Expect(serviceA.started).To(BeTrue())
			Expect(serviceB.started).To(BeTrue())
			Expect(serviceC.started).To(BeTrue())
			Expect(starter.servicesStarted).To(Equal([]Service{serviceA, serviceB, serviceC}))

			// 4. Stop the services and ensure an error was returned.
			Expect(starter.Stop()).To(Equal(errA))

			// 5. Ensure the services were stopped, but the first.
			Expect(serviceA.stopped).To(BeFalse())
			Expect(serviceB.stopped).To(BeTrue())
			Expect(serviceC.stopped).To(BeTrue())

			// 6. Ensure the started list did not get the service, that could not be stopped, removed from the list.
			Expect(starter.servicesStarted).To(Equal([]Service{serviceA}))
		})
	})

	Describe("ListenToSignals", func() {
		It("should stop all services when receive an interruption", func() {
			// 1. Create 3 services
			serviceA := &serviceStart{
				name: "A",
			}
			serviceB := &serviceStart{
				name: "B",
			}
			serviceC := &serviceStart{
				name: "C",
			}

			// 2. Create and Start the Starter
			starter := NewStarter(serviceA, serviceB, serviceC)
			Expect(starter.Start()).To(Succeed())

			// 3. Ensure the services were started.
			Expect(serviceA.started).To(BeTrue())
			Expect(serviceB.started).To(BeTrue())
			Expect(serviceC.started).To(BeTrue())

			mockSignalListener := signals.NewMockListener(os.Interrupt)

			go func() {
				time.Sleep(time.Millisecond * 75)
				mockSignalListener.Send(os.Interrupt)
			}()

			Expect(starter.ListenSignals(mockSignalListener)).To(Succeed())

			Expect(serviceA.stopped).To(BeTrue())
			Expect(serviceB.stopped).To(BeTrue())
			Expect(serviceC.stopped).To(BeTrue())

			// 4. Ensure the services were started at the right order.
			Expect(starter.servicesStarted).To(BeEmpty())
		})
	})
})
