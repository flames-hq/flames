package model

import "time"

// Event records a state change or action in the system, such as a VM being created,
// scheduled, or failing. Events are append-only and used for auditing and observability.
type Event struct {
	ID           string    `json:"id"`
	VMID         string    `json:"vm_id"`
	ControllerID string    `json:"controller_id"`
	Type         string    `json:"type"`
	Payload      []byte    `json:"payload"`
	CreatedAt    time.Time `json:"created_at"`
}

// EventFilter defines criteria for querying events. Zero-value fields are ignored,
// allowing callers to combine filters freely.
type EventFilter struct {
	VMID         string    `json:"vm_id"`
	ControllerID string    `json:"controller_id"`
	Type         string    `json:"type"`
	Since        time.Time `json:"since"`
	Limit        int       `json:"limit"`
}
