package tester

import (
	"testing"
)

func TestIsSafeURL(t *testing.T) {
	tests := []struct {
		url    string
		isSafe bool
	}{
		// 1. Legitimate Camera Streams (Must be ALLOWED)
		{"rtsp://192.168.1.100:554/live", true},
		{"http://10.0.0.50/mjpeg", true},
		{"dvrip://172.16.42.12:34567", true},
		{"rtmp://google.com/live", true}, // Public domains are fine

		// 2. Loopback (Must be BLOCKED)
		{"http://127.0.0.1", false},
		{"rtsp://localhost:554", false},
		{"http://[::1]", false},

		// 3. Link-Local / Cloud Metadata (Must be BLOCKED)
		{"http://169.254.169.254/latest/meta-data", false},
		{"rtsp://[fe80::1]", false},

		// 4. Multicast (Must be BLOCKED)
		{"rtsp://224.0.0.1", false},
		{"http://[ff02::1]", false},

		// 5. Invalid / Malformed URLs (Must fail closed and be BLOCKED)
		{"http://%", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := isSafeURL(tt.url); got != tt.isSafe {
				t.Errorf("isSafeURL(%q) = %v; want %v", tt.url, got, tt.isSafe)
			}
		})
	}
}
