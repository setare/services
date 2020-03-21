package services

// Configurable describes a service that should be loaded before started.
//
// This method will be used direct by `Starter`. Before starting a service,
// `Starter` will call `Load` (if available) before continuing starting the
// service. If it fails, `Starter.Start` will fail. Otherwise, the starting
// process will continue normally.
type Configurable interface {
	// Load will load the configuration
	Load() error
}
