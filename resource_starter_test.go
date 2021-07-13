package services_test

import (
	"context"
	"errors"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/setare/services"
)

func createController() *gomock.Controller {
	return gomock.NewController(GinkgoT(1))
}

var _ = Describe("ResourceStarter", func() {
	Describe("Start", func() {
		It("should start a service", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockResourceService(ctrl)
			serviceB := NewMockResourceService(ctrl)
			serviceC := NewMockResourceService(ctrl)

			// 2. Create and Start the ResourceStarter
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

			// 2. Create and Start the ResourceStarter
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

			// 2. Create and Start the ResourceStarter
			starter := services.NewManager()

			// 4. Checks if the start was cancelled.
			err := starter.Start(ctx, serviceA, serviceB, serviceC)
			Expect(err).To(MatchError(errB))
		})

		Describe("Close", func() {
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

				// 3. Close the resourceServices.
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

				// 2. Triggers the ResourceStarter
				starter := services.NewManager()
				Expect(starter.Start(ctx, serviceA, serviceB, serviceC)).To(Succeed())

				// 3. Close the resourceServices and ensure an error was returned.
				Expect(starter.Stop(ctx)).To(MatchError(errA))
			})
		})
	})
})
