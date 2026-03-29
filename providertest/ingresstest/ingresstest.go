package ingresstest

import (
	"context"
	"errors"
	"testing"

	"github.com/flames-hq/flames/model"
	"github.com/flames-hq/flames/provider/ingress"
	"github.com/flames-hq/flames/provider/providererr"
)

func Run(t *testing.T, newProvider func() ingress.IngressProvider) {
	t.Run("RegisterAndGet", func(t *testing.T) { testRegisterAndGet(t, newProvider()) })
	t.Run("Unregister", func(t *testing.T) { testUnregister(t, newProvider()) })
	t.Run("GetNotFound", func(t *testing.T) { testGetNotFound(t, newProvider()) })
	t.Run("OverwriteEndpoint", func(t *testing.T) { testOverwriteEndpoint(t, newProvider()) })
}

func testRegisterAndGet(t *testing.T, p ingress.IngressProvider) {
	ctx := context.Background()
	route := model.Route{Host: "app.example.com", Path: "/"}
	target := model.Target{Address: "10.0.0.1", Port: 8080}

	ep, err := p.RegisterEndpoint(ctx, "vm-1", route, target)
	if err != nil {
		t.Fatalf("RegisterEndpoint: %v", err)
	}
	if ep.VMID != "vm-1" {
		t.Errorf("got VMID %q, want %q", ep.VMID, "vm-1")
	}
	if ep.Route.Host != "app.example.com" {
		t.Errorf("got Host %q, want %q", ep.Route.Host, "app.example.com")
	}
	if ep.Target.Port != 8080 {
		t.Errorf("got Port %d, want 8080", ep.Target.Port)
	}

	got, err := p.GetEndpoint(ctx, "vm-1")
	if err != nil {
		t.Fatalf("GetEndpoint: %v", err)
	}
	if got.VMID != "vm-1" {
		t.Errorf("got VMID %q, want %q", got.VMID, "vm-1")
	}
}

func testUnregister(t *testing.T, p ingress.IngressProvider) {
	ctx := context.Background()
	route := model.Route{Host: "app.example.com", Path: "/"}
	target := model.Target{Address: "10.0.0.1", Port: 8080}

	_, _ = p.RegisterEndpoint(ctx, "vm-1", route, target)

	if err := p.UnregisterEndpoint(ctx, "vm-1"); err != nil {
		t.Fatalf("UnregisterEndpoint: %v", err)
	}

	_, err := p.GetEndpoint(ctx, "vm-1")
	if !errors.Is(err, providererr.ErrNotFound) {
		t.Errorf("expected ErrNotFound after unregister, got %v", err)
	}
}

func testGetNotFound(t *testing.T, p ingress.IngressProvider) {
	ctx := context.Background()

	_, err := p.GetEndpoint(ctx, "nonexistent")
	if !errors.Is(err, providererr.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func testOverwriteEndpoint(t *testing.T, p ingress.IngressProvider) {
	ctx := context.Background()

	route1 := model.Route{Host: "v1.example.com", Path: "/"}
	target1 := model.Target{Address: "10.0.0.1", Port: 8080}
	_, _ = p.RegisterEndpoint(ctx, "vm-1", route1, target1)

	route2 := model.Route{Host: "v2.example.com", Path: "/api"}
	target2 := model.Target{Address: "10.0.0.2", Port: 9090}
	_, _ = p.RegisterEndpoint(ctx, "vm-1", route2, target2)

	got, err := p.GetEndpoint(ctx, "vm-1")
	if err != nil {
		t.Fatalf("GetEndpoint: %v", err)
	}
	if got.Route.Host != "v2.example.com" {
		t.Errorf("got Host %q, want %q", got.Route.Host, "v2.example.com")
	}
	if got.Target.Port != 9090 {
		t.Errorf("got Port %d, want 9090", got.Target.Port)
	}
}
