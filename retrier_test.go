package services

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("Retrier", func() {
	Context("Startable", func() {
		It("should start and stop a service", func() {
			serviceA := &serviceStart{
				name: "A",
			}
			serviceARetrier := Retrier().Tries(3).Build(serviceA)
			Expect(serviceARetrier.Name()).To(Equal("Service A"))
			starter := NewStarter(serviceARetrier)
			Expect(starter.Start()).To(Succeed())
			Expect(serviceA.startCallCount).To(Equal(1))
			Expect(serviceA.started).To(BeTrue())
			Expect(starter.Stop()).To(Succeed())
			Expect(serviceA.stopped).To(BeTrue())
		})

		It("should start a service after the third try waiting between each", func() {
			serviceA := &serviceStart{
				startErr: []error{errors.New("any error"), errors.New("any error")},
			}
			serviceARetrier := Retrier().Tries(3).WaitBetweenTries(time.Millisecond * 100).Build(serviceA)
			starter := NewStarter(serviceARetrier)
			startedAt := time.Now()
			Expect(starter.Start()).To(Succeed())
			Expect(time.Since(startedAt)).To(BeNumerically("~", time.Millisecond*200, time.Millisecond*20))
			Expect(serviceA.startCallCount).To(Equal(3))
			Expect(serviceA.started).To(BeTrue())
		})

		It("should fail starting a service after reaching the tries limit", func() {
			serviceA := &serviceStart{
				startErr: []error{errors.New("any error"), errors.New("any error"), errors.New("any error")},
			}
			serviceARetrier := Retrier().Tries(3).Build(serviceA)
			starter := NewStarter(serviceARetrier)
			err := starter.Start()
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(ErrExhaustedAttempts, err)).To(BeTrue())
			Expect(serviceA.startCallCount).To(Equal(3))
			Expect(serviceA.started).To(BeFalse())
		})

		It("should fail when a service takes too long to start", func() {
			serviceA := &serviceStart{
				startErr:   []error{errors.New("any error"), errors.New("any error"), errors.New("any error")},
				startDelay: time.Millisecond * 100,
			}
			serviceARetrier := Retrier().Tries(3).Timeout(time.Millisecond * 150).Build(serviceA)
			starter := NewStarter(serviceARetrier)
			err := starter.Start()
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(context.DeadlineExceeded, err)).To(BeTrue())
			Expect(serviceA.startCallCount).To(Equal(2))
			Expect(serviceA.started).To(BeFalse())
		})

		It("should load a Configurable service", func() {
			serviceA := &serviceStartConfigurable{
				serviceStart: serviceStart{
					name: "A",
				},
			}
			serviceARetrier := Retrier().Tries(3).Build(serviceA)
			Expect(serviceARetrier.Name()).To(Equal("Service A"))
			starter := NewStarter(serviceARetrier)
			Expect(starter.Start()).To(Succeed())
			Expect(serviceA.startCallCount).To(Equal(1))
			Expect(serviceA.started).To(BeTrue())
			Expect(starter.Stop()).To(Succeed())
			Expect(serviceA.stopped).To(BeTrue())
			Expect(serviceA.stopped).To(BeTrue())
			Expect(serviceA.loaded).To(BeTrue())
		})
	})

	Context("StartableWithContext", func() {
		It("should start and stop a service", func() {
			serviceA := &serviceStartWithContext{
				name: "A",
			}
			serviceARetrier := Retrier().Tries(3).Build(serviceA)
			Expect(serviceARetrier.Name()).To(Equal("Service With Context A"))
			starter := NewStarter(serviceARetrier)
			Expect(starter.Start()).To(Succeed())
			Expect(serviceA.startCallCount).To(Equal(1))
			Expect(serviceA.started).To(BeTrue())
			Expect(starter.Stop()).To(Succeed())
			Expect(serviceA.stopped).To(BeTrue())
		})

		It("should start a service after the third try", func() {
			serviceA := &serviceStartWithContext{
				startErr: []error{errors.New("any error"), errors.New("any error")},
			}
			serviceARetrier := Retrier().Tries(3).Build(serviceA)
			starter := NewStarter(serviceARetrier)
			Expect(starter.Start()).To(Succeed())
			Expect(serviceA.startCallCount).To(Equal(3))
			Expect(serviceA.started).To(BeTrue())
		})

		It("should fail starting a service after reaching the tries limit", func() {
			serviceA := &serviceStartWithContext{
				startErr: []error{errors.New("any error"), errors.New("any error"), errors.New("any error")},
			}
			serviceARetrier := Retrier().Tries(3).Build(serviceA)
			starter := NewStarter(serviceARetrier)
			err := starter.Start()
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(ErrExhaustedAttempts, err)).To(BeTrue())
			Expect(serviceA.startCallCount).To(Equal(3))
			Expect(serviceA.started).To(BeFalse())
		})

		It("should fail when a service takes too long to start", func() {
			serviceA := &serviceStartWithContext{
				startErr:   []error{errors.New("any error"), errors.New("any error"), errors.New("any error")},
				startDelay: time.Millisecond * 100,
			}
			serviceARetrier := Retrier().Tries(3).Timeout(time.Millisecond * 150).Build(serviceA)
			starter := NewStarter(serviceARetrier)
			err := starter.Start()
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(context.DeadlineExceeded, err)).To(BeTrue())
			Expect(serviceA.startCallCount).To(Equal(2))
			Expect(serviceA.started).To(BeFalse())
		})
	})
})
