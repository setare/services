package services

import (
	"context"
)

// Configurable describes a service that should be loaded before started.
//
// This method will be used direct by `ResourceStarter`. Before starting a service, `ResourceStarter` will call `Load`
// (if available) before continuing starting the service. If it fails, `ResourceStarter.Start` will fail. Otherwise, the
// starting process will continue normally.
type Configurable interface {
	// Load will load the configuration
	Load(ctx context.Context) error
}
