---
name: add_generate_strix
description: Register a new protocol extractor for the Strix config generator. Use when adding support for Tuya, Tapo, Nest, Ring, Roborock and other camera protocols that need credentials in a separate YAML section of frigate-config.yaml. This skill adds the glue; it does NOT add the protocol itself -- that's /add_protocol_strix.
disable-model-invocation: true
argument-hint: [protocol-name]
---

# Add Generate Extractor to Strix

You are registering a new protocol with the Strix config generator so that `/api/generate` can produce a valid `frigate-config.yaml` for cameras of that protocol.

The protocol name is provided as argument (e.g. `/add_generate_strix tuya`). If no argument, use AskUserQuestion to ask which protocol.

This skill is NARROW. It adds ONE extractor function plus tests. Do NOT modify `pkg/generate/` internals (registry.go, config.go, writer.go, insert.go) -- they are protocol-agnostic by design. If you are tempted to change them, stop and ask the user.

## Repositories

- Strix: current working directory (`/home/user/Strix`)
- go2rtc: `/home/user/go2rtc` (reference for YAML section format)

---

## STEP 0: Study the xiaomi reference

Before writing anything, read these files completely. Xiaomi is the golden reference for this skill. Every new protocol copies its structure.

- `internal/xiaomi/xiaomi.go` -- how `extractForConfig` is written and where `generate.RegisterExtract` is called inside `Init()`
- `pkg/generate/xiaomi_test.go` -- the 16 test scenarios that every new protocol must pass
- `pkg/generate/registry.go` -- the `ExtractFunc` contract (do not modify, only understand)

Then glance at these to understand what NOT to touch:

- `pkg/generate/config.go` -- how `runExtract` is called for main/sub/go2rtc-override URLs
- `pkg/generate/writer.go` -- how `writeCredentials` nests sections under `go2rtc:`
- `pkg/generate/insert.go` -- `upsertCredentials` merges into existing configs

---

## STEP 1: Study the protocol in go2rtc

Read these files in the go2rtc repo:

- `internal/<proto>/README.md` -- authoritative YAML format
- `internal/<proto>/<proto>.go` -- how go2rtc unmarshals the credentials section (look for `yaml:"<proto>"` in a struct tag)
- `pkg/<proto>/` -- URL scheme: what goes into userinfo, what into query, what into path

You need to answer exactly three questions:

1. **Where does the secret live in the URL?**
   - `?token=X` in query (xiaomi, nest, ring)
   - `userinfo` password (tapo, roborock)
   - Custom (read the module)

2. **What is the YAML section name?** Usually the same as the URL scheme (`tuya:`, `tapo:`). Confirm from go2rtc's `yaml:` struct tag.

3. **What is the YAML key format?** Examples:
   - xiaomi: `"<userID>"` (quoted, matches `ci` field in device)
   - tapo: `<user>@<host>`
   - roborock: `<username>`
   - nest: `"<userID>"`

Write these three answers down. They drive the extractor.

---

## STEP 2: Verify the internal module already exists

Check `internal/<proto>/<proto>.go` exists and has a working `Init()` with `tester.RegisterSource` or similar. If the module does NOT exist, stop and tell the user to run `/add_protocol_strix <proto>` first. This skill only adds the generate hook to an existing module.

---

## STEP 3: Add the extractor to `internal/<proto>/<proto>.go`

Open `internal/<proto>/<proto>.go`.

**3a. Add import** (if not present):

```go
import "github.com/eduard256/strix/pkg/generate"
```

**3b. Register the extractor at the END of `Init()`:**

```go
generate.RegisterExtract("<proto>", extractForConfig)
```

**3c. Add the function.** Use the exact xiaomi style. Place it directly after `Init()`.

### Template: secret in query (xiaomi-like)

```go
// extractForConfig strips ?<secret>=... from <proto>:// URL and returns
// <key> + <token> for a go2rtc:<section>: block.
// ex. <proto>://<user>:<region>@<ip>?...&<secret>=T
//   -> <proto>://<user>:<region>@<ip>?..., "<section>", "<user>", "T"
func extractForConfig(rawURL string) (cleaned, section, key, value string) {
	u, err := url.Parse(rawURL)
	if err != nil || u.User == nil {
		return rawURL, "", "", ""
	}

	q := u.Query()
	token := q.Get("<secret>")
	if token == "" {
		return rawURL, "", "", ""
	}
	q.Del("<secret>")
	u.RawQuery = q.Encode()

	return u.String(), "<section>", u.User.Username(), token
}
```

### Template: secret in userinfo password (tapo-like)

```go
func extractForConfig(rawURL string) (cleaned, section, key, value string) {
	u, err := url.Parse(rawURL)
	if err != nil || u.User == nil {
		return rawURL, "", "", ""
	}

	pw, ok := u.User.Password()
	if !ok || pw == "" {
		return rawURL, "", "", ""
	}

	// ex. tapo: key = "admin@192.168.1.100"
	key = u.User.Username() + "@" + u.Host
	// URL stays as-is -- go2rtc reads credentials from userinfo directly
	return rawURL, "<section>", key, pw
}
```

### Rules for the extractor (strict)

- MUST return `(rawURL, "", "", "")` on any parse error or missing secret. Never return an empty `cleaned`.
- MUST NOT log, MUST NOT touch filesystem, MUST NOT call APIs.
- MUST be deterministic -- same input always returns same output.
- MUST NOT URL-encode the token value. `writeCredentials` emits it raw; YAML parses `V1:abc+/=` fine.
- Keep it under 20 lines. If it grows, you're doing too much.

---

## STEP 4: Write `pkg/generate/<proto>_test.go`

Copy the structure from `pkg/generate/xiaomi_test.go` exactly. Change:

- `registerXiaomi` -> `register<Proto>` (use a new `sync.Once` per file -- do NOT share `registerOnce` across files)
- `xurl(...)` -> `<proto>url(...)` -- builds a URL in this protocol's format
- Test function names: `TestXiaomi_*` -> `Test<Proto>_*`
- All string literals that reference `xiaomi://`, `"acc1"`, `V1:TOK_A` -> equivalents for this protocol

### The 16 tests to write (all MUST pass)

All scenarios from `xiaomi_test.go` are relevant and must be present:

| # | Test | What it verifies |
|---|---|---|
| 1 | NewConfig_SingleCamera | Nested `go2rtc:\n  <section>:\n    <key>: <value>` with correct indentation. URL in streams has no secret left. |
| 2 | SameAccount_TokenNotDuplicated | Two cameras, same key -> exactly one entry in the section. |
| 3 | TwoAccounts_SortedKeys | Two keys in the section appear in ASCII-sorted order. |
| 4 | TokenRefresh_OverwritesValue | Re-adding a camera with a new token replaces the stored value, exactly one key remains. |
| 5 | MainAndSub_SameAccount_OneToken | Main + Sub with identical credentials -> one key. |
| 6 | MainAndSub_DifferentAccounts | Main + Sub with two accounts -> two keys. |
| 7 | Scale_10Cameras_3Accounts | 10 cameras across 3 accounts sequentially added -> exactly 3 keys at the end, most-recent values. |
| 8 | URLWithoutToken_NoSection | URL missing the secret -> no `<section>:` header written (check `"\n  <section>:\n"`, not the URL scheme substring). |
| 9 | MalformedURL_DoesNotPanic | `<proto>://%%%bad` does not crash. |
| 10 | TokenSpecialChars_PreservedRaw | Secret with `+`, `/`, `=`, `:` is emitted verbatim. |
| 11 | Go2RTCOverride_PassesThroughExtractor | `req.Go2RTC.MainStreamSource` with this protocol is also extracted. |
| 12 | AddToConfig_NoExistingSection | Start from rtsp-only config, add this protocol -> new `<section>:` block created under `go2rtc:`. |
| 13 | AddToConfig_ExistingSection | Start from a config that already has `<section>:` -> new key merged, one header. |
| 14 | CustomName_URLStillClean | `req.Name` set, URL still has no secret in streams. |
| 15 | MixedProtocols | rtsp + this protocol together -- rtsp URL untouched, secret extracted. |
| 16 | SectionOrder | Order: go2rtc -> streams -> `<section>` -> cameras -> version. |

### Common pitfalls in tests

- `assertNotContains(cfg, "<section>:")` is WRONG when the URL scheme contains the same substring. Use `"\n  <section>:\n"` instead (nested form) to avoid matching `<proto>://`.
- For protocols where `extractForConfig` returns `rawURL` unchanged (userinfo template), the "URL cleaning" assertion `assertNotContains(cfg, "token=")` does not apply -- skip or adjust.
- Use `registerOnce` with `sync.Once` so running the whole package test suite twice does not duplicate-register the extractor.

---

## STEP 5: Build and run the tests

```bash
cd /home/user/Strix
go build ./...
go test ./pkg/generate/ -v -run Test<Proto>
```

Every test must pass. If any fails:

- Re-read the corresponding xiaomi test and the difference in your version.
- Re-read the extractor function -- usually the bug is there, not in generate internals.
- Do NOT "fix" `pkg/generate/` to make tests pass. The generator has 16 passing tests for xiaomi already -- your protocol must fit the same contract.

---

## STEP 6: Sanity check the full generator output

Run once with a realistic URL manually (Bash + `go run` inline tool) and eyeball the YAML. Confirm:

- `go2rtc:` is top-level
- `streams:` and `<section>:` are both siblings under `go2rtc:` with 2-space indent
- Keys under `<section>:` use 4-space indent
- No duplicate headers
- Secret appears verbatim (no `%2F`, no `%3D`)

Do NOT commit. Leave changes staged for the user to review and commit manually.

---

## ABSOLUTES -- DO NOT VIOLATE

1. **Never modify `pkg/generate/registry.go`, `config.go`, `writer.go`, `insert.go`.** If you think you need to, the protocol probably doesn't fit the extractor contract -- stop and discuss with the user.
2. **Never add a new extractor to `pkg/generate/`.** Extractors live in `internal/<proto>/`, the test file is the only thing in `pkg/generate/`.
3. **Never modify `pkg/<proto>/`.** That code comes from go2rtc; we don't own it.
4. **Never write tests in `internal/<proto>/`.** All generator tests go to `pkg/generate/<proto>_test.go`.
5. **Never call `generate.RegisterExtract` from inside a test.** Use `sync.Once` + a helper like `register<Proto>()` inside the test file.
6. **Never commit.** Leave the changes for the user.
7. **Never skip tests.** All 16 scenarios are mandatory -- they caught real regressions during development.
