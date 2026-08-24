---
name: add_protocol_strix
description: Add a new protocol support to Strix -- full flow from research to implementation. Covers stream handler registration, URL builder updates, database issues, and go2rtc integration.
disable-model-invocation: true
argument-hint: [protocol-name]
---

# Add Protocol to Strix

You are adding support for a new protocol to Strix. Follow every step in order. Be thorough -- read all referenced files completely before writing any code.

The protocol name is provided as argument (e.g. `/add_protocol_strix bubble`). If no argument, use AskUserQuestion to ask which protocol to add.

## Repositories

- Strix: current working directory (`/home/user/Strix`)
- go2rtc: `/home/user/go2rtc` (reference implementation, read-only)
- StrixCamDB: issues at https://github.com/eduard256/StrixCamDB/issues (for database updates)

## Related skills (know when to hand off)

- `/add_generate_strix <proto>` -- register a credentials extractor for the Frigate config generator. Run AFTER this skill if the protocol has tokens/passwords that must go into a separate YAML section of `frigate-config.yaml`.
- `/add_probe_detector_strix <proto>` -- add a device-type detector to `/api/probe` so the frontend auto-routes a matching IP to your new protocol page.

---

## STEP 0: Understand the existing implementations (REFERENCE)

Before doing anything, read these files completely to understand the patterns:

```
pkg/tester/source.go              -- handler registry + RTSP reference (Type A)
pkg/tester/worker.go              -- how handlers are called, screenshot logic
pkg/tester/session.go             -- session data structures
pkg/camdb/streams.go              -- URL builder, placeholder replacement
internal/test/test.go             -- API layer for tester
internal/search/search.go         -- search API (rarely needs changes)
internal/xiaomi/xiaomi.go         -- golden reference for Type B with cloud auth + token URLs
internal/homekit/homekit.go       -- reference for Type B with pairing-based custom source blocks
```

### How RTSP works (the reference pattern)

**Registration** in `pkg/tester/source.go`:
```go
var handlers = map[string]SourceHandler{}

func RegisterSource(scheme string, handler SourceHandler) {
    handlers[scheme] = handler
}

func init() {
    RegisterSource("rtsp", rtspHandler)
    RegisterSource("rtsps", rtspHandler)
    RegisterSource("rtspx", rtspHandler)
}
```

**Handler** -- receives a URL string, returns go2rtc `core.Producer`:
```go
func rtspHandler(rawURL string) (core.Producer, error) {
    rawURL, _, _ = strings.Cut(rawURL, "#")

    conn := rtsp.NewClient(rawURL)
    conn.Backchannel = false

    if err := conn.Dial(); err != nil {
        return nil, fmt.Errorf("rtsp: dial: %w", err)
    }

    if err := conn.Describe(); err != nil {
        _ = conn.Stop()
        return nil, fmt.Errorf("rtsp: describe: %w", err)
    }

    return conn, nil
}
```

**Data flow**: URL -> GetHandler(url) -> handler(url) -> core.Producer -> GetMedias() -> codecs, latency -> getScreenshot() -> jpegSize() -> Result (with width, height)

**Key**: The handler ONLY needs to return a `core.Producer`. Everything else (codecs extraction, screenshot capture, session management) is handled automatically by `worker.go`.

### How URLs are built in `pkg/camdb/streams.go`:

1. Database has URL templates like `/cam/realmonitor?channel=[CHANNEL]&subtype=0`
2. `replacePlaceholders()` substitutes `[CHANNEL]`, `[USERNAME]`, `[PASSWORD]`, etc.
3. `buildURL()` prepends `protocol://user:pass@host:port` to the path
4. Credentials are URL-encoded with `url.PathEscape` / `url.QueryEscape`

Default ports are defined in `defaultPorts` map:
```go
var defaultPorts = map[string]int{
    "rtsp": 554, "rtsps": 322, "http": 80, "https": 443,
    "rtmp": 1935, "mms": 554, "rtp": 5004,
}
```

---

## STEP 1: Research the protocol in go2rtc

go2rtc already implements most camera protocols. Study the implementation:

### Where to look in go2rtc

| What | Where |
|------|-------|
| Protocol client logic | `../go2rtc/pkg/{protocol}/` |
| Module registration | `../go2rtc/internal/{protocol}/` |
| Core interfaces | `../go2rtc/pkg/core/core.go` |
| Stream handler registry | `../go2rtc/internal/streams/handlers.go` |
| Keyframe capture | `../go2rtc/pkg/magic/keyframe.go` |

**Version note:** cloud-auth protocols like `xiaomi` require go2rtc >= 1.9.13. Frigate `stable` still ships with go2rtc 1.9.10; Frigate `dev`/0.18+ upgrades to 1.9.13+. A user on Frigate stable cannot stream a xiaomi camera even if Strix generates a perfect config -- the go2rtc binary inside Frigate will log `unsupported scheme`. Mention this in your handoff.

### Protocol map in go2rtc

| Protocol | pkg/ (Dial function) | internal/ (Init glue) |
|----------|---------------------|----------------------|
| rtsp/rtsps | `pkg/rtsp/client.go` | `internal/rtsp/rtsp.go` |
| http/https | `pkg/magic/producer.go`, `pkg/tcp/request.go` | `internal/http/http.go` |
| rtmp | `pkg/rtmp/` | `internal/rtmp/rtmp.go` |
| bubble | `pkg/bubble/` | `internal/bubble/bubble.go` |
| dvrip | `pkg/dvrip/` | `internal/dvrip/dvrip.go` |
| onvif | `pkg/onvif/` | `internal/onvif/onvif.go` |
| homekit | `pkg/homekit/`, `pkg/hap/` | `internal/homekit/homekit.go` |
| tapo | `pkg/tapo/` | `internal/tapo/tapo.go` |
| kasa | `pkg/kasa/` | `internal/kasa/kasa.go` |
| eseecloud | `pkg/eseecloud/` | `internal/eseecloud/eseecloud.go` |
| nest | `pkg/nest/` | `internal/nest/init.go` |
| ring | `pkg/ring/` | `internal/ring/ring.go` |
| wyze | `pkg/wyze/` | `internal/wyze/wyze.go` |
| xiaomi | `pkg/xiaomi/` | `internal/xiaomi/xiaomi.go` |
| tuya | `pkg/tuya/` | `internal/tuya/tuya.go` |
| doorbird | `pkg/doorbird/` | `internal/doorbird/doorbird.go` |
| isapi | `pkg/isapi/` | `internal/isapi/init.go` |
| flussonic | `pkg/flussonic/` | `internal/flussonic/flussonic.go` |
| gopro | `pkg/gopro/` | `internal/gopro/gopro.go` |
| roborock | `pkg/roborock/` | `internal/roborock/roborock.go` |

### What to read

1. Read `/home/user/go2rtc/internal/{protocol}/{protocol}.go` -- find `streams.HandleFunc` call, understand what function is called and how
2. Read `/home/user/go2rtc/pkg/{protocol}/` -- find the `Dial()` or `NewClient()` function, understand its signature and what it returns
3. Understand: does it return `core.Producer`? Does it need special setup before Dial? Does it need credentials differently?

### Typical go2rtc internal module (e.g. kasa -- simplest):
```go
package kasa

import (
    "github.com/AlexxIT/go2rtc/internal/streams"
    "github.com/AlexxIT/go2rtc/pkg/core"
    "github.com/AlexxIT/go2rtc/pkg/kasa"
)

func Init() {
    streams.HandleFunc("kasa", func(source string) (core.Producer, error) {
        return kasa.Dial(source)
    })
}
```

Most protocols follow this exact pattern: `pkg/{protocol}.Dial(url)` returns `core.Producer`.

---

## STEP 2: Classify the protocol

Use AskUserQuestion to discuss with the user. Determine the protocol type:

### Type A: Standard URL-based protocol (rtsp, rtmp, bubble, dvrip, http)

- Has URL scheme (ex. `bubble://host:port/path`)
- URLs stored in StrixCamDB database
- Flow: user searches camera -> gets URL templates -> URLs built with credentials -> sent to tester
- Needs: stream handler in tester + default port in URL builder + database issue
- Hands off to: no other skill needed (credentials live in userinfo, captured by the URL itself)

### Type B1: Custom pairing / discovery (homekit)

- Does NOT use URL templates from database
- mDNS discovery + multi-step pairing (PIN, PSK, etc.)
- Custom frontend page, custom API endpoint, custom source block in `/api/test`
- Data comes from `/api/probe` or direct user input
- Needs: `SourceBlockHandler` registration, pairing endpoint, dedicated HTML page
- Reference: `internal/homekit/homekit.go`, `www/homekit.html`
- Hands off to: `/add_probe_detector_strix` (if detectable by IP)

### Type B2: Cloud-auth with token URL (xiaomi, tapo, nest, ring, roborock, tuya)

- Has URL scheme (`xiaomi://userID:region@IP?did=X&model=Y&token=T`)
- Credentials come from a cloud API (Mi Cloud, Tapo Cloud, etc.) not from local discovery
- Stateless design: token is extracted server-side, embedded in URL, then consumed by a generator extractor that moves it to a dedicated YAML section
- Copy `internal/<proto>/<proto>.go` from go2rtc (adapt imports), add API endpoint for the cloud login flow, build a dedicated HTML page mirroring `www/xiaomi.html`
- Reference: `internal/xiaomi/xiaomi.go` (copy template), `www/xiaomi.html` (UI template)
- Hands off to: `/add_generate_strix <proto>` for the YAML credentials extractor, `/add_probe_detector_strix <proto>` if detectable by IP

### Type C: HTTP sub-protocol (mjpeg, jpeg snapshot, hls)

- Uses `http://` or `https://` URL scheme
- Already has URLs in database (same as HTTP)
- Needs special handling in tester based on Content-Type response
- Needs: stream handler that detects content type and handles accordingly

---

## STEP 3: For Type A -- Create StrixCamDB issue

ONLY for Type A protocols that have URL patterns stored in the database.

Create a GitHub issue using `gh` CLI for the new protocol:

```bash
cd /home/user/Strix
gh issue create --repo eduard256/StrixCamDB \
  --title "[New Protocol] {PROTOCOL_NAME}" \
  --label "new-protocol" \
  --body "$(cat <<'ISSUE_EOF'
```yaml
protocol: {PROTOCOL_NAME}
default_port: {PORT}
url_format: {EXAMPLE_URL_PATTERN}
```

## Description

{DESCRIPTION -- what cameras use this, what firmware, how it works}

## Known brands

- {BRAND1}
- {BRAND2}

## URL patterns

- {PATTERN1} -- main stream
- {PATTERN2} -- sub stream

## Where to research

- go2rtc source: https://github.com/AlexxIT/go2rtc/tree/master/pkg/{PROTOCOL_NAME}
- ispyconnect: search for "{PROTOCOL_NAME}" cameras

## Notes

{ANY_NOTES}
ISSUE_EOF
)"
```

If the protocol introduces new placeholders (e.g. `[STREAM]`), create a separate issue:

```bash
gh issue create --repo eduard256/StrixCamDB \
  --title "[New Placeholder] {PLACEHOLDER}" \
  --label "new-placeholder" \
  --body "$(cat <<'ISSUE_EOF'
placeholder: "{PLACEHOLDER}"
alternatives: ["{alt1}", "{alt2}"]
description: "{WHAT_IT_DOES}"
example_values: ["{VAL1}", "{VAL2}"]

## URL examples

- {URL_EXAMPLE_1}
- {URL_EXAMPLE_2}

## Known brands using this

- {BRAND1}
- {BRAND2}
ISSUE_EOF
)"
```

DO NOT wait for issue approval. Continue immediately to the next step.

---

## STEP 4: Update URL builder (Type A only)

If the protocol needs a new default port, edit `/home/user/Strix/pkg/camdb/streams.go`:

Add the port to `defaultPorts` map:
```go
var defaultPorts = map[string]int{
    "rtsp": 554, "rtsps": 322, "http": 80, "https": 443,
    "rtmp": 1935, "mms": 554, "rtp": 5004,
    // add new protocol here:
    "bubble": 80,
}
```

If the protocol needs new placeholders in `replacePlaceholders()`, add them to the pairs slice. Follow the existing pattern -- both `[UPPER]` and `[lower]` variants, plus `{curly}` variants.

### Files to edit for URL builder:
- `/home/user/Strix/pkg/camdb/streams.go` -- `defaultPorts` map and `replacePlaceholders()` function

---

## STEP 5: Add stream handler to tester

### Before writing code

1. Read ALL existing handlers in `/home/user/Strix/pkg/tester/source.go` completely
2. Read the go2rtc pkg/ implementation for this protocol (Step 1)
3. Understand what the `Dial()` function needs and returns

### For standard protocols (Type A, most Type C)

Most protocols follow the same pattern as RTSP. The handler:
1. Takes a URL string
2. Calls go2rtc's `pkg/{protocol}.Dial(url)` or equivalent
3. Returns `core.Producer`

Add the handler to `/home/user/Strix/pkg/tester/source.go`.

**Pattern for simple protocols** (bubble, dvrip, rtmp, kasa, etc.):

```go
import "github.com/AlexxIT/go2rtc/pkg/{protocol}"

// in init():
RegisterSource("{scheme}", {scheme}Handler)

// handler:
func {scheme}Handler(rawURL string) (core.Producer, error) {
    return {protocol}.Dial(rawURL)
}
```

If the protocol needs extra setup before Dial (like RTSP needs `Backchannel = false`), add it. Study the go2rtc internal module to see what setup is done.

**Pattern for protocols that need connection setup** (like RTSP):

```go
func {scheme}Handler(rawURL string) (core.Producer, error) {
    rawURL, _, _ = strings.Cut(rawURL, "#")

    conn := {protocol}.NewClient(rawURL)
    // any setup specific to this protocol

    if err := conn.Dial(); err != nil {
        return nil, fmt.Errorf("{scheme}: dial: %w", err)
    }

    // protocol-specific validation (like RTSP Describe)

    return conn, nil
}
```

### For Type B2 -- cloud-auth protocols (xiaomi, tapo, nest, ring, roborock, tuya)

Use xiaomi as the golden reference. These protocols fit the normal `tester.RegisterSource` contract -- their URL scheme IS routable, you just have to do cloud auth first and embed the resulting token in the URL.

**1. Copy `internal/<proto>/<proto>.go` from go2rtc** into Strix. Change imports:

```go
// go2rtc:
"github.com/AlexxIT/go2rtc/internal/api"
"github.com/AlexxIT/go2rtc/internal/app"
"github.com/AlexxIT/go2rtc/internal/streams"

// strix:
"github.com/eduard256/strix/internal/api"
"github.com/eduard256/strix/internal/app"
"github.com/eduard256/strix/pkg/tester"
```

Replace `streams.HandleFunc("<proto>", ...)` with `tester.RegisterSource("<proto>", ...)`. Drop `app.LoadConfig`/`app.PatchConfig` calls -- Strix is stateless, tokens live only in memory + URL (see xiaomi for the pattern).

**2. Stream handler extracts token from URL query** and seeds the in-memory cache:

```go
tester.RegisterSource("<proto>", func(rawURL string) (core.Producer, error) {
    u, _ := url.Parse(rawURL)
    // seed in-memory tokens cache from the URL so cloud-auth'd functions work
    if token := u.Query().Get("token"); token != "" && u.User != nil {
        ...
    }
    if u.User != nil {
        rawURL, _ = getCameraURL(u) // cloud call for p2p keys
    }
    return <proto>.Dial(rawURL)
})
```

**3. Cloud auth API endpoint** -- 4-step flow (username/password -> captcha -> 2FA -> success):

```go
api.HandleFunc("api/<proto>", apiHandler)
```

See `internal/xiaomi/xiaomi.go` for the exact switch on GET/POST and the 401+JSON-with-captcha/verify_phone response shape. The frontend mirrors this across several state transitions.

**4. Register with `main.go`:**

```go
modules := []module{
    ...
    {"<proto>", <proto>.Init},
}
```

**5. Build a frontend page `www/<proto>.html`** mirroring `www/xiaomi.html`. It has 6 states: loading, login, captcha, verify, region picker, not found. Also update `www/index.html`'s `navigateXiaomi`-style router to handle this protocol's probe type.

**6. Register credentials extractor with the config generator.** Do this in THE SAME `Init()` by calling `/add_generate_strix <proto>` (or hand off to that skill). The extractor strips `?token=...` from the URL and moves it into a top-level section under `go2rtc:` in the generated Frigate config.

### For Type B1 -- pairing-based protocols (homekit)

These don't fit the URL scheme contract -- data comes from a mDNS discovery plus a user-entered PIN. They use `SourceBlockHandler` instead of `SourceHandler`:

**1. Define block handler in `pkg/tester/source.go`:**

```go
type SourceBlockHandler func(data json.RawMessage, s *Session)

var sourceHandlers = map[string]SourceBlockHandler{}

func RegisterSourceBlock(name string, handler SourceBlockHandler) {
    sourceHandlers[name] = handler
}
```

**2. Update `internal/test/test.go:apiTestCreate()`** to parse and dispatch custom source blocks alongside `sources.streams`.

**3. Extended request format:**

```json
{
  "sources": {
    "streams": ["rtsp://...", "http://..."],
    "homekit": {"device_id": "AA:BB:CC", "pin": "123-45-678"}
  }
}
```

**4. Write the block handler** -- parses its params, runs pairing, calls `s.AddResult(...)` and `s.AddTested(...)` directly.

Reference: `internal/homekit/homekit.go`, `www/homekit.html`.

**IMPORTANT**: Before starting Type B1, discuss the approach with the user -- pairing flows are rare and each one is custom.

---

## STEP 6: Test the implementation

### Build and verify

```bash
cd /home/user/Strix
go build ./...
```

If it compiles, test with the running container:

```bash
# rebuild image
docker build -t strix:test .

# restart container
docker rm -f strix
docker run -d --name strix --network host --restart unless-stopped strix:test

# check logs
docker logs strix

# test the new protocol (example for bubble)
curl -s -X POST http://localhost:4567/api/test \
  -H 'Content-Type: application/json' \
  -d '{"sources":{"streams":["bubble://admin:password@192.168.1.100:80/"]}}'
```

### What to verify

1. Handler is registered -- check logs for no errors at startup
2. URLs with the new scheme are dispatched to the correct handler
3. If Type A: verify `/api/streams` returns URLs with correct scheme and port
4. Test with a real device if available

---

## STEP 7: Hand off to related skills

Once the tester handler works and the test returns a screenshot, the protocol is NOT fully wired yet. Check what else is needed:

- **Does the URL carry credentials (tokens, passwords)?** Run `/add_generate_strix <proto>` to register the extractor that moves them into a top-level section of the generated Frigate config. Without this, `frigate-config.yaml` embeds the full URL with the token, and a user pasting the config into Frigate directly will leak the secret (plus `go2rtc:xiaomi` section won't populate).
- **Is the device detectable by IP probe?** Run `/add_probe_detector_strix <proto>` so that `/api/probe?ip=X` returns `type: "<proto>"` and the frontend auto-routes to the protocol page.

Do NOT commit -- leave changes staged for the user to review.

---

## CODE STYLE RULES

All code MUST follow AlexxIT go2rtc style:

### File organization
- One handler per protocol is fine in `source.go` if it's a one-liner (`return pkg.Dial(url)`)
- If handler needs >10 lines of custom logic, create `source_{protocol}.go`
- Keep `source.go` as the registry + simple handlers
- Complex protocols get their own file

### Naming
- Handler: `{scheme}Handler` (e.g. `bubbleHandler`, `rtmpHandler`)
- Error prefix: `"{scheme}: dial: ..."` or `"{scheme}: ..."`
- Short var names: `conn` for connection, `prod` for producer

### Error handling
- Wrap errors with protocol prefix: `fmt.Errorf("bubble: dial: %w", err)`
- Close/stop connections on error: `_ = conn.Stop()`
- Return nil Producer on error, never a half-initialized one

### Comments
- Comment ONLY if the "why" is not obvious
- No docstrings on every function
- Inline examples: `// ex. "bubble://admin:pass@192.168.1.100:80/"`

### Imports
- go2rtc packages: `"github.com/AlexxIT/go2rtc/pkg/{protocol}"`
- Always import `"github.com/AlexxIT/go2rtc/pkg/core"` for Producer interface
- Group: stdlib, then go2rtc, then project packages

---

## go2rtc INTERNALS REFERENCE

### core.Producer interface (pkg/core/core.go)

Every protocol handler must return something that implements `core.Producer`:

```go
type Producer interface {
    GetMedias() []*Media    // what tracks are available (video/audio codecs)
    GetTrack(media *Media, codec *Codec) (*Receiver, error)  // get specific track
    Start() error           // start receiving packets (blocking)
    Stop() error            // close connection
}
```

The tester uses:
1. `GetMedias()` -- to list codecs (H264, AAC, etc.)
2. `GetTrack()` + `Start()` -- to capture screenshot (keyframe)
3. `Stop()` -- to clean up

### How screenshot and resolution work (pkg/tester/worker.go)

1. `getScreenshot(prod)` is called after successful Dial
2. Creates `magic.NewKeyframe()` consumer
3. Matches video media between producer and consumer
4. Gets track via `prod.GetTrack()`
5. Starts `prod.Start()` in goroutine (blocking -- reads packets)
6. Waits for first keyframe via `cons.WriteTo()` with 10s timeout
7. If H264/H265 -- converts to JPEG via ffmpeg
8. If already JPEG -- uses as-is
9. `jpegSize(jpeg)` extracts width and height from JPEG SOF0/SOF2 marker
10. Resolution stored in `Result.Width` and `Result.Height`

This works automatically for ANY protocol that returns a valid `core.Producer`. You do NOT need to implement screenshot or resolution logic per protocol.

### Result struct (pkg/tester/session.go)

```go
type Result struct {
    Source     string   `json:"source"`
    Screenshot string   `json:"screenshot,omitempty"`
    Codecs     []string `json:"codecs,omitempty"`
    Width      int      `json:"width,omitempty"`      // from JPEG screenshot
    Height     int      `json:"height,omitempty"`      // from JPEG screenshot
    LatencyMs  int64    `json:"latency_ms,omitempty"`
    Skipped    bool     `json:"skipped,omitempty"`
}
```

Resolution is extracted from the JPEG screenshot, not from SDP or protocol-specific data. This means width/height are only available when a screenshot was successfully captured. The frontend uses these values to classify streams as Main (HD) or Sub (SD).

### magic.NewKeyframe() (pkg/magic/keyframe.go)

Captures first video keyframe from any Producer. Supports H264, H265, JPEG, MJPEG. The tester uses this -- you never call it directly from a protocol handler.

### Connection patterns in go2rtc

**Simple Dial** (most protocols):
```go
// pkg/bubble/client.go
func Dial(rawURL string) (core.Producer, error) {
    // parse URL, connect, return producer
}
```

**Client with setup** (rtsp):
```go
// pkg/rtsp/client.go
conn := rtsp.NewClient(rawURL)
conn.Backchannel = false  // optional setup
conn.Dial()               // TCP connect
conn.Describe()           // RTSP DESCRIBE (gets SDP)
// conn is now a Producer
```

**HTTP-based** (complex -- content type detection):
```go
// pkg/magic/producer.go
// Opens HTTP connection, detects Content-Type:
// - multipart/x-mixed-replace -> MJPEG
// - image/jpeg -> single JPEG frame
// - application/vnd.apple.mpegurl -> HLS
// - video/mp2t -> MPEG-TS
// - etc.
```

### TCP/TLS connection (pkg/tcp/)

Many protocols use `pkg/tcp` for low-level connection:
- `tcp.Dial(rawURL)` -- TCP connect with timeout
- `tcp.Client` -- HTTP client with digest/basic auth
- Used by RTSP, HTTP, and others internally

---

## CHECKLIST BEFORE FINISHING

- [ ] Read all existing protocol handlers in `pkg/tester/source.go`
- [ ] Read xiaomi (`internal/xiaomi/xiaomi.go`) AND homekit (`internal/homekit/homekit.go`) as references
- [ ] Read go2rtc `pkg/` and `internal/` for this protocol
- [ ] Determined protocol type (A / B1 / B2 / C)
- [ ] For Type A: created StrixCamDB issue (protocol + placeholders if needed)
- [ ] For Type A: added default port to `defaultPorts` in `streams.go` (if not already there)
- [ ] Added handler registration (or full `internal/<proto>/` module for B2)
- [ ] Handler follows established pattern (RTSP for A, xiaomi for B2, homekit for B1)
- [ ] Error messages prefixed with protocol name
- [ ] Connections closed on error
- [ ] `go build ./...` compiles
- [ ] For B2: frontend page created (`www/<proto>.html`) and `www/index.html` router updated
- [ ] For B2: `/add_generate_strix <proto>` run to register the credentials extractor
- [ ] For detectable protocols: `/add_probe_detector_strix <proto>` run
- [ ] Changes LEFT STAGED (not committed -- user will review)
