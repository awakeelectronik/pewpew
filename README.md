<div align="center">

# 🔫 PewPew

### Your VPS is under attack right now. Let's watch.

**Real-time SSH attack map + VPS health monitor — single binary, zero config, ~20 MB RAM.**

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![Vue](https://img.shields.io/badge/Vue-3-42b883?logo=vue.js&logoColor=white)](https://vuejs.org)
[![SQLite](https://img.shields.io/badge/SQLite-local-003B57?logo=sqlite&logoColor=white)](https://sqlite.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![RAM](https://img.shields.io/badge/RAM-~10MB-success)](#-ram-usage)

_Drop one binary on your server. Watch the world try to hack you. Ban them with one click._

</div>

---

## The story

Every VPS on the internet gets **thousands of SSH brute-force attempts every single day**. Most people never know. PewPew reads your `/var/log/auth.log` in real time, plots every attack on a live world map, and lets you ban offenders with one click — all from a dark-mode dashboard served by a **single 10 MB Go binary with zero runtime dependencies**.

No Docker. No Elasticsearch. No agents. No config files. Just:

```bash
git clone https://github.com/awakeelectronik/pewpew.git
cd pewpew && make build
./bin/pewpew start
```

Open **http://127.0.0.1:9090** and watch the pew pew map light up. 🌍💥

---

## ✨ What it does

### 🗺️ Live Attack Map
Canvas world map with proper Mercator projection and Natural Earth GeoJSON. Every SSH attack animates in real time via WebSocket. Watching bots from Russia, China, and the US hammer your port 22 at 3am is oddly satisfying.

### 🔨 One-Click IP Ban
See an attacker? Click **BAN**. PewPew runs `ufw deny from <ip>`, logs the action with a full audit trail, and validates every IP before touching your firewall. Unban just as easily.

### 📡 Live Event Feed
Filterable real-time stream of SSH events: failed passwords, invalid users, accepted logins — every line from your auth log, enriched with country and city from offline GeoIP.

### 🏆 Top Attackers Leaderboard
Ranked by attempt count, with country, last seen timestamp, and ban button. See who's been the most persistent.

### 🖥️ VPS Health Metrics
Real-time CPU, RAM, swap, disk I/O, network RX/TX/drops, and load average — read directly from `/proc`. Zero deps, zero agents. Pay special attention to `st_percent` (CPU steal): if it's climbing, your hypervisor is throttling you and you're not getting what you paid for.

### 🔒 Port Scanner + Vuln Engine
Sees your open ports, flags risky ones with reasons, and gives you hardening recommendations based on your actual server state — not generic checklists.

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
- Linux VPS (Debian/Ubuntu/RHEL/Arch — autodetected)
- Read access to `/var/log/auth.log` (or `/var/log/secure`)
- `ufw` for IP banning (optional)
- `gcc` for SQLite CGO compilation

### 🌐 Attack Map coordinates

The live map shows attack origins when GeoIP data is available. Get the free MaxMind GeoLite2 database ([free registration here](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data)), then place `GeoLite2-City.mmdb` in any of:

```
/usr/share/GeoIP/GeoLite2-City.mmdb
/var/lib/GeoIP/GeoLite2-City.mmdb
~/.pewpew/GeoLite2-City.mmdb
```

Restart PewPew — new events get coordinates immediately.

---

## 🌐 Expose via Nginx (recommended)

PewPew binds to `127.0.0.1:9090` by default. Proxy it behind Nginx with HTTPS:

```nginx
server {
    server_name pewpew.yourdomain.com;
    listen 443 ssl;
    # ... your certs ...

    # Strongly recommended: protect with Basic Auth
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

> ⚠️ **Always protect with auth** before exposing publicly. The dashboard can ban IPs — you don't want that open to the internet.

---

## 🖥️ VPS Health Metrics

`GET /api/metrics` — a real-time snapshot computed from `/proc` files. All rate values (bytes/s, ops/s) are deltas against the previous call. The collector warms up at startup so the first response is already meaningful.

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

| Field | What it means |
|---|---|
| `st_percent` | CPU steal — high = your hypervisor is throttling you |
| `wa_percent` | I/O wait — high = disk bottleneck |
| `rx_drops_total` | Dropped packets lifetime — growing fast = network congestion at provider level |
| `swap.used_percent` | Any non-zero swap during normal use = RAM pressure |

---

## 📡 API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/events` | Recent SSH events (`?limit=N`) |
| `GET` | `/api/attackers` | Top attackers (IP, attempts, country, banned) |
| `GET` | `/api/bans` | Active bans list |
| `POST` | `/api/bans` | Create ban `{"ip":"1.2.3.4","reason":"..."}` |
| `DELETE` | `/api/bans/<ip>` | Remove ban |
| `GET` | `/api/metrics` | VPS health snapshot (CPU, RAM, disk, network, load) |
| `GET` | `/api/ports` | Open ports with risk assessment |
| `GET` | `/api/vulns` | Network vulnerability findings |
| `GET` | `/api/recommendations` | Hardening recommendations |
| `GET` | `/api/status` | System status (hostname, firewall backend, DB size, uptime) |
| `GET` | `/api/health` | Health check |
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
│       ├── store/       # SQLite repository (30-day auto-purge)
│       ├── firewall/    # UFW backend
│       ├── scan/        # Port scanner + vuln engine
│       └── metrics/     # /proc reader — CPU, RAM, disk, network, load
├── internal/transport/http/   # REST handlers + WebSocket broadcaster
├── static/              # embed.FS: compiled Vue SPA
└── web/                 # Vue 3 + Vite source
```

Clean Architecture layers, repository pattern, broadcaster pattern for WebSocket fan-out. Vue 3 + Vite frontend compiled and embedded into the binary via `embed.FS` — **zero Node.js in production**.

---

## 📊 RAM Usage

| Component | Typical |
|-----------|---|
| Go process | ~5–10 MB |
| SQLite (30d retention, ~100k events/day) | < 50 MB |
| Metrics collector (ring buffers) | < 1 MB |
| **Total** | **< 100 MB** |

Designed to run comfortably on the smallest VPS plans (512 MB – 2 GB RAM) without competing with your actual workloads.

---

## 🛣️ Roadmap

### Phase 1 — MVP ✅ Done
- [x] SSH log tailer (rotation-aware, no missed lines)
- [x] SSH log parser — extracts IP, user, port, event type
- [x] GeoIP resolver + LRU cache (offline, zero API calls)
- [x] SQLite store with 30-day auto-purge
- [x] Full REST API (events, attackers, bans, ports, vulns, recommendations, status)
- [x] WebSocket broadcaster with fan-out to multiple clients
- [x] UFW firewall backend — ban/unban + full audit trail
- [x] Port scanner + vulnerability engine
- [x] Hardening recommendations engine
- [x] Vue 3 + Vite SPA embedded in binary via `embed.FS`
- [x] Single-binary deploy (~10 MB)
- [x] Dark-mode dashboard (Map, Feed, Attackers, Bans, Ports, Vulns, Status)
- [x] World map with Mercator projection + Natural Earth GeoJSON polygons
- [x] VPS health metrics from `/proc` (CPU, RAM, disk I/O, network RX/TX/drops, load)

### Phase 2 — Extended Visibility 🚧 In Progress
- [ ] Metrics frontend widget — gauges + sparklines in Vue, alerts for steal/iowait/drops
- [ ] `install.sh` — one-liner: download release, create systemd unit, auto-start
- [ ] GitHub Releases: `linux-amd64` + `linux-arm64` prebuilt binaries + checksums
- [ ] Built-in auth — login (user/password + JWT cookie) so you can safely expose to internet
- [ ] Docker-aware — tail container logs (MySQL brute-force, Redis unauth, Nginx probes)
- [ ] Nginx scan detection — flag probes for `/.env`, `/wp-login.php`, `/phpmyadmin`, `/.git`

### Phase 3 — Pro
- [ ] Auto-ban rules (fail2ban-like, built-in): threshold + time window + action
- [ ] Notifications — Discord / Slack / Telegram webhooks
- [ ] File Integrity Monitoring (FIM)
- [ ] Multi-node — lightweight agent → central collector dashboard

---

## 🔧 Development

```bash
# Full build (Vue → Go embedded binary)
make build

# Frontend dev server (hot reload at :5173)
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

- **Local bind by default** — PewPew only listens on `127.0.0.1:9090`. Use Nginx + HTTPS for external access.
- **UFW ban** — runs `ufw deny from <ip>`. Requires root or a specific sudoers entry. IPs are always validated before use.
- **No built-in auth yet** — protect with Nginx Basic Auth until Phase 2 auth lands.
- **Full audit trail** — every ban/unban recorded in SQLite with timestamp, IP, reason, and actor.

---

## 🤝 Contributing

Pull requests are welcome! Check the [Roadmap](#️-roadmap) for what's coming.

1. Fork the repo
2. `git checkout -b feat/your-feature`
3. Commit and push
4. Open a PR

---

## 📄 License

MIT — see [LICENSE](LICENSE).

---

<div align="center">

Made with ☕ and a healthy paranoia about VPS security.

**[⭐ Star this repo](https://github.com/awakeelectronik/pewpew)** if PewPew helps you keep your server safe.

</div>
