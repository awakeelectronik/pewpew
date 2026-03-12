package http

import (
	"context"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pewpew/internal/infra/geoip"
	"pewpew/internal/infra/metrics"
	"pewpew/internal/infra/scan"
	"pewpew/internal/infra/store"
	"pewpew/internal/usecase/recommendations"
	"pewpew/static"
)

// Server maneja HTTP
type Server struct {
	addr      string
	db        store.Store
	geoip     geoip.Resolver
	collector *metrics.Collector
	srv       *http.Server
	mux       *http.ServeMux
	startedAt time.Time
}

// NewServer crea servidor HTTP
func NewServer(addr string, db store.Store, geoip geoip.Resolver) *Server {
	mux := http.NewServeMux()
	s := &Server{
		addr:      addr,
		db:        db,
		geoip:     geoip,
		collector: metrics.NewCollector(),
		srv:       &http.Server{Addr: addr, Handler: mux},
		mux:       mux,
		startedAt: time.Now(),
	}

	InitEventBroadcaster()
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/api/events", s.getEventsHandler)
	s.mux.HandleFunc("/api/health", s.healthHandler)
	s.mux.HandleFunc("/api/status", s.statusHandler)
	s.mux.HandleFunc("/api/metrics", s.metricsHandler)
	s.mux.HandleFunc("/api/attackers", s.attackersHandler)
	s.mux.HandleFunc("/api/bans", s.bansHandler)
	s.mux.HandleFunc("/api/bans/", s.banDetailsHandler)
	s.mux.HandleFunc("/api/ports", s.portsHandler)
	s.mux.HandleFunc("/api/vulns", s.vulnsHandler)
	s.mux.HandleFunc("/api/recommendations", s.recommendationsHandler)
	s.mux.HandleFunc("/ws/events", s.handleWebSocket)
	s.mux.HandleFunc("/", s.frontendHandler)
}

// healthHandler
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) attackersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	attackers, err := s.db.GetTopAttackers(50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(attackers)
}

func (s *Server) portsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ports, err := scan.DetectOpenPorts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(ports)
}

func (s *Server) vulnsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ports, err := scan.DetectOpenPorts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	findings := scan.RunVulnerabilityScan(ports)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(findings)
}

func (s *Server) recommendationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ports, _ := scan.DetectOpenPorts()
	recs := recommendations.Build(ports, detectFirewallBackend())
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(recs)
}

// metricsHandler returns a real-time VPS health snapshot (CPU, RAM, disk, net, load).
func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snap, err := s.collector.Collect()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(snap)
}

// frontendFS es el subárbol "dist" del FS embebido para servir en "/".
var frontendFS fs.FS

func init() {
	sub, err := fs.Sub(static.FS, "dist")
	if err != nil {
		log.Printf("[http] WARNING: frontend dist not found (run 'make build'): %v", err)
		return
	}
	frontendFS = sub
}

func (s *Server) frontendHandler(w http.ResponseWriter, r *http.Request) {
	if frontendFS == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("<!doctype html><html><body><h1>PewPew</h1><p>Frontend build not found. Run <code>make build</code>.</p></body></html>"))
		return
	}

	path := strings.TrimPrefix(filepath.Clean(r.URL.Path), "/")
	if path == "" || path == "." {
		path = "index.html"
	}

	f, err := frontendFS.Open(path)
	if err != nil {
		// SPA: rutas como /attackers → index.html
		if _, err2 := frontendFS.Open("index.html"); err2 == nil {
			http.ServeFileFS(w, r, frontendFS, "index.html")
			return
		}
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.ServeFileFS(w, r, frontendFS, "index.html")
		return
	}

	http.ServeFileFS(w, r, frontendFS, path)
}

func (s *Server) runtimeStatus() map[string]any {
	ports, _ := scan.DetectOpenPorts()
	findings := scan.RunVulnerabilityScan(ports)
	sshPath := detectSSHPathForStatus()
	sshSource := "file"
	if sshPath == "" {
		sshSource = "journald_or_unavailable"
	}
	return map[string]any{
		"hostname":         getHostname(),
		"ssh_log_path":     sshPath,
		"ssh_log_source":   sshSource,
		"firewall_backend": detectFirewallBackend(),
		"db_size":          s.db.GetDBSize(),
		"total_events":     s.db.GetEventCount(),
		"active_bans":      s.db.GetActiveBanCount(),
		"recent_findings":  len(findings),
		"open_ports":       len(ports),
		"version":          "1.0.0",
		"uptime_seconds":   int64(time.Since(s.startedAt).Seconds()),
		"bind_addr":        s.addr,
	}
}

// ListenAndServe inicia el servidor
func (s *Server) ListenAndServe() error {
	log.Printf("starting HTTP server on %s", s.addr)
	return s.srv.ListenAndServe()
}

// Close cierra el servidor de forma ordenada (Shutdown)
func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.srv.Shutdown(ctx)
}

func detectSSHPathForStatus() string {
	candidates := []string{"/var/log/auth.log", "/var/log/secure", "/var/log/authlog"}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
