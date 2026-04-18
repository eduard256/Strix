package generate

import (
	"net/url"
	"strings"
	"sync"
	"testing"
)

// registerXiaomi installs a xiaomi extractor identical to the one in
// internal/xiaomi. Tests live here (not in internal/xiaomi) because they
// validate generator behavior with xiaomi-style URLs.
var registerOnce sync.Once

func registerXiaomi() {
	registerOnce.Do(func() {
		RegisterExtract("xiaomi", func(rawURL string) (cleaned, section, key, value string) {
			u, err := url.Parse(rawURL)
			if err != nil || u.User == nil {
				return rawURL, "", "", ""
			}
			q := u.Query()
			token := q.Get("token")
			if token == "" {
				return rawURL, "", "", ""
			}
			q.Del("token")
			u.RawQuery = q.Encode()
			return u.String(), "xiaomi", u.User.Username(), token
		})
	})
}

// --- Helpers ---

func mustGen(t *testing.T, req *Request) string {
	t.Helper()
	r, err := Generate(req)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return r.Config
}

func assertContains(t *testing.T, cfg, substr string) {
	t.Helper()
	if !strings.Contains(cfg, substr) {
		t.Errorf("expected config to contain:\n  %q\n--- got ---\n%s", substr, cfg)
	}
}

func assertNotContains(t *testing.T, cfg, substr string) {
	t.Helper()
	if strings.Contains(cfg, substr) {
		t.Errorf("expected config NOT to contain:\n  %q\n--- got ---\n%s", substr, cfg)
	}
}

func countOccurrences(s, substr string) int {
	if substr == "" {
		return 0
	}
	n := 0
	for i := 0; ; {
		j := strings.Index(s[i:], substr)
		if j < 0 {
			return n
		}
		n++
		i += j + len(substr)
	}
}

// ex. "xiaomi://user:cn@ip?did=D&model=M&token=T"
func xurl(user, region, ip, did, model, token string) string {
	return "xiaomi://" + user + ":" + region + "@" + ip +
		"?did=" + did + "&model=" + model + "&token=" + url.QueryEscape(token)
}

// --- Tests ---

// Single xiaomi camera in a fresh config.
func TestXiaomi_NewConfig_SingleCamera(t *testing.T) {
	registerXiaomi()

	cfg := mustGen(t, &Request{
		MainStream: xurl("acc1", "cn", "10.0.20.229", "1", "chuangmi.camera.v1", "V1:TOK_A"),
	})

	assertContains(t, cfg, "go2rtc:\n  streams:\n")
	assertContains(t, cfg, "    '10_0_20_229_main':\n")
	assertContains(t, cfg, "- xiaomi://acc1:cn@10.0.20.229?did=1&model=chuangmi.camera.v1\n")
	assertContains(t, cfg, "  xiaomi:\n    \"acc1\": V1:TOK_A\n")
	assertNotContains(t, cfg, "token=")
	assertNotContains(t, cfg, "\nxiaomi:") // must be nested, not top-level
}

// Two cameras on the same account -- token appears only once.
func TestXiaomi_SameAccount_TokenNotDuplicated(t *testing.T) {
	registerXiaomi()

	c1 := mustGen(t, &Request{
		MainStream: xurl("acc1", "cn", "10.0.20.229", "1", "v1", "V1:TOK_A"),
	})
	c2 := mustGen(t, &Request{
		MainStream:     xurl("acc1", "cn", "10.0.20.230", "2", "v2", "V1:TOK_A"),
		ExistingConfig: c1,
	})

	if n := countOccurrences(c2, `"acc1":`); n != 1 {
		t.Errorf("expected exactly 1 \"acc1\" key, got %d\n---\n%s", n, c2)
	}
	assertContains(t, c2, `"acc1": V1:TOK_A`)
	assertContains(t, c2, "    '10_0_20_229_main':")
	assertContains(t, c2, "    '10_0_20_230_main':")
}

// Two accounts -- both tokens present, sorted by key.
func TestXiaomi_TwoAccounts_SortedKeys(t *testing.T) {
	registerXiaomi()

	c1 := mustGen(t, &Request{
		MainStream: xurl("zeta", "cn", "10.0.20.229", "1", "v1", "TOK_Z"),
	})
	c2 := mustGen(t, &Request{
		MainStream:     xurl("alpha", "de", "10.0.20.230", "2", "v2", "TOK_A"),
		ExistingConfig: c1,
	})

	iAlpha := strings.Index(c2, `"alpha":`)
	iZeta := strings.Index(c2, `"zeta":`)
	if iAlpha < 0 || iZeta < 0 {
		t.Fatalf("expected both keys:\n%s", c2)
	}
	if iAlpha >= iZeta {
		t.Errorf("expected alpha before zeta (sorted)\n%s", c2)
	}
}

// Re-login with a new token overwrites the existing value.
func TestXiaomi_TokenRefresh_OverwritesValue(t *testing.T) {
	registerXiaomi()

	c1 := mustGen(t, &Request{
		MainStream: xurl("acc1", "cn", "10.0.20.229", "1", "v1", "V1:OLD"),
	})
	c2 := mustGen(t, &Request{
		MainStream:     xurl("acc1", "cn", "10.0.20.230", "2", "v2", "V1:NEW"),
		ExistingConfig: c1,
	})

	assertContains(t, c2, `"acc1": V1:NEW`)
	assertNotContains(t, c2, `"acc1": V1:OLD`)
	if n := countOccurrences(c2, `"acc1":`); n != 1 {
		t.Errorf("expected 1 key after refresh, got %d", n)
	}
}

// Main + Sub stream with same credentials -- token deduped to one entry.
func TestXiaomi_MainAndSub_SameAccount_OneToken(t *testing.T) {
	registerXiaomi()

	cfg := mustGen(t, &Request{
		MainStream: xurl("acc1", "cn", "10.0.20.229", "1", "main", "V1:TOK"),
		SubStream:  xurl("acc1", "cn", "10.0.20.229", "1", "sub", "V1:TOK"),
	})

	if n := countOccurrences(cfg, `"acc1":`); n != 1 {
		t.Errorf("expected 1 acc1 key for main+sub, got %d\n%s", n, cfg)
	}
	assertContains(t, cfg, "    '10_0_20_229_main':")
	assertContains(t, cfg, "    '10_0_20_229_sub':")
	assertNotContains(t, cfg, "token=")
}

// Main and Sub from different accounts -- both tokens in the section.
func TestXiaomi_MainAndSub_DifferentAccounts(t *testing.T) {
	registerXiaomi()

	cfg := mustGen(t, &Request{
		MainStream: xurl("accA", "cn", "10.0.20.229", "1", "v1", "TOK_A"),
		SubStream:  xurl("accB", "de", "10.0.20.229", "1", "v1", "TOK_B"),
	})

	assertContains(t, cfg, `"accA": TOK_A`)
	assertContains(t, cfg, `"accB": TOK_B`)
}

// 10 cameras across 3 accounts added sequentially -- exactly 3 tokens at the end,
// correct token values, all streams present.
func TestXiaomi_Scale_10Cameras_3Accounts(t *testing.T) {
	registerXiaomi()

	cases := []struct{ user, ip, token string }{
		{"accA", "10.0.20.10", "TOK_A_v1"},
		{"accA", "10.0.20.11", "TOK_A_v1"},
		{"accB", "10.0.20.12", "TOK_B_v1"},
		{"accA", "10.0.20.13", "TOK_A_v1"},
		{"accC", "10.0.20.14", "TOK_C_v1"},
		{"accB", "10.0.20.15", "TOK_B_v2"}, // B gets refreshed
		{"accC", "10.0.20.16", "TOK_C_v1"},
		{"accA", "10.0.20.17", "TOK_A_v2"}, // A gets refreshed
		{"accB", "10.0.20.18", "TOK_B_v2"},
		{"accC", "10.0.20.19", "TOK_C_v2"}, // C gets refreshed
	}

	cfg := ""
	for i, c := range cases {
		req := &Request{
			MainStream: xurl(c.user, "cn", c.ip, "1", "v", c.token),
		}
		if i > 0 {
			req.ExistingConfig = cfg
		}
		cfg = mustGen(t, req)
	}

	if n := countOccurrences(cfg, `"accA":`); n != 1 {
		t.Errorf("accA: expected 1 key, got %d", n)
	}
	if n := countOccurrences(cfg, `"accB":`); n != 1 {
		t.Errorf("accB: expected 1 key, got %d", n)
	}
	if n := countOccurrences(cfg, `"accC":`); n != 1 {
		t.Errorf("accC: expected 1 key, got %d", n)
	}

	// final (latest) tokens
	assertContains(t, cfg, `"accA": TOK_A_v2`)
	assertContains(t, cfg, `"accB": TOK_B_v2`)
	assertContains(t, cfg, `"accC": TOK_C_v2`)

	for _, c := range cases {
		want := "xiaomi://" + c.user + ":cn@" + c.ip
		if !strings.Contains(cfg, want) {
			t.Errorf("missing stream URL %q", want)
		}
	}

	// only one xiaomi: section header, only one go2rtc:
	if n := countOccurrences(cfg, "\n  xiaomi:\n"); n != 1 {
		t.Errorf("expected 1 xiaomi: header, got %d", n)
	}
	if n := countOccurrences(cfg, "\ngo2rtc:\n"); n != 1 {
		t.Errorf("expected 1 go2rtc: header, got %d", n)
	}
}

// URL without ?token=... -- extractor returns empty section, no xiaomi: block written.
func TestXiaomi_URLWithoutToken_NoSection(t *testing.T) {
	registerXiaomi()

	cfg := mustGen(t, &Request{
		MainStream: "xiaomi://acc1:cn@10.0.20.229?did=1&model=v1",
	})

	// the nested section header would look like "\n  xiaomi:\n" -- URL scheme
	// "xiaomi://" must not trigger a false positive
	assertNotContains(t, cfg, "\n  xiaomi:\n")
	assertContains(t, cfg, "- xiaomi://acc1:cn@10.0.20.229?did=1&model=v1\n")
}

// Malformed URL must not crash the generator; URL is passed through as-is.
func TestXiaomi_MalformedURL_DoesNotPanic(t *testing.T) {
	registerXiaomi()

	_, err := Generate(&Request{
		MainStream: "xiaomi://%%%bad",
	})
	if err != nil {
		t.Logf("Generate returned error (ok): %v", err)
	}
}

// Token with base64 special chars (+ / =) must survive YAML write without escaping.
func TestXiaomi_TokenSpecialChars_PreservedRaw(t *testing.T) {
	registerXiaomi()

	raw := "V1:9d2w+abc/def=end="
	cfg := mustGen(t, &Request{
		MainStream: xurl("acc1", "cn", "10.0.20.229", "1", "v1", raw),
	})

	assertContains(t, cfg, `"acc1": `+raw)
}

// Go2RTC override MainStreamSource must also pass through the extractor.
func TestXiaomi_Go2RTCOverride_PassesThroughExtractor(t *testing.T) {
	registerXiaomi()

	cfg := mustGen(t, &Request{
		MainStream: "rtsp://placeholder:554/stream",
		Go2RTC: &Go2RTCOverride{
			MainStreamSource: xurl("acc1", "cn", "10.0.20.229", "1", "v1", "V1:OVR"),
		},
	})

	assertContains(t, cfg, `"acc1": V1:OVR`)
	assertNotContains(t, cfg, "token=")
	assertContains(t, cfg, "- xiaomi://acc1:cn@10.0.20.229?did=1&model=v1\n")
}

// addToConfig: existing config has no xiaomi: section -- must create one.
func TestXiaomi_AddToConfig_NoExistingSection(t *testing.T) {
	registerXiaomi()

	// start from a rtsp-only config
	c1 := mustGen(t, &Request{
		MainStream: "rtsp://user:pass@10.0.20.100/stream1",
	})
	assertNotContains(t, c1, "xiaomi:")

	c2 := mustGen(t, &Request{
		MainStream:     xurl("acc1", "cn", "10.0.20.229", "1", "v1", "V1:TOK"),
		ExistingConfig: c1,
	})

	assertContains(t, c2, "  xiaomi:\n    \"acc1\": V1:TOK\n")
	assertContains(t, c2, "- rtsp://user:pass@10.0.20.100/stream1")
	assertContains(t, c2, "- xiaomi://acc1:cn@10.0.20.229?did=1&model=v1")
}

// addToConfig: existing config already has xiaomi: section with other accounts.
func TestXiaomi_AddToConfig_ExistingSection(t *testing.T) {
	registerXiaomi()

	c1 := mustGen(t, &Request{
		MainStream: xurl("accA", "cn", "10.0.20.10", "1", "v1", "TOK_A"),
	})
	c2 := mustGen(t, &Request{
		MainStream:     xurl("accB", "de", "10.0.20.20", "2", "v2", "TOK_B"),
		ExistingConfig: c1,
	})

	assertContains(t, c2, `"accA": TOK_A`)
	assertContains(t, c2, `"accB": TOK_B`)
	// xiaomi: section stays nested, exactly one header
	if n := countOccurrences(c2, "\n  xiaomi:\n"); n != 1 {
		t.Errorf("expected 1 xiaomi header, got %d\n%s", n, c2)
	}
}

// Stream and camera names stay clean (no leftover tokens in URLs) with custom Name.
func TestXiaomi_CustomName_URLStillClean(t *testing.T) {
	registerXiaomi()

	cfg := mustGen(t, &Request{
		Name:       "my_cam",
		MainStream: xurl("acc1", "cn", "10.0.20.229", "1", "v1", "V1:TOK"),
	})

	assertContains(t, cfg, "    'my_cam_main':")
	assertContains(t, cfg, "  my_cam:")
	assertNotContains(t, cfg, "token=")
	assertContains(t, cfg, `"acc1": V1:TOK`)
}

// Mixed: rtsp + xiaomi -- rtsp URL untouched, xiaomi token extracted.
func TestXiaomi_MixedProtocols(t *testing.T) {
	registerXiaomi()

	c1 := mustGen(t, &Request{
		MainStream: "rtsp://admin:pw@10.0.20.100/Streaming/Channels/101",
	})
	c2 := mustGen(t, &Request{
		MainStream:     xurl("acc1", "cn", "10.0.20.229", "1", "v1", "V1:TOK"),
		ExistingConfig: c1,
	})

	assertContains(t, c2, "- rtsp://admin:pw@10.0.20.100/Streaming/Channels/101")
	assertContains(t, c2, "- xiaomi://acc1:cn@10.0.20.229?did=1&model=v1")
	assertContains(t, c2, `"acc1": V1:TOK`)
}

// Order in generated config: go2rtc -> (streams, xiaomi) -> cameras -> version.
func TestXiaomi_SectionOrder(t *testing.T) {
	registerXiaomi()

	cfg := mustGen(t, &Request{
		MainStream: xurl("acc1", "cn", "10.0.20.229", "1", "v1", "V1:TOK"),
	})

	iGo2rtc := strings.Index(cfg, "\ngo2rtc:\n")
	iStreams := strings.Index(cfg, "  streams:")
	iXiaomi := strings.Index(cfg, "  xiaomi:")
	iCameras := strings.Index(cfg, "\ncameras:\n")
	iVersion := strings.Index(cfg, "\nversion:")

	if iGo2rtc < 0 || iStreams < 0 || iXiaomi < 0 || iCameras < 0 || iVersion < 0 {
		t.Fatalf("missing section in config:\n%s", cfg)
	}
	if !(iGo2rtc < iStreams && iStreams < iXiaomi && iXiaomi < iCameras && iCameras < iVersion) {
		t.Errorf("wrong section order: go2rtc=%d streams=%d xiaomi=%d cameras=%d version=%d\n%s",
			iGo2rtc, iStreams, iXiaomi, iCameras, iVersion, cfg)
	}
}
