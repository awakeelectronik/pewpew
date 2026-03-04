# Phase 2 — Extended Visibility

This document tracks progress and design decisions for Phase 2 of PewPew.
It is the canonical reference for agent-assisted development sessions.

---

## Goals

Phase 2 extends PewPew beyond SSH attack detection into a broader
**VPS health and visibility platform** — still zero-config, still a
single binary, still reading from the kernel/filesystem without daemons.

---

## Deliverable 1 — VPS Health Metrics ✅ Implemented

### What it does

Exposes real-time VPS resource metrics via `GET /api/metrics` by reading
`/proc` files directly (no external library needed, no CGO).

### Package

`internal/infra/metrics/`

| File | Responsibility |
|---|---|
| `types.go` | Public structs: `Snapshot`, `CPUStats`, `MemStats`, `SwapStats`, `DiskStats`, `NetStats`, `LoadStats` |
| `collector.go` | `Collector` — reads `/proc`, computes deltas, auto-detects iface/disk |

### Key design decisions

- **Zero dependencies**: only `os`, `bufio`, `strconv`, `strings`, `sync`, `time` from stdlib.
- **Singleton collector** wired into `Server` struct at startup (`metrics.NewCollector()` in `NewServer`).
- **Warm-up on init**: constructor reads initial `/proc` values so first `Collect()` returns real deltas instead of zeros.
- **Auto-detection**:
  - Network interface: first non-`lo`/`docker*`/`veth*`/`br-*` entry in `/proc/net/dev`.
  - Disk device: first non-`loop*`/`sr*` whole-disk (no trailing digit) entry in `/proc/diskstats`.
- **Counter safety**: `safeRate()` returns 0 on counter resets (kernel reboot, overflow).
- **Thread-safe**: `sync.Mutex` guards all mutable state.

### JSON response schema

```json
{
  "cpu": {
    "us_percent": 1.2,
    "sy_percent": 0.3,
    "idle_percent": 98.1,
    "wa_percent": 0.4,
    "st_percent": 0.0
  },
  "memory": {
    "total_bytes": 2062352384,
    "used_bytes": 980000000,
    "available_bytes": 1082352384,
    "used_percent": 47.5
  },
  "swap": {
    "total_bytes": 0,
    "used_bytes": 0,
    "used_percent": 0
  },
  "disk": {
    "device": "vda",
    "reads_per_sec": 4,
    "writes_per_sec": 10
  },
  "net": {
    "interface": "eth0",
    "rx_bytes_per_sec": 12400,
    "tx_bytes_per_sec": 8800,
    "rx_packets_per_sec": 14,
    "tx_packets_per_sec": 12,
    "rx_drops_total": 705134,
    "tx_drops_total": 0
  },
  "load": {
    "load1": 0.15,
    "load5": 0.03,
    "load15": 0.01,
    "procs": 2
  }
}
```

### Frontend TODO (Vue 3)

- [ ] New `MetricsView` (or embedded widget in `StatusView`)
- [ ] Gauge components: CPU%, RAM%, Swap%
- [ ] Sparkline charts: RX/TX bytes/s over last 60 seconds (ring buffer in store)
- [ ] Alert highlight: `rx_drops_total` growing > 100/min → yellow warning badge
- [ ] Alert highlight: `st_percent` (steal) > 10% → red badge (hypervisor throttling)
- [ ] Alert highlight: `wa_percent` (iowait) > 20% → yellow badge (disk bottleneck)
- [ ] Poll `/api/metrics` every 2 seconds (or push via existing WebSocket)

---

## Deliverable 2 — Docker-Aware Log Monitoring ⏳ Planned

### What it does

Mount `/var/run/docker.sock` read-only, enumerate running containers,
tail their logs for known attack patterns (MySQL brute-force, Redis
unauth, etc.).

### Design notes

- New package `internal/infra/docker/` with `ContainerTailer`.
- Reuse existing `domain.SecurityEvent` and broadcaster pipeline.
- Optional: controlled by a `--docker` flag or `pewpew.yml` config key.
- Requires: Docker socket readable (user in `docker` group or root).

### Patterns to detect initially

| Service | Log pattern | Event type |
|---|---|---|
| MySQL | `Access denied for user` | `mysql_auth_failure` |
| Redis | `NOAUTH Authentication required` | `redis_unauth` |
| Nginx | `GET /.env`, `/wp-login.php`, `/phpmyadmin` | `web_probe` |

---

## Deliverable 3 — Nginx Scan Detection ⏳ Planned

Parse Nginx `access.log` for probe patterns targeting common
vulnerabilities. Flag them as `web_probe` events in the existing feed
and map.

- Log path candidates: `/var/log/nginx/access.log`, `/var/log/apache2/access.log`.
- Reuse existing `SSHTailer` pattern with a new `WebTailer`.
- Add regex patterns for: `.env`, `wp-login`, `phpmyadmin`, `xmlrpc.php`,
  `.git/config`, `actuator/`, `_profiler/`.

---

## Phase 1 recap (completed before Phase 2)

| Feature | Status |
|---|---|
| SSH log tailer (rotation-aware) | ✅ |
| SSH log parser (regex) | ✅ |
| GeoIP resolver + LRU cache | ✅ |
| SQLite store (30-day auto-purge) | ✅ |
| REST API (events, attackers, bans, ports, vulns, recommendations, status) | ✅ |
| WebSocket broadcaster (fan-out to multiple clients) | ✅ |
| UFW firewall backend (ban/unban) | ✅ |
| Port scanner + vulnerability engine | ✅ |
| Hardening recommendations engine | ✅ |
| Vue 3 + Vite SPA embedded in binary via `embed.FS` | ✅ |
| Single-binary deploy (10 MB) | ✅ |
| Dark-mode dashboard (Map, Feed, Attackers, Bans, Ports, Status) | ✅ |
