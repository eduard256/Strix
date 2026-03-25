package camdb

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

var defaultPorts = map[string]int{
	"rtsp": 554, "rtsps": 322, "http": 80, "https": 443,
	"rtmp": 1935, "mms": 554, "rtp": 5004,
}

type StreamParams struct {
	IDs     string
	IP      string
	User    string
	Pass    string
	Channel int
	Ports   map[int]bool // nil = no filter
}

type raw struct {
	url, protocol string
	port          int
}

// BuildStreams resolves IDs to full stream URLs with credentials and placeholders substituted
func BuildStreams(db *sql.DB, p *StreamParams) ([]string, error) {
	var raws []raw

	for _, id := range strings.Split(p.IDs, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		var rows *sql.Rows
		var err error

		switch {
		case strings.HasPrefix(id, "b:"):
			brandID := id[2:]
			rows, err = db.Query(
				"SELECT url, protocol, port FROM streams WHERE brand_id = ?", brandID,
			)

		case strings.HasPrefix(id, "m:"):
			parts := strings.SplitN(id[2:], ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("camdb: invalid model id: %s", id)
			}
			rows, err = db.Query(
				`SELECT s.url, s.protocol, s.port
				FROM stream_models sm
				JOIN streams s ON s.id = sm.stream_id
				WHERE s.brand_id = ? AND sm.model = ?`,
				parts[0], parts[1],
			)

		case strings.HasPrefix(id, "p:"):
			presetID := id[2:]
			rows, err = db.Query(
				"SELECT url, protocol, port FROM preset_streams WHERE preset_id = ?", presetID,
			)

		default:
			return nil, fmt.Errorf("camdb: unknown id prefix: %s", id)
		}

		if err != nil {
			return nil, err
		}

		found := false
		for rows.Next() {
			var r raw
			if err = rows.Scan(&r.url, &r.protocol, &r.port); err != nil {
				rows.Close()
				return nil, err
			}
			raws = append(raws, r)
			found = true
		}
		rows.Close()

		if !found {
			return nil, fmt.Errorf("camdb: not found: %s", id)
		}
	}

	// build full URLs, deduplicate
	seen := map[string]bool{}
	var streams []string

	for _, r := range raws {
		if len(streams) >= 20000 {
			break
		}

		port := r.port
		if port == 0 {
			if p, ok := defaultPorts[r.protocol]; ok {
				port = p
			} else {
				port = 80
			}
		}

		if p.Ports != nil && !p.Ports[port] {
			continue
		}

		u := buildURL(r.protocol, r.url, p.IP, port, p.User, p.Pass, p.Channel)
		if seen[u] {
			continue
		}
		seen[u] = true
		streams = append(streams, u)
	}

	return streams, nil
}


// internals

func buildURL(protocol, path, ip string, port int, user, pass string, channel int) string {
	path = replacePlaceholders(path, ip, port, user, pass, channel)

	var auth string
	if user != "" {
		auth = url.PathEscape(user) + ":" + url.PathEscape(pass) + "@"
	}

	host := ip
	if p, ok := defaultPorts[protocol]; !ok || p != port {
		host = ip + ":" + strconv.Itoa(port)
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return protocol + "://" + auth + host + path
}

func replacePlaceholders(s, ip string, port int, user, pass string, channel int) string {
	auth := ""
	if user != "" && pass != "" {
		auth = base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	}

	// URL-encode credentials for safe use in query parameters
	encUser := url.QueryEscape(user)
	encPass := url.QueryEscape(pass)

	pairs := []string{
		"[CHANNEL]", strconv.Itoa(channel),
		"[channel]", strconv.Itoa(channel),
		"{CHANNEL}", strconv.Itoa(channel),
		"{channel}", strconv.Itoa(channel),
		"[CHANNEL+1]", strconv.Itoa(channel + 1),
		"[channel+1]", strconv.Itoa(channel + 1),
		"{CHANNEL+1}", strconv.Itoa(channel + 1),
		"{channel+1}", strconv.Itoa(channel + 1),
		"[USERNAME]", encUser, "[username]", encUser,
		"[USER]", encUser, "[user]", encUser,
		"[PASSWORD]", encPass, "[password]", encPass,
		"[PASWORD]", encPass, "[pasword]", encPass,
		"[PASS]", encPass, "[pass]", encPass,
		"[PWD]", encPass, "[pwd]", encPass,
		"[WIDTH]", "640", "[width]", "640",
		"[HEIGHT]", "480", "[height]", "480",
		"[IP]", ip, "[ip]", ip,
		"[PORT]", strconv.Itoa(port), "[port]", strconv.Itoa(port),
		"[AUTH]", auth, "[auth]", auth,
		"[TOKEN]", "", "[token]", "",
	}

	r := strings.NewReplacer(pairs...)
	return r.Replace(s)
}
