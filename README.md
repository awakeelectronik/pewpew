# 🔫 PewPew

> Security Dashboard for VPS — Zero-config, single-binary, ultra-lightweight.

## Features (MVP)

- ✅ **Real-time SSH attack detection** via `/var/log/auth.log` streaming
- ✅ **GeoIP enrichment** (offline cache)
- ✅ **Low RAM footprint** (<100 MB estimated for 100k events/day)
- ✅ **Single binary** — no external dependencies in production
- ✅ **SQLite persistence** — local, no MySQL required
- ✅ **Live API** — REST endpoints for event querying
- 🚀 **Ban IP action** (UFW integration coming)
- 🚀 **Vue3 dashboard** (frontend embeded in binary)

## Quick Start

```bash
# Clone repo
git clone https://github.com/awakeelectronik/pewpew.git
cd pewpew

# Build
make build

# Run (requires read access to /var/log/auth.log)
./bin/pewpew start
```

Access dashboard at `http://127.0.0.1:8080`

## API

- `GET /api/events` — Recent security events (JSON)
- `GET /api/health` — Health check
- `GET /` — Dashboard UI (placeholder)

## Architecture

```
internal/
├── app/          # Wiring
├── domain/       # Entities
├── infra/
│   ├── tail/     # Streaming tailer
│   ├── parse/    # SSH parser
│   ├── geoip/    # GeoIP cache
│   ├── store/    # SQLite
│   └── http/     # API server
└── transport/    # HTTP routes
```

## Deployment

### Local / Systemd

```bash
sudo systemctl enable --now pewpew
```

### Behind Nginx (recommended)

```nginx
server {
    server_name pewpew.yourdomain.com;
    listen 443 ssl;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
    }
}
```

## RAM Usage

- **Estimated for 100k events/day:**
  - Process: 5-10 MB
  - SQLite DB: 50 MB (30-day retention)
  - **Total: <100 MB** ✓

## Roadmap

- [ ] Phase 1: MVP complete (SSH detection + UI + UFW ban)
- [ ] Phase 2: Docker-aware, Nginx scan detection
- [ ] Phase 3: Auto-ban, notifications, multi-node

## License

MIT
