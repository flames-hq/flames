package statetest

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flames-hq/flames/model"
	"github.com/flames-hq/flames/provider/providererr"
	"github.com/flames-hq/flames/provider/state"
)

func Run(t *testing.T, newStore func() state.StateStore) {
	t.Run("CreateAndGet", func(t *testing.T) { testCreateAndGet(t, newStore()) })
	t.Run("UpdateDesiredState", func(t *testing.T) { testUpdateDesiredState(t, newStore()) })
	t.Run("UpdateObservedState", func(t *testing.T) { testUpdateObservedState(t, newStore()) })
	t.Run("AssignVM", func(t *testing.T) { testAssignVM(t, newStore()) })
	t.Run("ConcurrentAssign", func(t *testing.T) { testConcurrentAssign(t, newStore()) })
	t.Run("ListPendingVMs", func(t *testing.T) { testListPendingVMs(t, newStore()) })
	t.Run("Events", func(t *testing.T) { testEvents(t, newStore()) })
	t.Run("Controllers", func(t *testing.T) { testControllers(t, newStore()) })
	t.Run("NotFoundErrors", func(t *testing.T) { testNotFoundErrors(t, newStore()) })
}

func testCreateAndGet(t *testing.T, s state.StateStore) {
	ctx := context.Background()
	spec := model.VMSpec{
		Resources: model.ResourceSpec{VCPUs: 2, MemoryMB: 512},
	}

	id, err := s.CreateVM(ctx, spec)
	if err != nil {
		t.Fatalf("CreateVM: %v", err)
	}
	if id == "" {
		t.Fatal("CreateVM returned empty ID")
	}

	vm, err := s.GetVM(ctx, id)
	if err != nil {
		t.Fatalf("GetVM: %v", err)
	}
	if vm.ID != id {
		t.Errorf("got ID %q, want %q", vm.ID, id)
	}
	if vm.DesiredState != model.DesiredRunning {
		t.Errorf("got DesiredState %q, want %q", vm.DesiredState, model.DesiredRunning)
	}
	if vm.ObservedState != model.ObservedPending {
		t.Errorf("got ObservedState %q, want %q", vm.ObservedState, model.ObservedPending)
	}
	if vm.Spec.Resources.VCPUs != 2 {
		t.Errorf("got VCPUs %d, want 2", vm.Spec.Resources.VCPUs)
	}
}

func testUpdateDesiredState(t *testing.T, s state.StateStore) {
	ctx := context.Background()
	id, _ := s.CreateVM(ctx, model.VMSpec{})

	if err := s.UpdateVMDesiredState(ctx, id, model.DesiredStopped); err != nil {
		t.Fatalf("UpdateVMDesiredState: %v", err)
	}

	vm, _ := s.GetVM(ctx, id)
	if vm.DesiredState != model.DesiredStopped {
		t.Errorf("got %q, want %q", vm.DesiredState, model.DesiredStopped)
	}
}

func testUpdateObservedState(t *testing.T, s state.StateStore) {
	ctx := context.Background()
	id, _ := s.CreateVM(ctx, model.VMSpec{})

	if err := s.UpdateVMObservedState(ctx, id, model.ObservedRunning, "ctrl-1"); err != nil {
		t.Fatalf("UpdateVMObservedState: %v", err)
	}

	vm, _ := s.GetVM(ctx, id)
	if vm.ObservedState != model.ObservedRunning {
		t.Errorf("got %q, want %q", vm.ObservedState, model.ObservedRunning)
	}
	if vm.ControllerID != "ctrl-1" {
		t.Errorf("got ControllerID %q, want %q", vm.ControllerID, "ctrl-1")
	}
}

func testAssignVM(t *testing.T, s state.StateStore) {
	ctx := context.Background()
	id, _ := s.CreateVM(ctx, model.VMSpec{})

	if err := s.AssignVM(ctx, id, "ctrl-1"); err != nil {
		t.Fatalf("AssignVM: %v", err)
	}

	vm, _ := s.GetVM(ctx, id)
	if vm.ControllerID != "ctrl-1" {
		t.Errorf("got ControllerID %q, want %q", vm.ControllerID, "ctrl-1")
	}

	// Second assign should conflict.
	err := s.AssignVM(ctx, id, "ctrl-2")
	if !errors.Is(err, providererr.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func testConcurrentAssign(t *testing.T, s state.StateStore) {
	ctx := context.Background()
	id, _ := s.CreateVM(ctx, model.VMSpec{})

	const goroutines = 10
	var successes atomic.Int32
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(n int) {
			defer wg.Done()
			err := s.AssignVM(ctx, id, "ctrl-"+string(rune('A'+n)))
			if err == nil {
				successes.Add(1)
			}
		}(i)
	}

	wg.Wait()

	if got := successes.Load(); got != 1 {
		t.Errorf("expected exactly 1 successful assign, got %d", got)
	}

	// VM should not be corrupted.
	vm, err := s.GetVM(ctx, id)
	if err != nil {
		t.Fatalf("GetVM after concurrent assign: %v", err)
	}
	if vm.ControllerID == "" {
		t.Error("ControllerID is empty after concurrent assign")
	}
}

func testListPendingVMs(t *testing.T, s state.StateStore) {
	ctx := context.Background()

	id1, _ := s.CreateVM(ctx, model.VMSpec{})
	id2, _ := s.CreateVM(ctx, model.VMSpec{})
	_ = s.UpdateVMObservedState(ctx, id2, model.ObservedRunning, "ctrl-1")

	pending, err := s.ListPendingVMs(ctx)
	if err != nil {
		t.Fatalf("ListPendingVMs: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending VM, got %d", len(pending))
	}
	if pending[0].ID != id1 {
		t.Errorf("expected pending VM %q, got %q", id1, pending[0].ID)
	}
}

func testEvents(t *testing.T, s state.StateStore) {
	ctx := context.Background()

	e1 := model.Event{VMID: "vm-1", Type: "vm.created", Payload: []byte(`{}`)}
	e2 := model.Event{VMID: "vm-2", Type: "vm.scheduled", Payload: []byte(`{}`)}
	e3 := model.Event{VMID: "vm-1", Type: "vm.started", Payload: []byte(`{}`)}

	for _, e := range []model.Event{e1, e2, e3} {
		if err := s.AppendEvent(ctx, e); err != nil {
			t.Fatalf("AppendEvent: %v", err)
		}
	}

	// Filter by VMID.
	events, err := s.ListEvents(ctx, model.EventFilter{VMID: "vm-1"})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events for vm-1, got %d", len(events))
	}

	// Filter by type.
	events, err = s.ListEvents(ctx, model.EventFilter{Type: "vm.scheduled"})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event of type vm.scheduled, got %d", len(events))
	}

	// Limit.
	events, err = s.ListEvents(ctx, model.EventFilter{Limit: 2})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events with limit, got %d", len(events))
	}
}

func testControllers(t *testing.T, s state.StateStore) {
	ctx := context.Background()

	ctrl := model.Controller{
		ID:     "ctrl-1",
		Status: "active",
		Capacity: model.CapacityInfo{
			TotalVCPUs:    16,
			TotalMemoryMB: 32768,
		},
	}

	if err := s.RegisterController(ctx, ctrl); err != nil {
		t.Fatalf("RegisterController: %v", err)
	}

	// Duplicate should fail.
	err := s.RegisterController(ctx, ctrl)
	if !errors.Is(err, providererr.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists, got %v", err)
	}

	// Heartbeat.
	hb := model.Heartbeat{
		Status:   "active",
		Capacity: model.CapacityInfo{TotalVCPUs: 16, TotalMemoryMB: 32768, UsedVCPUs: 4, UsedMemoryMB: 8192},
	}
	before := time.Now()
	if err := s.UpdateControllerHeartbeat(ctx, "ctrl-1", hb); err != nil {
		t.Fatalf("UpdateControllerHeartbeat: %v", err)
	}
	_ = before // heartbeat timestamp is set internally
}

func testNotFoundErrors(t *testing.T, s state.StateStore) {
	ctx := context.Background()

	_, err := s.GetVM(ctx, "nonexistent")
	if !errors.Is(err, providererr.ErrNotFound) {
		t.Errorf("GetVM: expected ErrNotFound, got %v", err)
	}

	err = s.UpdateVMDesiredState(ctx, "nonexistent", model.DesiredStopped)
	if !errors.Is(err, providererr.ErrNotFound) {
		t.Errorf("UpdateVMDesiredState: expected ErrNotFound, got %v", err)
	}

	err = s.UpdateVMObservedState(ctx, "nonexistent", model.ObservedRunning, "ctrl-1")
	if !errors.Is(err, providererr.ErrNotFound) {
		t.Errorf("UpdateVMObservedState: expected ErrNotFound, got %v", err)
	}

	err = s.AssignVM(ctx, "nonexistent", "ctrl-1")
	if !errors.Is(err, providererr.ErrNotFound) {
		t.Errorf("AssignVM: expected ErrNotFound, got %v", err)
	}

	err = s.UpdateControllerHeartbeat(ctx, "nonexistent", model.Heartbeat{})
	if !errors.Is(err, providererr.ErrNotFound) {
		t.Errorf("UpdateControllerHeartbeat: expected ErrNotFound, got %v", err)
	}
}
