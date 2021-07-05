package services

import (
	"os"

	signals "github.com/setare/go-os-signals"
)

type BootstrapRequest struct {
	Services       []Service
	Run            func() error
	SetupStarter   func(*Starter)
	SignalListener signals.Listener
}

// Bootstrap will start a signal listener listening all `request.Signals`. Then,
// start all services in `request.Services`. Then, at last, execute `request.Run`.
func Bootstrap(req BootstrapRequest) error {
	signalListener := req.SignalListener
	if signalListener == nil {
		signalListener = signals.NewListener(os.Interrupt)
	}

	s := NewStarter(req.Services...)
	if req.SetupStarter != nil {
		req.SetupStarter(s)
	}
	if err := s.Start(); err != nil {
		return err
	}
	go func() {
		s.ListenSignals(signalListener)
		os.Exit(0)
	}()

	return req.Run()
}
