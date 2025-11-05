package discovery

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/IOTechSystems/onvif"
	"github.com/IOTechSystems/onvif/media"
	xsdonvif "github.com/IOTechSystems/onvif/xsd/onvif"
	"github.com/strix-project/strix/internal/models"
)

// ONVIFDiscovery handles ONVIF device discovery and stream detection
type ONVIFDiscovery struct {
	logger interface{ Debug(string, ...any); Error(string, error, ...any) }
}

// NewONVIFDiscovery creates a new ONVIF discovery instance
func NewONVIFDiscovery(logger interface{ Debug(string, ...any); Error(string, error, ...any) }) *ONVIFDiscovery {
	return &ONVIFDiscovery{
		logger: logger,
	}
}

// DiscoverStreamsForIP discovers all possible streams for a given IP
func (o *ONVIFDiscovery) DiscoverStreamsForIP(ctx context.Context, ip, username, password string) ([]models.DiscoveredStream, error) {
	o.logger.Debug("=== ONVIF DiscoverStreamsForIP STARTED ===",
		"ip", ip,
		"username", username,
		"password_len", len(password))

	// Clean IP (remove port if present)
	if idx := strings.IndexByte(ip, ':'); idx > 0 {
		o.logger.Debug("cleaning IP address", "original", ip, "cleaned", ip[:idx])
		ip = ip[:idx]
	}

	var allStreams []models.DiscoveredStream

	// Try real ONVIF discovery first
	o.logger.Debug(">>> Starting ONVIF device discovery", "ip", ip)
	onvifStreams := o.discoverViaONVIF(ctx, ip, username, password)
	o.logger.Debug("<<< ONVIF device discovery completed", "streams_found", len(onvifStreams))

	if len(onvifStreams) > 0 {
		o.logger.Debug("ONVIF streams details:")
		for i, stream := range onvifStreams {
			o.logger.Debug("  ONVIF stream found",
				"index", i,
				"url", stream.URL,
				"protocol", stream.Protocol,
				"port", stream.Port,
				"type", stream.Type)
		}
	}
	allStreams = append(allStreams, onvifStreams...)

	// Add common RTSP streams
	o.logger.Debug(">>> Adding common RTSP streams", "ip", ip)
	commonStreams := o.getCommonRTSPStreams(ip, username, password)
	o.logger.Debug("<<< Common RTSP streams added", "count", len(commonStreams))
	allStreams = append(allStreams, commonStreams...)

	o.logger.Debug("=== ONVIF DiscoverStreamsForIP COMPLETED ===",
		"onvif_streams", len(onvifStreams),
		"common_streams", len(commonStreams),
		"total_streams", len(allStreams))

	return allStreams, nil
}

// discoverViaONVIF performs real ONVIF discovery
func (o *ONVIFDiscovery) discoverViaONVIF(ctx context.Context, ip, username, password string) []models.DiscoveredStream {
	o.logger.Debug(">>> discoverViaONVIF STARTED", "ip", ip)
	var streams []models.DiscoveredStream

	// Try standard ONVIF ports
	ports := []int{80, 8080, 8000}
	o.logger.Debug("Will try ONVIF ports", "ports", ports)

	for portIdx, port := range ports {
		o.logger.Debug("--- Trying ONVIF port ---",
			"port_index", portIdx+1,
			"total_ports", len(ports),
			"port", port)

		// Create timeout context for ONVIF connection
		onvifCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		xaddr := fmt.Sprintf("%s:%d", ip, port)
		o.logger.Debug("Creating ONVIF device",
			"xaddr", xaddr,
			"username", username,
			"has_password", password != "")

		// Create ONVIF device
		startTime := time.Now()
		dev, err := onvif.NewDevice(onvif.DeviceParams{
			Xaddr:    xaddr,
			Username: username,
			Password: password,
		})
		elapsed := time.Since(startTime)

		if err != nil {
			o.logger.Debug("❌ ONVIF device creation FAILED",
				"xaddr", xaddr,
				"error", err.Error(),
				"elapsed", elapsed.String())
			continue
		}

		o.logger.Debug("✅ ONVIF device created successfully",
			"xaddr", xaddr,
			"elapsed", elapsed.String())

		// Try to get profiles with context
		o.logger.Debug("Getting media profiles...", "xaddr", xaddr)
		profileStreams := o.getProfileStreams(onvifCtx, dev, ip)

		if len(profileStreams) > 0 {
			// Add ONVIF device service endpoint
			deviceServiceURL := fmt.Sprintf("http://%s/onvif/device_service", xaddr)
			streams = append(streams, models.DiscoveredStream{
				URL:      deviceServiceURL,
				Type:     "ONVIF",
				Protocol: "http",
				Port:     port,
				Working:  true, // Mark as working since ONVIF connection succeeded
				Metadata: map[string]interface{}{
					"source":      "onvif",
					"description": "ONVIF Device Service - used for PTZ control and device management",
				},
			})

			// Add profile streams
			streams = append(streams, profileStreams...)

			o.logger.Debug("🎉 ONVIF discovery SUCCESSFUL!",
				"xaddr", xaddr,
				"device_service", deviceServiceURL,
				"profiles_found", len(profileStreams))

			// Log device service
			o.logger.Debug("  Device Service",
				"url", deviceServiceURL)

			// Log each profile
			for i, stream := range profileStreams {
				o.logger.Debug("  Profile stream",
					"index", i+1,
					"url", stream.URL,
					"metadata", stream.Metadata)
			}
			break // Found working port, stop trying
		} else {
			o.logger.Debug("⚠️  No profiles returned from port", "xaddr", xaddr)
		}
	}

	o.logger.Debug("<<< discoverViaONVIF COMPLETED",
		"total_streams_found", len(streams))

	return streams
}

// getProfileStreams gets stream URIs from media profiles
func (o *ONVIFDiscovery) getProfileStreams(ctx context.Context, dev *onvif.Device, ip string) []models.DiscoveredStream {
	o.logger.Debug(">>> getProfileStreams STARTED", "ip", ip)
	var streams []models.DiscoveredStream

	// Get media profiles
	o.logger.Debug("Calling GetProfiles ONVIF method...")
	getProfilesReq := media.GetProfiles{}
	startTime := time.Now()
	profilesResp, err := dev.CallMethod(getProfilesReq)
	elapsed := time.Since(startTime)

	if err != nil {
		o.logger.Debug("❌ Failed to call GetProfiles",
			"error", err.Error(),
			"elapsed", elapsed.String())
		return streams
	}
	defer profilesResp.Body.Close()

	o.logger.Debug("✅ GetProfiles call successful",
		"elapsed", elapsed.String(),
		"status_code", profilesResp.StatusCode)

	// Read and parse XML response
	o.logger.Debug("Reading response body...")
	body, err := io.ReadAll(profilesResp.Body)
	if err != nil {
		o.logger.Debug("❌ Failed to read profiles response",
			"error", err.Error())
		return streams
	}

	o.logger.Debug("Response body read",
		"body_length", len(body),
		"body_preview", string(body[:min(200, len(body))]))

	// Parse SOAP envelope
	o.logger.Debug("Parsing SOAP envelope...")
	var envelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			GetProfilesResponse media.GetProfilesResponse `xml:"GetProfilesResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(body, &envelope); err != nil {
		o.logger.Debug("❌ Failed to parse profiles response",
			"error", err.Error())
		return streams
	}

	profileCount := len(envelope.Body.GetProfilesResponse.Profiles)
	o.logger.Debug("✅ SOAP envelope parsed successfully",
		"profiles_count", profileCount)

	// Get stream URI for each profile
	for i, profile := range envelope.Body.GetProfilesResponse.Profiles {
		o.logger.Debug("Processing profile",
			"index", i+1,
			"total", profileCount,
			"token", string(profile.Token),
			"name", string(profile.Name))

		streamURI := o.getStreamURI(dev, string(profile.Token))
		if streamURI != "" {
			o.logger.Debug("✅ Got stream URI for profile",
				"profile_token", string(profile.Token),
				"stream_uri", streamURI)

			streams = append(streams, models.DiscoveredStream{
				URL:      streamURI,
				Type:     "FFMPEG",
				Protocol: "rtsp",
				Port:     554,
				Working:  false, // Will be tested later
				Metadata: map[string]interface{}{
					"source":        "onvif",
					"profile_token": string(profile.Token),
					"profile_name":  string(profile.Name),
				},
			})
		} else {
			o.logger.Debug("⚠️  Failed to get stream URI for profile",
				"profile_token", string(profile.Token))
		}
	}

	o.logger.Debug("<<< getProfileStreams COMPLETED",
		"streams_collected", len(streams))

	return streams
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getStreamURI retrieves stream URI for a profile
func (o *ONVIFDiscovery) getStreamURI(dev *onvif.Device, profileToken string) string {
	o.logger.Debug(">>> getStreamURI STARTED", "profile_token", profileToken)

	stream := xsdonvif.StreamType("RTP-Unicast")
	protocol := xsdonvif.TransportProtocol("RTSP")
	token := xsdonvif.ReferenceToken(profileToken)

	getStreamURIReq := media.GetStreamUri{
		ProfileToken: &token,
		StreamSetup: &xsdonvif.StreamSetup{
			Stream: &stream,
			Transport: &xsdonvif.Transport{
				Protocol: &protocol,
			},
		},
	}

	o.logger.Debug("Calling GetStreamUri ONVIF method...", "profile_token", profileToken)
	startTime := time.Now()
	resp, err := dev.CallMethod(getStreamURIReq)
	elapsed := time.Since(startTime)

	if err != nil {
		o.logger.Debug("❌ Failed to get stream URI",
			"profile", profileToken,
			"error", err.Error(),
			"elapsed", elapsed.String())
		return ""
	}
	defer resp.Body.Close()

	o.logger.Debug("✅ GetStreamUri call successful",
		"profile", profileToken,
		"elapsed", elapsed.String(),
		"status_code", resp.StatusCode)

	// Read and parse XML response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		o.logger.Debug("❌ Failed to read stream URI response",
			"error", err.Error())
		return ""
	}

	o.logger.Debug("Response body read",
		"body_length", len(body),
		"body_preview", string(body[:min(200, len(body))]))

	// Parse SOAP envelope
	var envelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			GetStreamUriResponse media.GetStreamUriResponse `xml:"GetStreamUriResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(body, &envelope); err != nil {
		o.logger.Debug("❌ Failed to parse stream URI response",
			"error", err.Error())
		return ""
	}

	streamURI := string(envelope.Body.GetStreamUriResponse.MediaUri.Uri)
	o.logger.Debug("<<< getStreamURI COMPLETED",
		"stream_uri", streamURI)

	return streamURI
}

// getCommonRTSPStreams returns common RTSP stream URLs
func (o *ONVIFDiscovery) getCommonRTSPStreams(ip, username, password string) []models.DiscoveredStream {
	// Common RTSP paths that work with many cameras
	commonPaths := []struct {
		path  string
		notes string
	}{
		{"/stream1", "Common main stream"},
		{"/stream2", "Common sub stream"},
		{"/ch0", "Thingino main"},
		{"/ch1", "Thingino sub"},
		{"/live/main", "ONVIF standard main"},
		{"/live/sub", "ONVIF standard sub"},
		{"/Streaming/Channels/101", "Hikvision main"},
		{"/Streaming/Channels/102", "Hikvision sub"},
		{"/cam/realmonitor?channel=1&subtype=0", "Dahua main"},
		{"/cam/realmonitor?channel=1&subtype=1", "Dahua sub"},
		{"/h264/main", "Generic H264 main"},
		{"/h264/sub", "Generic H264 sub"},
		{"/media/video1", "Axis main"},
		{"/media/video2", "Axis sub"},
		{"/videoMain", "Foscam main"},
		{"/videoSub", "Foscam sub"},
		{"/11", "Simple numeric main"},
		{"/12", "Simple numeric sub"},
		{"/user=admin_password=tlJwpbo6_channel=1_stream=0.sdp", "Dahua alternative"},
		{"/live.sdp", "Generic live"},
		{"/stream", "Generic stream"},
		{"/video.h264", "Generic H264"},
		{"/live/0/MAIN", "Alternative main"},
		{"/live/0/SUB", "Alternative sub"},
		{"/MediaInput/h264", "Alternative H264"},
		{"/0/video0", "Alternative video0"},
		{"/0/video1", "Alternative video1"},
	}

	var streams []models.DiscoveredStream

	for _, cp := range commonPaths {
		var streamURL string
		if username != "" && password != "" {
			streamURL = fmt.Sprintf("rtsp://%s:%s@%s:554%s", url.QueryEscape(username), url.QueryEscape(password), ip, cp.path)
		} else {
			streamURL = fmt.Sprintf("rtsp://%s:554%s", ip, cp.path)
		}

		streams = append(streams, models.DiscoveredStream{
			URL:      streamURL,
			Type:     "FFMPEG",
			Protocol: "rtsp",
			Port:     554,
			Working:  false, // Will be tested later
			Metadata: map[string]interface{}{
				"source": "common",
				"notes":  cp.notes,
			},
		})
	}

	// Add some HTTP snapshot URLs too
	httpPaths := []struct {
		path  string
		notes string
	}{
		{"/snapshot.jpg", "Common snapshot"},
		{"/snap.jpg", "Alternative snapshot"},
		{"/image/jpeg.cgi", "CGI snapshot"},
		{"/cgi-bin/snapshot.cgi", "CGI bin snapshot"},
		{"/jpg/image.jpg", "JPEG image"},
		{"/tmpfs/auto.jpg", "Tmpfs snapshot"},
		{"/axis-cgi/jpg/image.cgi", "Axis snapshot"},
		{"/cgi-bin/viewer/video.jpg", "Viewer snapshot"},
		{"/Streaming/channels/1/picture", "Hikvision snapshot"},
		{"/onvif/snapshot", "ONVIF snapshot"},
	}

	for _, hp := range httpPaths {
		var streamURL string
		if username != "" && password != "" {
			// For HTTP, we'll rely on Basic Auth instead of URL embedding
			streamURL = fmt.Sprintf("http://%s%s", ip, hp.path)
		} else {
			streamURL = fmt.Sprintf("http://%s%s", ip, hp.path)
		}

		streams = append(streams, models.DiscoveredStream{
			URL:      streamURL,
			Type:     "JPEG",
			Protocol: "http",
			Port:     80,
			Working:  false, // Will be tested later
			Metadata: map[string]interface{}{
				"source":   "common",
				"notes":    hp.notes,
				"username": username,
				"password": password,
			},
		})
	}

	return streams
}