package model

// Endpoint represents an ingress entry point that routes external traffic to a VM.
type Endpoint struct {
	VMID   string `json:"vm_id"`
	Route  Route  `json:"route"`
	Target Target `json:"target"`
}

// Route defines the external host and path that incoming requests are matched against.
type Route struct {
	Host string `json:"host"`
	Path string `json:"path"`
}

// Target defines the internal address and port where traffic is forwarded to reach the VM.
type Target struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
}
