// Package api provides the transport-agnostic service layer for the Flames control plane.
package api

import (
	"context"

	"github.com/flames-hq/flames/model"
	"github.com/flames-hq/flames/provider/state"
	"github.com/flames-hq/flames/provider/queue"
)

// Service contains all API business logic. It speaks only in domain types and
// delegates persistence to provider interfaces injected via the constructor.
type Service struct {
	state state.StateStore
	queue queue.WorkQueue
}

// New creates a new Service with the given providers.
func New(state state.StateStore, queue queue.WorkQueue) *Service {
	return &Service{state: state, queue: queue}
}

// CreateVM creates a new VM from the given spec. It emits a "vm.created" event
// and returns the generated VM ID.
func (s *Service) CreateVM(ctx context.Context, spec model.VMSpec) (string, error) {
	id, err := s.state.CreateVM(ctx, spec)
	if err != nil {
		return "", err
	}

	_ = s.state.AppendEvent(ctx, model.Event{
		VMID: id,
		Type: "vm.created",
	})

	return id, nil
}

// GetVM returns the full VM resource by ID.
func (s *Service) GetVM(ctx context.Context, vmID string) (model.VM, error) {
	return s.state.GetVM(ctx, vmID)
}

// StopVM sets the VM's desired state to stopped and emits a "vm.stop_requested" event.
func (s *Service) StopVM(ctx context.Context, vmID string) error {
	if err := s.state.UpdateVMDesiredState(ctx, vmID, model.DesiredStopped); err != nil {
		return err
	}

	_ = s.state.AppendEvent(ctx, model.Event{
		VMID: vmID,
		Type: "vm.stop_requested",
	})

	return nil
}

// DeleteVM sets the VM's desired state to deleted and emits a "vm.delete_requested" event.
func (s *Service) DeleteVM(ctx context.Context, vmID string) error {
	if err := s.state.UpdateVMDesiredState(ctx, vmID, model.DesiredDeleted); err != nil {
		return err
	}

	_ = s.state.AppendEvent(ctx, model.Event{
		VMID: vmID,
		Type: "vm.delete_requested",
	})

	return nil
}

// RegisterController registers a new controller with the control plane.
func (s *Service) RegisterController(ctx context.Context, c model.Controller) error {
	return s.state.RegisterController(ctx, c)
}

// Heartbeat updates a controller's status and capacity.
func (s *Service) Heartbeat(ctx context.Context, controllerID string, hb model.Heartbeat) error {
	return s.state.UpdateControllerHeartbeat(ctx, controllerID, hb)
}

const (
	defaultListLimit = 100
	maxListLimit     = 1000
)

// ListControllers returns controllers matching the given filter.
func (s *Service) ListControllers(ctx context.Context, filter model.ControllerFilter) ([]model.Controller, error) {
	filter.Limit = clampLimit(filter.Limit)
	return s.state.ListControllers(ctx, filter)
}

// ListEvents returns events matching the given filter.
func (s *Service) ListEvents(ctx context.Context, filter model.EventFilter) ([]model.Event, error) {
	filter.Limit = clampLimit(filter.Limit)
	return s.state.ListEvents(ctx, filter)
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return defaultListLimit
	}
	if limit > maxListLimit {
		return maxListLimit
	}
	return limit
}
