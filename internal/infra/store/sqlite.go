package store

import (
	"database/sql"
	"net"
	"sync"
	"time"

	"pewpew/internal/domain"

	_ "github.com/mattn/go-sqlite3"
)

// Store interfaz de persistencia
type Store interface {
	Migrate() error
	InsertEventsBatch(events []*domain.SecurityEvent) error
	InsertBan(ban *domain.BanAction) error
	GetRecentEvents(limit int) ([]*domain.SecurityEvent, error)
	GetTopAttackers(limit int) ([]*domain.AttackerStat, error)
	GetDBSize() int64
	GetEventCount() int64
	GetActiveBanCount() int64
	GetActiveBans() ([]*domain.BanAction, error)
	GetRecentFindingCount(hours int) int64
	DeactivateBan(ip string) error
	DeleteEventsOlderThan(d time.Duration) (int64, error)
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

	CREATE TABLE IF NOT EXISTS findings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ts DATETIME DEFAULT CURRENT_TIMESTAMP,
		severity TEXT NOT NULL,
		category TEXT NOT NULL,
		target TEXT NOT NULL,
		evidence TEXT NOT NULL,
		recommendation TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_events_ip ON events(ip);
	CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
	CREATE INDEX IF NOT EXISTS idx_findings_ts ON findings(ts);
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

	return tx.Commit()
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

// GetTopAttackers retorna IPs con mas eventos recientes.
func (s *SQLiteStore) GetTopAttackers(limit int) ([]*domain.AttackerStat, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
		SELECT
			e.ip,
			COUNT(*) as total,
			MAX(e.timestamp) as last_seen,
			COALESCE(MAX(e.country), '') as country,
			COALESCE(MAX(e.city), '') as city,
			COALESCE(
				(
					SELECT e2.event_type
					FROM events e2
					WHERE e2.ip = e.ip
					ORDER BY e2.timestamp DESC
					LIMIT 1
				),
				''
			) as last_type,
			CASE WHEN EXISTS (
				SELECT 1 FROM bans b WHERE b.ip = e.ip AND b.active = 1
			) THEN 1 ELSE 0 END as banned
		FROM events e
		GROUP BY e.ip
		ORDER BY total DESC, last_seen DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.AttackerStat
	for rows.Next() {
		a := &domain.AttackerStat{}
		var banned int
		var lastSeenRaw string
		if err := rows.Scan(&a.IP, &a.Count, &lastSeenRaw, &a.Country, &a.City, &a.LastType, &banned); err != nil {
			return nil, err
		}
		// Excluir IPs inválidas (ej. caracteres sueltos "b", "f" por parseo erróneo)
		if a.IP == "" || net.ParseIP(a.IP) == nil {
			continue
		}
		a.LastSeen = parseSQLiteTime(lastSeenRaw)
		a.Banned = banned != 0
		out = append(out, a)
	}

	return out, rows.Err()
}

// GetDBSize retorna el tamaño de la base de datos en bytes
func (s *SQLiteStore) GetDBSize() int64 {
	var pageCount, pageSize int
	if err := s.db.QueryRow("PRAGMA page_count").Scan(&pageCount); err != nil {
		return 0
	}
	if err := s.db.QueryRow("PRAGMA page_size").Scan(&pageSize); err != nil {
		return 0
	}
	return int64(pageCount) * int64(pageSize)
}

// GetEventCount retorna el número total de eventos
func (s *SQLiteStore) GetEventCount() int64 {
	var n int64
	if err := s.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&n); err != nil {
		return 0
	}
	return n
}

// GetActiveBanCount retorna el número de bans activos
func (s *SQLiteStore) GetActiveBanCount() int64 {
	var n int64
	if err := s.db.QueryRow("SELECT COUNT(*) FROM bans WHERE active = 1").Scan(&n); err != nil {
		return 0
	}
	return n
}

// GetRecentFindingCount retorna findings recientes para status.
func (s *SQLiteStore) GetRecentFindingCount(hours int) int64 {
	var n int64
	if err := s.db.QueryRow("SELECT COUNT(*) FROM findings WHERE ts >= datetime('now', ?)", "-"+itoa(hours)+" hours").Scan(&n); err != nil {
		return 0
	}
	return n
}

// GetActiveBans retorna todos los bans activos
func (s *SQLiteStore) GetActiveBans() ([]*domain.BanAction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
		SELECT id, ip, timestamp, backend, reason, expires_at, created_by, active
		FROM bans
		WHERE active = 1
		ORDER BY timestamp DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bans []*domain.BanAction
	for rows.Next() {
		b := &domain.BanAction{}
		var active bool
		if err := rows.Scan(&b.ID, &b.IP, &b.Timestamp, &b.Backend, &b.Reason, &b.ExpiresAt, &b.CreatedBy, &active); err != nil {
			return nil, err
		}
		b.Active = active
		bans = append(bans, b)
	}
	return bans, rows.Err()
}

// DeactivateBan marca un ban como inactivo por IP
func (s *SQLiteStore) DeactivateBan(ip string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("UPDATE bans SET active = 0 WHERE ip = ?", ip)
	return err
}

// DeleteEventsOlderThan borra eventos con timestamp anterior a (now - d). Retorna el número de filas borradas.
func (s *SQLiteStore) DeleteEventsOlderThan(d time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-d)
	res, err := s.db.Exec("DELETE FROM events WHERE timestamp < ?", cutoff.Format("2006-01-02 15:04:05"))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Close cierra la conexión
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + (v % 10))
		v /= 10
	}
	return sign + string(buf[i:])
}

func parseSQLiteTime(v string) time.Time {
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t
		}
	}
	return time.Now()
}
