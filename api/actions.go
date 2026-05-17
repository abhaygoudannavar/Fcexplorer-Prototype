package api

import (
	"encoding/json"
	"fmt"
)

// CheckVM does GET / to see if Firecracker is running.
// This is the first thing you'd call to verify the socket is alive.
func CheckVM(c *Client) error {
	fmt.Println("==> Checking Firecracker status (GET /)")

	body, status, err := c.Request("GET", "/", nil)
	if err != nil {
		return fmt.Errorf("check vm: %v", err)
	}

	if !c.dryRun {
		fmt.Printf("    Status: %d\n", status)
		fmt.Printf("    Response: %s\n\n", string(body))
	}
	return nil
}

// SetMachineConfig configures the VM hardware.
// This must be done before booting.
func SetMachineConfig(c *Client, vcpus int, memMB int) error {
	fmt.Printf("==> Setting machine config: %d vCPUs, %d MB RAM (PUT /machine-config)\n", vcpus, memMB)

	cfg := MachineConfig{
		VcpuCount:  vcpus,
		MemSizeMib: memMB,
		HtEnabled:  false,
	}

	body, status, err := c.Request("PUT", "/machine-config", cfg)
	if err != nil {
		return fmt.Errorf("set machine config: %v", err)
	}

	if !c.dryRun {
		fmt.Printf("    Status: %d\n", status)
		if len(body) > 0 {
			fmt.Printf("    Response: %s\n", string(body))
		}
		fmt.Println()
	}
	return nil
}

// SetBootSource tells Firecracker which kernel to boot.
// The kernel must be an uncompressed vmlinux file, not a bzImage.
func SetBootSource(c *Client, kernelPath string) error {
	fmt.Printf("==> Setting boot source: %s (PUT /boot-source)\n", kernelPath)

	boot := BootSource{
		KernelImagePath: kernelPath,
		// console=ttyS0 sends kernel output to the serial port so we can see it
		BootArgs: "console=ttyS0 reboot=k panic=1 pci=off",
	}

	body, status, err := c.Request("PUT", "/boot-source", boot)
	if err != nil {
		return fmt.Errorf("set boot source: %v", err)
	}

	if !c.dryRun {
		fmt.Printf("    Status: %d\n", status)
		if len(body) > 0 {
			fmt.Printf("    Response: %s\n", string(body))
		}
		fmt.Println()
	}
	return nil
}

// AttachRootFS attaches a root filesystem image to the VM.
func AttachRootFS(c *Client, rootfsPath string) error {
	fmt.Printf("==> Attaching rootfs: %s (PUT /drives/rootfs)\n", rootfsPath)

	drive := Drive{
		DriveID:      "rootfs",
		PathOnHost:   rootfsPath,
		IsRootDevice: true,
		IsReadOnly:   false,
	}

	body, status, err := c.Request("PUT", "/drives/rootfs", drive)
	if err != nil {
		return fmt.Errorf("attach rootfs: %v", err)
	}

	if !c.dryRun {
		fmt.Printf("    Status: %d\n", status)
		if len(body) > 0 {
			fmt.Printf("    Response: %s\n", string(body))
		}
		fmt.Println()
	}
	return nil
}

// StartVM sends the InstanceStart action to boot the microVM.
// All config (machine, boot source, drives) must be set before calling this.
func StartVM(c *Client) error {
	fmt.Println("==> Starting VM (PUT /actions)")

	action := Action{
		ActionType: "InstanceStart",
	}

	body, status, err := c.Request("PUT", "/actions", action)
	if err != nil {
		return fmt.Errorf("start vm: %v", err)
	}

	if !c.dryRun {
		fmt.Printf("    Status: %d\n", status)
		if len(body) > 0 {
			fmt.Printf("    Response: %s\n", string(body))
		}
		fmt.Println()
	}
	return nil
}

// GetMachineConfig reads back the current VM config.
// Useful to verify that our PUT actually worked.
func GetMachineConfig(c *Client) error {
	fmt.Println("==> Reading back machine config (GET /machine-config)")

	body, status, err := c.Request("GET", "/machine-config", nil)
	if err != nil {
		return fmt.Errorf("get machine config: %v", err)
	}

	if !c.dryRun {
		fmt.Printf("    Status: %d\n", status)

		// pretty print the JSON
		var cfg MachineConfig
		if json.Unmarshal(body, &cfg) == nil {
			fmt.Printf("    vCPUs: %d\n", cfg.VcpuCount)
			fmt.Printf("    Memory: %d MB\n", cfg.MemSizeMib)
			fmt.Printf("    HT: %v\n", cfg.HtEnabled)
		} else {
			fmt.Printf("    Raw: %s\n", string(body))
		}
		fmt.Println()
	}
	return nil
}

// DemoRawRequest shows how to talk to the API using raw TCP.
// This is just for learning — the http.Client approach above is
// what you'd actually use in production.
func DemoRawRequest(c *Client) error {
	fmt.Println("==> Demo: Raw HTTP request (GET / using net.Dial)")

	cfg := MachineConfig{VcpuCount: 2, MemSizeMib: 256}
	data, _ := json.Marshal(cfg)

	resp, err := c.RawRequest("GET", "/", nil)
	if err != nil {
		return fmt.Errorf("raw request: %v", err)
	}

	if !c.dryRun {
		fmt.Printf("    Raw response:\n%s\n\n", resp)
	}

	// also show what a PUT would look like as raw bytes
	fmt.Println("==> Demo: Raw HTTP request (PUT /machine-config using net.Dial)")
	resp, err = c.RawRequest("PUT", "/machine-config", data)
	if err != nil {
		return fmt.Errorf("raw put request: %v", err)
	}

	if !c.dryRun {
		fmt.Printf("    Raw response:\n%s\n\n", resp)
	}

	return nil
}
