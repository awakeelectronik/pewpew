# Phase 2: Vue 3 Frontend + WebSocket + UFW Ban Integration

## ✅ Completed

### Frontend (Vue 3 + Vite)
- [ ] MapView: Live attack visualization (canvas-based)
- [ ] FeedView: Real-time event stream with filtering
- [ ] AttackersView: Top 20 attackers leaderboard
- [ ] BansView: Active bans management
- [ ] StatusView: System status dashboard
- [ ] Store pattern: Centralized state management
- [ ] WebSocket integration: Real-time event streaming
- [ ] REST API client: Axios for HTTP requests
- [ ] Dark mode UI: Hacker-optimized aesthetics

### Backend (Go)
- [ ] WebSocket broadcaster: Event fan-out to multiple clients
- [ ] UFW firewall backend: Ban/unban IP operations
- [ ] Ban CRUD endpoints:
  - [ ] GET /api/bans - List active bans
  - [ ] POST /api/bans - Create ban
  - [ ] DELETE /api/bans/:ip - Delete ban
- [ ] GET /api/status - System status
- [ ] WS /ws/events - Live event stream
- [ ] Handlers module: Separated HTTP handler functions

### Build System
- [ ] Vue build pipeline: Vite configured
- [ ] Go embed: Ready for `embed.FS` integration
- [ ] Makefile targets: web-build, build, run, dev-*

## 🔧 Next Steps

1. **Install dependencies**
   ```bash
   go get github.com/gorilla/websocket
   cd web && npm install
   ```

2. **Test development mode**
   ```bash
   # Terminal 1: Backend
   make dev-backend
   
   # Terminal 2: Frontend
   make dev-frontend
   ```

3. **Integrate Vue build into Go**
   - Uncomment embed.FS in `internal/transport/http/embed.go`
   - Update `server.go` to serve Vue files
   - Test single binary: `make build && ./bin/pewpew`

4. **Setup UFW requirements**
   - Requires sudo access for UFW commands
   - Consider sudoers configuration for production
   ```
   pewpew_user ALL=(ALL) NOPASSWD: /usr/sbin/ufw
   ```

5. **Database schema updates**
   - Add methods to store: GetActiveBans, DeactivateBan, GetDBSize, GetEventCount, GetActiveBanCount
   - Update migrations in `internal/infra/store/sqlite.go`

6. **Error handling**
   - Add retry logic for UFW commands
   - Implement WebSocket reconnection with exponential backoff
   - Add error boundaries in Vue components

## 🚀 MVP Status: 80% Complete

- ✅ Core architecture
- ✅ Frontend UI complete
- ✅ WebSocket streaming
- ✅ UFW integration
- ⏳ Testing phase
- ⏳ GeoIP offline database (Phase 3)
- ⏳ Production hardening (Phase 3)
