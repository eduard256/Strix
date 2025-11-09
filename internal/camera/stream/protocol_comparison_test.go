package stream

import (
	"strings"
	"testing"

	"github.com/eduard256/Strix/internal/models"
)

// TestProtocolAuthBehaviorComparison проверяет разницу в генерации auth вариантов
// между RTSP и HTTP протоколами
func TestProtocolAuthBehaviorComparison(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	ctx := BuildContext{
		IP:       "192.168.1.100",
		Username: "admin",
		Password: "12345",
		Port:     0, // Will use default for protocol
	}

	t.Log("\n" + strings.Repeat("=", 80))
	t.Log("PROTOCOL AUTH BEHAVIOR COMPARISON")
	t.Log(strings.Repeat("=", 80))

	// === RTSP ===
	t.Log("\n### RTSP Protocol ###")
	rtspEntry := models.CameraEntry{
		Type:     "FFMPEG",
		Protocol: "rtsp",
		Port:     554,
		URL:      "/live/ch0",
	}

	rtspURLs := builder.BuildURLsFromEntry(rtspEntry, ctx)

	t.Logf("\nRTSP with credentials (user=%s, pass=%s):", "admin", "***")
	t.Logf("Generated: %d URL(s)", len(rtspURLs))
	for i, url := range rtspURLs {
		t.Logf("  [%d] %s", i+1, url)
	}

	// Check RTSP behavior
	if len(rtspURLs) != 1 {
		t.Errorf("❌ RTSP: Expected 1 URL, got %d", len(rtspURLs))
	}

	hasRTSPAuth := false
	hasRTSPNoAuth := false
	for _, url := range rtspURLs {
		if strings.Contains(url, "@") {
			hasRTSPAuth = true
		} else {
			hasRTSPNoAuth = true
		}
	}

	if !hasRTSPAuth {
		t.Error("❌ RTSP: Should have URL WITH auth")
	}
	if hasRTSPNoAuth {
		t.Error("❌ RTSP: Should NOT have URL without auth when credentials provided")
	}

	if len(rtspURLs) == 1 && hasRTSPAuth && !hasRTSPNoAuth {
		t.Log("✅ RTSP: Correctly generates ONLY auth URL")
	}

	// === HTTP ===
	t.Log("\n### HTTP Protocol ###")
	httpEntry := models.CameraEntry{
		Type:     "JPEG",
		Protocol: "http",
		Port:     80,
		URL:      "snapshot.cgi",
	}

	httpURLs := builder.BuildURLsFromEntry(httpEntry, ctx)

	t.Logf("\nHTTP with credentials (user=%s, pass=%s):", "admin", "***")
	t.Logf("Generated: %d URL(s)", len(httpURLs))
	for i, url := range httpURLs {
		t.Logf("  [%d] %s", i+1, url)
	}

	// Check HTTP behavior
	if len(httpURLs) != 4 {
		t.Errorf("❌ HTTP: Expected 4 URLs, got %d", len(httpURLs))
	}

	// Analyze HTTP URLs
	type authVariant struct {
		name  string
		found bool
		url   string
	}

	variants := []authVariant{
		{name: "No auth", found: false},
		{name: "Basic auth only", found: false},
		{name: "Query params only", found: false},
		{name: "Basic auth + Query params", found: false},
	}

	for _, url := range httpURLs {
		hasBasicAuth := strings.Contains(url, "@")
		hasQueryParams := strings.Contains(url, "?")

		if !hasBasicAuth && !hasQueryParams {
			variants[0].found = true
			variants[0].url = url
		} else if hasBasicAuth && !hasQueryParams {
			variants[1].found = true
			variants[1].url = url
		} else if !hasBasicAuth && hasQueryParams {
			variants[2].found = true
			variants[2].url = url
		} else if hasBasicAuth && hasQueryParams {
			variants[3].found = true
			variants[3].url = url
		}
	}

	t.Log("\nHTTP Auth variants breakdown:")
	allFound := true
	for i, v := range variants {
		if v.found {
			t.Logf("  ✅ [%d] %s: %s", i+1, v.name, v.url)
		} else {
			t.Errorf("  ❌ [%d] %s: MISSING", i+1, v.name)
			allFound = false
		}
	}

	if allFound {
		t.Log("\n✅ HTTP: Correctly generates ALL 4 auth variants")
	} else {
		t.Error("\n❌ HTTP: Missing some auth variants")
	}

	// === COMPARISON SUMMARY ===
	t.Log("\n" + strings.Repeat("=", 80))
	t.Log("SUMMARY")
	t.Log(strings.Repeat("=", 80))
	t.Log("\nRTSP behavior:")
	t.Log("  • With credentials → 1 URL (WITH auth only)")
	t.Log("  • Without credentials → 1 URL (NO auth only)")
	t.Log("  • Rationale: RTSP auth is binary (works or doesn't)")
	t.Log("")
	t.Log("HTTP behavior:")
	t.Log("  • With credentials → 4 URLs:")
	t.Log("    1. No auth (try public access)")
	t.Log("    2. Basic auth only (user:pass@host)")
	t.Log("    3. Query params only (?user=X&pwd=Y)")
	t.Log("    4. Both methods combined")
	t.Log("  • Rationale: Different cameras support different auth methods")
	t.Log(strings.Repeat("=", 80))
}

// TestRTSPNoAuthWhenNoCredentials проверяет что RTSP без credentials НЕ генерирует auth URL
func TestRTSPNoAuthWhenNoCredentials(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	rtspEntry := models.CameraEntry{
		Type:     "FFMPEG",
		Protocol: "rtsp",
		Port:     554,
		URL:      "/live/main",
	}

	// Without credentials
	ctxNoAuth := BuildContext{
		IP:       "192.168.1.100",
		Username: "",
		Password: "",
		Port:     554,
	}

	urls := builder.BuildURLsFromEntry(rtspEntry, ctxNoAuth)

	t.Log("\n=== RTSP WITHOUT credentials ===")
	t.Logf("Generated: %d URL(s)", len(urls))
	for i, url := range urls {
		t.Logf("  [%d] %s", i+1, url)
	}

	if len(urls) != 1 {
		t.Errorf("Expected 1 URL, got %d", len(urls))
	}

	if len(urls) > 0 {
		if strings.Contains(urls[0], "@") {
			t.Error("❌ Should NOT have auth when no credentials provided")
		} else {
			t.Log("✅ Correctly generates URL without auth")
		}
	}
}

// TestHTTPNoAuthWhenNoCredentials проверяет что HTTP без credentials генерирует ТОЛЬКО 1 URL
func TestHTTPNoAuthWhenNoCredentials(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	httpEntry := models.CameraEntry{
		Type:     "JPEG",
		Protocol: "http",
		Port:     80,
		URL:      "snapshot.jpg",
	}

	// Without credentials
	ctxNoAuth := BuildContext{
		IP:       "192.168.1.100",
		Username: "",
		Password: "",
		Port:     80,
	}

	urls := builder.BuildURLsFromEntry(httpEntry, ctxNoAuth)

	t.Log("\n=== HTTP WITHOUT credentials ===")
	t.Logf("Generated: %d URL(s)", len(urls))
	for i, url := range urls {
		t.Logf("  [%d] %s", i+1, url)
	}

	if len(urls) != 1 {
		t.Errorf("Expected 1 URL, got %d", len(urls))
	}

	if len(urls) > 0 {
		if strings.Contains(urls[0], "@") || strings.Contains(urls[0], "?") {
			t.Error("❌ Should NOT have auth when no credentials provided")
		} else {
			t.Log("✅ Correctly generates URL without auth")
		}
	}
}

// TestCompleteProtocolMatrix проверяет полную матрицу протоколов и credentials
func TestCompleteProtocolMatrix(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	type testCase struct {
		protocol     string
		port         int
		url          string
		withCreds    bool
		expectedURLs int
		description  string
	}

	tests := []testCase{
		// RTSP
		{
			protocol:     "rtsp",
			port:         554,
			url:          "/live/ch0",
			withCreds:    true,
			expectedURLs: 1,
			description:  "RTSP with credentials",
		},
		{
			protocol:     "rtsp",
			port:         554,
			url:          "/live/ch0",
			withCreds:    false,
			expectedURLs: 1,
			description:  "RTSP without credentials",
		},
		// HTTP
		{
			protocol:     "http",
			port:         80,
			url:          "snapshot.cgi",
			withCreds:    true,
			expectedURLs: 4,
			description:  "HTTP with credentials",
		},
		{
			protocol:     "http",
			port:         80,
			url:          "snapshot.cgi",
			withCreds:    false,
			expectedURLs: 1,
			description:  "HTTP without credentials",
		},
		// HTTPS
		{
			protocol:     "https",
			port:         443,
			url:          "snapshot.cgi",
			withCreds:    true,
			expectedURLs: 4,
			description:  "HTTPS with credentials",
		},
		{
			protocol:     "https",
			port:         443,
			url:          "snapshot.cgi",
			withCreds:    false,
			expectedURLs: 1,
			description:  "HTTPS without credentials",
		},
	}

	t.Log("\n" + strings.Repeat("=", 80))
	t.Log("COMPLETE PROTOCOL MATRIX")
	t.Log(strings.Repeat("=", 80))

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			entry := models.CameraEntry{
				Type:     "FFMPEG",
				Protocol: tc.protocol,
				Port:     tc.port,
				URL:      tc.url,
			}

			ctx := BuildContext{
				IP:   "192.168.1.100",
				Port: tc.port,
			}

			if tc.withCreds {
				ctx.Username = "admin"
				ctx.Password = "12345"
			}

			urls := builder.BuildURLsFromEntry(entry, ctx)

			t.Logf("Protocol: %s, Creds: %v → Generated: %d URL(s)",
				tc.protocol, tc.withCreds, len(urls))

			if len(urls) != tc.expectedURLs {
				t.Errorf("❌ Expected %d URLs, got %d", tc.expectedURLs, len(urls))
				for i, url := range urls {
					t.Logf("    [%d] %s", i+1, url)
				}
			} else {
				t.Logf("✅ Correct: %d URL(s)", len(urls))
			}
		})
	}

	t.Log(strings.Repeat("=", 80))
}
