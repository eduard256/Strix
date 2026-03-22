package stream

import (
	"encoding/base64"
	"fmt"
	"net/url"
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

	// Replace placeholders in URL path (credentials are handled separately
	// to ensure proper encoding depending on their position in the URL).
	path := b.replacePlaceholders(entry.URL, ctx)
	b.logger.Debug("placeholders replaced", "original", entry.URL, "after_replacement", path)

	// Build the complete URL using url.URL struct for correct encoding
	var fullURL string

	// Check if the URL already contains authentication parameters
	hasAuthInURL := b.hasAuthenticationParams(path)
	b.logger.Debug("auth params detection", "has_auth_in_url", hasAuthInURL, "path", path)

	// Determine host string (omit default port for cleaner URLs)
	host := b.buildHost(ctx.IP, ctx.Port, ctx.Protocol)

	// Split path and query for url.URL (it expects them separately)
	pathPart, queryPart := b.splitPathQuery(path)

	// Ensure path starts with exactly one slash
	if !strings.HasPrefix(pathPart, "/") {
		pathPart = "/" + pathPart
	}

	u := &url.URL{
		Scheme:   ctx.Protocol,
		Host:     host,
		Path:     pathPart,
		RawQuery: queryPart,
	}

	switch ctx.Protocol {
	case "rtsp", "rtsps":
		if ctx.Username != "" && ctx.Password != "" && !hasAuthInURL {
			u.User = url.UserPassword(ctx.Username, ctx.Password)
		}

	case "http", "https":
		// For HTTP, credentials are NOT embedded in the URL by BuildURL.
		// BuildURLsFromEntry handles auth variants (userinfo, query params, etc.)
		// separately with url.UserPassword for proper encoding.

	default:
		// Generic: no credentials in URL
	}

	fullURL = u.String()

	b.logger.Debug("BuildURL complete",
		"final_url", fullURL,
		"entry_type", entry.Type,
		"entry_url_pattern", entry.URL,
		"protocol", ctx.Protocol,
		"port", ctx.Port,
		"has_auth_in_url", hasAuthInURL)

	return fullURL
}

// credentialPlaceholders lists all placeholder strings that represent
// username or password values. These must NOT be replaced via simple string
// substitution because they require context-aware encoding (different for
// query parameters, path segments, and userinfo).
var credentialPlaceholders = []string{
	"[USERNAME]", "[username]", "[USER]", "[user]",
	"[PASSWORD]", "[password]", "[PASWORD]", "[pasword]",
	"[PASS]", "[pass]", "[PWD]", "[pwd]",
}

// replacePlaceholders replaces all placeholders in the URL.
//
// Credential placeholders ([USERNAME], [PASSWORD], etc.) are handled in two
// phases to ensure correct encoding:
//  1. Non-credential placeholders (channel, resolution, IP, etc.) are replaced
//     first — these contain only safe characters.
//  2. Credential placeholders are then replaced with proper encoding:
//     - In query strings: via url.Values.Set + Encode (automatic encoding)
//     - In path segments: via url.PathEscape
func (b *Builder) replacePlaceholders(urlPath string, ctx BuildContext) string {
	result := urlPath

	// Generate base64 auth for [AUTH] placeholder (already safe — base64 has no
	// characters that need URL encoding)
	auth := ""
	if ctx.Username != "" && ctx.Password != "" {
		auth = base64.StdEncoding.EncodeToString([]byte(ctx.Username + ":" + ctx.Password))
	}

	// Phase 1: Replace non-credential placeholders (all values are safe strings)
	safeReplacements := map[string]string{
		"[CHANNEL]":   strconv.Itoa(ctx.Channel),
		"[channel]":   strconv.Itoa(ctx.Channel),
		"[CHANNEL+1]": strconv.Itoa(ctx.Channel + 1),
		"[channel+1]": strconv.Itoa(ctx.Channel + 1),
		"{CHANNEL}":   strconv.Itoa(ctx.Channel),
		"{channel}":   strconv.Itoa(ctx.Channel),
		"{CHANNEL+1}": strconv.Itoa(ctx.Channel + 1),
		"{channel+1}": strconv.Itoa(ctx.Channel + 1),
		"[WIDTH]":     strconv.Itoa(ctx.Width),
		"[width]":     strconv.Itoa(ctx.Width),
		"[HEIGHT]":    strconv.Itoa(ctx.Height),
		"[height]":    strconv.Itoa(ctx.Height),
		"[IP]":        ctx.IP,
		"[ip]":        ctx.IP,
		"[PORT]":      strconv.Itoa(ctx.Port),
		"[port]":      strconv.Itoa(ctx.Port),
		"[AUTH]":      auth,
		"[auth]":      auth,
		"[TOKEN]":     "",
		"[token]":     "",
	}

	for placeholder, value := range safeReplacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Phase 2: Replace credential placeholders with proper encoding.
	// First handle query parameters (via url.Values for safe encoding),
	// then handle any remaining credential placeholders in the path.
	result = b.replaceQueryCredentials(result, ctx)
	result = b.replacePathCredentials(result, ctx)

	return result
}

// replaceQueryCredentials handles credential replacement in query parameters.
// It parses the query string while credential placeholders are still intact
// (safe ASCII strings like "[PASSWORD]"), replaces them with real values via
// url.Values.Set, and re-encodes. This ensures special characters in passwords
// are always properly percent-encoded.
func (b *Builder) replaceQueryCredentials(urlPath string, ctx BuildContext) string {
	parts := strings.SplitN(urlPath, "?", 2)
	if len(parts) < 2 {
		return urlPath
	}

	basePath := parts[0]
	queryString := parts[1]

	// Parse the query string — placeholders like [PASSWORD] are safe to parse
	// because they contain no special URL characters.
	params, err := url.ParseQuery(queryString)
	if err != nil {
		return urlPath
	}

	// Username placeholder values that should be replaced
	usernamePlaceholders := map[string]bool{
		"[USERNAME]": true, "[username]": true,
		"[USER]": true, "[user]": true,
	}

	// Password placeholder values that should be replaced
	passwordPlaceholders := map[string]bool{
		"[PASSWORD]": true, "[password]": true,
		"[PASWORD]": true, "[pasword]": true,
		"[PASS]": true, "[pass]": true,
		"[PWD]": true, "[pwd]": true,
	}

	changed := false
	for key, values := range params {
		for _, val := range values {
			if usernamePlaceholders[val] {
				params.Set(key, ctx.Username)
				changed = true
			} else if passwordPlaceholders[val] {
				params.Set(key, ctx.Password)
				changed = true
			}
		}

		// Also handle auth-named keys whose values are still placeholders
		// or already contain the raw value from a previous step.
		// This covers patterns like "?user=admin&pwd=12345" that come from
		// replaceQueryParams in the old code.
		lowerKey := strings.ToLower(key)
		switch lowerKey {
		case "user", "username", "usr", "loginuse":
			if params.Get(key) == "" || isCredentialPlaceholder(params.Get(key)) {
				params.Set(key, ctx.Username)
				changed = true
			}
		case "password", "pass", "pwd", "loginpas", "passwd":
			if params.Get(key) == "" || isCredentialPlaceholder(params.Get(key)) {
				params.Set(key, ctx.Password)
				changed = true
			}
		}
	}

	if !changed {
		return urlPath
	}

	// params.Encode() automatically percent-encodes all values
	return basePath + "?" + params.Encode()
}

// replacePathCredentials replaces any remaining credential placeholders in the
// path portion of the URL using url.PathEscape for safe encoding.
func (b *Builder) replacePathCredentials(urlPath string, ctx BuildContext) string {
	// Map of credential placeholders to their escaped values for use in paths
	pathReplacements := map[string]string{
		"[USERNAME]": url.PathEscape(ctx.Username),
		"[username]": url.PathEscape(ctx.Username),
		"[USER]":     url.PathEscape(ctx.Username),
		"[user]":     url.PathEscape(ctx.Username),
		"[PASSWORD]": url.PathEscape(ctx.Password),
		"[password]": url.PathEscape(ctx.Password),
		"[PASWORD]":  url.PathEscape(ctx.Password),
		"[pasword]":  url.PathEscape(ctx.Password),
		"[PASS]":     url.PathEscape(ctx.Password),
		"[pass]":     url.PathEscape(ctx.Password),
		"[PWD]":      url.PathEscape(ctx.Password),
		"[pwd]":      url.PathEscape(ctx.Password),
	}

	for placeholder, value := range pathReplacements {
		urlPath = strings.ReplaceAll(urlPath, placeholder, value)
	}

	return urlPath
}

// isCredentialPlaceholder checks if a string is one of the known credential
// placeholder tokens.
func isCredentialPlaceholder(s string) bool {
	for _, p := range credentialPlaceholders {
		if s == p {
			return true
		}
	}
	return false
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

// buildHost returns the host:port string, omitting the port when it matches
// the default for the given protocol.
func (b *Builder) buildHost(ip string, port int, protocol string) string {
	isDefault := (protocol == "http" && port == 80) ||
		(protocol == "https" && port == 443) ||
		(protocol == "rtsp" && port == 554) ||
		(protocol == "rtsps" && port == 322)

	if isDefault || port == 0 {
		return ip
	}
	return fmt.Sprintf("%s:%d", ip, port)
}

// splitPathQuery splits a path string into path and raw query components.
// The input may contain "?" separating the path from the query string.
func (b *Builder) splitPathQuery(path string) (string, string) {
	if idx := strings.IndexByte(path, '?'); idx >= 0 {
		return path[:idx], path[idx+1:]
	}
	return path, ""
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