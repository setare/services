package services_test

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/golang/mock/gomock"
	signals "github.com/jamillosantos/go-os-signals"
	"github.com/jamillosantos/go-os-signals/signaltest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/setare/go-services"
)

func createController() *gomock.Controller {
	return gomock.NewController(GinkgoT(1))
}

var _ = Describe("Runner", func() {
	Describe("Run Resource instances", func() {
		It("should start Resouce instances", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockResource(ctrl)
			serviceB := NewMockResource(ctrl)
			serviceC := NewMockResource(ctrl)

			// 2. Create and Run the Runner
			runner := services.NewRunner()

			gomock.InOrder(
				serviceA.EXPECT().Start(gomock.Any()),
				serviceB.EXPECT().Start(gomock.Any()),
				serviceC.EXPECT().Start(gomock.Any()),
			)

			Expect(runner.Run(ctx, serviceA, serviceB, serviceC)).To(Succeed())
		})

		It("should not stop started Resource instances when one fails", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockResource(ctrl)
			serviceB := NewMockResource(ctrl)
			serviceC := NewMockResource(ctrl)

			// 2. Create and Run the Runner
			runner := services.NewRunner()

			wantErr := errors.New("random")

			gomock.InOrder(
				serviceA.EXPECT().Start(gomock.Any()),
				serviceB.EXPECT().Start(gomock.Any()),
				serviceC.EXPECT().Start(gomock.Any()).Return(wantErr),
			)

			Expect(runner.Run(ctx, serviceA, serviceB, serviceC)).To(MatchError(wantErr))
		})

		It("should interrupt starting Resource instances", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx, cancelFunc := context.WithCancel(context.TODO())
			defer cancelFunc()

			// 1. Create 3 resourceServices
			serviceA := NewMockResource(ctrl)
			serviceB := NewMockResource(ctrl)
			serviceC := NewMockResource(ctrl)

			gomock.InOrder(
				serviceA.EXPECT().Start(gomock.Any()),
				serviceB.EXPECT().Start(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Millisecond * 90)
				}),
				serviceB.EXPECT().Stop(gomock.Any()),
				serviceA.EXPECT().Stop(gomock.Any()).Do(func(_ context.Context) {
					time.Sleep(time.Second)
				}),
			)

			// 2. Create and Run the Runner
			runner := services.NewRunner()
			go func() {
				defer GinkgoRecover()

				// 3. Triggers the goroutine to interrupt the starting process before serviceC have chance
				// of finishing
				time.Sleep(time.Millisecond * 50)
				cancelFunc()
			}()

			// 4. Checks if the start was cancelled.
			err := runner.Run(ctx, serviceA, serviceB, serviceC)
			Expect(err).To(Equal(context.Canceled))
			now := time.Now()
			runner.Finish(context.Background())
			Expect(time.Since(now)).To(BeNumerically("~", time.Second, time.Millisecond*50))
		})

		It("should interrupt starting Resource instances when Start fails", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockResource(ctrl)
			serviceB := NewMockResource(ctrl)
			serviceC := NewMockResource(ctrl)

			// 1. Create 3 resourceServices

			errB := errors.New("random error")

			gomock.InOrder(
				serviceA.EXPECT().Start(gomock.Any()),
				serviceB.EXPECT().Start(gomock.Any()).Return(errB),
				serviceA.EXPECT().Stop(gomock.Any()),
			)

			// 2. Create and Run the Runner
			runner := services.NewRunner()

			// 4. Checks if the start was cancelled.
			err := runner.Run(ctx, serviceA, serviceB, serviceC)
			Expect(err).To(MatchError(errB))
			runner.Finish(context.Background())
		})
	})

	Describe("Finish", func() {
		It("should stop all resourceServices", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockResource(ctrl)
			serviceB := NewMockResource(ctrl)
			serviceC := NewMockResource(ctrl)

			gomock.InOrder(
				serviceA.EXPECT().Start(gomock.Any()),
				serviceB.EXPECT().Start(gomock.Any()),
				serviceC.EXPECT().Start(gomock.Any()),
				serviceC.EXPECT().Stop(gomock.Any()),
				serviceB.EXPECT().Stop(gomock.Any()),
				serviceA.EXPECT().Stop(gomock.Any()),
			)

			// 2. Triggers the runner
			runner := services.NewRunner()
			Expect(runner.Run(ctx, serviceA, serviceB, serviceC)).To(Succeed())

			// 3. Close the resourceServices.
			Expect(runner.Finish(ctx)).To(Succeed())
		})

		It("should fail when a resource service can't stop", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockResource(ctrl)
			serviceB := NewMockResource(ctrl)
			serviceC := NewMockResource(ctrl)

			errA := errors.New("any error")
			gomock.InOrder(
				serviceA.EXPECT().Start(gomock.Any()),
				serviceB.EXPECT().Start(gomock.Any()),
				serviceC.EXPECT().Start(gomock.Any()),
				serviceC.EXPECT().Stop(gomock.Any()),
				serviceB.EXPECT().Stop(gomock.Any()),
				serviceA.EXPECT().Stop(gomock.Any()).Return(errA),
			)

			// 2. Triggers the Runner
			runner := services.NewRunner()
			Expect(runner.Run(ctx, serviceA, serviceB, serviceC)).To(Succeed())

			// 3. Close the resourceServices and ensure an error was returned.
			Expect(runner.Finish(ctx)).To(MatchError(errA))
		})
	})

	Describe("Run Server instances", func() {
		It("should start a Server instance", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx, cancelFunc := context.WithCancel(context.TODO())
			defer cancelFunc()

			// 1. Create 3 resourceServices
			serviceA := NewMockServer(ctrl)
			serviceB := NewMockServer(ctrl)
			serviceC := NewMockServer(ctrl)

			serviceA.EXPECT().Listen(gomock.Any()).Return(nil)
			serviceB.EXPECT().Listen(gomock.Any()).Return(nil)
			serviceC.EXPECT().Listen(gomock.Any()).Return(nil)

			serviceA.EXPECT().Close(gomock.Any())
			serviceB.EXPECT().Close(gomock.Any())
			serviceC.EXPECT().Close(gomock.Any())

			// 2. Create and Run the Runner
			runner := services.NewRunner()

			go func() {
				defer GinkgoRecover()
				time.Sleep(time.Second)

				cancelFunc()
			}()

			Expect(runner.Run(ctx, serviceA, serviceB, serviceC)).To(MatchError(context.Canceled))
		})

		When("a Serve instance fail starting", func() {
			It("should interrupt starting a Serve instance", func() {
				ctrl := createController()
				defer ctrl.Finish()

				ctx := context.TODO()

				// 1. Create 3 resourceServices
				serviceA := NewMockServer(ctrl)
				serviceB := NewMockServer(ctrl)
				serviceC := NewMockServer(ctrl)

				wantErr := errors.New("random error")

				serviceA.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Millisecond * 10)
				}).AnyTimes()
				serviceB.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Millisecond * 50)
				}).AnyTimes()
				serviceC.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Millisecond * 100)
				}).Return(wantErr).AnyTimes()

				serviceA.EXPECT().Close(gomock.Any()).AnyTimes()
				serviceB.EXPECT().Close(gomock.Any()).AnyTimes()
				serviceC.EXPECT().Close(gomock.Any()).AnyTimes()

				// 2. Create and Run the Runner
				runner := services.NewRunner()

				Expect(runner.Run(ctx, serviceA, serviceB, serviceC)).To(Equal(services.MultiErrors{
					nil, nil, wantErr,
				}))
			})
		})

		When("receive a signal", func() {
			It("should interrupt starting services", func() {
				ctrl := createController()
				defer ctrl.Finish()

				ctx := context.TODO()

				// 1. Create 3 resourceServices
				serverA := NewMockServer(ctrl)
				serverB := NewMockServer(ctrl)
				serverC := NewMockServer(ctrl)

				serverA.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Second)
				})
				serverB.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Second)
				})
				serverC.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Second)
				})

				serverA.EXPECT().Close(gomock.Any()).AnyTimes()
				serverB.EXPECT().Close(gomock.Any()).AnyTimes()
				serverC.EXPECT().Close(gomock.Any()).AnyTimes()

				// 2. Create and Run the Runner
				listener := signaltest.NewMockListener(os.Interrupt)
				runner := services.NewRunner(services.WithListenerBuilder(func() signals.Listener {
					return listener
				}))

				go func() {
					defer GinkgoRecover()
					Expect(runner.Run(ctx, serverA, serverB, serverC)).To(Succeed())
				}()

				time.Sleep(time.Millisecond * 100)
				listener.Send(os.Interrupt)
				time.Sleep(time.Second)
			})
		})
	})
})
