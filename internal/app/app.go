package app

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"pewpew/internal/infra/geoip"
	"pewpew/internal/infra/parse"
	"pewpew/internal/infra/store"
	"pewpew/internal/infra/tail"
	"pewpew/internal/transport/http"
)

type App struct {
	db      store.Store
	tailer  *tail.SSHTailer
	geoip   geoip.Resolver
	server  *http.Server
	stopCh  chan struct{}
}

func New() (*App, error) {
	// 1. Init DB (SQLite)
	dbPath := filepath.Join(os.ExpandEnv("$HOME"), ".pewpew", "pewpew.db")
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
		log.Printf("warning: geoip resolver init failed: %v (will use placeholder)\n", err)
		geoipResolver = geoip.NewPlaceholderResolver()
	}

	// 4. Init tailer (SSH)
	sshLogPath := detectSSHLogPath()
	tailer := tail.NewSSHTailer(sshLogPath, db, geoipResolver, parse.ParseSSHLine)

	// 5. Init HTTP server
	httpServer := http.NewServer(":8080", db, geoipResolver)

	return &App{
		db:     db,
		tailer: tailer,
		geoip:  geoipResolver,
		server: httpServer,
		stopCh: make(chan struct{}),
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	log.Println("pewpew starting...")

	// Start tailer
	go func() {
		if err := a.tailer.Start(ctx); err != nil {
			log.Printf("tailer error: %v", err)
		}
	}()

	// Start HTTP server
	go func() {
		log.Println("HTTP server listening on http://127.0.0.1:8080")
		if err := a.server.ListenAndServe(); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	return nil
}

func (a *App) Stop() {
	log.Println("stopping app...")
	close(a.stopCh)
	a.tailer.Stop()
	a.server.Close()
	a.db.Close()
}

// detectSSHLogPath autodetecta la ruta del log SSH según distro
func detectSSHLogPath() string {
	candidates := []string{
		"/var/log/auth.log",     // Debian/Ubuntu
		"/var/log/secure",       // RHEL/CentOS/Fedora
		"/var/log/authlog",      // FreeBSD
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			log.Printf("detected SSH log: %s", path)
			return path
		}
	}

	log.Println("warning: SSH log not found, using default /var/log/auth.log")
	return "/var/log/auth.log"
}
