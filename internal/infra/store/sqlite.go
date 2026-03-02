package store

import (
	"database/sql"
	"sync"

	"pewpew/internal/domain"

	_ "github.com/mattn/go-sqlite3"
)

// Store interfaz de persistencia
type Store interface {
	Migrate() error
	InsertEventsBatch(events []*domain.SecurityEvent) error
	InsertBan(ban *domain.BanAction) error
	GetRecentEvents(limit int) ([]*domain.SecurityEvent, error)
	Close() error
}

// SQLiteStore implementa Store con SQLite
type SQLiteStore struct {
	db *sql.DB
	mu sync.Mutex
}

// NewSQLiteStore crea una nueva conexión SQLite
func NewSQLiteStore(dbPath string) (Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Tune SQLite para bajo overhead
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)

	return &SQLiteStore{db: db}, nil
}

// Migrate crea tablas si no existen
func (s *SQLiteStore) Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		source TEXT NOT NULL,
		event_type TEXT NOT NULL,
		ip TEXT NOT NULL,
		username TEXT,
		port INTEGER,
		message TEXT,
		country TEXT,
		city TEXT,
		latitude REAL,
		longitude REAL
	);

	CREATE TABLE IF NOT EXISTS bans (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ip TEXT NOT NULL UNIQUE,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		backend TEXT NOT NULL,
		reason TEXT,
		expires_at DATETIME,
		created_by TEXT,
		active BOOLEAN DEFAULT 1
	);

	CREATE INDEX IF NOT EXISTS idx_events_ip ON events(ip);
	CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
	`

	_, err := s.db.Exec(schema)
	return err
}

// InsertEventsBatch inserta un lote de eventos (transacción)
func (s *SQLiteStore) InsertEventsBatch(events []*domain.SecurityEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO events (timestamp, source, event_type, ip, username, port, message, country, city, latitude, longitude)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range events {
		if _, err := stmt.Exec(e.Timestamp, e.Source, e.EventType, e.IP, e.Username, e.Port, e.Message, e.Country, e.City, e.Latitude, e.Longitude); err != nil {
			return err
		}
	}

	return tx.Commit().Error
}

// InsertBan inserta una acción de baneo
func (s *SQLiteStore) InsertBan(ban *domain.BanAction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO bans (ip, timestamp, backend, reason, expires_at, created_by, active)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, ban.IP, ban.Timestamp, ban.Backend, ban.Reason, ban.ExpiresAt, ban.CreatedBy, ban.Active)

	return err
}

// GetRecentEvents retorna últimos N eventos
func (s *SQLiteStore) GetRecentEvents(limit int) ([]*domain.SecurityEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
		SELECT id, timestamp, source, event_type, ip, username, port, message, country, city, latitude, longitude
		FROM events
		ORDER BY timestamp DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.SecurityEvent
	for rows.Next() {
		e := &domain.SecurityEvent{}
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Source, &e.EventType, &e.IP, &e.Username, &e.Port, &e.Message, &e.Country, &e.City, &e.Latitude, &e.Longitude); err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, rows.Err()
}

// Close cierra la conexión
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
