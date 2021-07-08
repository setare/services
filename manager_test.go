package services_test

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	signals "github.com/setare/go-os-signals"

	"github.com/setare/services"
)

func createController() *gomock.Controller {
	return gomock.NewController(GinkgoT(1))
}

var _ = Describe("Manager", func() {
	Describe("Start", func() {
		It("should start a service", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockResourceService(ctrl)
			serviceB := NewMockResourceService(ctrl)
			serviceC := NewMockResourceService(ctrl)

			// 2. Create and Start the Manager
			starter := services.NewManager()

			gomock.InOrder(
				serviceA.EXPECT().Start(gomock.Any()),
				serviceB.EXPECT().Start(gomock.Any()),
				serviceC.EXPECT().Start(gomock.Any()),
			)

			Expect(starter.Start(ctx, serviceA, serviceB, serviceC)).To(Succeed())
		})

		It("should interrupt starting resourceServices", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockResourceService(ctrl)
			serviceB := NewMockResourceService(ctrl)
			serviceC := NewMockResourceService(ctrl)

			gomock.InOrder(
				serviceA.EXPECT().Start(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Millisecond * 10)
				}),
				serviceB.EXPECT().Start(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Millisecond * 90)
				}),
				serviceB.EXPECT().Stop(gomock.Any()),
				serviceA.EXPECT().Stop(gomock.Any()),
			)

			// 2. Create and Start the Manager
			starter := services.NewManager()
			go func() {
				// 3. Triggers the goroutine to interrupt the starting process before serviceC have chance
				// of finishing
				time.Sleep(time.Millisecond * 75)
				starter.Stop(context.TODO())
			}()

			// 4. Checks if the start was cancelled.
			err := starter.Start(ctx, serviceA, serviceB, serviceC)
			Expect(err).To(Equal(context.Canceled))
		})

		It("should interrupt starting resourceServices with a os.Interrupt", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockResourceService(ctrl)
			serviceB := NewMockResourceService(ctrl)
			serviceC := NewMockResourceService(ctrl)

			gomock.InOrder(
				serviceA.EXPECT().Start(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Millisecond * 10)
				}),
				serviceB.EXPECT().Start(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Millisecond * 90)
				}),
				serviceB.EXPECT().Stop(gomock.Any()),
				serviceA.EXPECT().Stop(gomock.Any()),
			)

			// 2. Create and Start the Manager
			mockListener := signals.NewMockListener(os.Interrupt)
			starter := services.NewManager(services.WithListener(mockListener))

			go func() {
				// 3. Triggers the goroutine to send a os.Interrupt signal before serviceC have chance
				// of finishing
				time.Sleep(time.Millisecond * 75)
				mockListener.Send(os.Interrupt)
			}()

			// 4. Checks if the start was cancelled.
			err := starter.Start(ctx, serviceA, serviceB, serviceC)
			Expect(err).To(Equal(context.Canceled))
		})

		It("should interrupt starting resourceServices when load fails", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockResourceService(ctrl)
			serviceB := NewMockResourceService(ctrl)
			serviceC := NewMockResourceService(ctrl)

			// 1. Create 3 resourceServices

			errB := errors.New("random error")

			gomock.InOrder(
				serviceA.EXPECT().Start(gomock.Any()),
				serviceB.EXPECT().Start(gomock.Any()).Return(errB),
				serviceA.EXPECT().Stop(gomock.Any()),
			)

			// 2. Create and Start the Manager
			starter := services.NewManager()

			// 4. Checks if the start was cancelled.
			err := starter.Start(ctx, serviceA, serviceB, serviceC)
			Expect(err).To(MatchError(errB))
		})

		Describe("Stop", func() {
			It("should stop all resourceServices", func() {
				ctrl := createController()
				defer ctrl.Finish()

				ctx := context.TODO()

				// 1. Create 3 resourceServices
				serviceA := NewMockResourceService(ctrl)
				serviceB := NewMockResourceService(ctrl)
				serviceC := NewMockResourceService(ctrl)

				gomock.InOrder(
					serviceA.EXPECT().Start(gomock.Any()),
					serviceB.EXPECT().Start(gomock.Any()),
					serviceC.EXPECT().Start(gomock.Any()),
					serviceC.EXPECT().Stop(gomock.Any()),
					serviceB.EXPECT().Stop(gomock.Any()),
					serviceA.EXPECT().Stop(gomock.Any()),
				)

				// 2. Triggers the starter
				starter := services.NewManager()
				Expect(starter.Start(ctx, serviceA, serviceB, serviceC)).To(Succeed())

				// 3. Stop the resourceServices.
				Expect(starter.Stop(ctx)).To(Succeed())
			})

			It("should fail when a resource service can't stop", func() {
				ctrl := createController()
				defer ctrl.Finish()

				ctx := context.TODO()

				// 1. Create 3 resourceServices
				serviceA := NewMockResourceService(ctrl)
				serviceB := NewMockResourceService(ctrl)
				serviceC := NewMockResourceService(ctrl)

				errA := errors.New("any error")
				gomock.InOrder(
					serviceA.EXPECT().Start(gomock.Any()),
					serviceB.EXPECT().Start(gomock.Any()),
					serviceC.EXPECT().Start(gomock.Any()),
					serviceC.EXPECT().Stop(gomock.Any()),
					serviceB.EXPECT().Stop(gomock.Any()),
					serviceA.EXPECT().Stop(gomock.Any()).Return(errA),
				)

				// 2. Triggers the Manager
				starter := services.NewManager()
				Expect(starter.Start(ctx, serviceA, serviceB, serviceC)).To(Succeed())

				// 3. Stop the resourceServices and ensure an error was returned.
				Expect(starter.Stop(ctx)).To(MatchError(errA))
			})
		})
	})

	Describe("ListenToSignals", func() {
		It("should stop all resourceServices when receive an interruption", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockResourceService(ctrl)
			serviceB := NewMockResourceService(ctrl)
			serviceC := NewMockResourceService(ctrl)

			gomock.InOrder(
				serviceA.EXPECT().Start(gomock.Any()),
				serviceB.EXPECT().Start(gomock.Any()),
				serviceC.EXPECT().Start(gomock.Any()),
				serviceC.EXPECT().Stop(gomock.Any()),
				serviceB.EXPECT().Stop(gomock.Any()),
				serviceA.EXPECT().Stop(gomock.Any()),
			)

			// 2. Create and Start the Manager
			fakeListener := signals.NewMockListener(os.Interrupt)
			starter := services.NewManager(services.WithListener(fakeListener))
			Expect(starter.Start(ctx, serviceA, serviceB, serviceC)).To(Succeed())

			time.Sleep(time.Millisecond * 50)
			fakeListener.Send(os.Interrupt)
			time.Sleep(time.Millisecond * 50)
		})
	})

	When("reporter defined", func() {
		It("should stop all resourceServices when receive an interruption", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockResourceService(ctrl)
			reporter := NewMockReporter(ctrl)

			gomock.InOrder(
				reporter.EXPECT().BeforeStart(gomock.Any(), serviceA),
				serviceA.EXPECT().Start(gomock.Any()),
				reporter.EXPECT().AfterStart(gomock.Any(), serviceA, nil),
				reporter.EXPECT().BeforeStop(gomock.Any(), serviceA),
				serviceA.EXPECT().Stop(gomock.Any()),
				reporter.EXPECT().AfterStop(gomock.Any(), serviceA, nil),
			)

			// 2. Create and Start the Manager
			fakeListener := signals.NewMockListener(os.Interrupt)
			manager := services.NewManager(
				services.WithListener(fakeListener),
				services.WithReporter(reporter),
			)
			Expect(manager.Start(ctx, serviceA)).To(Succeed())
			Expect(manager.Stop(ctx)).To(Succeed())
		})
	})
})
