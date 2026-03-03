package domain

import "time"

// SecurityEvent representa un evento de seguridad
type SecurityEvent struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"` // ssh, nginx, docker
	EventType string    `json:"event_type"` // failed_password, invalid_user, accepted_password
	IP        string    `json:"ip"`
	Username  string    `json:"username,omitempty"`
	Port      int       `json:"port,omitempty"`
	Message   string    `json:"message"`
	Country   string    `json:"country,omitempty"`
	City      string    `json:"city,omitempty"`
	Latitude  float64   `json:"latitude,omitempty"`
	Longitude float64   `json:"longitude,omitempty"`
}

// BanAction representa una acción de baneo
type BanAction struct {
	ID        int64     `json:"id"`
	IP        string    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
	Backend   string    `json:"backend"` // ufw, iptables, nft
	Reason    string    `json:"reason"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedBy string    `json:"created_by,omitempty"`
	Active    bool      `json:"active"`
}

// AttackerStat representa la agregacion por IP atacante.
type AttackerStat struct {
	IP         string    `json:"ip"`
	Count      int64     `json:"count"`
	LastSeen   time.Time `json:"last_seen"`
	Country    string    `json:"country,omitempty"`
	City       string    `json:"city,omitempty"`
	LastType   string    `json:"last_type,omitempty"`
	Banned     bool      `json:"banned"`
}

// OpenPort representa un puerto abierto detectado localmente.
type OpenPort struct {
	Proto      string `json:"proto"`
	Address    string `json:"address"`
	Port       int    `json:"port"`
	Process    string `json:"process,omitempty"`
	State      string `json:"state"`
	Risk       string `json:"risk"`
	Reason     string `json:"reason"`
	Externally bool   `json:"externally_reachable"`
}

// VulnerabilityFinding representa un hallazgo de riesgo de red.
type VulnerabilityFinding struct {
	ID         string `json:"id"`
	Severity   string `json:"severity"` // low, medium, high, critical
	Category   string `json:"category"`
	Target     string `json:"target"`
	Evidence   string `json:"evidence"`
	Recommendation string `json:"recommendation"`
}

// Recommendation representa una recomendacion de hardening.
type Recommendation struct {
	ID        string `json:"id"`
	Severity  string `json:"severity"` // ok, warn, critical
	Title     string `json:"title"`
	Details   string `json:"details"`
	Command   string `json:"command,omitempty"`
}
