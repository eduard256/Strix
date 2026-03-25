package frigate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/eduard256/strix/internal/api"
	"github.com/eduard256/strix/internal/app"
	"github.com/rs/zerolog"
)

var log zerolog.Logger
var frigateURL string

func Init() {
	log = app.GetLogger("frigate")

	frigateURL = detectFrigateURL()

	log.Info().Str("url", frigateURL).Msg("[frigate] target")

	api.HandleFunc("api/frigate/check", apiCheck)
}

func apiCheck(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{Timeout: 3 * time.Second}

	result := map[string]any{
		"url":          frigateURL,
		"detection":    "unknown",
		"ha_addons":    nil,
		"frigate_slug": "",
	}

	// show how URL was detected
	if env := os.Getenv("STRIX_FRIGATE_URL"); env != "" {
		result["detection"] = "env"
	} else if os.Getenv("SUPERVISOR_TOKEN") != "" {
		result["detection"] = "supervisor"
		// show what addons we found
		addons, frigateSlug := listHAAddons()
		result["ha_addons"] = addons
		result["frigate_slug"] = frigateSlug
	} else {
		result["detection"] = "fallback_localhost"
	}

	resp, err := client.Get(frigateURL + "/api/config")
	if err != nil {
		result["connected"] = false
		result["error"] = err.Error()
		api.ResponseJSON(w, result)
		return
	}
	resp.Body.Close()

	result["connected"] = true
	result["status_code"] = resp.StatusCode
	result["message"] = fmt.Sprintf("Frigate API responded with %d", resp.StatusCode)
	api.ResponseJSON(w, result)
}

// detectFrigateURL determines Frigate URL in priority order:
// 1. STRIX_FRIGATE_URL env var
// 2. HA Supervisor API autodiscovery
// 3. fallback to localhost:5000
func detectFrigateURL() string {
	// 1. explicit env
	if url := os.Getenv("STRIX_FRIGATE_URL"); url != "" {
		log.Info().Str("url", url).Msg("[frigate] using STRIX_FRIGATE_URL")
		return url
	}

	// 2. HA Supervisor autodiscovery
	if token := os.Getenv("SUPERVISOR_TOKEN"); token != "" {
		log.Info().Msg("[frigate] SUPERVISOR_TOKEN found, trying autodiscovery")

		slug := findFrigateAddon(token)
		if slug != "" {
			// HA addon hostname format: slug with _ replaced by - and prefixed
			url := fmt.Sprintf("http://%s:5000", slug)
			log.Info().Str("slug", slug).Str("url", url).Msg("[frigate] found via Supervisor")
			return url
		}

		log.Warn().Msg("[frigate] Supervisor available but Frigate addon not found")
	}

	// 3. fallback
	return "http://localhost:5000"
}

// findFrigateAddon queries HA Supervisor API for installed addons and finds Frigate
func findFrigateAddon(token string) string {
	client := &http.Client{Timeout: 3 * time.Second}

	req, err := http.NewRequest("GET", "http://supervisor/addons", nil)
	if err != nil {
		log.Debug().Err(err).Msg("[frigate] supervisor request failed")
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("[frigate] supervisor API unreachable")
		return ""
	}
	defer resp.Body.Close()

	var body struct {
		Data struct {
			Addons []struct {
				Slug    string `json:"slug"`
				Name    string `json:"name"`
				State   string `json:"state"`
				URL     string `json:"url"`
				Hostname string `json:"hostname"`
			} `json:"addons"`
		} `json:"data"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		log.Debug().Err(err).Msg("[frigate] supervisor response parse failed")
		return ""
	}

	for _, addon := range body.Data.Addons {
		// match by slug or name containing "frigate"
		if addon.Slug == "ccab4aaf_frigate" || addon.Slug == "frigate" {
			if addon.Hostname != "" {
				return addon.Hostname
			}
			return addon.Slug
		}
	}

	return ""
}

// listHAAddons returns addon list for debug purposes
func listHAAddons() ([]map[string]string, string) {
	token := os.Getenv("SUPERVISOR_TOKEN")
	if token == "" {
		return nil, ""
	}

	client := &http.Client{Timeout: 3 * time.Second}

	req, err := http.NewRequest("GET", "http://supervisor/addons", nil)
	if err != nil {
		return nil, ""
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return []map[string]string{{"error": err.Error()}}, ""
	}
	defer resp.Body.Close()

	var body struct {
		Data struct {
			Addons []struct {
				Slug     string `json:"slug"`
				Name     string `json:"name"`
				State    string `json:"state"`
				Hostname string `json:"hostname"`
			} `json:"addons"`
		} `json:"data"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return []map[string]string{{"error": err.Error()}}, ""
	}

	var addons []map[string]string
	var frigateSlug string
	for _, a := range body.Data.Addons {
		addons = append(addons, map[string]string{
			"slug":     a.Slug,
			"name":     a.Name,
			"state":    a.State,
			"hostname": a.Hostname,
		})
		if a.Slug == "ccab4aaf_frigate" || a.Slug == "frigate" {
			frigateSlug = a.Slug
		}
	}

	return addons, frigateSlug
}
