package api_test

import (
	"context"
	"errors"
	"testing"

	"github.com/flames-hq/flames/api"
	"github.com/flames-hq/flames/model"
	"github.com/flames-hq/flames/provider/providererr"
	"github.com/flames-hq/flames/provider/queue/memqueue"
	"github.com/flames-hq/flames/provider/state/memstate"
)

func newService() *api.Service {
	return api.New(memstate.New(), memqueue.New())
}

// AC-001: CreateVM returns an ID and GetVM returns pending state.
func TestCreateAndGetVM(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	spec := model.VMSpec{
		Resources: model.ResourceSpec{VCPUs: 2, MemoryMB: 512},
	}

	id, err := svc.CreateVM(ctx, spec)
	if err != nil {
		t.Fatalf("CreateVM: %v", err)
	}
	if id == "" {
		t.Fatal("CreateVM returned empty ID")
	}

	vm, err := svc.GetVM(ctx, id)
	if err != nil {
		t.Fatalf("GetVM: %v", err)
	}
	if vm.ObservedState != model.ObservedPending {
		t.Errorf("got observed state %q, want %q", vm.ObservedState, model.ObservedPending)
	}
	if vm.DesiredState != model.DesiredRunning {
		t.Errorf("got desired state %q, want %q", vm.DesiredState, model.DesiredRunning)
	}
	if vm.Spec.Resources.VCPUs != 2 {
		t.Errorf("got VCPUs %d, want 2", vm.Spec.Resources.VCPUs)
	}
}

// AC-002: StopVM sets desired state to stopped.
func TestStopVM(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	id, _ := svc.CreateVM(ctx, model.VMSpec{})
	if err := svc.StopVM(ctx, id); err != nil {
		t.Fatalf("StopVM: %v", err)
	}

	vm, _ := svc.GetVM(ctx, id)
	if vm.DesiredState != model.DesiredStopped {
		t.Errorf("got %q, want %q", vm.DesiredState, model.DesiredStopped)
	}
}

// AC-003: DeleteVM sets desired state to deleted.
func TestDeleteVM(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	id, _ := svc.CreateVM(ctx, model.VMSpec{})
	if err := svc.DeleteVM(ctx, id); err != nil {
		t.Fatalf("DeleteVM: %v", err)
	}

	vm, _ := svc.GetVM(ctx, id)
	if vm.DesiredState != model.DesiredDeleted {
		t.Errorf("got %q, want %q", vm.DesiredState, model.DesiredDeleted)
	}
}

// AC-004: GetVM returns not_found for nonexistent VM.
func TestGetVMNotFound(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	_, err := svc.GetVM(ctx, "nonexistent")
	if !errors.Is(err, providererr.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// AC-005: RegisterController makes the controller visible in ListControllers.
func TestRegisterAndListControllers(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	ctrl := model.Controller{
		ID:     "ctrl-1",
		Status: "active",
		Capacity: model.CapacityInfo{
			TotalVCPUs:    16,
			TotalMemoryMB: 32768,
		},
	}

	if err := svc.RegisterController(ctx, ctrl); err != nil {
		t.Fatalf("RegisterController: %v", err)
	}

	controllers, err := svc.ListControllers(ctx, model.ControllerFilter{})
	if err != nil {
		t.Fatalf("ListControllers: %v", err)
	}
	if len(controllers) != 1 {
		t.Fatalf("expected 1 controller, got %d", len(controllers))
	}
	if controllers[0].ID != "ctrl-1" {
		t.Errorf("got ID %q, want %q", controllers[0].ID, "ctrl-1")
	}
}

// AC-006: Heartbeat updates controller capacity.
func TestHeartbeatUpdatesCapacity(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	ctrl := model.Controller{
		ID:     "ctrl-1",
		Status: "active",
		Capacity: model.CapacityInfo{
			TotalVCPUs:    16,
			TotalMemoryMB: 32768,
		},
	}
	_ = svc.RegisterController(ctx, ctrl)

	hb := model.Heartbeat{
		Status: "active",
		Capacity: model.CapacityInfo{
			TotalVCPUs:    16,
			TotalMemoryMB: 32768,
			UsedVCPUs:     4,
			UsedMemoryMB:  8192,
		},
	}
	if err := svc.Heartbeat(ctx, "ctrl-1", hb); err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}

	controllers, _ := svc.ListControllers(ctx, model.ControllerFilter{})
	if controllers[0].Capacity.UsedVCPUs != 4 {
		t.Errorf("got UsedVCPUs %d, want 4", controllers[0].Capacity.UsedVCPUs)
	}
}

// Heartbeat for nonexistent controller returns not_found.
func TestHeartbeatNotFound(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	err := svc.Heartbeat(ctx, "nonexistent", model.Heartbeat{})
	if !errors.Is(err, providererr.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// AC-007: ListEvents filters by VM ID and respects limit.
func TestListEventsWithFilter(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	// Create two VMs to generate events.
	id1, _ := svc.CreateVM(ctx, model.VMSpec{})
	_, _ = svc.CreateVM(ctx, model.VMSpec{})
	_ = svc.StopVM(ctx, id1)

	// id1 should have vm.created + vm.stop_requested = 2 events
	events, err := svc.ListEvents(ctx, model.EventFilter{VMID: id1, Limit: 10})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events for vm %s, got %d", id1, len(events))
	}

	// Limit to 1.
	events, _ = svc.ListEvents(ctx, model.EventFilter{VMID: id1, Limit: 1})
	if len(events) != 1 {
		t.Errorf("expected 1 event with limit, got %d", len(events))
	}
}

// Verify event types are emitted correctly.
func TestEventEmission(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	id, _ := svc.CreateVM(ctx, model.VMSpec{})
	_ = svc.StopVM(ctx, id)
	_ = svc.DeleteVM(ctx, id)

	events, _ := svc.ListEvents(ctx, model.EventFilter{VMID: id})

	want := []string{"vm.created", "vm.stop_requested", "vm.delete_requested"}
	if len(events) != len(want) {
		t.Fatalf("expected %d events, got %d", len(want), len(events))
	}
	for i, e := range events {
		if e.Type != want[i] {
			t.Errorf("event[%d] type = %q, want %q", i, e.Type, want[i])
		}
	}
}

// Default limit is applied when zero.
func TestListEventsDefaultLimit(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	// Create 150 VMs (each emits 1 event).
	for range 150 {
		_, _ = svc.CreateVM(ctx, model.VMSpec{})
	}

	events, _ := svc.ListEvents(ctx, model.EventFilter{})
	if len(events) != 100 {
		t.Errorf("expected default limit 100, got %d", len(events))
	}
}
