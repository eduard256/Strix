package stream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

// AuthMethod represents an authentication method
type AuthMethod string

const (
	// AuthNone - no authentication
	AuthNone AuthMethod = "no_auth"
	// AuthBasicHeader - HTTP Basic Auth header only
	AuthBasicHeader AuthMethod = "basic_auth"
	// AuthQueryParams - credentials in query string parameters
	AuthQueryParams AuthMethod = "query_params"
	// AuthCombined - both Basic Auth header and query params (ZOSI requirement)
	AuthCombined AuthMethod = "combined"
	// AuthDigest - HTTP Digest authentication
	AuthDigest AuthMethod = "digest"
	// AuthURLEmbedded - credentials embedded in URL (rtsp://user:pass@host)
	AuthURLEmbedded AuthMethod = "url_embedded"
)

// Tester validates stream URLs
type Tester struct {
	httpClient     *http.Client
	ffprobeTimeout time.Duration
	logger         interface{ Debug(string, ...any); Error(string, error, ...any) }
}

// NewTester creates a new stream tester
func NewTester(ffprobeTimeout time.Duration, logger interface{ Debug(string, ...any); Error(string, error, ...any) }) *Tester {
	return &Tester{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		ffprobeTimeout: ffprobeTimeout,
		logger:         logger,
	}
}

// TestResult contains the test results for a stream
type TestResult struct {
	URL        string
	Working    bool
	Protocol   string
	Type       string
	AuthMethod AuthMethod
	Resolution string
	Codec      string
	FPS        int
	Bitrate    int
	HasAudio   bool
	Error      string
	TestTime   time.Duration
	Metadata   map[string]interface{}
}

// TestStreamWithAuthChain tests a stream URL with multiple authentication methods using smart fallback chain
func (t *Tester) TestStreamWithAuthChain(ctx context.Context, streamURL, username, password string) TestResult {
	startTime := time.Now()

	t.logger.Debug("TestStreamWithAuthChain started",
		"url", streamURL,
		"username", username,
		"has_password", password != "")

	// Parse URL to determine protocol
	u, err := url.Parse(streamURL)
	if err != nil {
		return TestResult{
			URL:      streamURL,
			Error:    fmt.Sprintf("invalid URL: %v", err),
			TestTime: time.Since(startTime),
			Metadata: make(map[string]interface{}),
		}
	}

	// For RTSP, use the original single-method approach (embedded credentials)
	if u.Scheme == "rtsp" || u.Scheme == "rtsps" {
		result := t.testWithAuthMethod(ctx, streamURL, username, password, AuthURLEmbedded)
		result.TestTime = time.Since(startTime)
		return result
	}

	// For HTTP/HTTPS, use smart auth chain
	if u.Scheme == "http" || u.Scheme == "https" {
		// Determine if URL already has auth parameters
		hasAuthParams := t.hasAuthenticationParams(streamURL)

		// Smart priority chain based on URL characteristics
		var authChain []AuthMethod

		if hasAuthParams {
			// URL has auth params - prioritize methods that use them
			authChain = []AuthMethod{
				AuthCombined,      // Try combined first (ZOSI fix!)
				AuthQueryParams,   // Query params only
				AuthBasicHeader,   // Basic Auth header only
				AuthNone,          // No auth (some cameras ignore auth)
			}
		} else {
			// URL doesn't have auth params - standard chain
			authChain = []AuthMethod{
				AuthNone,          // Try without auth first (fast)
				AuthBasicHeader,   // Most common method
				AuthDigest,        // Some older cameras
			}
		}

		t.logger.Debug("auth chain determined",
			"url", streamURL,
			"has_auth_params", hasAuthParams,
			"auth_chain", authChain,
			"chain_length", len(authChain))

		// Try each auth method
		for i, method := range authChain {
			t.logger.Debug("trying auth method",
				"method", method,
				"url", streamURL,
				"attempt", i+1,
				"of", len(authChain))

			result := t.testWithAuthMethod(ctx, streamURL, username, password, method)

			if result.Working {
				// Success! Return immediately
				result.TestTime = time.Since(startTime)
				t.logger.Debug("auth method SUCCEEDED",
					"url", streamURL,
					"method", method,
					"attempt", i+1,
					"of", len(authChain),
					"type", result.Type,
					"protocol", result.Protocol)
				return result
			}

			// Log failed attempt
			t.logger.Debug("auth method FAILED",
				"url", streamURL,
				"method", method,
				"attempt", i+1,
				"of", len(authChain),
				"error", result.Error)

			// Special cases: if we get certain errors, might want to continue or stop
			if result.Error != "" {
				// If 401 Unauthorized, definitely try next auth method
				if strings.Contains(result.Error, "401") || strings.Contains(result.Error, "authentication") {
					continue
				}

				// If connection refused, timeout, or other network errors, no point trying other auth methods
				if strings.Contains(result.Error, "connection refused") ||
				   strings.Contains(result.Error, "timeout") ||
				   strings.Contains(result.Error, "no route to host") {
					result.TestTime = time.Since(startTime)
					return result
				}
			}
		}

		// All methods failed, return last result
		result := TestResult{
			URL:      streamURL,
			Protocol: u.Scheme,
			Error:    fmt.Sprintf("all authentication methods failed"),
			TestTime: time.Since(startTime),
			Metadata: make(map[string]interface{}),
		}
		return result
	}

	// Unsupported protocol
	return TestResult{
		URL:      streamURL,
		Protocol: u.Scheme,
		Error:    fmt.Sprintf("unsupported protocol: %s", u.Scheme),
		TestTime: time.Since(startTime),
		Metadata: make(map[string]interface{}),
	}
}

// hasAuthenticationParams checks if URL contains auth parameters
func (t *Tester) hasAuthenticationParams(streamURL string) bool {
	authParams := []string{
		"user=", "username=", "usr=", "loginuse=",
		"password=", "pass=", "pwd=", "loginpas=", "passwd=",
	}

	lowerURL := strings.ToLower(streamURL)
	for _, param := range authParams {
		if strings.Contains(lowerURL, param) {
			return true
		}
	}
	return false
}

// testWithAuthMethod tests a stream with a specific authentication method
func (t *Tester) testWithAuthMethod(ctx context.Context, streamURL, username, password string, method AuthMethod) TestResult {
	result := TestResult{
		URL:        streamURL,
		AuthMethod: method,
		Metadata:   make(map[string]interface{}),
	}

	// Parse URL
	u, err := url.Parse(streamURL)
	if err != nil {
		result.Error = fmt.Sprintf("invalid URL: %v", err)
		return result
	}

	result.Protocol = u.Scheme

	// Handle based on protocol and auth method
	switch u.Scheme {
	case "rtsp", "rtsps":
		t.testRTSPWithAuth(ctx, streamURL, username, password, method, &result)
	case "http", "https":
		t.testHTTPWithAuth(ctx, streamURL, username, password, method, &result)
	default:
		result.Error = fmt.Sprintf("unsupported protocol: %s", u.Scheme)
	}

	return result
}

// testRTSPWithAuth tests RTSP stream with specific auth method
func (t *Tester) testRTSPWithAuth(ctx context.Context, streamURL, username, password string, method AuthMethod, result *TestResult) {
	// For RTSP, we only support embedded credentials
	if method == AuthURLEmbedded && username != "" && password != "" {
		u, _ := url.Parse(streamURL)
		u.User = url.UserPassword(username, password)
		streamURL = u.String()
	}

	// Use existing RTSP testing logic
	t.testRTSP(ctx, streamURL, username, password, result)
}

// testHTTPWithAuth tests HTTP stream with specific authentication method
func (t *Tester) testHTTPWithAuth(ctx context.Context, streamURL, username, password string, method AuthMethod, result *TestResult) {
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", streamURL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return
	}

	// Apply authentication based on method
	switch method {
	case AuthNone:
		// No authentication - do nothing

	case AuthBasicHeader:
		// Basic Auth header only
		if username != "" && password != "" {
			req.SetBasicAuth(username, password)
		}

	case AuthQueryParams:
		// Query params only (already in URL)
		// No additional action needed

	case AuthCombined:
		// Both Basic Auth header AND query params (ZOSI fix!)
		if username != "" && password != "" {
			req.SetBasicAuth(username, password)
		}
		// Query params already in URL

	case AuthDigest:
		// Digest auth requires a challenge-response flow
		// For now, we'll try basic auth and let the camera upgrade if needed
		if username != "" && password != "" {
			req.SetBasicAuth(username, password)
		}
	}

	// Add headers
	req.Header.Set("User-Agent", "Strix/1.0")

	t.logger.Debug("sending HTTP request",
		"url", streamURL,
		"method", method,
		"has_basic_auth_header", req.Header.Get("Authorization") != "",
		"user_agent", req.Header.Get("User-Agent"))

	// Send request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("HTTP request failed: %v", err)
		t.logger.Debug("HTTP request failed",
			"url", streamURL,
			"method", method,
			"error", err)
		return
	}
	defer resp.Body.Close()

	t.logger.Debug("HTTP response received",
		"url", streamURL,
		"status_code", resp.StatusCode,
		"status", resp.Status,
		"content_type", resp.Header.Get("Content-Type"),
		"content_length", resp.Header.Get("Content-Length"),
		"www_authenticate", resp.Header.Get("WWW-Authenticate"))

	// Check status code
	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)
		t.logger.Debug("HTTP non-200 response",
			"url", streamURL,
			"status_code", resp.StatusCode,
			"error", result.Error)

		// Special handling for 401
		if resp.StatusCode == http.StatusUnauthorized {
			result.Error = "authentication required"
		}
		return
	}

	// Check content type and validate stream
	t.validateHTTPStream(resp, result)
}

// validateHTTPStream validates the HTTP response as a valid stream
func (t *Tester) validateHTTPStream(resp *http.Response, result *TestResult) {
	contentType := resp.Header.Get("Content-Type")
	result.Metadata["content_type"] = contentType

	t.logger.Debug("validating HTTP stream",
		"url", resp.Request.URL.String(),
		"content_type", contentType,
		"status_code", resp.StatusCode)

	// Parse URL to check extension (some cameras don't set Content-Type correctly)
	urlPath := strings.ToLower(resp.Request.URL.Path)

	// Check URL extension first for cameras that don't set Content-Type
	if strings.Contains(urlPath, ".jpg") || strings.Contains(urlPath, ".jpeg") || strings.Contains(urlPath, "snapshot") {
		// Likely a JPEG snapshot - verify with magic bytes
		buffer := make([]byte, 3)
		n, _ := resp.Body.Read(buffer)
		t.logger.Debug("JPEG detection by URL",
			"url", urlPath,
			"bytes_read", n,
			"valid_magic_bytes", n >= 3 && buffer[0] == 0xFF && buffer[1] == 0xD8 && buffer[2] == 0xFF)
		if n >= 3 && buffer[0] == 0xFF && buffer[1] == 0xD8 && buffer[2] == 0xFF {
			result.Type = "JPEG"
			result.Working = true
			t.logger.Debug("stream validated as JPEG by URL extension", "url", urlPath)
			return
		}
	}

	if strings.Contains(urlPath, ".m3u8") {
		result.Type = "HLS"
		result.Working = true
		return
	}

	if strings.Contains(urlPath, ".mpd") {
		result.Type = "MPEG-DASH"
		result.Working = true
		return
	}

	if strings.Contains(urlPath, ".mjpg") || strings.Contains(urlPath, ".mjpeg") {
		result.Type = "MJPEG"
		result.Working = true
		return
	}

	// Determine stream type based on content type
	switch {
	case strings.Contains(contentType, "multipart"):
		result.Type = "MJPEG"
		result.Working = true

		// Read first few bytes to verify
		buffer := make([]byte, 512)
		n, _ := resp.Body.Read(buffer)
		if n > 0 {
			// Check for MJPEG boundary
			if bytes.Contains(buffer[:n], []byte("--")) {
				result.Working = true
			}
		}

	case strings.Contains(contentType, "image/jpeg"), strings.Contains(contentType, "image/jpg"):
		result.Type = "JPEG"
		result.Working = true

		// Read first few bytes to verify JPEG magic bytes
		buffer := make([]byte, 3)
		n, _ := resp.Body.Read(buffer)
		if n >= 3 && buffer[0] == 0xFF && buffer[1] == 0xD8 && buffer[2] == 0xFF {
			result.Working = true
		} else {
			result.Working = false
			result.Error = "invalid JPEG data"
		}

	case strings.Contains(contentType, "video"):
		result.Type = "HTTP_VIDEO"
		result.Working = true

	case strings.Contains(contentType, "application/vnd.apple.mpegurl"), strings.Contains(contentType, "application/x-mpegurl"):
		// HLS stream
		result.Type = "HLS"
		result.Working = true

	case strings.Contains(contentType, "application/dash+xml"):
		// MPEG-DASH stream
		result.Type = "MPEG-DASH"
		result.Working = true

	case strings.Contains(contentType, "text/html"), strings.Contains(contentType, "text/plain"):
		// Ignore web interfaces and plain text responses
		result.Working = false
		result.Error = "web interface, not a video stream"

	default:
		result.Type = "HTTP_UNKNOWN"
		result.Working = true // Assume it works if we got 200 OK
		result.Metadata["note"] = "unknown content type, may still be valid"
	}
}

// TestStream tests if a stream URL is working (legacy method, now uses auth chain)
func (t *Tester) TestStream(ctx context.Context, streamURL, username, password string) TestResult {
	// Delegate to the new auth chain method for better coverage
	return t.TestStreamWithAuthChain(ctx, streamURL, username, password)
}

// testRTSP tests an RTSP stream using ffprobe
func (t *Tester) testRTSP(ctx context.Context, streamURL, username, password string, result *TestResult) {
	// Build ffprobe command
	cmdCtx, cancel := context.WithTimeout(ctx, t.ffprobeTimeout)
	defer cancel()

	// Build URL with credentials if provided
	testURL := streamURL
	if username != "" && password != "" {
		u, _ := url.Parse(streamURL)
		u.User = url.UserPassword(username, password)
		testURL = u.String()
	}

	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		"-rtsp_transport", "tcp",
		testURL,
	}

	cmd := exec.CommandContext(cmdCtx, "ffprobe", args...)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	t.logger.Debug("testing RTSP stream", "url", streamURL)

	// Execute command
	err := cmd.Run()
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			result.Error = "timeout while testing stream"
		} else {
			result.Error = fmt.Sprintf("ffprobe failed: %v", err)
			if stderr.Len() > 0 {
				result.Error += fmt.Sprintf(" (stderr: %s)", stderr.String())
			}
		}
		return
	}

	// Parse ffprobe output
	var probeResult struct {
		Streams []struct {
			CodecName  string `json:"codec_name"`
			CodecType  string `json:"codec_type"`
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			AvgFrameRate string `json:"avg_frame_rate"`
			BitRate    string `json:"bit_rate"`
		} `json:"streams"`
		Format struct {
			BitRate string `json:"bit_rate"`
		} `json:"format"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &probeResult); err != nil {
		result.Error = fmt.Sprintf("failed to parse ffprobe output: %v", err)
		return
	}

	// Extract stream information
	result.Working = len(probeResult.Streams) > 0
	result.Type = "FFMPEG"

	for _, stream := range probeResult.Streams {
		if stream.CodecType == "video" {
			result.Codec = stream.CodecName
			result.Resolution = fmt.Sprintf("%dx%d", stream.Width, stream.Height)

			// Parse frame rate
			if stream.AvgFrameRate != "" {
				parts := strings.Split(stream.AvgFrameRate, "/")
				if len(parts) == 2 {
					// Calculate FPS from fraction
					var num, den int
					fmt.Sscanf(parts[0], "%d", &num)
					fmt.Sscanf(parts[1], "%d", &den)
					if den > 0 {
						result.FPS = num / den
					}
				}
			}

			// Parse bitrate
			if stream.BitRate != "" {
				fmt.Sscanf(stream.BitRate, "%d", &result.Bitrate)
			}
		} else if stream.CodecType == "audio" {
			result.HasAudio = true
		}
	}

	// Use format bitrate if stream bitrate not available
	if result.Bitrate == 0 && probeResult.Format.BitRate != "" {
		fmt.Sscanf(probeResult.Format.BitRate, "%d", &result.Bitrate)
	}

	if !result.Working {
		result.Error = "no streams found"
	}
}

// testHTTP tests an HTTP stream
func (t *Tester) testHTTP(ctx context.Context, streamURL, username, password string, result *TestResult) {
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", streamURL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return
	}

	// Add Basic Auth if credentials provided
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	// Add headers
	req.Header.Set("User-Agent", "Strix/1.0")

	t.logger.Debug("testing HTTP stream", "url", streamURL)

	// Send request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("HTTP request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)

		// Special handling for 401
		if resp.StatusCode == http.StatusUnauthorized {
			result.Error = "authentication required"
		}
		return
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	result.Metadata["content_type"] = contentType

	// Determine stream type based on content type
	switch {
	case strings.Contains(contentType, "multipart"):
		result.Type = "MJPEG"
		result.Working = true

		// Read first few bytes to verify
		buffer := make([]byte, 512)
		n, _ := resp.Body.Read(buffer)
		if n > 0 {
			// Check for MJPEG boundary
			if bytes.Contains(buffer[:n], []byte("--")) {
				result.Working = true
			}
		}

	case strings.Contains(contentType, "image/jpeg"):
		result.Type = "JPEG"
		result.Working = true

		// Read first few bytes to verify JPEG magic bytes
		buffer := make([]byte, 3)
		n, _ := resp.Body.Read(buffer)
		if n >= 3 && buffer[0] == 0xFF && buffer[1] == 0xD8 && buffer[2] == 0xFF {
			result.Working = true
		} else {
			result.Working = false
			result.Error = "invalid JPEG data"
		}

	case strings.Contains(contentType, "video"):
		result.Type = "HTTP_VIDEO"
		result.Working = true

		// Try to probe with ffprobe for more details
		t.probeHTTPVideo(ctx, streamURL, username, password, result)

	case strings.Contains(contentType, "text/html"), strings.Contains(contentType, "text/plain"):
		// Ignore web interfaces and plain text responses
		result.Working = false
		result.Error = "web interface, not a video stream"

	default:
		result.Type = "HTTP_UNKNOWN"
		result.Working = true // Assume it works if we got 200 OK
		result.Metadata["note"] = "unknown content type, may still be valid"
	}
}

// probeHTTPVideo uses ffprobe to get more details about HTTP video stream
func (t *Tester) probeHTTPVideo(ctx context.Context, streamURL, username, password string, result *TestResult) {
	cmdCtx, cancel := context.WithTimeout(ctx, t.ffprobeTimeout)
	defer cancel()

	// Build URL with credentials if needed
	testURL := streamURL
	if username != "" && password != "" && !strings.Contains(streamURL, "@") {
		u, _ := url.Parse(streamURL)
		u.User = url.UserPassword(username, password)
		testURL = u.String()
	}

	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		testURL,
	}

	cmd := exec.CommandContext(cmdCtx, "ffprobe", args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err == nil {
		var probeResult struct {
			Streams []struct {
				CodecName string `json:"codec_name"`
				CodecType string `json:"codec_type"`
				Width     int    `json:"width"`
				Height    int    `json:"height"`
			} `json:"streams"`
		}

		if json.Unmarshal(stdout.Bytes(), &probeResult) == nil {
			for _, stream := range probeResult.Streams {
				if stream.CodecType == "video" {
					result.Codec = stream.CodecName
					result.Resolution = fmt.Sprintf("%dx%d", stream.Width, stream.Height)
					break
				}
			}
		}
	}
}

// TestMultiple tests multiple URLs concurrently
func (t *Tester) TestMultiple(ctx context.Context, urls []string, username, password string, maxConcurrent int) []TestResult {
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}

	results := make([]TestResult, len(urls))
	sem := make(chan struct{}, maxConcurrent)

	for i, url := range urls {
		i, url := i, url // Capture for goroutine

		sem <- struct{}{} // Acquire semaphore
		go func() {
			defer func() { <-sem }() // Release semaphore

			results[i] = t.TestStream(ctx, url, username, password)
		}()
	}

	// Wait for all to complete
	for i := 0; i < maxConcurrent; i++ {
		sem <- struct{}{}
	}

	return results
}

// IsFFProbeAvailable checks if ffprobe is available
func (t *Tester) IsFFProbeAvailable() bool {
	cmd := exec.Command("ffprobe", "-version")
	err := cmd.Run()
	return err == nil
}