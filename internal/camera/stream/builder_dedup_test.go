package stream

import (
	"strings"
	"testing"

	"github.com/eduard256/Strix/internal/models"
)

// mockLogger implements the logger interface for testing
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, args ...any) {}
func (m *mockLogger) Error(msg string, err error, args ...any) {}

// TestCurrentDeduplicationProblems демонстрирует проблемы текущей дедупликации
func TestCurrentDeduplicationProblems(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	tests := []struct {
		name               string
		entry              models.CameraEntry
		ctx                BuildContext
		expectedURLCount   int // Сколько Builder генерирует
		realUniqueCount    int // Сколько реально уникальных
		description        string
	}{
		{
			name: "HTTP auth variants - same endpoint, 4 different URLs",
			entry: models.CameraEntry{
				Type:     "JPEG",
				Protocol: "http",
				Port:     80,
				URL:      "snapshot.jpg",
			},
			ctx: BuildContext{
				IP:       "192.168.1.100",
				Username: "admin",
				Password: "12345",
				Port:     80,
			},
			expectedURLCount: 4, // Builder генерирует 4 варианта
			realUniqueCount:  1, // Но это ОДИН поток
			description:      "PROBLEM: 4 authentication variants of the same HTTP endpoint",
		},
		{
			name: "HTTP with auth placeholders - generates duplicates",
			entry: models.CameraEntry{
				Type:     "JPEG",
				Protocol: "http",
				Port:     80,
				URL:      "snapshot.cgi?user=[USERNAME]&pwd=[PASSWORD]",
			},
			ctx: BuildContext{
				IP:       "192.168.1.100",
				Username: "admin",
				Password: "12345",
				Port:     80,
			},
			expectedURLCount: 4,
			realUniqueCount:  1,
			description:      "PROBLEM: Placeholder replacement + auth variants = duplicates",
		},
		{
			name: "RTSP with/without credentials",
			entry: models.CameraEntry{
				Type:     "FFMPEG",
				Protocol: "rtsp",
				Port:     554,
				URL:      "/live/main",
			},
			ctx: BuildContext{
				IP:       "192.168.1.100",
				Username: "admin",
				Password: "12345",
				Port:     554,
			},
			expectedURLCount: 2, // С credentials и без
			realUniqueCount:  1, // Это один поток
			description:      "PROBLEM: RTSP with and without credentials are both generated",
		},
		{
			name: "RTSP without credentials - only one URL",
			entry: models.CameraEntry{
				Type:     "FFMPEG",
				Protocol: "rtsp",
				Port:     554,
				URL:      "/live/main",
			},
			ctx: BuildContext{
				IP:       "192.168.1.100",
				Username: "",
				Password: "",
				Port:     554,
			},
			expectedURLCount: 1,
			realUniqueCount:  1,
			description:      "OK: No credentials = only one URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls := builder.BuildURLsFromEntry(tt.entry, tt.ctx)

			t.Logf("\n=== %s ===", tt.description)
			t.Logf("Entry: %s://%s", tt.entry.Protocol, tt.entry.URL)
			t.Logf("Expected URL count: %d", tt.expectedURLCount)
			t.Logf("Real unique streams: %d", tt.realUniqueCount)
			t.Logf("Generated URLs:")
			for i, url := range urls {
				t.Logf("  [%d] %s", i+1, url)
			}

			if len(urls) != tt.expectedURLCount {
				t.Errorf("FAILED: Expected %d URLs, got %d", tt.expectedURLCount, len(urls))
			}

			// Демонстрация проблемы
			if len(urls) > tt.realUniqueCount {
				duplicateCount := len(urls) - tt.realUniqueCount
				t.Logf("\n⚠️  PROBLEM: %d semantic duplicates generated", duplicateCount)
				t.Logf("These are different URL strings pointing to the SAME stream!")
				t.Logf("Waste: %d unnecessary tests", duplicateCount)
			}

			// Показать канонические URL
			canonicalURLs := make(map[string][]string)
			for _, url := range urls {
				canonical := makeCanonical(url)
				canonicalURLs[canonical] = append(canonicalURLs[canonical], url)
			}

			t.Logf("\nCanonical URL analysis:")
			for canonical, variants := range canonicalURLs {
				t.Logf("  Canonical: %s", canonical)
				if len(variants) > 1 {
					t.Logf("    ⚠️  Has %d variants (DUPLICATES!):", len(variants))
					for _, v := range variants {
						t.Logf("      - %s", v)
					}
				} else {
					t.Logf("    ✓ Unique")
				}
			}
		})
	}
}

// TestMultipleSourcesDuplication тестирует дубликаты от разных источников
func TestMultipleSourcesDuplication(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	// Симуляция: один и тот же паттерн из двух источников
	entry1 := models.CameraEntry{
		Type:     "FFMPEG",
		Protocol: "rtsp",
		Port:     554,
		URL:      "/Streaming/Channels/101",
	}

	entry2 := models.CameraEntry{
		Type:     "FFMPEG",
		Protocol: "rtsp",
		Port:     554,
		URL:      "/Streaming/Channels/101",
	}

	ctx := BuildContext{
		IP:       "192.168.1.100",
		Username: "admin",
		Password: "12345",
		Port:     554,
	}

	urls1 := builder.BuildURLsFromEntry(entry1, ctx)
	urls2 := builder.BuildURLsFromEntry(entry2, ctx)

	t.Logf("\n=== Multiple Sources Generate Same URLs ===")
	t.Logf("Source 1 (e.g., Popular Patterns):")
	for i, url := range urls1 {
		t.Logf("  [%d] %s", i+1, url)
	}

	t.Logf("\nSource 2 (e.g., Model Patterns):")
	for i, url := range urls2 {
		t.Logf("  [%d] %s", i+1, url)
	}

	// Симуляция текущей дедупликации (простое сравнение строк)
	urlMap := make(map[string]bool)
	var combined []string

	for _, url := range urls1 {
		if !urlMap[url] {
			combined = append(combined, url)
			urlMap[url] = true
		}
	}

	detectedDuplicates := 0
	for _, url := range urls2 {
		if !urlMap[url] {
			combined = append(combined, url)
			urlMap[url] = true
		} else {
			detectedDuplicates++
		}
	}

	t.Logf("\nCurrent deduplication results:")
	t.Logf("  Source 1 URLs: %d", len(urls1))
	t.Logf("  Source 2 URLs: %d", len(urls2))
	t.Logf("  Combined URLs: %d", len(combined))
	t.Logf("  Duplicates detected by string comparison: %d", detectedDuplicates)

	// Канонический анализ
	canonicalMap := make(map[string][]string)
	for _, url := range combined {
		canonical := makeCanonical(url)
		canonicalMap[canonical] = append(canonicalMap[canonical], url)
	}

	realUnique := len(canonicalMap)
	semanticDuplicates := len(combined) - realUnique

	t.Logf("\nCanonical URL analysis:")
	t.Logf("  Real unique streams: %d", realUnique)
	t.Logf("  Semantic duplicates: %d", semanticDuplicates)
	t.Logf("  Current dedup effectiveness: %.1f%%",
		float64(detectedDuplicates)/float64(len(urls1)+len(urls2))*100)
	t.Logf("  Should be dedup effectiveness: %.1f%%",
		float64(semanticDuplicates+detectedDuplicates)/float64(len(urls1)+len(urls2))*100)

	if semanticDuplicates > 0 {
		t.Logf("\n⚠️  PROBLEM: %d semantic duplicates NOT detected", semanticDuplicates)
	}
}

// TestWorstCaseScenario показывает худший сценарий
func TestWorstCaseScenario(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	// Паттерн, который есть везде: Popular + Model + ONVIF
	entry := models.CameraEntry{
		Type:     "JPEG",
		Protocol: "http",
		Port:     80,
		URL:      "snapshot.jpg",
	}

	ctx := BuildContext{
		IP:       "192.168.1.100",
		Username: "admin",
		Password: "12345",
		Port:     80,
	}

	// Симуляция 3 источников
	popularURLs := builder.BuildURLsFromEntry(entry, ctx)
	modelURLs := builder.BuildURLsFromEntry(entry, ctx)

	// ONVIF может вернуть URL без credentials
	onvifURL := "http://192.168.1.100/snapshot.jpg"

	t.Logf("\n=== WORST CASE: Same pattern from 3 sources ===")
	t.Logf("Popular patterns generates: %d URLs", len(popularURLs))
	t.Logf("Model patterns generates: %d URLs", len(modelURLs))
	t.Logf("ONVIF returns: 1 URL")

	// Текущая дедупликация
	urlMap := make(map[string]bool)
	var all []string

	add := func(url string) {
		if !urlMap[url] {
			all = append(all, url)
			urlMap[url] = true
		}
	}

	for _, url := range popularURLs {
		add(url)
	}
	for _, url := range modelURLs {
		add(url)
	}
	add(onvifURL)

	t.Logf("\nAfter current deduplication:")
	t.Logf("  Total URLs to test: %d", len(all))

	for i, url := range all {
		t.Logf("  [%d] %s", i+1, url)
	}

	// Канонический анализ
	canonicalMap := make(map[string][]string)
	for _, url := range all {
		canonical := makeCanonical(url)
		canonicalMap[canonical] = append(canonicalMap[canonical], url)
	}

	t.Logf("\nCanonical analysis:")
	t.Logf("  Real unique streams: %d", len(canonicalMap))
	t.Logf("  URLs being tested: %d", len(all))
	t.Logf("  Waste: %d unnecessary tests (%.1f%%)",
		len(all)-len(canonicalMap),
		float64(len(all)-len(canonicalMap))/float64(len(all))*100)

	if len(all) > 1 {
		t.Logf("\n⚠️  CRITICAL: Testing the same stream %d times!", len(all))
		t.Logf("Expected time waste: ~%d seconds (assuming 2s per test)", (len(all)-1)*2)
	}
}

// makeCanonical - упрощенная нормализация URL для теста
func makeCanonical(rawURL string) string {
	url := rawURL

	// 1. Убрать credentials (user:pass@)
	if idx := strings.Index(url, "://"); idx >= 0 {
		protocol := url[:idx+3]
		rest := url[idx+3:]

		if atIdx := strings.Index(rest, "@"); atIdx >= 0 {
			rest = rest[atIdx+1:]
		}

		url = protocol + rest
	}

	// 2. Убрать auth query параметры
	authParams := []string{
		"user=", "username=", "usr=",
		"pwd=", "password=", "pass=",
	}

	for _, param := range authParams {
		if idx := strings.Index(url, "?"+param); idx >= 0 {
			// Найти конец параметра
			endIdx := strings.Index(url[idx+1:], "&")
			if endIdx >= 0 {
				url = url[:idx+1] + url[idx+1+endIdx+1:]
			} else {
				url = url[:idx]
			}
		}

		if idx := strings.Index(url, "&"+param); idx >= 0 {
			endIdx := strings.Index(url[idx+1:], "&")
			if endIdx >= 0 {
				url = url[:idx] + url[idx+1+endIdx:]
			} else {
				url = url[:idx]
			}
		}
	}

	// 3. Убрать trailing ?
	url = strings.TrimSuffix(url, "?")

	return url
}
