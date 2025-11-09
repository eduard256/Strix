package stream

import (
	"testing"

	"github.com/eduard256/Strix/internal/models"
)

// TestRealWorldDeduplication тестирует реальный сценарий:
// 5 одинаковых URL из 3 разных источников (ONVIF, Model patterns, Popular patterns)
func TestRealWorldDeduplication(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	ctx := BuildContext{
		IP:       "192.168.1.100",
		Username: "admin",
		Password: "12345",
		Channel:  1,
		Port:     554,
	}

	t.Log("\n========================================")
	t.Log("REAL WORLD SCENARIO: Same stream from 3 sources")
	t.Log("========================================\n")

	// === SOURCE 1: ONVIF Discovery ===
	t.Log("=== SOURCE 1: ONVIF Discovery ===")
	onvifStreams := []models.DiscoveredStream{
		{
			URL:      "rtsp://192.168.1.100:554/Streaming/Channels/101",
			Type:     "ONVIF",
			Protocol: "rtsp",
			Port:     554,
			Working:  true, // ONVIF streams are pre-verified
		},
	}
	t.Logf("ONVIF discovered: %d URLs", len(onvifStreams))
	for i, s := range onvifStreams {
		t.Logf("  [ONVIF-%d] %s", i+1, s.URL)
	}

	// === SOURCE 2: Model-specific patterns (Hikvision) ===
	t.Log("\n=== SOURCE 2: Model-specific patterns (Hikvision DS-2CD2086) ===")
	modelEntry := models.CameraEntry{
		Models:   []string{"DS-2CD2086G2-I", "DS-2CD2042WD"},
		Type:     "FFMPEG",
		Protocol: "rtsp",
		Port:     554,
		URL:      "/Streaming/Channels/101",
	}
	modelURLs := builder.BuildURLsFromEntry(modelEntry, ctx)
	t.Logf("Model patterns generated: %d URLs", len(modelURLs))
	for i, url := range modelURLs {
		t.Logf("  [MODEL-%d] %s", i+1, url)
	}

	// === SOURCE 3: Popular patterns ===
	t.Log("\n=== SOURCE 3: Popular patterns (generic RTSP) ===")
	popularEntry := models.CameraEntry{
		Type:     "FFMPEG",
		Protocol: "rtsp",
		Port:     554,
		URL:      "/Streaming/Channels/101",
	}
	popularURLs := builder.BuildURLsFromEntry(popularEntry, ctx)
	t.Logf("Popular patterns generated: %d URLs", len(popularURLs))
	for i, url := range popularURLs {
		t.Logf("  [POPULAR-%d] %s", i+1, url)
	}

	// === CURRENT DEDUPLICATION (как в scanner.go:235-395) ===
	t.Log("\n=== CURRENT DEDUPLICATION (string comparison) ===")
	urlMap := make(map[string]bool)
	var allStreams []models.DiscoveredStream

	// Add ONVIF streams
	for _, stream := range onvifStreams {
		if !urlMap[stream.URL] {
			allStreams = append(allStreams, stream)
			urlMap[stream.URL] = true
			t.Logf("✓ Added: %s (from ONVIF)", stream.URL)
		} else {
			t.Logf("✗ Skipped: %s (duplicate from ONVIF)", stream.URL)
		}
	}

	// Add Model URLs
	for _, url := range modelURLs {
		if !urlMap[url] {
			allStreams = append(allStreams, models.DiscoveredStream{
				URL:      url,
				Type:     modelEntry.Type,
				Protocol: modelEntry.Protocol,
				Port:     modelEntry.Port,
			})
			urlMap[url] = true
			t.Logf("✓ Added: %s (from Model)", url)
		} else {
			t.Logf("✗ Skipped: %s (duplicate from Model)", url)
		}
	}

	// Add Popular URLs
	for _, url := range popularURLs {
		if !urlMap[url] {
			allStreams = append(allStreams, models.DiscoveredStream{
				URL:      url,
				Type:     popularEntry.Type,
				Protocol: popularEntry.Protocol,
				Port:     popularEntry.Port,
			})
			urlMap[url] = true
			t.Logf("✓ Added: %s (from Popular)", url)
		} else {
			t.Logf("✗ Skipped: %s (duplicate from Popular)", url)
		}
	}

	// === RESULTS ===
	t.Log("\n========================================")
	t.Log("DEDUPLICATION RESULTS")
	t.Log("========================================")

	totalGenerated := len(onvifStreams) + len(modelURLs) + len(popularURLs)
	t.Logf("Total URLs generated: %d", totalGenerated)
	t.Logf("  - From ONVIF: %d", len(onvifStreams))
	t.Logf("  - From Model: %d", len(modelURLs))
	t.Logf("  - From Popular: %d", len(popularURLs))
	t.Logf("\nURLs after deduplication: %d", len(allStreams))
	t.Logf("Duplicates removed: %d", totalGenerated-len(allStreams))

	// List final URLs
	t.Log("\nFinal URLs to test:")
	for i, stream := range allStreams {
		t.Logf("  [%d] %s (type: %s)", i+1, stream.URL, stream.Type)
	}

	// === CANONICAL ANALYSIS (показывает реальные дубликаты) ===
	t.Log("\n========================================")
	t.Log("CANONICAL ANALYSIS (semantic duplicates)")
	t.Log("========================================")

	canonicalMap := make(map[string][]string)
	for _, stream := range allStreams {
		canonical := normalizeURLForComparison(stream.URL)
		canonicalMap[canonical] = append(canonicalMap[canonical], stream.URL)
	}

	realUnique := len(canonicalMap)
	semanticDuplicates := len(allStreams) - realUnique

	t.Logf("Real unique streams: %d", realUnique)
	t.Logf("Semantic duplicates: %d", semanticDuplicates)

	if semanticDuplicates > 0 {
		t.Log("\n⚠️  PROBLEM: Multiple URLs point to the SAME stream:")
		for canonical, variants := range canonicalMap {
			if len(variants) > 1 {
				t.Logf("\n  Canonical: %s", canonical)
				t.Logf("  Variants (%d):", len(variants))
				for _, v := range variants {
					t.Logf("    - %s", v)
				}
				t.Logf("  ⚠️  This stream will be tested %d times!", len(variants))
			}
		}

		t.Logf("\n⚠️  WASTE: %d unnecessary tests", semanticDuplicates)
		t.Logf("Time waste: ~%d seconds (assuming 2s per test)", semanticDuplicates*2)
		t.Logf("Bandwidth waste: ~%d KB (assuming 100KB per test)", semanticDuplicates*100)
	} else {
		t.Log("\n✓ No semantic duplicates found")
	}

	// === ASSERTION ===
	if semanticDuplicates > 0 {
		t.Errorf("DEDUPLICATION FAILED: %d semantic duplicates not removed", semanticDuplicates)
	}
}

// TestHTTPAuthVariantsDuplication проверяет дубликаты от HTTP auth вариантов
func TestHTTPAuthVariantsDuplication(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	ctx := BuildContext{
		IP:       "192.168.1.100",
		Username: "admin",
		Password: "12345",
		Port:     80,
	}

	t.Log("\n========================================")
	t.Log("HTTP AUTHENTICATION VARIANTS TEST")
	t.Log("========================================\n")

	// Один entry для HTTP
	entry := models.CameraEntry{
		Type:     "JPEG",
		Protocol: "http",
		Port:     80,
		URL:      "snapshot.cgi",
	}

	t.Log("Entry: http://192.168.1.100/snapshot.cgi")
	t.Log("\nBuilder generates auth variants:")

	urls := builder.BuildURLsFromEntry(entry, ctx)
	for i, url := range urls {
		t.Logf("  [%d] %s", i+1, url)
	}

	t.Logf("\nTotal URLs generated: %d", len(urls))

	// Canonical analysis
	canonicalMap := make(map[string][]string)
	for _, url := range urls {
		canonical := normalizeURLForComparison(url)
		canonicalMap[canonical] = append(canonicalMap[canonical], url)
	}

	t.Logf("Real unique endpoints: %d", len(canonicalMap))
	semanticDuplicates := len(urls) - len(canonicalMap)
	t.Logf("Semantic duplicates: %d", semanticDuplicates)

	if semanticDuplicates > 0 {
		t.Log("\n⚠️  PROBLEM: Multiple auth variants for the SAME endpoint:")
		for canonical, variants := range canonicalMap {
			if len(variants) > 1 {
				t.Logf("\n  Endpoint: %s", canonical)
				t.Logf("  Auth variants (%d):", len(variants))
				for j, v := range variants {
					t.Logf("    [%d] %s", j+1, v)
				}
			}
		}

		t.Logf("\n⚠️  All %d variants will be tested, but only 1 will likely work", len(urls))
		t.Logf("Expected success rate: ~25%% (1 out of 4)")
		t.Logf("Expected failures: ~%d", len(urls)-1)
	}

	// Note: это НЕ ошибка - это feature для повышения шансов найти рабочий вариант auth
	t.Log("\nNOTE: This is intentional - trying multiple auth methods increases success rate")
	t.Log("But it does mean testing the same stream multiple times with different credentials")
}

// TestFiveIdenticalURLsFromThreeSources - главный тест: ровно 5 одинаковых URL
func TestFiveIdenticalURLsFromThreeSources(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	ctx := BuildContext{
		IP:       "192.168.1.100",
		Username: "admin",
		Password: "password123",
		Port:     554,
	}

	t.Log("\n========================================")
	t.Log("TEST: 5 IDENTICAL URLs from 3 SOURCES")
	t.Log("========================================\n")

	// SOURCE 1: ONVIF - returns 1 URL without auth
	onvifURL := "rtsp://192.168.1.100:554/live/ch0"
	t.Log("SOURCE 1 - ONVIF Discovery:")
	t.Logf("  Returns: %s", onvifURL)

	// SOURCE 2: Model patterns - generates 2 URLs (with/without auth)
	modelEntry := models.CameraEntry{
		Type:     "FFMPEG",
		Protocol: "rtsp",
		Port:     554,
		URL:      "/live/ch0",
	}
	modelURLs := builder.BuildURLsFromEntry(modelEntry, ctx)
	t.Log("\nSOURCE 2 - Model Patterns (Hikvision):")
	t.Logf("  Generates: %d URLs", len(modelURLs))
	for i, url := range modelURLs {
		t.Logf("    [%d] %s", i+1, url)
	}

	// SOURCE 3: Popular patterns - generates 2 URLs (with/without auth)
	popularEntry := models.CameraEntry{
		Type:     "FFMPEG",
		Protocol: "rtsp",
		Port:     554,
		URL:      "/live/ch0",
	}
	popularURLs := builder.BuildURLsFromEntry(popularEntry, ctx)
	t.Log("\nSOURCE 3 - Popular Patterns:")
	t.Logf("  Generates: %d URLs", len(popularURLs))
	for i, url := range popularURLs {
		t.Logf("    [%d] %s", i+1, url)
	}

	// Simulate current deduplication
	urlMap := make(map[string]bool)
	var combined []string

	// Add ONVIF
	if !urlMap[onvifURL] {
		combined = append(combined, onvifURL)
		urlMap[onvifURL] = true
	}

	// Add Model
	for _, url := range modelURLs {
		if !urlMap[url] {
			combined = append(combined, url)
			urlMap[url] = true
		}
	}

	// Add Popular
	for _, url := range popularURLs {
		if !urlMap[url] {
			combined = append(combined, url)
			urlMap[url] = true
		}
	}

	t.Log("\n========================================")
	t.Log("RESULTS")
	t.Log("========================================")

	totalGenerated := 1 + len(modelURLs) + len(popularURLs)
	t.Logf("Total URLs from all sources: %d", totalGenerated)
	t.Logf("  ONVIF: 1")
	t.Logf("  Model: %d", len(modelURLs))
	t.Logf("  Popular: %d", len(popularURLs))

	t.Logf("\nAfter string-based deduplication: %d URLs", len(combined))
	t.Logf("Removed by string comparison: %d", totalGenerated-len(combined))

	t.Log("\nFinal URLs to test:")
	for i, url := range combined {
		t.Logf("  [%d] %s", i+1, url)
	}

	// Canonical analysis
	canonicalMap := make(map[string][]string)
	for _, url := range combined {
		canonical := normalizeURLForComparison(url)
		canonicalMap[canonical] = append(canonicalMap[canonical], url)
	}

	realUnique := len(canonicalMap)
	semanticDuplicates := len(combined) - realUnique

	t.Log("\n========================================")
	t.Log("SEMANTIC ANALYSIS")
	t.Log("========================================")
	t.Logf("Real unique streams: %d", realUnique)
	t.Logf("Semantic duplicates: %d", semanticDuplicates)

	if semanticDuplicates > 0 {
		t.Log("\n⚠️  CRITICAL ISSUE:")
		t.Logf("The same stream will be tested %d times!", len(combined))
		t.Log("\nBreakdown:")
		for canonical, variants := range canonicalMap {
			t.Logf("\n  Stream: %s", canonical)
			t.Logf("  Will be tested %d times as:", len(variants))
			for i, v := range variants {
				t.Logf("    [%d] %s", i+1, v)
			}
		}

		t.Log("\n⚠️  IMPACT:")
		t.Logf("  - Wasted tests: %d", semanticDuplicates)
		t.Logf("  - Wasted time: ~%d seconds", semanticDuplicates*2)
		t.Logf("  - Efficiency: %.1f%% (should be 100%%)",
			float64(realUnique)/float64(len(combined))*100)

		t.Errorf("\nDEDUPLICATION FAILED: %d duplicates not detected", semanticDuplicates)
	} else {
		t.Log("\n✓ SUCCESS: All duplicates properly detected")
	}
}

// normalizeURLForComparison убирает различия в auth для сравнения
func normalizeURLForComparison(rawURL string) string {
	// Простая нормализация: убираем user:pass@ из URL
	url := rawURL

	// Найти protocol://
	protocolEnd := 0
	for i := 0; i < len(url)-3; i++ {
		if url[i:i+3] == "://" {
			protocolEnd = i + 3
			break
		}
	}

	if protocolEnd == 0 {
		return url
	}

	protocol := url[:protocolEnd]
	rest := url[protocolEnd:]

	// Убрать user:pass@
	atIndex := -1
	for i := 0; i < len(rest); i++ {
		if rest[i] == '@' {
			atIndex = i
			break
		}
		if rest[i] == '/' {
			break
		}
	}

	if atIndex >= 0 {
		rest = rest[atIndex+1:]
	}

	return protocol + rest
}
