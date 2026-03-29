package model

import "time"

type DesiredState string

const (
	DesiredRunning DesiredState = "running"
	DesiredStopped DesiredState = "stopped"
	DesiredDeleted DesiredState = "deleted"
)

type ObservedState string

const (
	ObservedPending   ObservedState = "pending"
	ObservedScheduled ObservedState = "scheduled"
	ObservedPreparing ObservedState = "preparing"
	ObservedStarting  ObservedState = "starting"
	ObservedRunning   ObservedState = "running"
	ObservedStopping  ObservedState = "stopping"
	ObservedStopped   ObservedState = "stopped"
	ObservedFailed    ObservedState = "failed"
	ObservedDeleted   ObservedState = "deleted"
)

// VM represents a virtual machine managed by the control plane. It tracks both the
// user-requested desired state and the actual observed state as reported by the assigned controller.
type VM struct {
	ID            string        `json:"id"`
	DesiredState  DesiredState  `json:"desired_state"`
	ObservedState ObservedState `json:"observed_state"`
	ImageID       string        `json:"image_id"`
	ControllerID  string        `json:"controller_id"`
	Spec          VMSpec        `json:"spec"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// VMSpec defines the full configuration for a VM including compute resources, lifecycle behavior,
// runtime parameters, networking, storage, and placement constraints. Treated as opaque by the
// StateStore — it is stored and returned as-is, never interpreted by the storage layer.
type VMSpec struct {
	Resources ResourceSpec      `json:"resources"`
	Lifecycle LifecycleSpec     `json:"lifecycle"`
	Runtime   RuntimeSpec       `json:"runtime"`
	Network   NetworkSpec       `json:"network"`
	Storage   StorageSpec       `json:"storage"`
	Placement PlacementSpec     `json:"placement"`
	Metadata  map[string]string `json:"metadata"`
}

// ResourceSpec defines the compute resources allocated to a VM.
type ResourceSpec struct {
	VCPUs    int `json:"vcpus"`
	MemoryMB int `json:"memory_mb"`
}

// RestartPolicy determines how the controller handles a VM that exits.
type RestartPolicy string

const (
	RestartNever     RestartPolicy = "never"
	RestartOnFailure RestartPolicy = "on_failure"
	RestartAlways    RestartPolicy = "always"
)

// LifecycleSpec controls VM lifetime behavior: maximum runtime, automatic cleanup, and restart policy.
type LifecycleSpec struct {
	TimeoutSeconds int           `json:"timeout_seconds"`
	AutoDelete     bool          `json:"auto_delete"`
	RestartPolicy  RestartPolicy `json:"restart_policy"`
}

// RuntimeSpec defines the command, arguments, and environment variables to execute inside the VM.
type RuntimeSpec struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// NetworkSpec defines the network configuration for a VM, including port mappings.
type NetworkSpec struct {
	Ports []PortMapping `json:"ports"`
}

// PortMapping maps a guest port inside the VM to a host port with a given protocol.
type PortMapping struct {
	Guest    int    `json:"guest"`
	Host     int    `json:"host"`
	Protocol string `json:"protocol"`
}

// StorageSpec defines the storage configuration for a VM.
type StorageSpec struct {
	RootDiskSizeMB int `json:"root_disk_size_mb"`
}

// PlacementSpec defines constraints for scheduling the VM onto a controller node via label matching.
type PlacementSpec struct {
	Labels map[string]string `json:"labels"`
}
