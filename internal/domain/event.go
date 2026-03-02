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
