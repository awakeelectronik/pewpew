package http

import (
	"encoding/json"
	"log"
	"net/http"

	"pewpew/internal/infra/geoip"
	"pewpew/internal/infra/store"
)

// Server maneja HTTP
type Server struct {
	addr  string
	db    store.Store
	geoip geoip.Resolver
	mux   *http.ServeMux
}

// NewServer crea servidor HTTP
func NewServer(addr string, db store.Store, geoip geoip.Resolver) *Server {
	s := &Server{
		addr:  addr,
		db:    db,
		geoip: geoip,
		mux:   http.NewServeMux(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/api/events", s.getEventsHandler)
	s.mux.HandleFunc("/api/health", s.healthHandler)
	s.mux.HandleFunc("/", s.indexHandler)
}

// getEventsHandler retorna últimos eventos
func (s *Server) getEventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	events, err := s.db.GetRecentEvents(100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// healthHandler
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// indexHandler sirve UI (placeholder por ahora)
func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
  <title>PewPew</title>
</head>
<body>
  <h1>🔫 PewPew</h1>
  <p>Security Dashboard</p>
  <a href="/api/events">Live Events (JSON)</a>
</body>
</html>`))
}

// ListenAndServe inicia el servidor
func (s *Server) ListenAndServe() error {
	log.Printf("starting HTTP server on %s", s.addr)
	return http.ListenAndServe(s.addr, s.mux)
}

// Close cierra el servidor
func (s *Server) Close() error {
	return nil
}
