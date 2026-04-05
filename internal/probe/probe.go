package probe

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/eduard256/strix/internal/api"
	"github.com/eduard256/strix/internal/app"
	"github.com/eduard256/strix/pkg/probe"
	"github.com/rs/zerolog"

	_ "modernc.org/sqlite"
)

const probeTimeout = 120 * time.Millisecond

var log zerolog.Logger
var db *sql.DB
var ports []int
var detectors []func(*probe.Response) string

func Init() {
	log = app.GetLogger("probe")

	var err error
	db, err = sql.Open("sqlite", "file:"+app.DB+"?mode=ro&immutable=1")
	if err != nil {
		log.Error().Err(err).Msg("[probe] db open")
	}

	ports = loadPorts()
	// HomeKit detector
	detectors = append(detectors, func(r *probe.Response) string {
		if r.Probes.MDNS != nil && !r.Probes.MDNS.Paired {
			if r.Probes.MDNS.Category == "camera" || r.Probes.MDNS.Category == "doorbell" {
				return "homekit"
			}
		}
		return ""
	})

	api.HandleFunc("api/probe", apiProbe)
}

func apiProbe(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		http.Error(w, "missing ip parameter", http.StatusBadRequest)
		return
	}

	if net.ParseIP(ip) == nil {
		http.Error(w, "invalid ip: "+ip, http.StatusBadRequest)
		return
	}

	result := runProbe(r.Context(), ip)
	api.ResponseJSON(w, result)
}

func runProbe(parent context.Context, ip string) *probe.Response {
	ctx, cancel := context.WithTimeout(parent, probeTimeout)
	defer cancel()

	resp := &probe.Response{IP: ip}
	var mu sync.Mutex
	var wg sync.WaitGroup

	run := func(fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	fastCtx, fastCancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer fastCancel()

	run(func() {
		r, _ := probe.ScanPorts(fastCtx, ip, ports)
		mu.Lock()
		resp.Probes.Ports = r
		mu.Unlock()
	})
	run(func() {
		r, _ := probe.ReverseDNS(fastCtx, ip)
		mu.Lock()
		resp.Probes.DNS = r
		mu.Unlock()
	})
	run(func() {
		mac := probe.LookupARP(ip)
		if mac == "" {
			return
		}
		vendor := probe.LookupOUI(db, mac)
		mu.Lock()
		resp.Probes.ARP = &probe.ARPResult{MAC: mac, Vendor: vendor}
		mu.Unlock()
	})
	run(func() {
		r, _ := probe.QueryHAP(ctx, ip)
		mu.Lock()
		resp.Probes.MDNS = r
		mu.Unlock()
	})
	run(func() {
		r, _ := probe.ProbeHTTP(fastCtx, ip, nil)
		mu.Lock()
		resp.Probes.HTTP = r
		mu.Unlock()
	})

	wg.Wait()

	// determine reachable
	resp.Reachable = (resp.Probes.Ports != nil && len(resp.Probes.Ports.Open) > 0) ||
		resp.Probes.MDNS != nil

	// determine type
	resp.Type = "standard"
	if !resp.Reachable {
		resp.Type = "unreachable"
	} else {
		for _, detect := range detectors {
			if t := detect(resp); t != "" {
				resp.Type = t
				break
			}
		}
	}

	return resp
}

func loadPorts() []int {
	if db == nil {
		return defaultPorts()
	}

	rows, err := db.Query("SELECT DISTINCT port FROM streams WHERE port > 0 UNION SELECT DISTINCT port FROM preset_streams WHERE port > 0")
	if err != nil {
		log.Warn().Err(err).Msg("[probe] failed to load ports from db, using defaults")
		return defaultPorts()
	}
	defer rows.Close()

	var result []int
	for rows.Next() {
		var port int
		if err = rows.Scan(&port); err == nil {
			result = append(result, port)
		}
	}

	if len(result) == 0 {
		return defaultPorts()
	}

	result = append(result, 51826)
	log.Info().Int("count", len(result)).Msg("[probe] loaded ports from db")
	return result
}

func defaultPorts() []int {
	return []int{554, 80, 8080, 443, 8554, 5544, 10554, 1935, 81, 88, 8090, 8001, 8081, 7070, 7447, 34567, 51826}
}
