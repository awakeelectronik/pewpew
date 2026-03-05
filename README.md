<div align="center">

# 🔫 PewPew

**Real-time SSH attack dashboard + VPS health monitor for your server.**

_One binary. Zero config. Live map. One-click IP ban. VPS metrics._

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![Vue](https://img.shields.io/badge/Vue-3-42b883?logo=vue.js&logoColor=white)](https://vuejs.org)
[![SQLite](https://img.shields.io/badge/SQLite-local-003B57?logo=sqlite&logoColor=white)](https://sqlite.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![RAM](https://img.shields.io/badge/RAM-~10MB-success)](#ram-usage)

</div>

---

## What is PewPew?

Every VPS on the internet receives **thousands of SSH brute-force attempts per day**. PewPew turns your `/var/log/auth.log` into a live security dashboard — so you can see who's hitting you, where they're from, and ban them with a single click.

It also exposes real-time VPS health metrics (CPU, RAM, disk I/O, network throughput and drops, load average) directly from `/proc` — no agents, no external dependencies.

> ⚡ Go backend + Vue 3 frontend compiled into **a single 10 MB binary**. Drop it on your server and it just works.

---

## ✨ Features

- **🗺️ Live Attack Map** — canvas-based world map, attacks animate in real time via WebSocket
- **📡 Event Feed** — filterable stream of SSH events (failed passwords, invalid users, accepted logins)
- **🏆 Top Attackers** — leaderboard with country, attempt count and ban/unban button
- **🔨 One-Click Ban** — integrates with UFW (`ufw deny from <ip>`), logged with full audit trail
- **🔒 Open Ports Scanner** — lists your exposed ports with risk level and reason
- **🛡️ Vuln Findings** — flags sensitive services exposed to the internet
- **💡 Hardening Recommendations** — actionable advice based on your actual server state
- **📊 System Status** — hostname, firewall backend, DB size, event count, uptime
- **🖥️ VPS Health Metrics** — CPU %, RAM %, swap, disk I/O ops/s, network RX/TX bytes/s and drops, load average — all from `/proc`, zero deps
- **🪶 Ultra lightweight** — ~10 MB RAM, SQLite with 30-day auto-purge, no external dependencies in production
- **🔁 Log rotation aware** — detects logrotate and reopens the file without missing lines
- **📦 Single binary deploy** — Vue build embedded via `embed.FS`, zero Node in production

---

## 🚀 Quick Start

```bash
git clone https://github.com/awakeelectronik/pewpew.git
cd pewpew
make build
./bin/pewpew start
```

Open **http://127.0.0.1:9090** → done.

**Requirements:**
- Linux VPS (Debian/Ubuntu/RHEL/Arch)
- Read access to `/var/log/auth.log` (or `/var/log/secure`, `/var/log/authlog`)
- `ufw` installed for IP banning (optional)
- `gcc` (for SQLite CGO compilation)

**Attack map (lat/lon):** The live map shows attack origins only when GeoIP data is available. Install MaxMind GeoLite2-City (free, registration required at [MaxMind](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data)), then place `GeoLite2-City.mmdb` in one of:

- `/usr/share/GeoIP/GeoLite2-City.mmdb`
- `/var/lib/GeoIP/GeoLite2-City.mmdb`
- `~/.pewpew/GeoLite2-City.mmdb`

Restart PewPew; **new** events will get coordinates. Existing DB events keep `lat=0 lon=0` until new attacks arrive.

---

## 🌐 Expose via Nginx (recommended)

PewPew binds to `127.0.0.1:9090` by default. To access it from the internet, proxy it through Nginx with HTTPS:

```nginx
server {
    server_name pewpew.yourdomain.com;
    listen 443 ssl;
    # ... your certs ...

    # Protect with Basic Auth (highly recommended)
    auth_basic "PewPew";
    auth_basic_user_file /etc/nginx/.htpasswd;

    location / {
        proxy_pass http://127.0.0.1:9090;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

> ⚠️ **Always protect with auth** if you expose the dashboard publicly. The panel can ban IPs — you don't want that open.

---

## 📡 API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/events` | Recent events (`?limit=N`) |
| `GET` | `/api/health` | Health check |
| `GET` | `/api/status` | System status (hostname, firewall, DB size, uptime) |
| `GET` | `/api/metrics` | **VPS health snapshot** (CPU, RAM, disk I/O, network, load) |
| `GET` | `/api/attackers` | Top attackers (IP, attempts, country, banned) |
| `GET` | `/api/bans` | Active bans list |
| `POST` | `/api/bans` | Create ban `{"ip":"1.2.3.4","reason":"..."}` |
| `DELETE` | `/api/bans/<ip>` | Remove ban |
| `GET` | `/api/ports` | Open ports with risk assessment |
| `GET` | `/api/vulns` | Network vulnerability findings |
| `GET` | `/api/recommendations` | Hardening recommendations |
| `GET` | `/ws/events` | WebSocket live event stream |

---

## 🖥️ VPS Health Metrics

`GET /api/metrics` returns a point-in-time snapshot computed from `/proc` files.
All rate values (bytes/s, ops/s) are deltas against the previous call — the
collector warms up at startup so the first response is already meaningful.

```json
{
  "cpu":    { "us_percent": 1.2, "sy_percent": 0.3, "idle_percent": 98.1, "wa_percent": 0.4, "st_percent": 0.0 },
  "memory": { "total_bytes": 2062352384, "used_bytes": 980000000, "available_bytes": 1082352384, "used_percent": 47.5 },
  "swap":   { "total_bytes": 0, "used_bytes": 0, "used_percent": 0 },
  "disk":   { "device": "vda", "reads_per_sec": 4, "writes_per_sec": 10 },
  "net":    { "interface": "eth0", "rx_bytes_per_sec": 12400, "tx_bytes_per_sec": 8800,
              "rx_packets_per_sec": 14, "tx_packets_per_sec": 12,
              "rx_drops_total": 705134, "tx_drops_total": 0 },
  "load":   { "load1": 0.15, "load5": 0.03, "load15": 0.01, "procs": 2 }
}
```

**Key fields to watch:**
- `st_percent` — CPU steal time; high values mean the hypervisor is throttling your VPS.
- `wa_percent` — I/O wait; high values indicate disk bottleneck.
- `rx_drops_total` — Lifetime dropped packets on the main interface; a rapidly growing counter means network congestion at the provider level.
- `swap.used_percent` — Any non-zero swap usage under SSH/SCP workloads will cause severe transfer slowdowns.

---

## 🏗️ Architecture

```
pewpew/
├── cmd/pewpew/           # CLI entrypoint (start, version, help)
├── internal/
│   ├── app/             # Wiring: DB → tailer → geoip → HTTP
│   ├── domain/          # SecurityEvent, BanAction, AttackerStat
│   ├── usecase/         # Hardening recommendations engine
│   └── infra/
│       ├── tail/        # SSHTailer — streaming, rotation-aware
│       ├── parse/       # SSH log parser (regex)
│       ├── geoip/       # GeoIP resolver + LRU cache
│       ├── store/       # SQLite repository
│       ├── firewall/    # UFW backend
│       ├── scan/        # Port scanner + vuln engine
│       └── metrics/     # VPS health collector (reads /proc, zero deps)
├── internal/transport/http/   # REST handlers + WebSocket broadcaster
├── static/              # embed.FS: compiled Vue SPA
└── web/                 # Vue 3 + Vite source
```

**Design principles:** Clean Architecture layers, repository pattern for data access, broadcaster pattern for WebSocket fan-out.

---

## 📊 RAM Usage

| Component | Typical usage |
|-----------|---------------|
| Go process | ~5–10 MB |
| SQLite (30d retention, ~100k events/day) | < 50 MB |
| Metrics collector (ring buffers) | < 1 MB |
| **Total target** | **< 100 MB** |

Designed to run comfortably on the smallest VPS plans (1–2 GB RAM).

---

## 🛣️ Roadmap

### MVP & Distribution
- [ ] `install.sh` — one-liner install: download release, systemd unit, auto-start
- [ ] `pewpew.service` systemd unit
- [ ] GitHub Releases: `linux-amd64` and `linux-arm64` binaries + checksums
- [ ] Optional config: `pewpew.yml` / CLI flags for port, log path, retention days
- [ ] **Auth** — simple login (user/password in SQLite) + JWT cookie
- [ ] **GeoIP real** — MaxMind GeoLite2 offline database (country/city/lat/lon)
- [ ] Dashboard screenshot + live map GIF in README

### Phase 2 — Extended Visibility
- [x] **VPS Health Metrics** — CPU, RAM, swap, disk I/O, network RX/TX/drops, load average via `/api/metrics`
- [ ] **Metrics frontend widget** — gauge + sparklines in Vue dashboard, alerts for steal/iowait/drops
- [ ] **Docker-aware** — mount `/var/run/docker.sock`, monitor container logs
- [ ] MySQL brute-force detection from Docker container logs
- [ ] **Nginx scan detection** — flag probes for `/.env`, `/wp-login.php`, `/phpmyadmin`

### Phase 3 — Pro
- [ ] **Auto-ban** rules (like fail2ban, but built-in): threshold + window + action
- [ ] **Notifications** — Discord / Slack / Telegram
- [ ] **FIM** — optional file integrity monitoring
- [ ] **Multi-node** — lightweight agent → central collector

---

## 🔧 Development

```bash
# Full build (Vue → Go with embedded UI)
make build

# Frontend dev server (hot reload)
make dev-frontend

# Backend only (API, no embedded UI)
make dev-backend

# Tests
make test

# Clean
make clean
```

---

## 🔐 Security Notes

- **Local bind by default** — PewPew listens only on `127.0.0.1:9090`. Expose via reverse proxy with HTTPS + auth.
- **UFW ban** — runs `ufw deny from <ip>`. Requires root or sudoers permission. IPs are validated before use.
- **No auth yet** — if you expose the port without a proxy, anyone can see the dashboard and ban IPs. Use Nginx Basic Auth until the built-in auth lands.
- **Audit trail** — every ban/unban is recorded in SQLite with timestamp, IP, reason and actor.

---

## 🤝 Contributing

Pull requests are welcome! Check the [Roadmap](#️-roadmap) for what's next.

1. Fork the repo
2. Create a branch: `git checkout -b feat/your-feature`
3. Commit your changes
4. Push and open a PR

---

## 📄 License

MIT — see [LICENSE](LICENSE).

---

<div align="center">

Made with ☕ and paranoia about VPS security.

**[⭐ Star this repo](https://github.com/awakeelectronik/pewpew)** if PewPew helps you keep your server safe.

</div>
