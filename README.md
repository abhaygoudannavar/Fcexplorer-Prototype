# fc-explorer

A small proof-of-concept that demonstrates how to communicate with the [Firecracker](https://github.com/firecracker-microvm/firecracker) VMM through its HTTP API over a Unix domain socket.

## Why I built this

I'm applying for an LFX mentorship project that involves managing sandbox monitors and their lifecycles. The core of that work is talking to VM monitors through socket APIs — configuring VMs, starting them, watching their state. I built this PoC to prove to myself (and to the mentors) that I understand how this works at the socket level, not just through wrapper libraries.

## How Firecracker's API works

Firecracker exposes a REST API, but instead of listening on a TCP port, it listens on a **Unix domain socket** (usually at `/tmp/firecracker.socket`). The API is plain HTTP/1.1 — you send JSON payloads with PUT requests to configure the VM, then trigger actions.

The boot sequence looks like this:

```
┌────────────────┐         HTTP/1.1 over         ┌──────────────────┐
│  fc-explorer   │ ────── Unix Socket ──────────> │    Firecracker   │
│  (this tool)   │ <───── JSON responses ──────── │    VMM process   │
└────────────────┘                                └────────┬─────────┘
                                                           │
                                                     manages a
                                                           │
                                                  ┌────────▼─────────┐
                                                  │     microVM      │
                                                  │  (KVM-backed)    │
                                                  └──────────────────┘
```

The API calls happen in this order:

1. **GET /** — Check if Firecracker is running
2. **PUT /machine-config** — Set vCPU count and memory
3. **PUT /boot-source** — Tell it which kernel to boot
4. **PUT /drives/rootfs** — Attach a root filesystem
5. **PUT /actions** — Send `{"action_type": "InstanceStart"}` to boot the VM
6. **GET /machine-config** — Read back config to verify it took

Every PUT expects `Content-Type: application/json`. Responses come back as JSON too.

## HTTP over Unix sockets — how it actually works

This was the tricky part to figure out. A Unix socket is just a file on disk that two processes can connect through. The HTTP protocol doesn't care whether the transport is TCP or a Unix socket — it's the same bytes either way.

In Go, there are two ways to do this:

### Approach 1: http.Client with custom Transport (what you'd use in practice)

```go
transport := &http.Transport{
    DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
        return net.DialTimeout("unix", socketPath, 3*time.Second)
    },
}
client := &http.Client{Transport: transport}
resp, err := client.Get("http://localhost/machine-config")
```

The URL says `http://localhost` but the custom transport ignores the hostname and dials the Unix socket instead. The HTTP library handles all the framing, chunking, headers, etc.

### Approach 2: Raw net.Dial (to understand what's happening underneath)

```go
conn, _ := net.Dial("unix", socketPath)
raw := "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"
conn.Write([]byte(raw))
```

This is literally just writing HTTP bytes into a file descriptor. I included this approach in the code (`RawRequest` in `client.go`) to show I understand what the HTTP library is doing under the hood.

## Build

```bash
make build
```

## Usage

Since most people don't have Firecracker installed (it only runs on Linux with KVM), this tool has a `--dry-run` mode that shows what it *would* send without needing a real Firecracker instance.

```bash
# dry run — shows all HTTP requests without sending them
./bin/fc-explorer --dry-run

# dry run with custom config
./bin/fc-explorer --dry-run --vcpus 4 --mem 512

# dry run with raw HTTP demo
./bin/fc-explorer --dry-run --raw-demo

# real usage (requires a running Firecracker instance)
./bin/fc-explorer --socket /tmp/firecracker.socket \
                  --kernel /opt/vmlinux \
                  --rootfs /opt/rootfs.ext4
```

## Demo (dry run)

```
$ ./bin/fc-explorer --dry-run --vcpus 4 --mem 512 --raw-demo
fc-explorer — Firecracker API proof of concept
Socket: /tmp/firecracker.socket
Dry run: true

--- Step 1/6: check firecracker ---
==> Checking Firecracker status (GET /)
[DRY RUN] GET /

--- Step 2/6: set machine config ---
==> Setting machine config: 4 vCPUs, 512 MB RAM (PUT /machine-config)
[DRY RUN] PUT /machine-config
  Body: {"vcpu_count":4,"mem_size_mib":512,"ht_enabled":false}

--- Step 3/6: set boot source ---
==> Setting boot source: /path/to/vmlinux (PUT /boot-source)
[DRY RUN] PUT /boot-source
  Body: {"kernel_image_path":"/path/to/vmlinux","boot_args":"console=ttyS0 reboot=k panic=1 pci=off"}

--- Step 4/6: attach rootfs ---
==> Attaching rootfs: /path/to/rootfs.ext4 (PUT /drives/rootfs)
[DRY RUN] PUT /drives/rootfs
  Body: {"drive_id":"rootfs","path_on_host":"/path/to/rootfs.ext4","is_root_device":true,"is_read_only":false}

--- Step 5/6: start vm ---
==> Starting VM (PUT /actions)
[DRY RUN] PUT /actions
  Body: {"action_type":"InstanceStart"}

--- Step 6/6: read back config ---
==> Reading back machine config (GET /machine-config)
[DRY RUN] GET /machine-config

--- Bonus: Raw HTTP Demo ---
==> Demo: Raw HTTP request (GET / using net.Dial)
[DRY RUN] Raw HTTP request:
GET / HTTP/1.1
Host: localhost
Accept: application/json


==> Demo: Raw HTTP request (PUT /machine-config using net.Dial)
[DRY RUN] Raw HTTP request:
PUT /machine-config HTTP/1.1
Host: localhost
Accept: application/json
Content-Type: application/json
Content-Length: 51

{"vcpu_count":2,"mem_size_mib":256,"ht_enabled":false}

Done!
```

## How the code is organized

- `main.go` — CLI flags and the boot sequence loop
- `api/types.go` — Go structs that match the Firecracker API (from their swagger spec)
- `api/client.go` — Unix socket HTTP client. Has both approaches: `http.Client` with custom transport, and raw `net.Dial` for learning.
- `api/actions.go` — High-level operations (configure, boot, attach drive, etc.)

## What I learned

- Firecracker's API is really clean. Each resource (machine config, boot source, drives) has its own endpoint. You configure everything first, then send a single action to boot.
- The kernel must be an uncompressed `vmlinux`, not a `bzImage`. I found this out the hard way from the docs.
- HTTP over Unix sockets is simpler than I expected. The Go standard library makes it almost invisible — you just swap the dialer.
- The API is stateless-ish: once you start the VM, you can't change the machine config. You'd need to stop and recreate.

## What I'd do next

- **Snapshot/restore support** — Firecracker has a snapshot API (`PUT /snapshot/create`, `PUT /snapshot/load`). This is how you'd do fast VM cloning.
- **Network interface setup** — Add `PUT /network-interfaces/{id}` to attach a TAP device for networking.
- **Metrics endpoint** — Firecracker can expose metrics via a Unix socket. Would be cool to read those.
- **VM lifecycle watcher** — Poll the API periodically and detect state changes (running → paused → stopped).

## Dependencies

None. Standard library only.
