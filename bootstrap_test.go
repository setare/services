package services

import (
	"errors"
	"os"

	. "github.com/onsi/gomega"
	signals "github.com/setare/go-os-signals"
)

var _ = Describe("Bootstrap", func() {
	It("should succeed without SetupStarter", func() {
		run := false
		setupStarter := false
		err := Bootstrap(BootstrapRequest{
			SetupStarter: func(s *Starter) {
				Expect(s).NotTo(BeNil())
				setupStarter = true
			},
			Run: func() error {
				run = true
				return nil
			},
		})

		Expect(run).To(BeTrue())
		Expect(setupStarter).To(BeTrue())
		Expect(err).ToNot(HaveOccurred())
	})

	It("shuold succeed with SetupStarter", func() {
		listener := signals.NewMockListener(os.Interrupt)
		run := false
		setupStarter := false
		err := Bootstrap(BootstrapRequest{
			SignalListener: listener,
			SetupStarter: func(s *Starter) {
				Expect(s).NotTo(BeNil())
				setupStarter = true
			},
			Run: func() error {
				run = true
				return nil
			},
		})

		Expect(run).To(BeTrue())
		Expect(setupStarter).To(BeTrue())
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fail when starter failed", func() {
		wantErr := errors.New("random error")

		failedServer := serviceStart{
			name:     "failedServer",
			startErr: []error{wantErr},
		}

		listener := signals.NewMockListener(os.Interrupt)
		run := false
		err := Bootstrap(BootstrapRequest{
			SignalListener: listener,
			Services: []Service{
				&failedServer,
			},
			Run: func() error {
				run = true
				return nil
			},
		})

		Expect(run).To(BeFalse())
		Expect(err).To(MatchError(wantErr))
	})

})
