# Changelog

All notable changes to PewPew are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-03-12

### Added

- **Live Attack Map** — Canvas world map with Mercator projection and Natural Earth GeoJSON. SSH attacks animate in real time via WebSocket.
- **One-click IP ban** — Ban attackers with a single click. PewPew runs `ufw deny from <ip>`, with full audit trail and IP validation.
- **Live event feed** — Filterable real-time stream of SSH events (failed passwords, invalid users, accepted logins) with country/city from offline GeoIP.
- **Top Attackers leaderboard** — Ranked by attempt count, with country, last seen, and ban button.
- **VPS health metrics** — Real-time CPU, RAM, swap, disk I/O, network RX/TX/drops, and load average from `/proc`. No agents, zero deps.
- **Port scanner + vulnerability engine** — Detects open ports, flags risky ones, and provides hardening recommendations based on server state.
- **SSH log tailer** — Rotation-aware, no missed lines. Reads `/var/log/auth.log` (or `/var/log/secure`).
- **SSH log parser** — Extracts IP, user, port, and event type from log lines.
- **GeoIP resolver** — Offline MaxMind GeoLite2, LRU cache, zero API calls.
- **SQLite store** — 30-day auto-purge, full event and ban history.
- **REST API** — Events, attackers, bans, ports, vulns, recommendations, status, health.
- **WebSocket broadcaster** — Live event stream with fan-out to multiple clients.
- **UFW firewall backend** — Ban/unban with full audit trail.
- **Single-binary deploy** — Vue 3 + Vite SPA embedded via `embed.FS`, ~10 MB binary, zero Node.js in production.
- **Dark-mode dashboard** — Map, Feed, Attackers, Bans, Ports, Vulns, Status views.

### Security

- Binds to `127.0.0.1:9090` by default. Use Nginx + HTTPS (and Basic Auth) for external access.
- Every ban/unban recorded in SQLite with timestamp, IP, reason, and actor.

---

## [0.1.0-alpha] - Pre-release

Initial alpha with MVP features. Superseded by 1.0.0.

[1.0.0]: https://github.com/awakeelectronik/pewpew/releases/tag/v1.0.0
[0.1.0-alpha]: https://github.com/awakeelectronik/pewpew/compare/initial...v0.1.0-alpha
