package stream

import (
	"net/url"
	"strings"
	"testing"

	"github.com/eduard256/Strix/internal/models"
)

// Passwords with various special characters that real users might use.
// Each one exercises a different URL-parsing edge case.
var specialPasswords = []struct {
	name     string
	password string
	breaking string // which URL component this character breaks without escaping
}{
	{"at sign", "p@ssword", "userinfo delimiter — splits user:pass from host"},
	{"colon", "p:ssword", "userinfo separator — splits username from password"},
	{"hash", "p#ssword", "fragment delimiter — truncates everything after it"},
	{"ampersand", "p&ssword", "query param separator — splits password into two params"},
	{"equals", "p=ssword", "query value delimiter — corrupts key=value parsing"},
	{"question mark", "p?ssword", "query start — creates phantom query string"},
	{"slash", "p/ssword", "path separator — changes URL path structure"},
	{"percent", "p%ssword", "escape prefix — creates invalid percent-encoding"},
	{"space", "p ssword", "whitespace — breaks URL parsing entirely"},
	{"plus", "p+ssword", "query space encoding — decoded as space in query strings"},
	{"dollar", "p$ssword", "shell/URI special character"},
	{"exclamation", "p!ssword", "sub-delimiter in RFC 3986"},
	{"mixed special", "p@ss:w#rd$1&2", "multiple special characters combined"},
	{"all dangerous", "P@:?#&=+$ !", "all URL-breaking characters at once"},
	{"url-like", "http://evil", "password that looks like a URL"},
	{"chinese", "密码test", "unicode characters in password"},
}

// ---------------------------------------------------------------------------
// RTSP URL tests
// ---------------------------------------------------------------------------

// TestRTSP_SpecialCharsInPassword_URLMustBeParseable verifies that RTSP URLs
// built with special-character passwords can be parsed back by url.Parse
// without losing or corrupting the host, scheme, or userinfo.
func TestRTSP_SpecialCharsInPassword_URLMustBeParseable(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	entry := models.CameraEntry{
		Type:     "FFMPEG",
		Protocol: "rtsp",
		Port:     554,
		URL:      "/live/main",
	}

	for _, sp := range specialPasswords {
		t.Run(sp.name, func(t *testing.T) {
			ctx := BuildContext{
				IP:       "192.168.1.100",
				Username: "admin",
				Password: sp.password,
				Port:     554,
			}

			urls := builder.BuildURLsFromEntry(entry, ctx)
			if len(urls) == 0 {
				t.Fatal("no URLs generated")
			}

			for i, rawURL := range urls {
				u, err := url.Parse(rawURL)
				if err != nil {
					t.Errorf("[%d] url.Parse failed: %v\n  raw URL: %s", i, err, rawURL)
					continue
				}

				// Scheme must be rtsp
				if u.Scheme != "rtsp" {
					t.Errorf("[%d] wrong scheme %q, want \"rtsp\"\n  raw URL: %s", i, u.Scheme, rawURL)
				}

				// Host must be the camera IP, not garbage from a mis-parsed password
				host := u.Hostname()
				if host != "192.168.1.100" {
					t.Errorf("[%d] wrong host %q, want \"192.168.1.100\"\n  raw URL: %s", i, host, rawURL)
				}

				// Password must round-trip correctly
				if u.User != nil {
					got, ok := u.User.Password()
					if !ok {
						t.Errorf("[%d] password not present in parsed URL\n  raw URL: %s", i, rawURL)
					} else if got != sp.password {
						t.Errorf("[%d] password mismatch: got %q, want %q\n  raw URL: %s", i, got, sp.password, rawURL)
					}
				}

				// Path must start with /live/main
				if !strings.HasPrefix(u.Path, "/live/main") {
					t.Errorf("[%d] wrong path %q, want prefix \"/live/main\"\n  raw URL: %s", i, u.Path, rawURL)
				}

				// Fragment must be empty (# in password must not leak)
				if u.Fragment != "" {
					t.Errorf("[%d] unexpected fragment %q — '#' in password leaked\n  raw URL: %s", i, u.Fragment, rawURL)
				}
			}
		})
	}
}

// TestRTSP_SpecialCharsInPassword_CountUnchanged verifies that the number
// of generated URLs does not change based on password content.
// A simple password and a complex one should produce the same URL count.
func TestRTSP_SpecialCharsInPassword_CountUnchanged(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	entry := models.CameraEntry{
		Type:     "FFMPEG",
		Protocol: "rtsp",
		Port:     554,
		URL:      "/stream1",
	}

	// Baseline: simple password
	baseCtx := BuildContext{
		IP:       "192.168.1.100",
		Username: "admin",
		Password: "simple123",
		Port:     554,
	}
	baseURLs := builder.BuildURLsFromEntry(entry, baseCtx)
	baseCount := len(baseURLs)

	for _, sp := range specialPasswords {
		t.Run(sp.name, func(t *testing.T) {
			ctx := BuildContext{
				IP:       "192.168.1.100",
				Username: "admin",
				Password: sp.password,
				Port:     554,
			}

			urls := builder.BuildURLsFromEntry(entry, ctx)
			if len(urls) != baseCount {
				t.Errorf("URL count changed: simple password produces %d, %q produces %d",
					baseCount, sp.password, len(urls))
				t.Logf("  simple URLs: %v", baseURLs)
				t.Logf("  special URLs: %v", urls)
			}
		})
	}
}

// TestRTSP_NormalPassword_NoChange ensures that encoding does not alter URLs
// when the password contains only safe characters (letters, digits, - . _ ~).
func TestRTSP_NormalPassword_NoChange(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	entry := models.CameraEntry{
		Type:     "FFMPEG",
		Protocol: "rtsp",
		Port:     554,
		URL:      "/Streaming/Channels/101",
	}

	normalPasswords := []string{
		"admin",
		"Admin123",
		"test-password",
		"hello_world",
		"dots.in.password",
		"tilde~ok",
		"UPPERCASE",
		"1234567890",
	}

	for _, pass := range normalPasswords {
		t.Run(pass, func(t *testing.T) {
			ctx := BuildContext{
				IP:       "192.168.1.100",
				Username: "admin",
				Password: pass,
				Port:     554,
			}

			urls := builder.BuildURLsFromEntry(entry, ctx)
			if len(urls) == 0 {
				t.Fatal("no URLs generated")
			}

			for _, rawURL := range urls {
				// Normal passwords must NOT contain any percent-encoding
				// because all their characters are unreserved.
				if strings.Contains(rawURL, "%") {
					t.Errorf("normal password %q was percent-encoded in URL: %s", pass, rawURL)
				}

				// Must contain the literal password string
				if !strings.Contains(rawURL, pass) {
					t.Errorf("URL does not contain literal password %q: %s", pass, rawURL)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// HTTP query string tests
// ---------------------------------------------------------------------------

// TestHTTP_SpecialCharsInPassword_QueryPlaceholders tests URLs where
// the password goes into a query parameter via [PASSWORD] placeholder.
// These are patterns like "snapshot.cgi?user=[USERNAME]&pwd=[PASSWORD]".
func TestHTTP_SpecialCharsInPassword_QueryPlaceholders(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	entry := models.CameraEntry{
		Type:     "JPEG",
		Protocol: "http",
		Port:     80,
		URL:      "snapshot.cgi?user=[USERNAME]&pwd=[PASSWORD]",
	}

	for _, sp := range specialPasswords {
		t.Run(sp.name, func(t *testing.T) {
			ctx := BuildContext{
				IP:       "192.168.1.100",
				Username: "admin",
				Password: sp.password,
				Port:     80,
			}

			urls := builder.BuildURLsFromEntry(entry, ctx)
			if len(urls) == 0 {
				t.Fatal("no URLs generated")
			}

			for i, rawURL := range urls {
				u, err := url.Parse(rawURL)
				if err != nil {
					t.Errorf("[%d] url.Parse failed: %v\n  raw URL: %s", i, err, rawURL)
					continue
				}

				// Host must be correct
				if u.Hostname() != "192.168.1.100" {
					t.Errorf("[%d] wrong host %q\n  raw URL: %s", i, u.Hostname(), rawURL)
				}

				// Fragment must be empty
				if u.Fragment != "" {
					t.Errorf("[%d] fragment leak %q — '#' in password broke URL\n  raw URL: %s",
						i, u.Fragment, rawURL)
				}

				// If URL has query params, check pwd round-trips
				q := u.Query()
				if pwd := q.Get("pwd"); pwd != "" {
					if pwd != sp.password {
						t.Errorf("[%d] pwd param mismatch: got %q, want %q\n  raw URL: %s",
							i, pwd, sp.password, rawURL)
					}
				}

				// Ampersand in password must NOT create extra query params
				// e.g. password "p&ssword" must not produce key "ssword"
				if strings.Contains(sp.password, "&") {
					// Extract the part after & as potential rogue key
					parts := strings.SplitN(sp.password, "&", 2)
					rogueKey := strings.SplitN(parts[1], "&", 2)[0]
					rogueKey = strings.SplitN(rogueKey, "=", 2)[0]
					if rogueKey != "" && q.Has(rogueKey) {
						t.Errorf("[%d] ampersand in password created rogue query param %q\n  raw URL: %s",
							i, rogueKey, rawURL)
					}
				}
			}
		})
	}
}

// TestHTTP_SpecialCharsInPassword_PathPlaceholders tests patterns where
// credentials appear in the URL path, e.g.
// "/user=[USERNAME]_password=[PASSWORD]_channel=1_stream=0.sdp"
func TestHTTP_SpecialCharsInPassword_PathPlaceholders(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	entry := models.CameraEntry{
		Type:     "FFMPEG",
		Protocol: "rtsp",
		Port:     554,
		URL:      "/user=[USERNAME]_password=[PASSWORD]_channel=1_stream=0.sdp",
	}

	for _, sp := range specialPasswords {
		t.Run(sp.name, func(t *testing.T) {
			ctx := BuildContext{
				IP:       "192.168.1.100",
				Username: "admin",
				Password: sp.password,
				Port:     554,
			}

			urls := builder.BuildURLsFromEntry(entry, ctx)
			if len(urls) == 0 {
				t.Fatal("no URLs generated")
			}

			for i, rawURL := range urls {
				u, err := url.Parse(rawURL)
				if err != nil {
					t.Errorf("[%d] url.Parse failed: %v\n  raw URL: %s", i, err, rawURL)
					continue
				}

				// Host must be correct
				if u.Hostname() != "192.168.1.100" {
					t.Errorf("[%d] wrong host %q, want \"192.168.1.100\"\n  raw URL: %s",
						i, u.Hostname(), rawURL)
				}

				// Scheme must be rtsp
				if u.Scheme != "rtsp" {
					t.Errorf("[%d] wrong scheme %q, want \"rtsp\"\n  raw URL: %s",
						i, u.Scheme, rawURL)
				}

				// Fragment must be empty
				if u.Fragment != "" {
					t.Errorf("[%d] fragment leak %q\n  raw URL: %s", i, u.Fragment, rawURL)
				}
			}
		})
	}
}

// TestHTTP_SpecialCharsInPassword_UserInfo tests HTTP URLs where
// credentials are embedded in the userinfo part (user:pass@host).
func TestHTTP_SpecialCharsInPassword_UserInfo(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	entry := models.CameraEntry{
		Type:     "JPEG",
		Protocol: "http",
		Port:     80,
		URL:      "snapshot.jpg",
	}

	for _, sp := range specialPasswords {
		t.Run(sp.name, func(t *testing.T) {
			ctx := BuildContext{
				IP:       "192.168.1.100",
				Username: "admin",
				Password: sp.password,
				Port:     80,
			}

			urls := builder.BuildURLsFromEntry(entry, ctx)
			if len(urls) == 0 {
				t.Fatal("no URLs generated")
			}

			for i, rawURL := range urls {
				u, err := url.Parse(rawURL)
				if err != nil {
					t.Errorf("[%d] url.Parse failed: %v\n  raw URL: %s", i, err, rawURL)
					continue
				}

				// Host must be correct
				if u.Hostname() != "192.168.1.100" {
					t.Errorf("[%d] wrong host %q\n  raw URL: %s", i, u.Hostname(), rawURL)
				}

				// If userinfo present, password must round-trip
				if u.User != nil {
					if got, ok := u.User.Password(); ok {
						if got != sp.password {
							t.Errorf("[%d] userinfo password mismatch: got %q, want %q\n  raw URL: %s",
								i, got, sp.password, rawURL)
						}
					}
				}

				// Fragment must be empty
				if u.Fragment != "" {
					t.Errorf("[%d] fragment leak %q\n  raw URL: %s", i, u.Fragment, rawURL)
				}
			}
		})
	}
}

// TestHTTP_SpecialCharsInPassword_CountUnchanged ensures HTTP URL count
// stays the same regardless of password content.
func TestHTTP_SpecialCharsInPassword_CountUnchanged(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	entry := models.CameraEntry{
		Type:     "JPEG",
		Protocol: "http",
		Port:     80,
		URL:      "snapshot.jpg",
	}

	baseCtx := BuildContext{
		IP:       "192.168.1.100",
		Username: "admin",
		Password: "simple123",
		Port:     80,
	}
	baseURLs := builder.BuildURLsFromEntry(entry, baseCtx)
	baseCount := len(baseURLs)

	for _, sp := range specialPasswords {
		t.Run(sp.name, func(t *testing.T) {
			ctx := BuildContext{
				IP:       "192.168.1.100",
				Username: "admin",
				Password: sp.password,
				Port:     80,
			}

			urls := builder.BuildURLsFromEntry(entry, ctx)
			if len(urls) != baseCount {
				t.Errorf("URL count changed: simple=%d, special(%q)=%d",
					baseCount, sp.password, len(urls))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Username special char tests
// ---------------------------------------------------------------------------

// TestSpecialCharsInUsername verifies that usernames with special characters
// are also handled correctly (less common but possible).
func TestSpecialCharsInUsername(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	entry := models.CameraEntry{
		Type:     "FFMPEG",
		Protocol: "rtsp",
		Port:     554,
		URL:      "/stream1",
	}

	specialUsernames := []string{
		"user@domain",
		"user:name",
		"user#1",
		"admin&root",
	}

	for _, username := range specialUsernames {
		t.Run(username, func(t *testing.T) {
			ctx := BuildContext{
				IP:       "192.168.1.100",
				Username: username,
				Password: "password123",
				Port:     554,
			}

			urls := builder.BuildURLsFromEntry(entry, ctx)
			if len(urls) == 0 {
				t.Fatal("no URLs generated")
			}

			for i, rawURL := range urls {
				u, err := url.Parse(rawURL)
				if err != nil {
					t.Errorf("[%d] url.Parse failed: %v\n  raw URL: %s", i, err, rawURL)
					continue
				}

				if u.Hostname() != "192.168.1.100" {
					t.Errorf("[%d] wrong host %q\n  raw URL: %s", i, u.Hostname(), rawURL)
				}

				if u.User != nil {
					if got := u.User.Username(); got != username {
						t.Errorf("[%d] username mismatch: got %q, want %q\n  raw URL: %s",
							i, got, username, rawURL)
					}
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Regression: normal passwords must not be affected
// ---------------------------------------------------------------------------

// TestHTTP_NormalPassword_NoPercentEncoding ensures that simple passwords
// do not get percent-encoded in the userinfo part, so we don't break
// cameras that might do byte-level comparison.
func TestHTTP_NormalPassword_NoPercentEncoding(t *testing.T) {
	logger := &mockLogger{}
	builder := NewBuilder([]string{}, logger)

	entry := models.CameraEntry{
		Type:     "JPEG",
		Protocol: "http",
		Port:     80,
		URL:      "snapshot.cgi?user=[USERNAME]&pwd=[PASSWORD]",
	}

	normalPasswords := []string{
		"admin123",
		"Password",
		"test-pass",
		"hello_world",
		"dots.dots",
		"tilde~ok",
	}

	for _, pass := range normalPasswords {
		t.Run(pass, func(t *testing.T) {
			ctx := BuildContext{
				IP:       "192.168.1.100",
				Username: "admin",
				Password: pass,
				Port:     80,
			}

			urls := builder.BuildURLsFromEntry(entry, ctx)

			for _, rawURL := range urls {
				u, err := url.Parse(rawURL)
				if err != nil {
					t.Errorf("url.Parse failed: %v\n  URL: %s", err, rawURL)
					continue
				}

				// Check query params: value must match exactly.
				// Skip URLs where pwd is empty (the no-auth variant).
				q := u.Query()
				if pwd := q.Get("pwd"); pwd != "" && pwd != pass {
					t.Errorf("pwd param %q != expected %q\n  URL: %s", pwd, pass, rawURL)
				}

				// Only check raw query encoding on URLs that actually have
				// the password in query params (skip no-auth and userinfo-only variants).
				if q.Get("pwd") != "" && !strings.Contains(u.RawQuery, pass) {
					t.Errorf("safe password %q was percent-encoded in query: %s\n  URL: %s",
						pass, u.RawQuery, rawURL)
				}
			}
		})
	}
}
