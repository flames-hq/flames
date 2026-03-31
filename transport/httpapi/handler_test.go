package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/flames-hq/flames/api"
	"github.com/flames-hq/flames/model"
	"github.com/flames-hq/flames/provider/queue/memqueue"
	"github.com/flames-hq/flames/provider/state/memstate"
	"github.com/flames-hq/flames/transport/httpapi"
)

// RequestOptions configures an HTTP request (method, headers, etc.).
type RequestOptions struct {
	Method  string
	Headers map[string]string
}

func newTestServer() *httptest.Server {
	svc := api.New(memstate.New(), memqueue.New())
	handler := httpapi.NewHandler(svc)
	return httptest.NewServer(handler)
}

func request(ts *httptest.Server, path string, body any, opts *RequestOptions) *http.Response {
	method := "GET"
	var headers map[string]string

	if opts != nil {
		if opts.Method != "" {
			method = opts.Method
		}
		headers = opts.Headers
	}

	var reqBody *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req, _ := http.NewRequest(method, ts.URL+path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, _ := http.DefaultClient.Do(req)
	return resp
}

func post(ts *httptest.Server, path string, body any, opts *RequestOptions) *http.Response {
	if opts == nil {
		opts = &RequestOptions{}
	}
	opts.Method = "POST"
	return request(ts, path, body, opts)
}

func get(ts *httptest.Server, path string) *http.Response {
	return request(ts, path, nil, nil)
}

func del(ts *httptest.Server, path string, opts *RequestOptions) *http.Response {
	if opts == nil {
		opts = &RequestOptions{}
	}
	opts.Method = "DELETE"
	return request(ts, path, nil, opts)
}

func decodeBody(resp *http.Response, v any) {
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(v)
}

// AC-008: POST /v1/vms returns 202 with id.
func TestCreateVM(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	spec := model.VMSpec{Resources: model.ResourceSpec{VCPUs: 2, MemoryMB: 512}}
	resp := post(ts, "/v1/vms", spec, nil)

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("got status %d, want 202", resp.StatusCode)
	}

	var body map[string]string
	decodeBody(resp, &body)
	if body["id"] == "" {
		t.Fatal("response missing 'id' field")
	}

	// Verify the VM is retrievable.
	getResp := get(ts, "/v1/vms/"+body["id"])
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/vms/%s: got status %d, want 200", body["id"], getResp.StatusCode)
	}

	var vm model.VM
	decodeBody(getResp, &vm)
	if vm.ObservedState != model.ObservedPending {
		t.Errorf("got observed state %q, want %q", vm.ObservedState, model.ObservedPending)
	}
}

// AC-009: GET /v1/vms/{nonexistent} returns 404 with structured error.
func TestGetVMNotFound(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	resp := get(ts, "/v1/vms/nonexistent")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("got status %d, want 404", resp.StatusCode)
	}

	var body map[string]string
	decodeBody(resp, &body)
	if body["code"] != "not_found" {
		t.Errorf("got code %q, want %q", body["code"], "not_found")
	}
	if body["resource_type"] != "vm" {
		t.Errorf("got resource_type %q, want %q", body["resource_type"], "vm")
	}
	if body["resource_id"] != "nonexistent" {
		t.Errorf("got resource_id %q, want %q", body["resource_id"], "nonexistent")
	}
}

// Test stop and delete return 202.
func TestStopAndDeleteVM(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	// Create a VM.
	resp := post(ts, "/v1/vms", model.VMSpec{}, nil)
	var created map[string]string
	decodeBody(resp, &created)
	id := created["id"]

	// Stop.
	stopResp := post(ts, "/v1/vms/"+id+"/stop", nil, nil)
	if stopResp.StatusCode != http.StatusAccepted {
		t.Errorf("stop: got status %d, want 202", stopResp.StatusCode)
	}
	stopResp.Body.Close()

	// Verify desired state changed.
	getResp := get(ts, "/v1/vms/"+id)
	var vm model.VM
	decodeBody(getResp, &vm)
	if vm.DesiredState != model.DesiredStopped {
		t.Errorf("after stop: got desired state %q, want %q", vm.DesiredState, model.DesiredStopped)
	}

	// Delete.
	delResp := del(ts, "/v1/vms/"+id, nil)
	if delResp.StatusCode != http.StatusAccepted {
		t.Errorf("delete: got status %d, want 202", delResp.StatusCode)
	}
	delResp.Body.Close()

	getResp = get(ts, "/v1/vms/"+id)
	decodeBody(getResp, &vm)
	if vm.DesiredState != model.DesiredDeleted {
		t.Errorf("after delete: got desired state %q, want %q", vm.DesiredState, model.DesiredDeleted)
	}
}

// Test controller registration, heartbeat, and listing.
func TestControllerLifecycle(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	ctrl := model.Controller{
		ID:     "ctrl-1",
		Status: "active",
		Capacity: model.CapacityInfo{
			TotalVCPUs:    16,
			TotalMemoryMB: 32768,
		},
	}

	// Register.
	resp := post(ts, "/v1/controllers", ctrl, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: got status %d, want 201", resp.StatusCode)
	}
	resp.Body.Close()

	// Heartbeat.
	hb := model.Heartbeat{
		Status:   "active",
		Capacity: model.CapacityInfo{TotalVCPUs: 16, TotalMemoryMB: 32768, UsedVCPUs: 4, UsedMemoryMB: 8192},
	}
	hbResp := post(ts, "/v1/controllers/ctrl-1/heartbeat", hb, nil)
	if hbResp.StatusCode != http.StatusOK {
		t.Fatalf("heartbeat: got status %d, want 200", hbResp.StatusCode)
	}
	hbResp.Body.Close()

	// List.
	listResp := get(ts, "/v1/controllers")
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list: got status %d, want 200", listResp.StatusCode)
	}
	var controllers []model.Controller
	decodeBody(listResp, &controllers)
	if len(controllers) != 1 {
		t.Fatalf("expected 1 controller, got %d", len(controllers))
	}
	if controllers[0].Capacity.UsedVCPUs != 4 {
		t.Errorf("got UsedVCPUs %d, want 4", controllers[0].Capacity.UsedVCPUs)
	}
}

// Test events listing and filtering.
func TestListEvents(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	// Create two VMs to generate events.
	resp1 := post(ts, "/v1/vms", model.VMSpec{}, nil)
	var c1 map[string]string
	decodeBody(resp1, &c1)

	resp2 := post(ts, "/v1/vms", model.VMSpec{}, nil)
	resp2.Body.Close()

	// Stop the first VM (adds another event).
	post(ts, "/v1/vms/"+c1["id"]+"/stop", nil, nil).Body.Close()

	// Filter by vm_id.
	evResp := get(ts, "/v1/events?vm_id="+c1["id"]+"&limit=10")
	if evResp.StatusCode != http.StatusOK {
		t.Fatalf("list events: got status %d, want 200", evResp.StatusCode)
	}
	var events []model.Event
	decodeBody(evResp, &events)
	if len(events) != 2 {
		t.Errorf("expected 2 events for vm %s, got %d", c1["id"], len(events))
	}
}

// AC-010: Idempotency-Key replay and conflict.
func TestIdempotency(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	spec := model.VMSpec{Resources: model.ResourceSpec{VCPUs: 1, MemoryMB: 256}}

	idemOpts := &RequestOptions{
		Headers: map[string]string{"Idempotency-Key": "abc123"},
	}

	// First request with key.
	resp1 := post(ts, "/v1/vms", spec, idemOpts)
	var body1 map[string]string
	decodeBody(resp1, &body1)

	if resp1.StatusCode != http.StatusAccepted {
		t.Fatalf("first request: got status %d, want 202", resp1.StatusCode)
	}

	// Replay same key, same body → same response.
	resp2 := post(ts, "/v1/vms", spec, idemOpts)
	var body2 map[string]string
	decodeBody(resp2, &body2)

	if resp2.StatusCode != http.StatusAccepted {
		t.Fatalf("replay: got status %d, want 202", resp2.StatusCode)
	}
	if body1["id"] != body2["id"] {
		t.Errorf("replay returned different id: %q vs %q", body1["id"], body2["id"])
	}

	// Same key, different body → 409.
	differentSpec := model.VMSpec{Resources: model.ResourceSpec{VCPUs: 4, MemoryMB: 1024}}
	resp3 := post(ts, "/v1/vms", differentSpec, idemOpts)
	resp3.Body.Close()

	if resp3.StatusCode != http.StatusConflict {
		t.Errorf("different body: got status %d, want 409", resp3.StatusCode)
	}
}

// AC-011: GET /healthz returns 200 with {"status":"ok"}.
func TestHealthz(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	resp := get(ts, "/healthz")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want 200", resp.StatusCode)
	}

	var body map[string]string
	decodeBody(resp, &body)
	if body["status"] != "ok" {
		t.Errorf("got status %q, want %q", body["status"], "ok")
	}
}

// AC-012: Concurrent requests don't race (verified by go test -race).
func TestConcurrentRequests(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			spec := model.VMSpec{Resources: model.ResourceSpec{VCPUs: 1, MemoryMB: 256}}
			resp := post(ts, "/v1/vms", spec, nil)
			resp.Body.Close()
		}()
	}
	wg.Wait()

	// All VMs created — list events to verify no corruption.
	resp := get(ts, "/v1/events?limit=1000")
	var events []model.Event
	decodeBody(resp, &events)
	if len(events) != 20 {
		t.Errorf("expected 20 events from concurrent creates, got %d", len(events))
	}
}
