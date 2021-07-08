package services

import (
	"context"
)

// Configurable describes a service that should be loaded before started.
//
// This method will be used direct by `Manager`. Before starting a service,
// `Manager` will call `Load` (if available) before continuing starting the
// service. If it fails, `Manager.Start` will fail. Otherwise, the starting
// process will continue normally.
type Configurable interface {
	// Load will load the configuration
	Load(ctx context.Context) error
}
