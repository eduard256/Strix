package probe

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ProbeONVIF sends unicast WS-Discovery probe to ip:3702.
// Returns nil, nil if the device does not support ONVIF.
func ProbeONVIF(ctx context.Context, ip string) (*ONVIFResult, error) {
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(100 * time.Millisecond)
	}
	_ = conn.SetDeadline(deadline)

	// WS-Discovery Probe message
	// https://www.onvif.org/wp-content/uploads/2016/12/ONVIF_Feature_Discovery_Specification_16.07.pdf
	msg := `<?xml version="1.0" ?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
	<s:Header xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing">
		<a:Action>http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</a:Action>
		<a:MessageID>urn:uuid:` + randUUID() + `</a:MessageID>
		<a:To>urn:schemas-xmlsoap-org:ws:2005:04:discovery</a:To>
	</s:Header>
	<s:Body>
		<d:Probe xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery">
			<d:Types />
			<d:Scopes />
		</d:Probe>
	</s:Body>
</s:Envelope>`

	addr := &net.UDPAddr{IP: net.ParseIP(ip), Port: 3702}
	if _, err = conn.WriteTo([]byte(msg), addr); err != nil {
		return nil, err
	}

	buf := make([]byte, 8192)
	for {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			return nil, nil // timeout -- device doesn't support ONVIF
		}

		body := string(buf[:n])
		if !strings.Contains(body, "onvif") {
			continue
		}

		xaddrs := findXMLTag(body, "XAddrs")
		if xaddrs == "" {
			continue
		}

		// fix buggy cameras reporting 0.0.0.0
		// ex. <wsdd:XAddrs>http://0.0.0.0:8080/onvif/device_service</wsdd:XAddrs>
		if s, ok := strings.CutPrefix(xaddrs, "http://0.0.0.0"); ok {
			xaddrs = "http://" + ip + s
		}

		port := 80
		if u, err := url.Parse(xaddrs); err == nil && u.Port() != "" {
			fmt.Sscanf(u.Port(), "%d", &port)
		}

		scopes := findXMLTag(body, "Scopes")

		return &ONVIFResult{
			URL:      xaddrs,
			Port:     port,
			Name:     findScope(scopes, "onvif://www.onvif.org/name/"),
			Hardware: findScope(scopes, "onvif://www.onvif.org/hardware/"),
		}, nil
	}
}

// internals

var reXMLTag = map[string]*regexp.Regexp{}

func findXMLTag(s, tag string) string {
	re, ok := reXMLTag[tag]
	if !ok {
		re = regexp.MustCompile(`(?s)<(?:\w+:)?` + tag + `\b[^>]*>([^<]+)`)
		reXMLTag[tag] = re
	}
	m := re.FindStringSubmatch(s)
	if len(m) != 2 {
		return ""
	}
	return m[1]
}

func findScope(s, prefix string) string {
	i := strings.Index(s, prefix)
	if i < 0 {
		return ""
	}
	s = s[i+len(prefix):]
	if j := strings.IndexByte(s, ' '); j >= 0 {
		s = s[:j]
	}
	s, _ = url.QueryUnescape(s)
	return s
}

func randUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	s := hex.EncodeToString(b)
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}
