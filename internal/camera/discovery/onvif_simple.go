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
	// Clean IP (remove port if present)
	if idx := strings.IndexByte(ip, ':'); idx > 0 {
		ip = ip[:idx]
	}

	var allStreams []models.DiscoveredStream

	// Try real ONVIF discovery first
	onvifStreams := o.discoverViaONVIF(ctx, ip, username, password)
	allStreams = append(allStreams, onvifStreams...)

	// Add common RTSP streams
	commonStreams := o.getCommonRTSPStreams(ip, username, password)
	allStreams = append(allStreams, commonStreams...)

	o.logger.Debug("collected streams", "onvif", len(onvifStreams), "common", len(commonStreams), "total", len(allStreams))

	return allStreams, nil
}

// discoverViaONVIF performs real ONVIF discovery
func (o *ONVIFDiscovery) discoverViaONVIF(ctx context.Context, ip, username, password string) []models.DiscoveredStream {
	var streams []models.DiscoveredStream

	// Try standard ONVIF ports
	ports := []int{80, 8080, 8000}

	for _, port := range ports {
		// Create timeout context for ONVIF connection
		onvifCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		xaddr := fmt.Sprintf("%s:%d", ip, port)

		o.logger.Debug("trying ONVIF connection", "xaddr", xaddr)

		// Create ONVIF device
		dev, err := onvif.NewDevice(onvif.DeviceParams{
			Xaddr:    xaddr,
			Username: username,
			Password: password,
		})
		if err != nil {
			o.logger.Debug("ONVIF device creation failed", "xaddr", xaddr, "error", err.Error())
			continue
		}

		// Try to get profiles with context
		profileStreams := o.getProfileStreams(onvifCtx, dev, ip)
		if len(profileStreams) > 0 {
			streams = append(streams, profileStreams...)
			o.logger.Debug("ONVIF discovery successful", "xaddr", xaddr, "profiles", len(profileStreams))
			break // Found working port, stop trying
		}
	}

	return streams
}

// getProfileStreams gets stream URIs from media profiles
func (o *ONVIFDiscovery) getProfileStreams(ctx context.Context, dev *onvif.Device, ip string) []models.DiscoveredStream {
	var streams []models.DiscoveredStream

	// Get media profiles
	getProfilesReq := media.GetProfiles{}
	profilesResp, err := dev.CallMethod(getProfilesReq)
	if err != nil {
		o.logger.Debug("failed to get ONVIF profiles", "error", err.Error())
		return streams
	}
	defer profilesResp.Body.Close()

	// Read and parse XML response
	body, err := io.ReadAll(profilesResp.Body)
	if err != nil {
		o.logger.Debug("failed to read profiles response", "error", err.Error())
		return streams
	}

	// Parse SOAP envelope
	var envelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			GetProfilesResponse media.GetProfilesResponse `xml:"GetProfilesResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(body, &envelope); err != nil {
		o.logger.Debug("failed to parse profiles response", "error", err.Error())
		return streams
	}

	// Get stream URI for each profile
	for _, profile := range envelope.Body.GetProfilesResponse.Profiles {
		streamURI := o.getStreamURI(dev, string(profile.Token))
		if streamURI != "" {
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
		}
	}

	return streams
}

// getStreamURI retrieves stream URI for a profile
func (o *ONVIFDiscovery) getStreamURI(dev *onvif.Device, profileToken string) string {
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

	resp, err := dev.CallMethod(getStreamURIReq)
	if err != nil {
		o.logger.Debug("failed to get stream URI", "profile", profileToken, "error", err.Error())
		return ""
	}
	defer resp.Body.Close()

	// Read and parse XML response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		o.logger.Debug("failed to read stream URI response", "error", err.Error())
		return ""
	}

	// Parse SOAP envelope
	var envelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			GetStreamUriResponse media.GetStreamUriResponse `xml:"GetStreamUriResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(body, &envelope); err != nil {
		o.logger.Debug("failed to parse stream URI response", "error", err.Error())
		return ""
	}

	return string(envelope.Body.GetStreamUriResponse.MediaUri.Uri)
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