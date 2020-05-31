package services

import "os"

// Reporter will be called Before and After some actions by a `Starter`.
type Reporter interface {
	BeforeStart(Service)
	AfterStart(Service, error)
	BeforeStop(Service)
	AfterStop(Service, error)
	BeforeLoad(Configurable)
	AfterLoad(Configurable, error)

	SignalReceived(os.Signal)
}

type RetrierReporter interface {
	Reporter
	BeforeRetry(Service, int)
	AfterGiveUp(Service, int, error)
}
