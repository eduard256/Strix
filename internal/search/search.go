package search

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/eduard256/strix/internal/api"
	"github.com/eduard256/strix/internal/app"
	"github.com/eduard256/strix/pkg/camdb"
	"github.com/rs/zerolog"

	_ "github.com/mattn/go-sqlite3"
)

var log zerolog.Logger
var db *sql.DB

func Init() {
	log = app.GetLogger("search")

	var err error
	db, err = sql.Open("sqlite3", "file:"+app.DB+"?mode=ro&immutable=1")
	if err != nil {
		log.Fatal().Err(err).Msg("[search] db open")
	}

	// verify DB is readable
	var count int
	if err = db.QueryRow("SELECT COUNT(*) FROM brands").Scan(&count); err != nil {
		log.Fatal().Err(err).Msg("[search] db verify")
	}
	log.Info().Int("brands", count).Msg("[search] loaded")

	api.HandleFunc("api/search", apiSearch)
	api.HandleFunc("api/streams", apiStreams)
}

func apiSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	var results []camdb.Result
	var err error

	if q == "" {
		results, err = camdb.SearchAll(db)
	} else {
		results, err = camdb.SearchQuery(db, q)
	}

	if err != nil {
		api.Error(w, err, http.StatusInternalServerError)
		return
	}

	api.ResponseJSON(w, map[string]any{"results": results})
}

func apiStreams(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	ids := q.Get("ids")
	if ids == "" {
		http.Error(w, "ids required", http.StatusBadRequest)
		return
	}

	ip := q.Get("ip")
	if ip == "" {
		http.Error(w, "ip required", http.StatusBadRequest)
		return
	}

	channel, _ := strconv.Atoi(q.Get("channel"))

	var portFilter map[int]bool
	if ps := q.Get("ports"); ps != "" {
		portFilter = map[int]bool{}
		for _, p := range strings.Split(ps, ",") {
			if v, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
				portFilter[v] = true
			}
		}
	}

	streams, err := camdb.BuildStreams(db, &camdb.StreamParams{
		IDs:     ids,
		IP:      ip,
		User:    q.Get("user"),
		Pass:    q.Get("pass"),
		Channel: channel,
		Ports:   portFilter,
	})

	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		} else if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "unknown") {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	api.ResponseJSON(w, map[string]any{"streams": streams})
}
