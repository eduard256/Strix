package stream

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/eduard256/Strix/internal/models"
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
		"{channel}":  strconv.Itoa(ctx.Channel), // BUBBLE protocol uses {channel}
		"{CHANNEL}":  strconv.Itoa(ctx.Channel),
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
	urlMap := make(map[string]bool)
	var urls []string

	// Helper to add unique URLs
	addURL := func(url string) {
		if !urlMap[url] {
			urls = append(urls, url)
			urlMap[url] = true
		}
	}

	switch entry.Protocol {
	case "bubble":
		// BUBBLE protocol: proprietary Chinese NVR/DVR protocol
		// Always use HTTP with embedded credentials
		if ctx.Username != "" && ctx.Password != "" {
			// Build HTTP URL with credentials embedded
			ctxHTTP := ctx
			ctxHTTP.Protocol = "http"

			baseURL := b.BuildURL(entry, ctxHTTP)

			// Parse and add credentials to URL
			if u, err := url.Parse(baseURL); err == nil {
				u.User = url.UserPassword(ctx.Username, ctx.Password)
				addURL(u.String())
			}
		} else {
			// No credentials - try anyway (some cameras might work)
			ctxHTTP := ctx
			ctxHTTP.Protocol = "http"
			addURL(b.BuildURL(entry, ctxHTTP))
		}

	case "rtsp", "rtsps":
		// For RTSP: generate ONLY with credentials if provided, otherwise without
		if ctx.Username != "" && ctx.Password != "" {
			// Credentials provided - generate ONLY URL with auth
			addURL(b.BuildURL(entry, ctx))
		} else {
			// No credentials - generate ONLY URL without auth
			ctxNoAuth := ctx
			ctxNoAuth.Username = ""
			ctxNoAuth.Password = ""
			addURL(b.BuildURL(entry, ctxNoAuth))
		}

	case "http", "https":
		// For HTTP/HTTPS: ALWAYS generate 4 authentication variants
		if ctx.Username != "" && ctx.Password != "" {
			// 1. No authentication
			ctxNoAuth := ctx
			ctxNoAuth.Username = ""
			ctxNoAuth.Password = ""
			urlNoAuth := b.BuildURL(entry, ctxNoAuth)
			addURL(urlNoAuth)

			// 2. Basic Auth only (embedded credentials)
			urlBasic := b.BuildURL(entry, ctxNoAuth) // Use clean URL
			if u, err := url.Parse(urlBasic); err == nil {
				u.User = url.UserPassword(ctx.Username, ctx.Password)
				addURL(u.String())
			}

			// 3. Query parameters only
			urlWithParams := b.BuildURL(entry, ctx) // This will replace placeholders if any

			// If URL has auth placeholders, they're already replaced
			if strings.Contains(entry.URL, "[USERNAME]") || strings.Contains(entry.URL, "[PASSWORD]") {
				addURL(urlWithParams)
			} else {
				// No placeholders - add query params for auth (don't overwrite existing params)
				if u, err := url.Parse(urlWithParams); err == nil {
					q := u.Query()

					// Add user/pwd if not already present
					if !q.Has("user") && !q.Has("usr") && !q.Has("username") {
						q.Set("user", ctx.Username)
					}
					if !q.Has("pwd") && !q.Has("password") && !q.Has("pass") {
						q.Set("pwd", ctx.Password)
					}
					u.RawQuery = q.Encode()
					addURL(u.String())

					// Try alternative names too
					q2 := url.Values{}
					for k, v := range u.Query() {
						q2[k] = v
					}
					if !q2.Has("username") && !q2.Has("user") && !q2.Has("usr") {
						q2.Set("username", ctx.Username)
					}
					if !q2.Has("password") && !q2.Has("pwd") && !q2.Has("pass") {
						q2.Set("password", ctx.Password)
					}
					u.RawQuery = q2.Encode()
					addURL(u.String())
				}
			}

			// 4. Basic Auth + Query parameters (combined)
			if strings.Contains(entry.URL, "[USERNAME]") || strings.Contains(entry.URL, "[PASSWORD]") {
				// URL has placeholders - add Basic Auth to the URL with replaced params
				if u, err := url.Parse(urlWithParams); err == nil {
					u.User = url.UserPassword(ctx.Username, ctx.Password)
					addURL(u.String())
				}
			} else {
				// No placeholders - add both Basic Auth and query params (without overwriting existing)
				if u, err := url.Parse(urlNoAuth); err == nil {
					u.User = url.UserPassword(ctx.Username, ctx.Password)
					q := u.Query()

					// Add auth params only if not already present
					if !q.Has("user") && !q.Has("usr") && !q.Has("username") {
						q.Set("user", ctx.Username)
					}
					if !q.Has("pwd") && !q.Has("password") && !q.Has("pass") {
						q.Set("pwd", ctx.Password)
					}
					u.RawQuery = q.Encode()
					addURL(u.String())
				}
			}
		} else {
			// No credentials provided - just one URL
			addURL(b.BuildURL(entry, ctx))
		}

	default:
		// Other protocols - single URL
		addURL(b.BuildURL(entry, ctx))
	}


	b.logger.Debug("BuildURLsFromEntry complete",
		"entry_url_pattern", entry.URL,
		"entry_type", entry.Type,
		"entry_protocol", entry.Protocol,
		"total_urls_generated", len(urls),
		"urls", urls)

	return urls
}