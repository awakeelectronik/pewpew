package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"pewpew/internal/infra/geoip"
	"pewpew/internal/infra/parse"
	"pewpew/internal/infra/store"
	"pewpew/internal/infra/tail"
	httptransport "pewpew/internal/transport/http"
)

type App struct {
	db      store.Store
	tailer  tailer
	geoip   geoip.Resolver
	server  *httptransport.Server
	stopCh  chan struct{}
}

type tailer interface {
	Start(ctx context.Context) error
	Stop()
}

func New() (*App, error) {
	// 1. Init DB (SQLite)
	dbPath := filepath.Join(os.ExpandEnv("$HOME"), ".pewpew", "pewpew.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}
	db, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		return nil, err
	}

	// 2. Migrate DB
	if err := db.Migrate(); err != nil {
		return nil, err
	}

	// 3. Init GeoIP (offline + cache)
	geoipResolver, err := geoip.NewResolver()
	if err != nil {
		log.Printf("warning: geoip resolver init failed: %v (will use placeholder; map will show no coordinates). Install GeoLite2-City.mmdb in /usr/share/GeoIP/ or ~/.pewpew/ and restart.", err)
		geoipResolver = geoip.NewPlaceholderResolver()
	}

	// 4. Init tailer (archivo o journald)
	var sshTailer tailer
	sshLogPath, hasSSHLog := detectSSHLogPath()
	if hasSSHLog {
		sshTailer = tail.NewSSHTailer(sshLogPath, db, geoipResolver, parse.ParseSSHLine, httptransport.BroadcastEvent)
	} else if hasJournald() {
		sshTailer = tail.NewSSHJournalTailer(db, geoipResolver, parse.ParseSSHLine, httptransport.BroadcastEvent)
	} else {
		log.Println("SSH log not found and journald unavailable; SSH detection disabled")
	}

	// 5. Init HTTP server
	httpServer := httptransport.NewServer("127.0.0.1:9090", db, geoipResolver)

	return &App{
		db:     db,
		tailer: sshTailer,
		geoip:  geoipResolver,
		server: httpServer,
		stopCh: make(chan struct{}),
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	log.Println("pewpew starting...")

	if a.tailer != nil {
		go func() {
			if err := a.tailer.Start(ctx); err != nil {
				log.Printf("tailer error: %v", err)
			}
		}()
	}

	// Retención: purgar eventos más antiguos que 30 días (cada hora)
	const retentionDays = 30
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-a.stopCh:
				return
			case <-ticker.C:
				n, err := a.db.DeleteEventsOlderThan(retentionDays * 24 * time.Hour)
				if err != nil {
					log.Printf("retention purge error: %v", err)
				} else if n > 0 {
					log.Printf("retention: deleted %d events older than %d days", n, retentionDays)
				}
			}
		}
	}()

	go func() {
		log.Println("HTTP server listening on http://127.0.0.1:9090")
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
		}
	}()

	return nil
}

func (a *App) Stop() {
	log.Println("stopping app...")
	close(a.stopCh)
	if a.tailer != nil {
		a.tailer.Stop()
	}
	a.server.Close()
	a.db.Close()
}

// detectSSHLogPath autodetecta la ruta del log SSH según distro.
// Devuelve (path, true) si existe; ("", false) si no hay ningún log disponible.
func detectSSHLogPath() (string, bool) {
	candidates := []string{
		"/var/log/auth.log",     // Debian/Ubuntu
		"/var/log/secure",       // RHEL/CentOS/Fedora
		"/var/log/authlog",      // FreeBSD
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			log.Printf("detected SSH log: %s", path)
			return path, true
		}
	}

	return "", false
}

func hasJournald() bool {
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return false
	}
	cmd := exec.Command("journalctl", "--version")
	return cmd.Run() == nil
}

