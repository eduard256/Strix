package discovery

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/strix-project/strix/internal/camera/database"
	"github.com/strix-project/strix/internal/camera/stream"
	"github.com/strix-project/strix/internal/models"
	"github.com/strix-project/strix/pkg/sse"
)

// Scanner orchestrates stream discovery
type Scanner struct {
	loader       *database.Loader
	searchEngine *database.SearchEngine
	builder      *stream.Builder
	tester       *stream.Tester
	onvif        *ONVIFDiscovery
	config       ScannerConfig
	logger       interface{ Debug(string, ...any); Error(string, error, ...any); Info(string, ...any) }
}

// ScannerConfig contains scanner configuration
type ScannerConfig struct {
	WorkerPoolSize   int
	DefaultTimeout   time.Duration
	MaxStreams       int
	ModelSearchLimit int
	FFProbeTimeout   time.Duration
}

// NewScanner creates a new stream scanner
func NewScanner(
	loader *database.Loader,
	searchEngine *database.SearchEngine,
	builder *stream.Builder,
	tester *stream.Tester,
	onvif *ONVIFDiscovery,
	config ScannerConfig,
	logger interface{ Debug(string, ...any); Error(string, error, ...any); Info(string, ...any) },
) *Scanner {
	return &Scanner{
		loader:       loader,
		searchEngine: searchEngine,
		builder:      builder,
		tester:       tester,
		onvif:        onvif,
		config:       config,
		logger:       logger,
	}
}

// ScanResult contains the scan results
type ScanResult struct {
	Streams      []models.DiscoveredStream
	TotalTested  int
	TotalFound   int
	Duration     time.Duration
	Error        error
}

// Scan performs stream discovery
func (s *Scanner) Scan(ctx context.Context, req models.StreamDiscoveryRequest, streamWriter *sse.StreamWriter) (*ScanResult, error) {
	startTime := time.Now()
	result := &ScanResult{}

	// Set defaults
	if req.Timeout <= 0 {
		req.Timeout = int(s.config.DefaultTimeout.Seconds())
	}
	if req.MaxStreams <= 0 {
		req.MaxStreams = s.config.MaxStreams
	}
	if req.ModelLimit <= 0 {
		req.ModelLimit = s.config.ModelSearchLimit
	}

	// Create context with timeout
	scanCtx, cancel := context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Second)
	defer cancel()

	s.logger.Info("starting stream discovery",
		"target", req.Target,
		"model", req.Model,
		"timeout", req.Timeout,
		"max_streams", req.MaxStreams,
	)

	// Send initial message
	streamWriter.SendJSON("scan_started", map[string]interface{}{
		"target":      req.Target,
		"model":       req.Model,
		"max_streams": req.MaxStreams,
		"timeout":     req.Timeout,
	})

	// Check if target is a direct stream URL
	if s.isDirectStreamURL(req.Target) {
		return s.scanDirectStream(scanCtx, req, streamWriter, result)
	}

	// Extract IP from target
	ip := s.extractIP(req.Target)
	if ip == "" {
		err := fmt.Errorf("invalid target IP: %s", req.Target)
		streamWriter.SendError(err)
		result.Error = err
		return result, err
	}

	// Collect all URLs to test
	urls, err := s.collectURLs(scanCtx, req, ip)
	if err != nil {
		streamWriter.SendError(err)
		result.Error = err
		return result, err
	}

	s.logger.Info("collected URLs for testing", "count", len(urls))

	// Send progress update
	streamWriter.SendJSON("progress", models.ProgressMessage{
		Tested:    0,
		Found:     0,
		Remaining: len(urls),
	})

	// Test URLs concurrently
	s.testURLsConcurrently(scanCtx, urls, req, streamWriter, result)

	// Calculate duration
	result.Duration = time.Since(startTime)

	// Send completion message
	streamWriter.SendJSON("complete", models.CompleteMessage{
		TotalTested: result.TotalTested,
		TotalFound:  result.TotalFound,
		Duration:    result.Duration.Seconds(),
	})

	// Send final done event to signal proper stream closure
	streamWriter.SendJSON("done", map[string]interface{}{
		"message": "Stream discovery finished",
	})

	// Small delay to ensure all data is flushed to client
	time.Sleep(100 * time.Millisecond)

	s.logger.Info("stream discovery completed",
		"tested", result.TotalTested,
		"found", result.TotalFound,
		"duration", result.Duration,
	)

	return result, nil
}

// isDirectStreamURL checks if target is a direct stream URL
func (s *Scanner) isDirectStreamURL(target string) bool {
	u, err := url.Parse(target)
	if err != nil {
		return false
	}
	return u.Scheme == "rtsp" || u.Scheme == "http" || u.Scheme == "https"
}

// scanDirectStream scans a direct stream URL
func (s *Scanner) scanDirectStream(ctx context.Context, req models.StreamDiscoveryRequest, streamWriter *sse.StreamWriter, result *ScanResult) (*ScanResult, error) {
	s.logger.Debug("testing direct stream URL", "url", req.Target)

	testResult := s.tester.TestStream(ctx, req.Target, req.Username, req.Password)
	result.TotalTested = 1

	if testResult.Working {
		result.TotalFound = 1

		// Embed credentials in URL for basic_auth and combined methods
		finalURL := s.embedCredentialsInURL(testResult.URL, req.Username, req.Password, string(testResult.AuthMethod))

		discoveredStream := models.DiscoveredStream{
			URL:        finalURL,
			Type:       testResult.Type,
			Protocol:   testResult.Protocol,
			Working:    true,
			AuthMethod: string(testResult.AuthMethod),
			Resolution: testResult.Resolution,
			Codec:      testResult.Codec,
			FPS:        testResult.FPS,
			Bitrate:    testResult.Bitrate,
			HasAudio:   testResult.HasAudio,
			TestTime:   testResult.TestTime,
			Metadata:   testResult.Metadata,
		}

		result.Streams = append(result.Streams, discoveredStream)

		// Send to SSE
		streamWriter.SendJSON("stream_found", map[string]interface{}{
			"stream": discoveredStream,
		})
	} else {
		streamWriter.SendJSON("stream_failed", map[string]interface{}{
			"url":   req.Target,
			"error": testResult.Error,
		})
	}

	return result, nil
}

// extractIP extracts IP address from target
func (s *Scanner) extractIP(target string) string {
	// Remove protocol if present
	if u, err := url.Parse(target); err == nil && u.Host != "" {
		target = u.Host
	}

	// Remove port if present
	if idx := len(target) - 1; idx >= 0 && target[idx] == ']' {
		// IPv6 address
		return target
	}

	for i := len(target) - 1; i >= 0; i-- {
		if target[i] == ':' {
			return target[:i]
		}
	}

	return target
}

// embedCredentialsInURL embeds username and password in URL for basic_auth and combined methods
func (s *Scanner) embedCredentialsInURL(streamURL, username, password, authMethod string) string {
	// Only apply for basic_auth and combined methods
	if authMethod != "basic_auth" && authMethod != "combined" {
		return streamURL
	}

	// Check if credentials are provided
	if username == "" || password == "" {
		return streamURL
	}

	// Parse URL
	u, err := url.Parse(streamURL)
	if err != nil {
		s.logger.Debug("failed to parse URL for credential embedding",
			"url", streamURL,
			"error", err.Error())
		return streamURL
	}

	// Check if credentials already exist in URL
	if u.User != nil {
		s.logger.Debug("credentials already exist in URL, skipping embedding",
			"url", streamURL)
		return streamURL
	}

	// Embed credentials
	u.User = url.UserPassword(username, password)
	embeddedURL := u.String()

	s.logger.Debug("credentials embedded in URL",
		"original_url", streamURL,
		"embedded_url", embeddedURL,
		"auth_method", authMethod)

	return embeddedURL
}

// collectURLs collects all URLs to test
func (s *Scanner) collectURLs(ctx context.Context, req models.StreamDiscoveryRequest, ip string) ([]string, error) {
	var allURLs []string
	urlMap := make(map[string]bool) // For deduplication
	var onvifCount, modelCount, popularCount int

	s.logger.Debug("collectURLs started",
		"ip", ip,
		"model", req.Model,
		"username", req.Username,
		"channel", req.Channel)

	// Build context for URL generation
	buildCtx := stream.BuildContext{
		IP:       ip,
		Username: req.Username,
		Password: req.Password,
		Channel:  req.Channel,
	}

	// 1. ONVIF discovery (always first)
	s.logger.Debug("phase 1: starting ONVIF discovery", "ip", ip)
	onvifStreams, err := s.onvif.DiscoverStreamsForIP(ctx, ip, req.Username, req.Password)
	if err != nil {
		s.logger.Error("ONVIF discovery failed", err)
	} else {
		for _, stream := range onvifStreams {
			if !urlMap[stream.URL] {
				allURLs = append(allURLs, stream.URL)
				urlMap[stream.URL] = true
				onvifCount++
			}
		}
		s.logger.Debug("ONVIF discovery completed",
			"streams_found", len(onvifStreams),
			"unique_urls_added", onvifCount)
	}

	// 2. Model-specific patterns
	if req.Model != "" {
		s.logger.Debug("phase 2: searching model-specific patterns",
			"model", req.Model,
			"limit", req.ModelLimit)

		// Search for similar models
		cameras, err := s.searchEngine.SearchByModel(req.Model, 0.8, req.ModelLimit)
		if err != nil {
			s.logger.Error("model search failed", err)
		} else {
			// Collect entries from all matching cameras
			var entries []models.CameraEntry
			for _, camera := range cameras {
				entries = append(entries, camera.Entries...)
			}

			s.logger.Debug("model entries collected",
				"cameras_matched", len(cameras),
				"total_entries", len(entries))

			// Build URLs from entries
			for _, entry := range entries {
				buildCtx.Port = entry.Port
				buildCtx.Protocol = entry.Protocol

				urls := s.builder.BuildURLsFromEntry(entry, buildCtx)
				for _, url := range urls {
					if !urlMap[url] {
						allURLs = append(allURLs, url)
						urlMap[url] = true
						modelCount++
					}
				}
			}

			s.logger.Debug("model patterns URLs built",
				"total_unique_model_urls", modelCount)
		}
	}

	// 3. Popular patterns (always add as fallback)
	s.logger.Debug("phase 3: adding popular patterns")
	patterns, err := s.loader.LoadPopularPatterns()
	if err != nil {
		s.logger.Error("failed to load popular patterns", err)
	} else {
		s.logger.Debug("popular patterns loaded", "count", len(patterns))

		for _, pattern := range patterns {
			entry := models.CameraEntry{
				Type:     pattern.Type,
				Protocol: pattern.Protocol,
				Port:     pattern.Port,
				URL:      pattern.URL,
			}

			buildCtx.Port = pattern.Port
			buildCtx.Protocol = pattern.Protocol

			url := s.builder.BuildURL(entry, buildCtx)
			if !urlMap[url] {
				allURLs = append(allURLs, url)
				urlMap[url] = true
				popularCount++
			}
		}
	}

	totalBeforeDedup := onvifCount + modelCount + popularCount
	duplicatesRemoved := totalBeforeDedup - len(allURLs)

	s.logger.Debug("URL collection complete",
		"total_unique_urls", len(allURLs),
		"from_onvif", onvifCount,
		"from_model_patterns", modelCount,
		"from_popular_patterns", popularCount,
		"total_before_dedup", totalBeforeDedup,
		"duplicates_removed", duplicatesRemoved)

	return allURLs, nil
}

// testURLsConcurrently tests URLs concurrently
func (s *Scanner) testURLsConcurrently(ctx context.Context, urls []string, req models.StreamDiscoveryRequest, streamWriter *sse.StreamWriter, result *ScanResult) {
	var wg sync.WaitGroup
	var tested int32
	var found int32

	// Create worker pool
	sem := make(chan struct{}, s.config.WorkerPoolSize)
	streamsChan := make(chan models.DiscoveredStream, 100)

	// Start periodic progress updates
	progressCtx, cancelProgress := context.WithCancel(ctx)
	defer cancelProgress()

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		lastTested := int32(0)

		for {
			select {
			case <-progressCtx.Done():
				return
			case <-ticker.C:
				currentTested := atomic.LoadInt32(&tested)
				// Only send if there's been progress
				if currentTested != lastTested {
					streamWriter.SendJSON("progress", models.ProgressMessage{
						Tested:    int(currentTested),
						Found:     int(atomic.LoadInt32(&found)),
						Remaining: len(urls) - int(currentTested),
					})
					lastTested = currentTested
				}
			}
		}
	}()

	// Start result collector
	go func() {
		for stream := range streamsChan {
			result.Streams = append(result.Streams, stream)

			// Send to SSE
			streamWriter.SendJSON("stream_found", map[string]interface{}{
				"stream": stream,
			})

			// Send progress (immediate update when stream is found)
			streamWriter.SendJSON("progress", models.ProgressMessage{
				Tested:    int(atomic.LoadInt32(&tested)),
				Found:     int(atomic.LoadInt32(&found)),
				Remaining: len(urls) - int(atomic.LoadInt32(&tested)),
			})

			// Check if we've found enough streams
			if int(atomic.LoadInt32(&found)) >= req.MaxStreams {
				s.logger.Debug("max streams reached", "count", req.MaxStreams)
			}
		}
	}()

	// Test each URL
	for _, url := range urls {
		// Check if context is done or max streams reached
		select {
		case <-ctx.Done():
			s.logger.Debug("scan cancelled or timeout")
			break
		default:
		}

		if int(atomic.LoadInt32(&found)) >= req.MaxStreams {
			break
		}

		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Test the stream
			testResult := s.tester.TestStream(ctx, url, req.Username, req.Password)
			atomic.AddInt32(&tested, 1)

			if testResult.Working {
				atomic.AddInt32(&found, 1)

				// Embed credentials in URL for basic_auth and combined methods
				finalURL := s.embedCredentialsInURL(testResult.URL, req.Username, req.Password, string(testResult.AuthMethod))

				discoveredStream := models.DiscoveredStream{
					URL:        finalURL,
					Type:       testResult.Type,
					Protocol:   testResult.Protocol,
					Port:       0, // Will be extracted from URL if needed
					Working:    true,
					AuthMethod: string(testResult.AuthMethod),
					Resolution: testResult.Resolution,
					Codec:      testResult.Codec,
					FPS:        testResult.FPS,
					Bitrate:    testResult.Bitrate,
					HasAudio:   testResult.HasAudio,
					TestTime:   testResult.TestTime,
					Metadata:   testResult.Metadata,
				}

				streamsChan <- discoveredStream
			} else {
				s.logger.Debug("stream test failed", "url", url, "error", testResult.Error)
			}
		}(url)
	}

	// Wait for all tests to complete
	wg.Wait()
	close(streamsChan)

	// Update final counts
	result.TotalTested = int(atomic.LoadInt32(&tested))
	result.TotalFound = int(atomic.LoadInt32(&found))
}