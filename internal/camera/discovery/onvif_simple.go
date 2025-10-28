package discovery

import (
	"context"
	"fmt"
	"net/url"
	"strings"

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
	// Clean IP (remove port if present)
	if idx := strings.IndexByte(ip, ':'); idx > 0 {
		ip = ip[:idx]
	}

	// Return common RTSP streams as we can't use complex ONVIF due to API changes
	streams := o.getCommonRTSPStreams(ip, username, password)

	o.logger.Debug("generated common RTSP streams", "count", len(streams))

	return streams, nil
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