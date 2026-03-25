---
name: add_probe_detector_strix
description: Add a new device type detector to the Strix probe system. Covers adding new probers, result types, and detector functions.
disable-model-invocation: true
argument-hint: [detector-name]
---

# Add Probe Detector to Strix

You are adding a new device type detector to the Strix probe system. The probe system runs when a user enters an IP address -- it discovers what's at that IP and determines the device type. The device type drives the frontend flow.

The detector name is provided as argument (e.g. `/add_probe_detector_strix onvif`). If no argument, use AskUserQuestion to ask which detector to add.

## Repository

- Strix: `/home/user/Strix`
- go2rtc (reference): `/home/user/go2rtc`

---

## STEP 0: Understand the probe system

Before writing anything, read these files COMPLETELY:

```
/home/user/Strix/internal/probe/probe.go   -- glue: Init(), runProbe(), detectors, API handler
/home/user/Strix/pkg/probe/models.go       -- all data structures (Response, Probes, result types)
/home/user/Strix/pkg/probe/ping.go         -- prober example: ICMP ping
/home/user/Strix/pkg/probe/ports.go        -- prober example: TCP port scan
/home/user/Strix/pkg/probe/arp.go          -- prober example: ARP lookup
/home/user/Strix/pkg/probe/dns.go          -- prober example: reverse DNS
/home/user/Strix/pkg/probe/http.go         -- prober example: HTTP HEAD request
/home/user/Strix/pkg/probe/mdns.go         -- prober example: HomeKit mDNS query
/home/user/Strix/pkg/probe/oui.go          -- prober example: OUI vendor lookup
```

Read ALL of them. Every prober is different. Understand the full picture before proceeding.

### How the probe system works

The probe has three layers:

**Layer 1: Probers** (`pkg/probe/`)

Pure functions that gather raw data about an IP address. Each runs in parallel with a shared 100ms timeout context. They do NOT interpret results -- just collect facts.

Current probers:
- `Ping()` -- ICMP echo, returns latency
- `ScanPorts()` -- TCP connect to all known camera ports, returns open ports
- `ReverseDNS()` -- reverse DNS lookup, returns hostname
- `LookupARP()` -- reads /proc/net/arp, returns MAC address
- `LookupOUI()` -- looks up MAC prefix in SQLite, returns vendor name
- `ProbeHTTP()` -- HTTP HEAD to ports 80/8080, returns status + server header
- `QueryHAP()` -- mDNS query for HomeKit Accessory Protocol, returns device info

Every prober writes its result into `resp.Probes.{Name}` via mutex.

**Layer 2: Detectors** (`internal/probe/probe.go`)

Functions registered in the `detectors` slice. They run AFTER all probers complete. Each detector receives the full `*probe.Response` with all probe results and returns a device type string (or empty string to pass).

```go
var detectors []func(*probe.Response) string
```

Detectors are checked in order. First non-empty result wins and sets `resp.Type`.

Default type is `"standard"`. If device is unreachable, type is `"unreachable"`.

**Layer 3: API** (`internal/probe/probe.go`)

`GET /api/probe?ip=192.168.1.100` returns the full Response JSON. The frontend uses `type` field to decide which UI flow to show.

### Data flow

```
IP address
  |
  v
[All probers run in parallel, 100ms timeout]
  |
  v
probe.Response filled with results
  |
  v
[Detectors run in order on the Response]
  |
  v
resp.Type = "homekit" | "standard" | "unreachable" | ...
  |
  v
JSON response to frontend
```

### API response example

```json
{
  "ip": "192.168.1.100",
  "reachable": true,
  "latency_ms": 2.5,
  "type": "homekit",
  "probes": {
    "ping": {"latency_ms": 2.5},
    "ports": {"open": [80, 554, 5353]},
    "dns": {"hostname": "camera.local"},
    "arp": {"mac": "C0:56:E3:AA:BB:CC", "vendor": "Hikvision"},
    "mdns": {
      "name": "My Camera",
      "device_id": "AA:BB:CC:DD:EE:FF",
      "model": "Camera 1080p",
      "category": "camera",
      "paired": false,
      "port": 80
    },
    "http": {"port": 80, "status_code": 200, "server": "nginx"}
  }
}
```

---

## STEP 1: Determine what you need

Use AskUserQuestion to discuss with the user. There are two scenarios:

### Scenario A: Detector only (using existing probe data)

The detector can determine device type from data already collected by existing probers. No new prober needed.

Examples:
- Detect ONVIF cameras by checking if port 80 is open and HTTP server header contains "onvif" or specific vendor strings
- Detect specific brands by ARP vendor name
- Detect UPnP devices by checking specific open ports

In this case: skip to STEP 3.

### Scenario B: New prober + detector

Need to collect new data that existing probers don't provide. Requires adding a new prober to `pkg/probe/` and a new result type to `models.go`.

Examples:
- ONVIF discovery (send ONVIF GetCapabilities request)
- UPnP SSDP discovery
- Specific protocol handshake

In this case: proceed to STEP 2.

---

## STEP 2: Add new prober (Scenario B only)

### 2a: Add result type to models.go

Edit `/home/user/Strix/pkg/probe/models.go`:

1. Add new result struct:
```go
type {Name}Result struct {
    // fields specific to this probe
}
```

2. Add field to `Probes` struct:
```go
type Probes struct {
    Ping  *PingResult  `json:"ping"`
    Ports *PortsResult `json:"ports"`
    DNS   *DNSResult   `json:"dns"`
    ARP   *ARPResult   `json:"arp"`
    MDNS  *MDNSResult  `json:"mdns"`
    HTTP  *HTTPResult  `json:"http"`
    {Name} *{Name}Result `json:"{name}"`  // add here
}
```

### 2b: Write prober function

Create `/home/user/Strix/pkg/probe/{name}.go`.

Rules:
- Pure function, no app/api imports
- Takes `context.Context` and `ip string` as first params
- Returns `(*{Name}Result, error)`
- Respects context deadline (timeout comes from runProbe)
- Returns `nil, nil` when device doesn't support this (NOT an error)
- Keep it simple -- one file, one function

Pattern:
```go
package probe

import "context"

func Probe{Name}(ctx context.Context, ip string) (*{Name}Result, error) {
    // respect context deadline
    deadline, ok := ctx.Deadline()
    if !ok {
        // set sensible default
    }

    // do the probe work...

    // not supported = nil, nil (not an error)
    // found = &{Name}Result{...}, nil
    // actual error = nil, err
}
```

### 2c: Wire prober into runProbe

Edit `/home/user/Strix/internal/probe/probe.go`, add to `runProbe()` alongside other probers:

```go
run(func() {
    r, _ := probe.Probe{Name}(ctx, ip)
    mu.Lock()
    resp.Probes.{Name} = r
    mu.Unlock()
})
```

All probers run in parallel inside the same `run()` pattern. The mutex protects writes to `resp.Probes`.

---

## STEP 3: Add detector function

Edit `/home/user/Strix/internal/probe/probe.go`, add detector in `Init()`:

```go
// {Name} detector
detectors = append(detectors, func(r *probe.Response) string {
    // check probe results to determine device type
    // return type string or "" to pass
    if r.Probes.{Something} != nil && {condition} {
        return "{type_name}"
    }
    return ""
})
```

### Detector rules

1. Return a SHORT type string: `"homekit"`, `"onvif"`, `"tapo"`, etc.
2. Return `""` (empty) to pass to the next detector
3. Detectors run in order -- put more specific detectors BEFORE generic ones
4. A detector can use ANY combination of probe results (ports, HTTP, ARP, mDNS, custom)
5. Don't do network I/O in detectors -- all data should come from probers

### Type string convention

The type string is used by the frontend to select UI flow:
- `"unreachable"` -- device not found (set automatically, don't return this)
- `"standard"` -- default, normal camera (set automatically if no detector matches)
- `"homekit"` -- Apple HomeKit device
- Custom types: lowercase, one word, matches the protocol/brand name

---

## STEP 4: Build and test

```bash
cd /home/user/Strix
go build ./...
```

If it compiles, rebuild Docker and test:

```bash
docker build -t strix:test .
docker rm -f strix
docker run -d --name strix --network host --restart unless-stopped strix:test
sleep 2

# test probe on a known device
curl -s "http://localhost:4567/api/probe?ip={DEVICE_IP}" | python3 -m json.tool
```

Verify:
1. New probe data appears in `probes` object (if new prober added)
2. `type` field correctly identifies the device
3. No errors in `docker logs strix`

---

## STEP 5: Commit and push

```bash
cd /home/user/Strix
git add -A
git commit -m "Add {name} probe detector"
git push origin develop
```

---

## CODE STYLE

### pkg/probe/ files
- One file per prober
- Pure functions, no globals, no app imports
- `context.Context` as first param for anything with I/O
- Return `nil, nil` for "not applicable" (not an error)
- Short names: `conn`, `resp`, `buf`

### internal/probe/probe.go
- Detectors are inline anonymous functions in Init()
- Keep detector logic minimal -- just check fields and return type
- If detector logic is complex (>10 lines), extract to a named function in the same file

### models.go
- All result structs in one file
- JSON tags use lowercase with underscores
- Optional fields use `omitempty`
- Pointer types for probe results (nil = not collected)
