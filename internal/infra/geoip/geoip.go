package geoip

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/oschwald/geoip2-golang"
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

// CachedResolver usa caché local; sin DB devuelve placeholder (0,0)
type CachedResolver struct {
	cache map[string]GeoData
	mu    sync.RWMutex
}

// MaxMindResolver usa la base de datos GeoLite2-City para lat/lon reales
type MaxMindResolver struct {
	db   *geoip2.Reader
	cache map[string]GeoData
	mu    sync.RWMutex
}

// geoLite2Paths orden de búsqueda de la base de datos
var geoLite2Paths = []string{
	"/usr/share/GeoIP/GeoLite2-City.mmdb",
	"/var/lib/GeoIP/GeoLite2-City.mmdb",
}

func init() {
	home, err := os.UserHomeDir()
	if err == nil {
		geoLite2Paths = append(geoLite2Paths,
			filepath.Join(home, ".pewpew", "GeoLite2-City.mmdb"),
			filepath.Join(home, "GeoLite2-City.mmdb"),
		)
	}
	geoLite2Paths = append(geoLite2Paths, "GeoLite2-City.mmdb")
}

// NewResolver crea un resolver con MaxMind si existe la DB; si no, error (usar NewPlaceholderResolver)
func NewResolver() (Resolver, error) {
	for _, path := range geoLite2Paths {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		db, err := geoip2.Open(path)
		if err != nil {
			log.Printf("[geoip] open %s: %v", path, err)
			continue
		}
		log.Printf("[geoip] using database: %s", path)
		return &MaxMindResolver{
			db:    db,
			cache: make(map[string]GeoData),
		}, nil
	}
	return nil, fmt.Errorf("no GeoLite2-City.mmdb found (looked in %v)", geoLite2Paths)
}

// Resolve implementa Resolver usando la DB MaxMind (con caché)
func (m *MaxMindResolver) Resolve(ip string) GeoData {
	m.mu.RLock()
	if cached, ok := m.cache[ip]; ok {
		m.mu.RUnlock()
		return cached
	}
	m.mu.RUnlock()

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return m.storePlaceholder(ip, GeoData{CountryCode: "XX", City: "Unknown", Latitude: 0, Longitude: 0})
	}

	record, err := m.db.City(parsed)
	if err != nil || record == nil {
		return m.storePlaceholder(ip, GeoData{CountryCode: "XX", City: "Unknown", Latitude: 0, Longitude: 0})
	}

	countryCode := "XX"
	if record.Country.IsoCode != "" {
		countryCode = record.Country.IsoCode
	}
	city := "Unknown"
	if name, ok := record.City.Names["en"]; ok && name != "" {
		city = name
	}
	lat := float64(0)
	lon := float64(0)
	if record.Location.Latitude != 0 || record.Location.Longitude != 0 {
		lat = record.Location.Latitude
		lon = record.Location.Longitude
	}

	geoData := GeoData{
		CountryCode: countryCode,
		City:        city,
		Latitude:    lat,
		Longitude:   lon,
	}
	return m.storePlaceholder(ip, geoData)
}

func (m *MaxMindResolver) storePlaceholder(ip string, g GeoData) GeoData {
	m.mu.Lock()
	m.cache[ip] = g
	m.mu.Unlock()
	return g
}

// NewPlaceholderResolver para cuando no hay DB GeoIP (lat/lon siempre 0)
func NewPlaceholderResolver() Resolver {
	return &CachedResolver{
		cache: make(map[string]GeoData),
	}
}

// Resolve (CachedResolver) devuelve siempre placeholder 0,0
func (c *CachedResolver) Resolve(ip string) GeoData {
	c.mu.RLock()
	if cached, ok := c.cache[ip]; ok {
		c.mu.RUnlock()
		return cached
	}
	c.mu.RUnlock()

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
