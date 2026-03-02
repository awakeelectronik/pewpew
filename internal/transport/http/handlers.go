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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// statusHandler retorna estado del sistema
func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]interface{}{
		"hostname":         getHostname(),
		"ssh_log_path":     "/var/log/auth.log", // TODO: desde config
		"firewall_backend": detectFirewallBackend(),
		"db_size":          s.db.GetDBSize(),
		"total_events":     s.db.GetEventCount(),
		"active_bans":      s.db.GetActiveBanCount(),
		"version":          "0.1.0-alpha",
		"uptime_seconds":   getUptime(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// bansHandler gestiona operaciones de ban
func (s *Server) bansHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Listar bans
		bans, err := s.db.GetActiveBans()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bans)

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

// banDetailsHandler maneja DELETE de ban específico
func (s *Server) banDetailsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := strings.TrimPrefix(r.URL.Path, "/api/bans/")
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

func getUptime() int64 {
	// TODO: implementar uptime tracking
	return 0
}
