package http

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"pewpew/internal/domain"
	"pewpew/internal/infra/firewall"
)

// getEventsHandler retorna eventos recientes
func (s *Server) getEventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 100
	if q := r.URL.Query().Get("limit"); q != "" {
		if l, err := strconv.Atoi(q); err == nil && l > 0 && l <= 10000 {
			limit = l
		}
	}

	events, err := s.db.GetRecentEvents(limit)
	if err != nil {
		log.Printf("[events] ERROR GetRecentEvents: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if events == nil {
		events = []*domain.SecurityEvent{}
	}

	withGeo := 0
	for _, e := range events {
		if e.Latitude != 0 || e.Longitude != 0 {
			withGeo++
		}
	}
	log.Printf("[events] GET /api/events limit=%d -> %d events (%d with lat/lon)", limit, len(events), withGeo)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if errEncode := json.NewEncoder(w).Encode(events); errEncode != nil {
		log.Printf("[events] ERROR encoding response: %v", errEncode)
	}
}

// statusHandler retorna estado del sistema
func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]interface{}{
		"hostname": getHostname(),
	}
	for k, v := range s.runtimeStatus() {
		status[k] = v
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(status)
}

// bansHandler gestiona operaciones de ban
func (s *Server) bansHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		log.Printf("[bans] GET /api/bans requested")
		// Listar bans: DB + IPs que UFW tiene en DENY (p. ej. ufw deny from X por CLI)
		bans, err := s.db.GetActiveBans()
		if err != nil {
			log.Printf("[bans] ERROR GetActiveBans: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("[bans] GetActiveBans ok, count=%d", len(bans))
		if bans == nil {
			bans = []*domain.BanAction{}
		}
		inDB := make(map[string]bool)
		for _, b := range bans {
			inDB[b.IP] = true
		}
		ufw := firewall.NewUFWBackend()
		avail := ufw.IsAvailable()
		log.Printf("[bans] ufw.IsAvailable()=%v", avail)
		if avail {
			ufwIPs, errUFW := ufw.BannedIPs()
			if errUFW != nil {
				log.Printf("[bans] ERROR UFW BannedIPs: %v", errUFW)
			} else {
				log.Printf("[bans] UFW BannedIPs ok, count=%d, ips=%v", len(ufwIPs), ufwIPs)
			}
			for _, ip := range ufwIPs {
				if !inDB[ip] {
					bans = append(bans, &domain.BanAction{
						IP:        ip,
						Timestamp: time.Now(),
						Backend:   "ufw",
						Reason:    "Blocked by UFW",
						Active:    true,
						CreatedBy:  "ufw",
					})
					inDB[ip] = true
				}
			}
		} else {
			log.Printf("[bans] ufw not available (path empty)")
		}
		log.Printf("[bans] returning total bans=%d", len(bans))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if errEncode := json.NewEncoder(w).Encode(bans); errEncode != nil {
			log.Printf("[bans] ERROR encoding response: %v", errEncode)
		}

	case http.MethodPost:
		// Crear ban
		var req struct {
			IP     string `json:"ip"`
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		// Validar IP
		if net.ParseIP(req.IP) == nil {
			http.Error(w, "invalid IP", http.StatusBadRequest)
			return
		}

		// Ban via firewall
		ufw := firewall.NewUFWBackend()
		if err := ufw.Ban(req.IP); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Guardar en DB
		ban := &domain.BanAction{
			IP:        req.IP,
			Timestamp: time.Now(),
			Backend:   "ufw",
			Reason:    req.Reason,
			Active:    true,
			CreatedBy: "dashboard",
		}
		if err := s.db.InsertBan(ban); err != nil {
			log.Printf("failed to save ban: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ban)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// banDetailsHandler maneja DELETE de ban específico. GET /api/bans/ (solo barra) → lista (por si el cliente pide con trailing slash).
func (s *Server) banDetailsHandler(w http.ResponseWriter, r *http.Request) {
	pathSuffix := strings.TrimPrefix(r.URL.Path, "/api/bans/")
	if r.Method == http.MethodGet && (pathSuffix == "" || pathSuffix == "/") {
		s.bansHandler(w, r)
		return
	}
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := pathSuffix
	if net.ParseIP(ip) == nil {
		http.Error(w, "invalid IP", http.StatusBadRequest)
		return
	}

	// Unban via firewall
	ufw := firewall.NewUFWBackend()
	if err := ufw.Unban(ip); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Marcar como inactivo en DB
	if err := s.db.DeactivateBan(ip); err != nil {
		log.Printf("failed to deactivate ban: %v", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helpers
func getHostname() string {
	host, err := os.Hostname()
	if err != nil {
		return "localhost"
	}
	return host
}

func detectFirewallBackend() string {
	ufw := firewall.NewUFWBackend()
	if ufw.IsAvailable() {
		return "ufw"
	}
	return "none"
}

