package services

import (
	"os"

	signals "github.com/setare/go-os-signals"
)

type BootstrapRequest struct {
	Services []Service
	Signals  []os.Signal
	Run      func()
}

// Bootstrap will start a signal listener listening all `request.Signals`. Then,
// start all services in `request.Services`. Then, at last, execute `request.Run`.
func Bootstrap(req BootstrapRequest) {
	signalListener := signals.NewListener(req.Signals...)

	s := NewStarter(req.Services...)
	if err := s.Start(); err != nil {
		panic(err)
	}
	go func() {
		s.ListenSignals(signalListener)
		os.Exit(0)
	}()

	req.Run()
}
