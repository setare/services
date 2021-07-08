package services_test

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/setare/services"
)

var _ = Describe("Retrier", func() {
	Context("ResourceService", func() {
		It("should start and stop a service", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			serviceA := NewMockResourceService(ctrl)

			gomock.InOrder(
				serviceA.EXPECT().Name().Return("Service A"),
				serviceA.EXPECT().Start(gomock.Any()),
				serviceA.EXPECT().Stop(gomock.Any()),
			)

			serviceARetrier := services.Retrier().Backoff(backoff.NewExponentialBackOff()).Build(serviceA)
			Expect(serviceARetrier.Name()).To(Equal("Service A"))
			manager := services.NewManager()
			Expect(manager.Start(ctx, serviceARetrier)).To(Succeed())
			Expect(manager.Stop(ctx)).To(Succeed())
		})

		It("should start a service after the third try waiting between each", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			serviceA := NewMockResourceService(ctrl)

			reporter := NewMockRetrierReporter(ctrl)

			randomErr := errors.New("any error")
			gomock.InOrder(
				reporter.EXPECT().BeforeRetry(gomock.Any(), serviceA, 1),
				serviceA.EXPECT().Start(gomock.Any()).Return(randomErr),
				reporter.EXPECT().BeforeRetry(gomock.Any(), serviceA, 2),
				serviceA.EXPECT().Start(gomock.Any()).Return(randomErr),
				reporter.EXPECT().BeforeRetry(gomock.Any(), serviceA, 3),
				serviceA.EXPECT().Start(gomock.Any()).Return(randomErr),
				reporter.EXPECT().BeforeRetry(gomock.Any(), serviceA, 4),
				serviceA.EXPECT().Start(gomock.Any()).Return(nil),
			)

			serviceARetrier := services.Retrier().Reporter(reporter).Backoff(backoff.NewConstantBackOff(time.Millisecond * 50)).Build(serviceA)
			starter := services.NewManager()
			startedAt := time.Now()
			Expect(starter.Start(ctx, serviceARetrier)).To(Succeed())
			Expect(time.Since(startedAt)).To(BeNumerically("~", time.Millisecond*150, time.Millisecond*20))
		})

		It("should load a Configurable service", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			serviceA := &struct {
				*MockResourceService
				*MockConfigurable
			}{
				MockResourceService: NewMockResourceService(ctrl),
				MockConfigurable:    NewMockConfigurable(ctrl),
			}

			gomock.InOrder(
				serviceA.MockConfigurable.EXPECT().Load(gomock.Any()),
				serviceA.MockResourceService.EXPECT().Start(gomock.Any()),
			)

			serviceARetrier := services.Retrier().Build(serviceA)
			starter := services.NewManager()
			Expect(starter.Start(ctx, serviceARetrier)).To(Succeed())
		})
	})

	Context("StartableWithContext", func() {
		/*
			It("should start and stop a service", func() {
				serviceA := &serviceStartWithContext{
					name: "A",
				}
				serviceARetrier := Retrier().Tries(3).Build(serviceA)
				Expect(serviceARetrier.Name()).To(Equal("Service With Context A"))
				starter := NewManager(serviceARetrier)
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
				starter := NewManager(serviceARetrier)
				Expect(starter.Start()).To(Succeed())
				Expect(serviceA.startCallCount).To(Equal(3))
				Expect(serviceA.started).To(BeTrue())
			})

			It("should fail starting a service after reaching the tries limit", func() {
				serviceA := &serviceStartWithContext{
					startErr: []error{errors.New("any error"), errors.New("any error"), errors.New("any error")},
				}
				serviceARetrier := Retrier().Tries(3).Build(serviceA)
				starter := NewManager(serviceARetrier)
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
				starter := NewManager(serviceARetrier)
				err := starter.Start()
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(context.DeadlineExceeded, err)).To(BeTrue())
				Expect(serviceA.startCallCount).To(Equal(2))
				Expect(serviceA.started).To(BeFalse())
			})
		*/
	})
})
