package stream

import (
	"testing"

	"github.com/eduard256/Strix/internal/models"
)

// TestRTSPAuthLogic проверяет логику генерации RTSP URL с авторизацией
func TestRTSPAuthLogic(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	entry := models.CameraEntry{
		Type:     "FFMPEG",
		Protocol: "rtsp",
		Port:     554,
		URL:      "/live/ch0",
	}

	tests := []struct {
		name              string
		ctx               BuildContext
		expectedURLCount  int
		shouldHaveNoAuth  bool
		shouldHaveAuth    bool
		description       string
	}{
		{
			name: "RTSP with credentials - should generate ONLY with auth",
			ctx: BuildContext{
				IP:       "192.168.1.100",
				Username: "admin",
				Password: "12345",
				Port:     554,
			},
			expectedURLCount:  1,
			shouldHaveNoAuth:  false,
			shouldHaveAuth:    true,
			description:       "When credentials provided, generate ONLY URL with auth",
		},
		{
			name: "RTSP without credentials - should generate ONLY without auth",
			ctx: BuildContext{
				IP:       "192.168.1.100",
				Username: "",
				Password: "",
				Port:     554,
			},
			expectedURLCount:  1,
			shouldHaveNoAuth:  true,
			shouldHaveAuth:    false,
			description:       "When NO credentials provided, generate ONLY URL without auth",
		},
		{
			name: "RTSP with only username (no password) - should generate without auth",
			ctx: BuildContext{
				IP:       "192.168.1.100",
				Username: "admin",
				Password: "",
				Port:     554,
			},
			expectedURLCount:  1,
			shouldHaveNoAuth:  true,
			shouldHaveAuth:    false,
			description:       "Username without password = no credentials",
		},
		{
			name: "RTSP with only password (no username) - should generate without auth",
			ctx: BuildContext{
				IP:       "192.168.1.100",
				Username: "",
				Password: "12345",
				Port:     554,
			},
			expectedURLCount:  1,
			shouldHaveNoAuth:  true,
			shouldHaveAuth:    false,
			description:       "Password without username = no credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls := builder.BuildURLsFromEntry(entry, tt.ctx)

			t.Logf("\n=== %s ===", tt.description)
			t.Logf("Context: IP=%s, User=%s, Pass=%s",
				tt.ctx.IP,
				maskString(tt.ctx.Username),
				maskString(tt.ctx.Password))
			t.Logf("Generated URLs: %d", len(urls))

			for i, url := range urls {
				t.Logf("  [%d] %s", i+1, url)
			}

			// Check count
			if len(urls) != tt.expectedURLCount {
				t.Errorf("FAILED: Expected %d URLs, got %d", tt.expectedURLCount, len(urls))
			}

			// Check for auth presence
			hasNoAuth := false
			hasAuth := false

			for _, url := range urls {
				if containsAuth(url) {
					hasAuth = true
				} else {
					hasNoAuth = true
				}
			}

			if tt.shouldHaveNoAuth && !hasNoAuth {
				t.Errorf("FAILED: Expected URL without auth, but none found")
			}
			if !tt.shouldHaveNoAuth && hasNoAuth {
				t.Errorf("FAILED: Expected NO URL without auth, but found one")
			}
			if tt.shouldHaveAuth && !hasAuth {
				t.Errorf("FAILED: Expected URL with auth, but none found")
			}
			if !tt.shouldHaveAuth && hasAuth {
				t.Errorf("FAILED: Expected NO URL with auth, but found one")
			}

			t.Logf("✓ Test passed")
		})
	}
}

// TestHTTPAuthLogic проверяет что HTTP НЕ изменился (все 4 варианта)
func TestHTTPAuthLogic(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	entry := models.CameraEntry{
		Type:     "JPEG",
		Protocol: "http",
		Port:     80,
		URL:      "snapshot.cgi",
	}

	t.Log("\n=== HTTP should generate ALL 4 auth variants (unchanged behavior) ===")

	ctx := BuildContext{
		IP:       "192.168.1.100",
		Username: "admin",
		Password: "12345",
		Port:     80,
	}

	urls := builder.BuildURLsFromEntry(entry, ctx)

	t.Logf("Generated URLs: %d", len(urls))
	for i, url := range urls {
		t.Logf("  [%d] %s", i+1, url)
	}

	expectedCount := 4
	if len(urls) != expectedCount {
		t.Errorf("FAILED: Expected %d URLs for HTTP, got %d", expectedCount, len(urls))
		t.Errorf("HTTP auth variant generation should NOT be changed!")
	} else {
		t.Log("✓ HTTP still generates 4 auth variants (correct)")
	}

	// Verify we have different auth methods
	hasNoAuth := false
	hasBasicAuth := false
	hasQueryAuth := false
	hasBothAuth := false

	for _, url := range urls {
		hasAuth := containsAuth(url)
		hasQuery := containsString(url, "?")

		if !hasAuth && !hasQuery {
			hasNoAuth = true
		} else if hasAuth && !hasQuery {
			hasBasicAuth = true
		} else if !hasAuth && hasQuery {
			hasQueryAuth = true
		} else if hasAuth && hasQuery {
			hasBothAuth = true
		}
	}

	if !hasNoAuth || !hasBasicAuth || !hasQueryAuth || !hasBothAuth {
		t.Error("FAILED: HTTP should have all 4 auth variants:")
		t.Logf("  No auth: %v", hasNoAuth)
		t.Logf("  Basic auth: %v", hasBasicAuth)
		t.Logf("  Query auth: %v", hasQueryAuth)
		t.Logf("  Both: %v", hasBothAuth)
	} else {
		t.Log("✓ All 4 HTTP auth variants present (correct)")
	}
}

// TestHTTPSAuthLogic проверяет что HTTPS работает как HTTP
func TestHTTPSAuthLogic(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	entry := models.CameraEntry{
		Type:     "JPEG",
		Protocol: "https",
		Port:     443,
		URL:      "snapshot.cgi",
	}

	t.Log("\n=== HTTPS should generate ALL 4 auth variants (same as HTTP) ===")

	ctx := BuildContext{
		IP:       "192.168.1.100",
		Username: "admin",
		Password: "12345",
		Port:     443,
	}

	urls := builder.BuildURLsFromEntry(entry, ctx)

	t.Logf("Generated URLs: %d", len(urls))
	for i, url := range urls {
		t.Logf("  [%d] %s", i+1, url)
	}

	expectedCount := 4
	if len(urls) != expectedCount {
		t.Errorf("FAILED: Expected %d URLs for HTTPS, got %d", expectedCount, len(urls))
	} else {
		t.Log("✓ HTTPS generates 4 auth variants (correct)")
	}
}

// TestBUBBLEProtocolUnchanged проверяет что BUBBLE протокол не изменился
func TestBUBBLEProtocolUnchanged(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	entry := models.CameraEntry{
		Type:     "BUBBLE",
		Protocol: "bubble",
		Port:     34567,
		URL:      "/{channel}?stream=0",
	}

	t.Log("\n=== BUBBLE protocol should remain unchanged ===")

	ctx := BuildContext{
		IP:       "192.168.1.100",
		Username: "admin",
		Password: "12345",
		Port:     34567,
		Channel:  1,
	}

	urls := builder.BuildURLsFromEntry(entry, ctx)

	t.Logf("Generated URLs: %d", len(urls))
	for i, url := range urls {
		t.Logf("  [%d] %s", i+1, url)
	}

	if len(urls) < 1 {
		t.Error("FAILED: BUBBLE should generate at least 1 URL")
	} else {
		t.Log("✓ BUBBLE protocol works")
	}
}

// TestRTSPDeduplicationAcrossSources проверяет дедупликацию между источниками
func TestRTSPDeduplicationAcrossSources(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	ctx := BuildContext{
		IP:       "192.168.1.100",
		Username: "admin",
		Password: "12345",
		Port:     554,
	}

	entry := models.CameraEntry{
		Type:     "FFMPEG",
		Protocol: "rtsp",
		Port:     554,
		URL:      "/live/ch0",
	}

	t.Log("\n=== RTSP Deduplication: Each source generates ONLY auth URL ===")

	// Source 1: Model patterns
	modelURLs := builder.BuildURLsFromEntry(entry, ctx)
	t.Logf("Model patterns: %d URLs", len(modelURLs))
	for i, url := range modelURLs {
		t.Logf("  [%d] %s", i+1, url)
	}

	// Source 2: Popular patterns
	popularURLs := builder.BuildURLsFromEntry(entry, ctx)
	t.Logf("Popular patterns: %d URLs", len(popularURLs))
	for i, url := range popularURLs {
		t.Logf("  [%d] %s", i+1, url)
	}

	// Source 3: ONVIF (manual simulation - without auth)
	onvifURL := "rtsp://192.168.1.100:554/live/ch0"
	t.Logf("ONVIF: 1 URL")
	t.Logf("  [1] %s", onvifURL)

	// Current deduplication
	urlMap := make(map[string]bool)
	var combined []string

	for _, url := range modelURLs {
		if !urlMap[url] {
			combined = append(combined, url)
			urlMap[url] = true
		}
	}

	for _, url := range popularURLs {
		if !urlMap[url] {
			combined = append(combined, url)
			urlMap[url] = true
		}
	}

	if !urlMap[onvifURL] {
		combined = append(combined, onvifURL)
		urlMap[onvifURL] = true
	}

	t.Logf("\nAfter deduplication: %d URLs", len(combined))
	for i, url := range combined {
		t.Logf("  [%d] %s", i+1, url)
	}

	// Verify: should have exactly 2 URLs
	// 1. From Model/Popular (with auth): rtsp://admin:12345@192.168.1.100/live/ch0
	// 2. From ONVIF (without auth, with port): rtsp://192.168.1.100:554/live/ch0
	expectedCount := 2
	if len(combined) != expectedCount {
		t.Errorf("FAILED: Expected %d unique URLs, got %d", expectedCount, len(combined))
		t.Log("Expected:")
		t.Log("  1. rtsp://admin:12345@192.168.1.100/live/ch0 (from Model/Popular)")
		t.Log("  2. rtsp://192.168.1.100:554/live/ch0 (from ONVIF)")
	} else {
		t.Log("✓ Deduplication works correctly")
		t.Log("  Model/Popular URLs are identical → deduplicated to 1")
		t.Log("  ONVIF URL is different (has :554 port) → kept as separate")
		t.Log("  Total: 2 unique URLs (correct!)")
	}
}

// TestRTSPWithoutCredentialsSingleURL проверяет что без credentials генерируется 1 URL
func TestRTSPWithoutCredentialsSingleURL(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	entry := models.CameraEntry{
		Type:     "FFMPEG",
		Protocol: "rtsp",
		Port:     554,
		URL:      "/live/main",
	}

	ctx := BuildContext{
		IP:       "192.168.1.100",
		Username: "",
		Password: "",
		Port:     554,
	}

	t.Log("\n=== RTSP without credentials should generate SINGLE URL ===")

	urls := builder.BuildURLsFromEntry(entry, ctx)

	t.Logf("Generated URLs: %d", len(urls))
	for i, url := range urls {
		t.Logf("  [%d] %s", i+1, url)
	}

	if len(urls) != 1 {
		t.Errorf("FAILED: Expected 1 URL without credentials, got %d", len(urls))
	}

	if len(urls) > 0 && containsAuth(urls[0]) {
		t.Error("FAILED: URL should NOT contain auth when no credentials provided")
	}

	t.Log("✓ Single URL without auth generated (correct)")
}

// Helper functions

func containsAuth(url string) bool {
	// Check for user:pass@ pattern
	for i := 0; i < len(url)-3; i++ {
		if url[i:i+3] == "://" {
			// Found protocol, check for @
			for j := i + 3; j < len(url); j++ {
				if url[j] == '@' {
					return true
				}
				if url[j] == '/' {
					break
				}
			}
			break
		}
	}
	return false
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func maskString(s string) string {
	if s == "" {
		return "(empty)"
	}
	return "***"
}
