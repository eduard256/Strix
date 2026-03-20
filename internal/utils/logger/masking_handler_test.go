package logger

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

func TestSecretStore_AddRemoveMask(t *testing.T) {
	store := NewSecretStore()

	// No secrets: text unchanged
	if got := store.Mask("password=secret123"); got != "password=secret123" {
		t.Errorf("expected unchanged text, got %q", got)
	}

	// Add a secret
	store.Add("secret123")
	if got := store.Mask("password=secret123"); got != "password=***" {
		t.Errorf("expected masked, got %q", got)
	}

	// Remove the secret
	store.Remove("secret123")
	if got := store.Mask("password=secret123"); got != "password=secret123" {
		t.Errorf("expected unmasked after remove, got %q", got)
	}
}

func TestSecretStore_EmptyString(t *testing.T) {
	store := NewSecretStore()
	store.Add("")
	if got := store.Mask("test"); got != "test" {
		t.Errorf("empty secret should be ignored, got %q", got)
	}
	store.Remove("") // should not panic
}

func TestSecretStore_MultipleSecrets(t *testing.T) {
	store := NewSecretStore()
	store.Add("pass1")
	store.Add("pass2")

	got := store.Mask("url=rtsp://user:pass1@host and also pwd=pass2&rate=0")
	if strings.Contains(got, "pass1") || strings.Contains(got, "pass2") {
		t.Errorf("both passwords should be masked, got %q", got)
	}
}

func TestSecretStore_ConcurrentAccess(t *testing.T) {
	store := NewSecretStore()
	var wg sync.WaitGroup

	// Simulate concurrent scans adding/removing/masking
	for i := 0; i < 100; i++ {
		wg.Add(3)
		secret := "secret" + string(rune('A'+i%26))

		go func(s string) {
			defer wg.Done()
			store.Add(s)
		}(secret)

		go func() {
			defer wg.Done()
			_ = store.Mask("some text with secretA in it")
		}()

		go func(s string) {
			defer wg.Done()
			store.Remove(s)
		}(secret)
	}

	wg.Wait()
}

func TestSecretMaskingHandler_MasksStringAttrs(t *testing.T) {
	var buf bytes.Buffer
	store := NewSecretStore()
	store.Add("mypassword")

	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := NewSecretMaskingHandler(inner, store)
	log := slog.New(handler)

	log.Debug("testing stream", "url", "rtsp://admin:mypassword@192.168.1.10/stream")

	output := buf.String()
	if strings.Contains(output, "mypassword") {
		t.Errorf("password should be masked in output: %s", output)
	}
	if !strings.Contains(output, "***") {
		t.Errorf("expected *** in output: %s", output)
	}
}

func TestSecretMaskingHandler_MasksMessage(t *testing.T) {
	var buf bytes.Buffer
	store := NewSecretStore()
	store.Add("secretpwd")

	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := NewSecretMaskingHandler(inner, store)
	log := slog.New(handler)

	log.Debug("failed with secretpwd in message")

	output := buf.String()
	if strings.Contains(output, "secretpwd") {
		t.Errorf("password should be masked in message: %s", output)
	}
}

func TestSecretMaskingHandler_MasksErrorValues(t *testing.T) {
	var buf bytes.Buffer
	store := NewSecretStore()
	store.Add("r6wnm0wlix")

	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := NewSecretMaskingHandler(inner, store)
	log := slog.New(handler)

	err := errors.New(`Get "http://10.0.20.111/cgi-bin/encoder?PWD=r6wnm0wlix&USER=admin": dial tcp`)
	log.Debug("request failed", "error", err)

	output := buf.String()
	if strings.Contains(output, "r6wnm0wlix") {
		t.Errorf("password should be masked in error: %s", output)
	}
}

func TestSecretMaskingHandler_NoSecretsPassthrough(t *testing.T) {
	var buf bytes.Buffer
	store := NewSecretStore()

	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := NewSecretMaskingHandler(inner, store)
	log := slog.New(handler)

	log.Debug("normal message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "normal message") || !strings.Contains(output, "value") {
		t.Errorf("output should pass through unchanged: %s", output)
	}
}

func TestSecretMaskingHandler_MasksMultipleOccurrences(t *testing.T) {
	var buf bytes.Buffer
	store := NewSecretStore()
	store.Add("secret123")

	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := NewSecretMaskingHandler(inner, store)
	log := slog.New(handler)

	log.Debug("test",
		"url1", "rtsp://user:secret123@host1/stream",
		"url2", "http://host2/snap?pwd=secret123",
		"path", "/user=admin_password=secret123_channel=1",
	)

	output := buf.String()
	if strings.Contains(output, "secret123") {
		t.Errorf("all occurrences should be masked: %s", output)
	}
}

func TestSecretMaskingHandler_Enabled(t *testing.T) {
	store := NewSecretStore()
	inner := slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler := NewSecretMaskingHandler(inner, store)

	if handler.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("debug should be disabled when level is info")
	}
	if !handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("info should be enabled")
	}
}

func TestSecretMaskingHandler_SpecialCharsPassword(t *testing.T) {
	var buf bytes.Buffer
	store := NewSecretStore()
	store.Add("p@ss:w0rd#1")

	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := NewSecretMaskingHandler(inner, store)
	log := slog.New(handler)

	// Simulate URLs built by builder.go and onvif_simple.go
	// 1. RTSP with url.QueryEscape (onvif_simple.go:395)
	log.Debug("testing RTSP stream", "url", "rtsp://admin:p%40ss%3Aw0rd%231@192.168.1.10:554/stream1")

	// 2. HTTP with url.UserPassword (builder.go:355) -- Go encodes special chars
	log.Debug("testing HTTP stream", "url", "http://admin:p%40ss%3Aw0rd%231@192.168.1.10/snap.jpg")

	// 3. Query params with url.Values.Encode (builder.go:377)
	log.Debug("testing HTTP stream", "url", "http://192.168.1.10/snap.jpg?pwd=p%40ss%3Aw0rd%231&user=admin")

	// 4. Error from Go http.Client (contains encoded URL)
	log.Debug("stream test failed",
		"url", "http://admin:p%40ss%3Aw0rd%231@192.168.1.10/camera",
		"error", `HTTP request failed: Get "http://admin:***@192.168.1.10/camera": connection refused`)

	output := buf.String()
	t.Logf("Output:\n%s", output)

	if strings.Contains(output, "p@ss:w0rd#1") {
		t.Errorf("plain text password should be masked: %s", output)
	}
	if strings.Contains(output, "p%40ss%3Aw0rd%231") {
		t.Errorf("URL-encoded password should be masked: %s", output)
	}
}

func TestSecretMaskingHandler_PlainPassword(t *testing.T) {
	var buf bytes.Buffer
	store := NewSecretStore()
	store.Add("simplepass123")

	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := NewSecretMaskingHandler(inner, store)
	log := slog.New(handler)

	// Plain password without special chars -- no encoding difference
	log.Debug("testing RTSP stream", "url", "rtsp://admin:simplepass123@192.168.1.10:554/stream")
	log.Debug("testing HTTP stream", "url", "http://192.168.1.10/snap.jpg?pwd=simplepass123&user=admin")
	log.Debug("stream test failed",
		"url", "http://admin:simplepass123@192.168.1.10/camera",
		"error", `HTTP request failed: Get "http://192.168.1.10/snap.jpg?pwd=simplepass123&user=admin": connection refused`)

	output := buf.String()
	t.Logf("Output:\n%s", output)

	if strings.Contains(output, "simplepass123") {
		t.Errorf("password should be masked everywhere: %s", output)
	}
}

func TestSecretMaskingHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	store := NewSecretStore()
	store.Add("secretval")

	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := NewSecretMaskingHandler(inner, store)
	child := handler.WithAttrs([]slog.Attr{slog.String("static", "has secretval inside")})
	log := slog.New(child)

	log.Debug("test")

	output := buf.String()
	if strings.Contains(output, "secretval") {
		t.Errorf("pre-set attr should be masked: %s", output)
	}
}
