package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/abhaygoudannavar/fc-explorer/api"
)

func main() {
	socketPath := flag.String("socket", "/tmp/firecracker.socket", "path to the Firecracker API socket")
	dryRun := flag.Bool("dry-run", false, "print requests without sending them")
	kernelPath := flag.String("kernel", "/path/to/vmlinux", "path to uncompressed kernel image")
	rootfsPath := flag.String("rootfs", "/path/to/rootfs.ext4", "path to root filesystem image")
	vcpus := flag.Int("vcpus", 2, "number of vCPUs")
	mem := flag.Int("mem", 256, "memory in MB")
	rawDemo := flag.Bool("raw-demo", false, "demonstrate raw HTTP over Unix socket")
	flag.Parse()

	fmt.Println("fc-explorer — Firecracker API proof of concept")
	fmt.Printf("Socket: %s\n", *socketPath)
	fmt.Printf("Dry run: %v\n\n", *dryRun)

	client := api.NewClient(*socketPath, *dryRun)

	// The boot sequence for a Firecracker microVM is:
	// 1. Check if Firecracker is running (GET /)
	// 2. Set machine config — vCPUs, memory (PUT /machine-config)
	// 3. Set boot source — kernel path (PUT /boot-source)
	// 4. Attach root filesystem (PUT /drives/rootfs)
	// 5. Start the VM (PUT /actions with InstanceStart)
	// 6. Optionally read back config to verify (GET /machine-config)

	steps := []struct {
		name string
		fn   func() error
	}{
		{"check firecracker", func() error { return api.CheckVM(client) }},
		{"set machine config", func() error { return api.SetMachineConfig(client, *vcpus, *mem) }},
		{"set boot source", func() error { return api.SetBootSource(client, *kernelPath) }},
		{"attach rootfs", func() error { return api.AttachRootFS(client, *rootfsPath) }},
		{"start vm", func() error { return api.StartVM(client) }},
		{"read back config", func() error { return api.GetMachineConfig(client) }},
	}

	for i, step := range steps {
		fmt.Printf("--- Step %d/%d: %s ---\n", i+1, len(steps), step.name)
		if err := step.fn(); err != nil {
			fmt.Fprintf(os.Stderr, "error at step '%s': %v\n", step.name, err)
			if !*dryRun {
				// in dry-run mode we keep going even on errors
				os.Exit(1)
			}
		}
	}

	// optional: show raw HTTP approach
	if *rawDemo {
		fmt.Println("--- Bonus: Raw HTTP Demo ---")
		if err := api.DemoRawRequest(client); err != nil {
			fmt.Fprintf(os.Stderr, "raw demo error: %v\n", err)
		}
	}

	fmt.Println("Done!")
}
