<div align="center">

# 🔫 PewPew

**Real-time SSH attack dashboard for your VPS.**

_One binary. Zero config. Live map. One-click IP ban._

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![Vue](https://img.shields.io/badge/Vue-3-42b883?logo=vue.js&logoColor=white)](https://vuejs.org)
[![SQLite](https://img.shields.io/badge/SQLite-local-003B57?logo=sqlite&logoColor=white)](https://sqlite.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![RAM](https://img.shields.io/badge/RAM-~10MB-success)](#ram-usage)

![PewPew Dashboard](https://pewpew.hipocondria.co)

</div>

---

## What is PewPew?

Every VPS on the internet receives **thousands of SSH brute-force attempts per day**. PewPew turns your `/var/log/auth.log` into a live security dashboard — so you can see who's hitting you, where they're from, and ban them with a single click.

> ⚡ Go backend + Vue 3 frontend compiled into **a single 10MB binary**. Drop it on your server and it just works.

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
| `GET` | `/api/attackers` | Top attackers (IP, attempts, country, banned) |
| `GET` | `/api/bans` | Active bans list |
| `POST` | `/api/bans` | Create ban `{"ip":"1.2.3.4","reason":"..."}` |
| `DELETE` | `/api/bans/<ip>` | Remove ban |
| `GET` | `/api/ports` | Open ports with risk assessment |
| `GET` | `/api/vulns` | Network vulnerability findings |
| `GET` | `/api/recommendations` | Hardening recommendations |
| `GET` | `/ws/events` | WebSocket live event stream |

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
│       └── scan/        # Port scanner + vuln engine
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
