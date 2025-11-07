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
	Resolution string
	Codec      string
	FPS        int
	Bitrate    int
	HasAudio   bool
	Error      string
	TestTime   time.Duration
	Metadata   map[string]interface{}
}



// validateHTTPStream validates the HTTP response as a valid stream
func (t *Tester) validateHTTPStream(resp *http.Response, result *TestResult) {
	contentType := resp.Header.Get("Content-Type")
	result.Metadata["content_type"] = contentType
	urlPath := strings.ToLower(resp.Request.URL.Path)

	t.logger.Debug("validating HTTP stream",
		"url", resp.Request.URL.String(),
		"content_type", contentType,
		"status_code", resp.StatusCode)

	// Read first bytes to check magic bytes (up to 512 bytes for MJPEG boundary detection)
	buffer := make([]byte, 512)
	n, _ := resp.Body.Read(buffer)

	// Check for JPEG magic bytes (FF D8 FF)
	hasJPEGMagic := n >= 3 && buffer[0] == 0xFF && buffer[1] == 0xD8 && buffer[2] == 0xFF
	// Check for MJPEG boundary
	hasMJPEGBoundary := n > 0 && bytes.Contains(buffer[:n], []byte("--"))

	t.logger.Debug("stream content analysis",
		"bytes_read", n,
		"has_jpeg_magic", hasJPEGMagic,
		"has_mjpeg_boundary", hasMJPEGBoundary)

	// 1. Check Content-Type for multipart (MJPEG)
	if strings.Contains(contentType, "multipart") {
		result.Type = "MJPEG"
		result.Working = hasMJPEGBoundary
		if !hasMJPEGBoundary {
			result.Error = "no MJPEG boundary found"
		}
		t.logger.Debug("detected MJPEG by content-type", "working", result.Working)
		return
	}

	// 2. Check for JPEG by magic bytes (most reliable)
	if hasJPEGMagic {
		// Verify it's not MJPEG
		if hasMJPEGBoundary {
			result.Type = "MJPEG"
			result.Working = true
			t.logger.Debug("detected MJPEG by magic bytes and boundary")
		} else {
			result.Type = "JPEG"
			result.Working = true
			t.logger.Debug("detected JPEG by magic bytes")
		}
		return
	}

	// 3. Check Content-Type for image/jpeg
	if strings.Contains(contentType, "image/jpeg") || strings.Contains(contentType, "image/jpg") {
		result.Type = "JPEG"
		result.Working = true
		t.logger.Debug("detected JPEG by content-type")
		return
	}

	// 4. Check URL patterns for JPEG (fallback for cameras with wrong Content-Type)
	jpegPatterns := []string{".jpg", ".jpeg", "snapshot", "image", "picture", "snap", "photo", "capture"}
	for _, pattern := range jpegPatterns {
		if strings.Contains(urlPath, pattern) {
			result.Type = "JPEG"
			result.Working = true
			t.logger.Debug("detected JPEG by URL pattern", "pattern", pattern, "url", urlPath)
			result.Metadata["detection_method"] = "url_pattern"
			return
		}
	}

	// 5. Check for MJPEG by extension
	if strings.Contains(urlPath, ".mjpg") || strings.Contains(urlPath, ".mjpeg") {
		result.Type = "MJPEG"
		result.Working = true
		t.logger.Debug("detected MJPEG by URL extension")
		return
	}

	// 6. Check for HLS
	if strings.Contains(urlPath, ".m3u8") ||
		strings.Contains(contentType, "application/vnd.apple.mpegurl") ||
		strings.Contains(contentType, "application/x-mpegurl") {
		result.Type = "HLS"
		result.Working = true
		return
	}

	// 7. Check for MPEG-DASH
	if strings.Contains(urlPath, ".mpd") || strings.Contains(contentType, "application/dash+xml") {
		result.Type = "MPEG-DASH"
		result.Working = true
		return
	}

	// 8. Check for video content type
	if strings.Contains(contentType, "video") {
		result.Type = "HTTP_VIDEO"
		result.Working = true
		return
	}

	// 9. Check for web interface
	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "text/plain") {
		result.Working = false
		result.Error = "web interface, not a video stream"
		return
	}

	// 10. Unknown - but still working if we got 200 OK
	result.Type = "HTTP_UNKNOWN"
	result.Working = true
	result.Metadata["note"] = "unknown content type, may still be valid"
}

// TestStream tests if a stream URL is working
func (t *Tester) TestStream(ctx context.Context, streamURL string) TestResult {
	startTime := time.Now()

	result := TestResult{
		URL:      streamURL,
		Metadata: make(map[string]interface{}),
	}

	// Parse URL to determine protocol
	u, err := url.Parse(streamURL)
	if err != nil {
		result.Error = fmt.Sprintf("invalid URL: %v", err)
		result.TestTime = time.Since(startTime)
		return result
	}

	result.Protocol = u.Scheme

	// Test based on protocol
	switch u.Scheme {
	case "rtsp", "rtsps":
		t.testRTSP(ctx, streamURL, &result)
	case "http", "https":
		t.testHTTP(ctx, streamURL, &result)
	default:
		result.Error = fmt.Sprintf("unsupported protocol: %s", u.Scheme)
	}

	result.TestTime = time.Since(startTime)
	return result
}

// testRTSP tests an RTSP stream using ffprobe
func (t *Tester) testRTSP(ctx context.Context, streamURL string, result *TestResult) {
	// Build ffprobe command
	cmdCtx, cancel := context.WithTimeout(ctx, t.ffprobeTimeout)
	defer cancel()

	// Use URL as-is - credentials already embedded if needed
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		"-rtsp_transport", "tcp",
		streamURL,
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
func (t *Tester) testHTTP(ctx context.Context, streamURL string, result *TestResult) {
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", streamURL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return
	}

	// Extract credentials from URL if present
	u, _ := url.Parse(streamURL)
	if u.User != nil {
		username := u.User.Username()
		password, _ := u.User.Password()
		if username != "" && password != "" {
			req.SetBasicAuth(username, password)
			// Remove credentials from URL for logging
			u.User = nil
			streamURL = u.String()
		}
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

	// Use validateHTTPStream to determine stream type
	t.validateHTTPStream(resp, result)

	// Try to probe with ffprobe for HTTP_VIDEO type for more details
	if result.Type == "HTTP_VIDEO" && result.Working {
		t.probeHTTPVideo(ctx, streamURL, result)
	}
}

// probeHTTPVideo uses ffprobe to get more details about HTTP video stream
func (t *Tester) probeHTTPVideo(ctx context.Context, streamURL string, result *TestResult) {
	cmdCtx, cancel := context.WithTimeout(ctx, t.ffprobeTimeout)
	defer cancel()

	// Use URL as-is - credentials already in URL if needed

	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		streamURL,
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
func (t *Tester) TestMultiple(ctx context.Context, urls []string, maxConcurrent int) []TestResult {
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

			results[i] = t.TestStream(ctx, url)
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