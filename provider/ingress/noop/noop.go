// Package noop provides a no-op IngressProvider implementation for development mode.
// It accepts all calls without performing network operations, but tracks registered
// endpoints so GetEndpoint can distinguish registered from unregistered VMs.
package noop

import (
	"context"
	"sync"

	"github.com/flames-hq/flames/model"
	"github.com/flames-hq/flames/provider/ingress"
	"github.com/flames-hq/flames/provider/providererr"
)

var _ ingress.IngressProvider = (*Provider)(nil)

// Provider is a no-op IngressProvider that stores endpoints in memory without configuring real networking.
type Provider struct {
	mu        sync.RWMutex
	endpoints map[string]model.Endpoint
}

// New creates a new no-op IngressProvider.
func New() *Provider {
	return &Provider{
		endpoints: make(map[string]model.Endpoint),
	}
}

// RegisterEndpoint stores the endpoint mapping in memory. Always succeeds.
func (p *Provider) RegisterEndpoint(_ context.Context, vmID string, route model.Route, target model.Target) (model.Endpoint, error) {
	ep := model.Endpoint{
		VMID:   vmID,
		Route:  route,
		Target: target,
	}

	p.mu.Lock()
	p.endpoints[vmID] = ep
	p.mu.Unlock()

	return ep, nil
}

// UnregisterEndpoint removes the endpoint mapping for a VM. Always succeeds.
func (p *Provider) UnregisterEndpoint(_ context.Context, vmID string) error {
	p.mu.Lock()
	delete(p.endpoints, vmID)
	p.mu.Unlock()
	return nil
}

// GetEndpoint returns the endpoint for a VM. Returns ErrNotFound if the VM has no registered endpoint.
func (p *Provider) GetEndpoint(_ context.Context, vmID string) (model.Endpoint, error) {
	p.mu.RLock()
	ep, ok := p.endpoints[vmID]
	p.mu.RUnlock()

	if !ok {
		return model.Endpoint{}, providererr.NotFound("endpoint", vmID)
	}
	return ep, nil
}
