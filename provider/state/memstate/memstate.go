// Package memstate provides an in-memory StateStore implementation for development mode.
package memstate

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/flames-hq/flames/model"
	"github.com/flames-hq/flames/provider/providererr"
	"github.com/flames-hq/flames/provider/state"
)

var _ state.StateStore = (*Store)(nil)

// Store is a mutex-protected in-memory StateStore backed by Go maps and slices.
type Store struct {
	mu          sync.RWMutex
	vms         map[string]model.VM
	controllers map[string]model.Controller
	events      []model.Event
}

// New creates a new in-memory StateStore.
func New() *Store {
	return &Store{
		vms:         make(map[string]model.VM),
		controllers: make(map[string]model.Controller),
	}
}

// CreateVM creates a new VM with the given spec, generating a random ID and setting initial state to pending.
func (s *Store) CreateVM(_ context.Context, spec model.VMSpec) (string, error) {
	id := newID()
	now := time.Now()

	vm := model.VM{
		ID:            id,
		DesiredState:  model.DesiredRunning,
		ObservedState: model.ObservedPending,
		Spec:          spec,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	s.mu.Lock()
	s.vms[id] = vm
	s.mu.Unlock()

	return id, nil
}

// GetVM returns a VM by ID. Returns ErrNotFound if the VM does not exist.
func (s *Store) GetVM(_ context.Context, vmID string) (model.VM, error) {
	s.mu.RLock()
	vm, ok := s.vms[vmID]
	s.mu.RUnlock()

	if !ok {
		return model.VM{}, providererr.NotFound("vm", vmID)
	}
	return vm, nil
}

// UpdateVMDesiredState sets the desired state for a VM. Returns ErrNotFound if the VM does not exist.
func (s *Store) UpdateVMDesiredState(_ context.Context, vmID string, desired model.DesiredState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	vm, ok := s.vms[vmID]
	if !ok {
		return providererr.NotFound("vm", vmID)
	}
	vm.DesiredState = desired
	vm.UpdatedAt = time.Now()
	s.vms[vmID] = vm
	return nil
}

// UpdateVMObservedState sets the observed state and reporting controller for a VM. Returns ErrNotFound if the VM does not exist.
func (s *Store) UpdateVMObservedState(_ context.Context, vmID string, observed model.ObservedState, controllerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	vm, ok := s.vms[vmID]
	if !ok {
		return providererr.NotFound("vm", vmID)
	}
	vm.ObservedState = observed
	vm.ControllerID = controllerID
	vm.UpdatedAt = time.Now()
	s.vms[vmID] = vm
	return nil
}

// AssignVM atomically assigns a VM to a controller. Returns ErrConflict if the VM is already assigned.
func (s *Store) AssignVM(_ context.Context, vmID string, controllerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	vm, ok := s.vms[vmID]
	if !ok {
		return providererr.NotFound("vm", vmID)
	}
	if vm.ControllerID != "" {
		return providererr.Conflict("vm", vmID, "vm already assigned to controller "+vm.ControllerID)
	}
	vm.ControllerID = controllerID
	vm.UpdatedAt = time.Now()
	s.vms[vmID] = vm
	return nil
}

// ListPendingVMs returns all VMs with observed state "pending".
func (s *Store) ListPendingVMs(_ context.Context) ([]model.VM, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []model.VM
	for _, vm := range s.vms {
		if vm.ObservedState == model.ObservedPending {
			result = append(result, vm)
		}
	}
	return result, nil
}

// AppendEvent appends an event to the event log, auto-generating an ID and timestamp if not set.
func (s *Store) AppendEvent(_ context.Context, event model.Event) error {
	if event.ID == "" {
		event.ID = newID()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	s.mu.Lock()
	s.events = append(s.events, event)
	s.mu.Unlock()

	return nil
}

// ListEvents returns events matching the given filter criteria.
func (s *Store) ListEvents(_ context.Context, filter model.EventFilter) ([]model.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []model.Event
	for _, e := range s.events {
		if filter.VMID != "" && e.VMID != filter.VMID {
			continue
		}
		if filter.ControllerID != "" && e.ControllerID != filter.ControllerID {
			continue
		}
		if filter.Type != "" && e.Type != filter.Type {
			continue
		}
		if !filter.Since.IsZero() && e.CreatedAt.Before(filter.Since) {
			continue
		}
		result = append(result, e)
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}
	return result, nil
}

// RegisterController registers a new controller. Returns ErrAlreadyExists if the controller ID is already registered.
func (s *Store) RegisterController(_ context.Context, controller model.Controller) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.controllers[controller.ID]; ok {
		return providererr.AlreadyExists("controller", controller.ID)
	}
	s.controllers[controller.ID] = controller
	return nil
}

// UpdateControllerHeartbeat updates a controller's status, capacity, and last heartbeat timestamp.
func (s *Store) UpdateControllerHeartbeat(_ context.Context, controllerID string, heartbeat model.Heartbeat) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.controllers[controllerID]
	if !ok {
		return providererr.NotFound("controller", controllerID)
	}
	c.Status = heartbeat.Status
	c.Capacity = heartbeat.Capacity
	c.LastHeartbeatAt = time.Now()
	s.controllers[controllerID] = c
	return nil
}

// ListControllers returns controllers matching the given filter criteria.
func (s *Store) ListControllers(_ context.Context, filter model.ControllerFilter) ([]model.Controller, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []model.Controller
	for _, c := range s.controllers {
		if filter.Status != "" && c.Status != filter.Status {
			continue
		}
		result = append(result, c)
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}
	return result, nil
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
