// Package server serves the page and its live data on the loopback address.
package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/johncbrubaker/connection-check/internal/monitor"
	"github.com/johncbrubaker/connection-check/internal/report"
	"github.com/johncbrubaker/connection-check/internal/sites"
)

// Config is what the server needs besides the monitor.
type Config struct {
	Page       fs.FS    // the embedded page
	ReportBase string   // URL of the hosted report page
	SitesPath  string   // where to save sites added from the page
	Hosts      []string // Host header values to accept, e.g. "127.0.0.1:52100"
	Started    time.Time
}

type server struct {
	m   *monitor.Monitor
	cfg Config
}

// New returns the handler for the page and its API.
func New(m *monitor.Monitor, cfg Config) http.Handler {
	s := &server{m: m, cfg: cfg}
	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(cfg.Page))
	mux.HandleFunc("GET /api/stream", s.stream)
	mux.HandleFunc("GET /api/report", s.report)
	mux.HandleFunc("POST /api/window", s.window)
	mux.HandleFunc("POST /api/sites", s.addSite)
	return guard(cfg.Hosts, mux)
}

// guard rejects requests for other host names, so a web page can't reach this server
// through a DNS trick, and POSTs that aren't JSON, so other pages can't send them without permission.
func guard(hosts []string, next http.Handler) http.Handler {
	ok := map[string]bool{}
	for _, h := range hosts {
		ok[h] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ok[r.Host] || (r.Method == http.MethodPost && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// stream sends every site's snapshot once a second as Server-Sent Events.
func (s *server) stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		payload, _ := json.Marshal(map[string]any{
			"now": time.Now().UnixMilli(), "started": s.cfg.Started.UnixMilli(), "sites": s.m.Snapshots(),
		})
		if _, err := fmt.Fprintf(w, "event: state\ndata: %s\n\n", payload); err != nil {
			return
		}
		flusher.Flush()
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
		}
	}
}

func (s *server) report(w http.ResponseWriter, r *http.Request) {
	snaps := s.m.Snapshots()
	i, err := strconv.Atoi(r.URL.Query().Get("site"))
	if err != nil || i < 0 || i >= len(snaps) {
		http.Error(w, "no such site", http.StatusNotFound)
		return
	}
	link, err := report.Link(s.cfg.ReportBase, snaps[i])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"link": link, "text": report.Text(snaps[i])})
}

func (s *server) window(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Site     int    `json:"site"`
		Window   string `json:"window"`
		From, To int64
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err := s.m.SetWindow(req.Site, req.Window, time.UnixMilli(req.From), time.UnixMilli(req.To))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *server) addSite(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name, Host string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	site, err := sites.Parse(req.Name + "=" + req.Host)
	if err == nil {
		err = s.m.Add(site)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := sites.Save(s.cfg.SitesPath, s.m.Sites()); err != nil {
		http.Error(w, "added, but not saved: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
