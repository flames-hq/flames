// Package ingress defines the IngressProvider interface for exposing VM services to external traffic.
package ingress

import (
	"context"

	"github.com/flames-hq/flames/model"
)

// IngressProvider manages external routing to VMs. The no-op default accepts all calls
// without configuring real networking, making it safe for local development.
type IngressProvider interface {
	RegisterEndpoint(ctx context.Context, vmID string, route model.Route, target model.Target) (model.Endpoint, error)
	UnregisterEndpoint(ctx context.Context, vmID string) error
	GetEndpoint(ctx context.Context, vmID string) (model.Endpoint, error)
}
