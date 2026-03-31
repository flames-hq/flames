// Package httpapi provides the HTTP transport adapter for the Flames API service.
package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/flames-hq/flames/api"
	"github.com/flames-hq/flames/model"
)

// Handler is the HTTP transport adapter. It implements http.Handler.
type Handler struct {
	svc  *api.Service
	mux  *http.ServeMux
	idem *idempotencyStore
}

// NewHandler creates a new HTTP handler wired to the given service.
func NewHandler(svc *api.Service) *Handler {
	h := &Handler{
		svc:  svc,
		mux:  http.NewServeMux(),
		idem: newIdempotencyStore(),
	}

	h.mux.HandleFunc("POST /v1/vms", h.idem.wrap(h.createVM))
	h.mux.HandleFunc("GET /v1/vms/{vm_id}", h.getVM)
	h.mux.HandleFunc("POST /v1/vms/{vm_id}/stop", h.idem.wrap(h.stopVM))
	h.mux.HandleFunc("DELETE /v1/vms/{vm_id}", h.idem.wrap(h.deleteVM))
	h.mux.HandleFunc("POST /v1/controllers", h.idem.wrap(h.registerController))
	h.mux.HandleFunc("POST /v1/controllers/{controller_id}/heartbeat", h.heartbeat)
	h.mux.HandleFunc("GET /v1/controllers", h.listControllers)
	h.mux.HandleFunc("GET /v1/events", h.listEvents)
	h.mux.HandleFunc("GET /healthz", h.healthz)

	return h
}

// ServeHTTP dispatches requests through the logging middleware and router.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
	h.mux.ServeHTTP(rec, r)
	log.Printf(`{"method":%q,"path":%q,"status":%d,"duration_ms":%.1f}`,
		r.Method, r.URL.Path, rec.statusCode, float64(time.Since(start).Microseconds())/1000)
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.statusCode = code
		r.wroteHeader = true
		r.ResponseWriter.WriteHeader(code)
	}
}

// --- VM Handlers ---

func (h *Handler) createVM(w http.ResponseWriter, r *http.Request) {
	var spec model.VMSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeErrorMessage(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	id, err := h.svc.CreateVM(r.Context(), spec)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"id": id})
}

func (h *Handler) getVM(w http.ResponseWriter, r *http.Request) {
	vmID := r.PathValue("vm_id")

	vm, err := h.svc.GetVM(r.Context(), vmID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, vm)
}

func (h *Handler) stopVM(w http.ResponseWriter, r *http.Request) {
	vmID := r.PathValue("vm_id")

	if err := h.svc.StopVM(r.Context(), vmID); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"id": vmID})
}

func (h *Handler) deleteVM(w http.ResponseWriter, r *http.Request) {
	vmID := r.PathValue("vm_id")

	if err := h.svc.DeleteVM(r.Context(), vmID); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"id": vmID})
}

// --- Controller Handlers ---

func (h *Handler) registerController(w http.ResponseWriter, r *http.Request) {
	var ctrl model.Controller
	if err := json.NewDecoder(r.Body).Decode(&ctrl); err != nil {
		writeErrorMessage(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	if err := h.svc.RegisterController(r.Context(), ctrl); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, ctrl)
}

func (h *Handler) heartbeat(w http.ResponseWriter, r *http.Request) {
	controllerID := r.PathValue("controller_id")

	var hb model.Heartbeat
	if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
		writeErrorMessage(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	if err := h.svc.Heartbeat(r.Context(), controllerID, hb); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) listControllers(w http.ResponseWriter, r *http.Request) {
	filter := model.ControllerFilter{
		Status: r.URL.Query().Get("status"),
		Limit:  queryInt(r, "limit"),
	}

	controllers, err := h.svc.ListControllers(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, controllers)
}

// --- Event Handlers ---

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := model.EventFilter{
		VMID:         q.Get("vm_id"),
		ControllerID: q.Get("controller_id"),
		Type:         q.Get("type"),
		Limit:        queryInt(r, "limit"),
	}

	if s := q.Get("since"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, "bad_request", "invalid 'since' parameter, expected RFC 3339")
			return
		}
		filter.Since = t
	}

	events, err := h.svc.ListEvents(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, events)
}

// --- Health ---

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func queryInt(r *http.Request, key string) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(s)
	return n
}
