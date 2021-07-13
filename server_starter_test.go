package services_test

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/jamillosantos/go-os-signals/signaltest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/setare/services"
)

var _ = Describe("ServerStarter", func() {
	Describe("Listen", func() {
		It("should start a service", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockServerService(ctrl)
			serviceB := NewMockServerService(ctrl)
			serviceC := NewMockServerService(ctrl)

			serviceA.EXPECT().Listen(gomock.Any()).Return(nil)
			serviceB.EXPECT().Listen(gomock.Any()).Return(nil)
			serviceC.EXPECT().Listen(gomock.Any()).Return(nil)

			serviceA.EXPECT().Close(gomock.Any())
			serviceB.EXPECT().Close(gomock.Any())
			serviceC.EXPECT().Close(gomock.Any())

			// 2. Create and Start the ResourceStarter
			starter := services.ServerStarter{}

			go func() {
				defer GinkgoRecover()
				time.Sleep(time.Second)

				starter.Close(ctx)
			}()

			Expect(starter.Listen(ctx, serviceA, serviceB, serviceC)).To(BeNil())
		})

		It("should interrupt starting services", func() {
			ctrl := createController()
			defer ctrl.Finish()

			ctx := context.TODO()

			// 1. Create 3 resourceServices
			serviceA := NewMockServerService(ctrl)
			serviceB := NewMockServerService(ctrl)
			serviceC := NewMockServerService(ctrl)

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

			// 2. Create and Start the ResourceStarter
			starter := services.ServerStarter{}

			go func() {
				defer GinkgoRecover()
				time.Sleep(time.Second)

				starter.Close(ctx)
			}()

			Expect(starter.Listen(ctx, serviceA, serviceB, serviceC)).To(Equal([]error{
				nil, nil, wantErr,
			}))
		})

		When("stop is called", func() {
			It("should interrupt starting services", func() {
				ctrl := createController()
				defer ctrl.Finish()

				ctx := context.TODO()

				// 1. Create 3 resourceServices
				serviceA := NewMockServerService(ctrl)
				serviceB := NewMockServerService(ctrl)
				serviceC := NewMockServerService(ctrl)

				wantErr := errors.New("random error")

				serviceA.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Second)
				}).AnyTimes()
				serviceB.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Second)
				}).AnyTimes()
				serviceC.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Second)
				}).Return(wantErr).AnyTimes()

				serviceA.EXPECT().Close(gomock.Any()).AnyTimes()
				serviceB.EXPECT().Close(gomock.Any()).AnyTimes()
				serviceC.EXPECT().Close(gomock.Any()).AnyTimes()

				// 2. Create and Start the ResourceStarter
				starter := services.ServerStarter{}

				go func() {
					defer GinkgoRecover()
					Expect(starter.Listen(ctx, serviceA, serviceB, serviceC)).To(Equal([]error{
						nil, nil, wantErr,
					}))
				}()

				time.Sleep(time.Millisecond * 100)
				starter.Close(ctx)
				time.Sleep(time.Second)
			})
		})

		When("receive a signal", func() {
			It("should interrupt starting services", func() {
				ctrl := createController()
				defer ctrl.Finish()

				ctx := context.TODO()

				// 1. Create 3 resourceServices
				serviceA := NewMockServerService(ctrl)
				serviceB := NewMockServerService(ctrl)
				serviceC := NewMockServerService(ctrl)

				wantErr := errors.New("random error")

				serviceA.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Second)
				}).AnyTimes()
				serviceB.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Second)
				}).AnyTimes()
				serviceC.EXPECT().Listen(gomock.Any()).Do(func(context.Context) {
					time.Sleep(time.Second)
				}).Return(wantErr).AnyTimes()

				serviceA.EXPECT().Close(gomock.Any()).AnyTimes()
				serviceB.EXPECT().Close(gomock.Any()).AnyTimes()
				serviceC.EXPECT().Close(gomock.Any()).AnyTimes()

				// 2. Create and Start the ResourceStarter
				listener := signaltest.NewMockListener(os.Interrupt)
				starter := services.NewServerStarter(listener)

				go func() {
					defer GinkgoRecover()
					Expect(starter.Listen(ctx, serviceA, serviceB, serviceC)).To(Equal([]error{
						nil, nil, wantErr,
					}))
				}()

				time.Sleep(time.Millisecond * 100)
				listener.Send(os.Interrupt)
				time.Sleep(time.Second)
			})
		})
	})
})
