package stream

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/strix-project/strix/internal/models"
)

// Builder handles stream URL construction
type Builder struct {
	queryParams []string
	logger      interface{ Debug(string, ...any) }
}

// NewBuilder creates a new stream URL builder
func NewBuilder(queryParams []string, logger interface{ Debug(string, ...any) }) *Builder {
	return &Builder{
		queryParams: queryParams,
		logger:      logger,
	}
}

// BuildContext contains parameters for URL building
type BuildContext struct {
	IP       string
	Port     int
	Username string
	Password string
	Channel  int
	Width    int
	Height   int
	Protocol string
	Path     string
}

// BuildURL builds a complete URL from an entry and context
func (b *Builder) BuildURL(entry models.CameraEntry, ctx BuildContext) string {
	b.logger.Debug("BuildURL called",
		"entry_type", entry.Type,
		"entry_url", entry.URL,
		"entry_port", entry.Port,
		"entry_protocol", entry.Protocol,
		"ctx_ip", ctx.IP,
		"ctx_port", ctx.Port,
		"ctx_username", ctx.Username,
		"ctx_channel", ctx.Channel)

	// Set defaults
	if ctx.Width == 0 {
		ctx.Width = 640
	}
	if ctx.Height == 0 {
		ctx.Height = 480
	}
	// NOTE: Channel default is 0 - will only be used for [CHANNEL] placeholder replacement
	// Literal channel values in URLs (like "channel=1") are preserved as-is

	// Use entry's port if not specified
	if ctx.Port == 0 {
		ctx.Port = entry.Port

		// If entry port is also 0, use default port for the protocol
		if ctx.Port == 0 {
			// Use entry's protocol if not specified for port determination
			protocol := ctx.Protocol
			if protocol == "" {
				protocol = entry.Protocol
			}

			switch protocol {
			case "http":
				ctx.Port = 80
			case "https":
				ctx.Port = 443
			case "rtsp", "rtsps":
				ctx.Port = 554
			default:
				ctx.Port = 80 // Default to 80 if unknown
			}

			b.logger.Debug("using default port for protocol",
				"protocol", protocol,
				"default_port", ctx.Port)
		}
	}

	// Use entry's protocol if not specified
	if ctx.Protocol == "" {
		ctx.Protocol = entry.Protocol
	}

	// Replace placeholders in URL path
	path := b.replacePlaceholders(entry.URL, ctx)
	b.logger.Debug("placeholders replaced", "original", entry.URL, "after_replacement", path)

	// Build the complete URL
	var fullURL string

	// Check if the URL already contains authentication parameters
	hasAuthInURL := b.hasAuthenticationParams(path)
	b.logger.Debug("auth params detection", "has_auth_in_url", hasAuthInURL, "path", path)

	switch ctx.Protocol {
	case "rtsp":
		if ctx.Username != "" && ctx.Password != "" && !hasAuthInURL {
			// Standard ports can be omitted
			if ctx.Port == 554 {
				fullURL = fmt.Sprintf("rtsp://%s:%s@%s/%s",
					ctx.Username, ctx.Password, ctx.IP, path)
			} else {
				fullURL = fmt.Sprintf("rtsp://%s:%s@%s:%d/%s",
					ctx.Username, ctx.Password, ctx.IP, ctx.Port, path)
			}
		} else {
			if ctx.Port == 554 {
				fullURL = fmt.Sprintf("rtsp://%s/%s", ctx.IP, path)
			} else {
				fullURL = fmt.Sprintf("rtsp://%s:%d/%s", ctx.IP, ctx.Port, path)
			}
		}

	case "http", "https":
		// For HTTP, check if auth should be in URL or parameters
		if ctx.Username != "" && ctx.Password != "" && !hasAuthInURL {
			// Don't put auth in URL for HTTP, will use Basic Auth header
			if (ctx.Protocol == "http" && ctx.Port == 80) ||
			   (ctx.Protocol == "https" && ctx.Port == 443) {
				fullURL = fmt.Sprintf("%s://%s/%s", ctx.Protocol, ctx.IP, path)
			} else {
				fullURL = fmt.Sprintf("%s://%s:%d/%s", ctx.Protocol, ctx.IP, ctx.Port, path)
			}
		} else {
			if (ctx.Protocol == "http" && ctx.Port == 80) ||
			   (ctx.Protocol == "https" && ctx.Port == 443) {
				fullURL = fmt.Sprintf("%s://%s/%s", ctx.Protocol, ctx.IP, path)
			} else {
				fullURL = fmt.Sprintf("%s://%s:%d/%s", ctx.Protocol, ctx.IP, ctx.Port, path)
			}
		}

	default:
		// Generic URL construction
		fullURL = fmt.Sprintf("%s://%s:%d/%s", ctx.Protocol, ctx.IP, ctx.Port, path)
	}

	// Clean up double slashes (except after protocol://)
	fullURL = b.cleanURL(fullURL)

	b.logger.Debug("BuildURL complete",
		"final_url", fullURL,
		"entry_type", entry.Type,
		"entry_url_pattern", entry.URL,
		"protocol", ctx.Protocol,
		"port", ctx.Port,
		"has_auth_in_url", hasAuthInURL)

	return fullURL
}

// replacePlaceholders replaces all placeholders in the URL
func (b *Builder) replacePlaceholders(urlPath string, ctx BuildContext) string {
	result := urlPath

	// Generate base64 auth for [AUTH] placeholder
	auth := ""
	if ctx.Username != "" && ctx.Password != "" {
		auth = base64.StdEncoding.EncodeToString([]byte(ctx.Username + ":" + ctx.Password))
	}

	// Common placeholders
	replacements := map[string]string{
		"[CHANNEL]":  strconv.Itoa(ctx.Channel),
		"[channel]":  strconv.Itoa(ctx.Channel),
		"[WIDTH]":    strconv.Itoa(ctx.Width),
		"[width]":    strconv.Itoa(ctx.Width),
		"[HEIGHT]":   strconv.Itoa(ctx.Height),
		"[height]":   strconv.Itoa(ctx.Height),
		"[USERNAME]": ctx.Username,
		"[username]": ctx.Username,
		"[PASSWORD]": ctx.Password,
		"[password]": ctx.Password,
		"[PASWORD]":  ctx.Password, // Handle typo in database
		"[pasword]":  ctx.Password,
		"[USER]":     ctx.Username,
		"[user]":     ctx.Username,
		"[PASS]":     ctx.Password,
		"[pass]":     ctx.Password,
		"[PWD]":      ctx.Password,
		"[pwd]":      ctx.Password,
		"[IP]":       ctx.IP,
		"[ip]":       ctx.IP,
		"[PORT]":     strconv.Itoa(ctx.Port),
		"[port]":     strconv.Itoa(ctx.Port),
		"[AUTH]":     auth, // base64(username:password) for basic auth
		"[auth]":     auth,
		"[TOKEN]":    "", // Empty for now
		"[token]":    "",
	}

	// Replace all placeholders
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Handle query parameter placeholders (only for auth params)
	result = b.replaceQueryParams(result, ctx)

	return result
}


// replaceQueryParams handles query parameter replacements
func (b *Builder) replaceQueryParams(urlPath string, ctx BuildContext) string {
	// Parse URL to handle query params
	parts := strings.SplitN(urlPath, "?", 2)
	if len(parts) < 2 {
		return urlPath
	}

	basePath := parts[0]
	queryString := parts[1]

	// Parse query parameters
	params, err := url.ParseQuery(queryString)
	if err != nil {
		return urlPath
	}

	// ONLY replace authentication parameters
	// DO NOT replace channel, width, height - they should stay as-is from URL patterns
	for key := range params {
		lowerKey := strings.ToLower(key)

		switch lowerKey {
		case "user", "username", "usr", "loginuse":
			params.Set(key, ctx.Username)
		case "password", "pass", "pwd", "loginpas", "passwd":
			params.Set(key, ctx.Password)
		// Removed: channel, width, height replacements - they were breaking working URLs
		}
	}

	// Rebuild URL
	return basePath + "?" + params.Encode()
}


// hasAuthenticationParams checks if URL contains auth parameters
func (b *Builder) hasAuthenticationParams(urlPath string) bool {
	authParams := []string{
		"user=", "username=", "usr=", "loginuse=",
		"password=", "pass=", "pwd=", "loginpas=", "passwd=",
	}

	lowerPath := strings.ToLower(urlPath)
	for _, param := range authParams {
		if strings.Contains(lowerPath, param) {
			return true
		}
	}

	return false
}

// cleanURL cleans up the URL
func (b *Builder) cleanURL(fullURL string) string {
	// Remove double slashes except after protocol://
	protocolEnd := strings.Index(fullURL, "://")
	if protocolEnd > 0 {
		protocol := fullURL[:protocolEnd+3]
		rest := fullURL[protocolEnd+3:]

		// Replace multiple slashes with single slash
		rest = regexp.MustCompile(`/{2,}`).ReplaceAllString(rest, "/")

		return protocol + rest
	}

	return fullURL
}

// BuildURLsFromEntry generates all possible URLs from a camera entry
func (b *Builder) BuildURLsFromEntry(entry models.CameraEntry, ctx BuildContext) []string {
	var urls []string

	// Build main URL
	mainURL := b.BuildURL(entry, ctx)
	b.logger.Debug("BuildURLsFromEntry: main URL built", "url", mainURL, "entry_type", entry.Type)
	urls = append(urls, mainURL)

	// For NVR systems, try multiple channels
	if ctx.Channel == 0 && strings.Contains(strings.ToLower(entry.Notes), "channel") {
		for ch := 1; ch <= 4; ch++ {
			altCtx := ctx
			altCtx.Channel = ch
			altURL := b.BuildURL(entry, altCtx)
			if altURL != mainURL {
				urls = append(urls, altURL)
			}
		}
	}

	// Try different resolutions for snapshot URLs
	if entry.Type == "JPEG" || entry.Type == "MJPEG" {
		resolutions := [][2]int{
			{640, 480},
			{1280, 720},
			{1920, 1080},
		}

		for _, res := range resolutions {
			if res[0] != ctx.Width || res[1] != ctx.Height {
				altCtx := ctx
				altCtx.Width = res[0]
				altCtx.Height = res[1]
				altURL := b.BuildURL(entry, altCtx)
				if altURL != mainURL {
					urls = append(urls, altURL)
				}
			}
		}
	}

	b.logger.Debug("BuildURLsFromEntry complete",
		"entry_url_pattern", entry.URL,
		"entry_type", entry.Type,
		"total_urls_generated", len(urls),
		"urls", urls)

	return urls
}