package geoip

import (
	"sync"
)

// GeoData contiene información geográfica
type GeoData struct {
	CountryCode string
	City        string
	Latitude    float64
	Longitude   float64
	ASN         string
}

// Resolver interfaz
type Resolver interface {
	Resolve(ip string) GeoData
}

// CachedResolver usa caché local + lookup offline (MVP: placeholder)
type CachedResolver struct {
	cache map[string]GeoData
	mu    sync.RWMutex
}

// NewResolver crea un resolver (MVP sin DB GeoIP aún)
func NewResolver() (Resolver, error) {
	return &CachedResolver{
		cache: make(map[string]GeoData),
	}, nil
}

// NewPlaceholderResolver para cuando no hay DB GeoIP
func NewPlaceholderResolver() Resolver {
	return &CachedResolver{
		cache: make(map[string]GeoData),
	}
}

// Resolve resuelve IP a GeoData (con caché)
func (c *CachedResolver) Resolve(ip string) GeoData {
	c.mu.RLock()
	if cached, ok := c.cache[ip]; ok {
		c.mu.RUnlock()
		return cached
	}
	c.mu.RUnlock()

	// MVP: placeholder; luego integrar MaxMind GeoLite2 offline
	geoData := GeoData{
		CountryCode: "XX",
		City:        "Unknown",
		Latitude:    0,
		Longitude:   0,
	}

	c.mu.Lock()
	c.cache[ip] = geoData
	c.mu.Unlock()

	return geoData
}
