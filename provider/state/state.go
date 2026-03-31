// Package state defines the StateStore interface for persisting VMs, controllers, and events.
package state

import (
	"context"

	"github.com/flames-hq/flames/model"
)

// StateStore is the source of truth for VM, controller, and event records.
// Implementations must be safe for concurrent use and guarantee atomic updates.
type StateStore interface {
	CreateVM(ctx context.Context, spec model.VMSpec) (string, error)
	GetVM(ctx context.Context, vmID string) (model.VM, error)
	UpdateVMDesiredState(ctx context.Context, vmID string, state model.DesiredState) error
	UpdateVMObservedState(ctx context.Context, vmID string, state model.ObservedState, controllerID string) error
	AssignVM(ctx context.Context, vmID string, controllerID string) error
	ListPendingVMs(ctx context.Context) ([]model.VM, error)

	AppendEvent(ctx context.Context, event model.Event) error
	ListEvents(ctx context.Context, filter model.EventFilter) ([]model.Event, error)

	RegisterController(ctx context.Context, controller model.Controller) error
	UpdateControllerHeartbeat(ctx context.Context, controllerID string, heartbeat model.Heartbeat) error
	ListControllers(ctx context.Context, filter model.ControllerFilter) ([]model.Controller, error)
}
