package model

import "time"

// Controller represents a node in the cluster that runs VMs and reports back to the control plane.
type Controller struct {
	ID              string            `json:"id"`
	Status          string            `json:"status"`
	Labels          map[string]string `json:"labels"`
	Capacity        CapacityInfo      `json:"capacity"`
	LastHeartbeatAt time.Time         `json:"last_heartbeat_at"`
}

// CapacityInfo reports a controller's total and currently used compute resources,
// used by the scheduler to make placement decisions.
type CapacityInfo struct {
	TotalVCPUs    int `json:"total_vcpus"`
	TotalMemoryMB int `json:"total_memory_mb"`
	UsedVCPUs     int `json:"used_vcpus"`
	UsedMemoryMB  int `json:"used_memory_mb"`
}

// Heartbeat is the payload sent by a controller to report its current status and resource usage.
type Heartbeat struct {
	Status   string       `json:"status"`
	Capacity CapacityInfo `json:"capacity"`
}
