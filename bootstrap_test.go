package services

import (
	"errors"
	"os"
	"testing"

	signals "github.com/setare/go-os-signals"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrap(t *testing.T) {
	t.Run("without SetupStarter", func(t *testing.T) {
		run := false
		setupStarter := false
		err := Bootstrap(BootstrapRequest{
			SetupStarter: func(s *Starter) {
				require.NotNil(t, s)
				setupStarter = true
			},
			Run: func() error {
				run = true
				return nil
			},
		})

		assert.True(t, run)
		assert.True(t, setupStarter)
		assert.NoError(t, err, nil)
	})

	t.Run("with SetupStarter", func(t *testing.T) {
		listener := signals.NewMockListener(os.Interrupt)
		run := false
		setupStarter := false
		err := Bootstrap(BootstrapRequest{
			SignalListener: listener,
			SetupStarter: func(s *Starter) {
				require.NotNil(t, s)
				setupStarter = true
			},
			Run: func() error {
				run = true
				return nil
			},
		})

		assert.True(t, run)
		assert.True(t, setupStarter)
		assert.NoError(t, err, nil)
	})

	t.Run("starter failed", func(t *testing.T) {
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

		require.False(t, run)
		require.ErrorIs(t, err, wantErr)
	})
}
