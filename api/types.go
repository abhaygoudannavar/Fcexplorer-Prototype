package api

// These structs match the Firecracker API spec.
// I got the field names from: https://github.com/firecracker-microvm/firecracker/blob/main/src/api_server/swagger/firecracker.yaml
//
// Not all fields are included — just the ones needed for a basic boot sequence.

// MachineConfig sets the VM's hardware: how many CPUs and how much memory.
// Note: Firecracker v1.15+ uses "smt" instead of the old "ht_enabled" field.
// I found this out when the real API returned a 400 with:
//   "unknown field `ht_enabled`, expected one of `vcpu_count`, `mem_size_mib`, `smt`..."
type MachineConfig struct {
	VcpuCount  int  `json:"vcpu_count"`
	MemSizeMib int  `json:"mem_size_mib"`
	Smt        bool `json:"smt"`
}

// BootSource tells Firecracker where to find the kernel and what boot args to pass.
// KernelImagePath must point to an uncompressed kernel (vmlinux, not bzImage).
type BootSource struct {
	KernelImagePath string `json:"kernel_image_path"`
	BootArgs        string `json:"boot_args"`
}

// Drive represents a block device attached to the VM.
// IsRootDevice=true means this is the root filesystem.
type Drive struct {
	DriveID      string `json:"drive_id"`
	PathOnHost   string `json:"path_on_host"`
	IsRootDevice bool   `json:"is_root_device"`
	IsReadOnly   bool   `json:"is_read_only"`
}

// Action is used to tell Firecracker to do something (start, stop, etc).
// For booting, ActionType is "InstanceStart".
type Action struct {
	ActionType string `json:"action_type"`
}

// VMInfo is what Firecracker returns when you GET /.
// It's pretty minimal — mostly just tells you the VM state.
type VMInfo struct {
	ID      string `json:"id"`
	State   string `json:"state"`
	VMState string `json:"vmm_version"`
}

// ErrorResponse is what Firecracker sends back when something goes wrong.
type ErrorResponse struct {
	FaultMessage string `json:"fault_message"`
}
