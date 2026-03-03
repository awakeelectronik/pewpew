# 🔫 PewPew

> Security dashboard para VPS: zero-config, un solo binario, ultra liviano. Mapa de ataques en tiempo real, top atacantes y ban IP con un clic.

---

## Qué hace PewPew

- **Un solo binario**: Backend Go + frontend Vue embebido (`embed.FS`). Sin Node ni dependencias en producción.
- **Zero-config**: Autodetección del log SSH (`/var/log/auth.log`, `/var/log/secure`, `/var/log/authlog`) y fallback a journald si no hay archivo.
- **Tail en tiempo real**: Lee el auth log en streaming (sin cargar el archivo entero). Detecta rotación (logrotate) y reabre el archivo para no perder líneas.
- **Parsing SSH**: Detecta `Failed password`, `Invalid user`, `Accepted password`/`publickey`, errores de handshake (kex) y desconexiones; extrae IP, usuario y puerto.
- **GeoIP**: Resolver con caché (actualmente placeholder; preparado para MaxMind GeoLite2 offline).
- **SQLite local**: Eventos, bans y hallazgos. Retención configurada: eventos más antiguos de 30 días se purgan automáticamente cada hora.
- **API REST**: Eventos, estado, top atacantes, bans, puertos abiertos, hallazgos de vulnerabilidad y recomendaciones de hardening.
- **WebSocket**: Feed en vivo de eventos (`/ws/events`) para el mapa y el feed de la UI.
- **Dashboard Vue 3**: Modo oscuro; pantallas: Mapa live, Feed, Top Attackers, Bans, Status, Puertos, Vulnerabilidades, Recomendaciones.
- **Ban / Unban IP**: Integración con UFW; validación de IP; cada acción se registra en la base de datos (auditoría).
- **Seguridad por defecto**: Servidor escucha solo en `127.0.0.1:9090`. Acceso desde internet vía Nginx (o similar) como reverse proxy.
- **Cierre ordenado**: Al parar, el servidor HTTP hace `Shutdown` para no dejar conexiones colgadas.

**CLI:** `pewpew`, `pewpew start`, `pewpew version`, `pewpew help`.

---

## Quick Start

```bash
git clone https://github.com/awakeelectronik/pewpew.git
cd pewpew
make build
./bin/pewpew start
```

Abre **http://127.0.0.1:9090** en el navegador.

**Requisitos:** Lectura de `/var/log/auth.log` (o equivalente). Para Ban IP: UFW instalado y permisos para ejecutar `ufw deny/delete`.

**Por qué 9090:** Evita conflicto con el puerto 8080 (proxies/desarrollo). 9090 es habitual en dashboards y métricas.

**Tail y rotación:** Si logrotate renombra el archivo de log, PewPew detecta el cambio y reabre el archivo para seguir leyendo sin perder líneas.

---

## API

| Método / Ruta | Descripción |
|----------------|-------------|
| `GET /api/events` | Últimos eventos (query `?limit=N`) |
| `GET /api/health` | Health check |
| `GET /api/status` | Estado: hostname, log path, firewall, DB size, eventos, bans, versión, uptime |
| `GET /api/attackers` | Top atacantes (IP, intentos, país, último evento, banned) |
| `GET /api/bans` | Lista de bans activos |
| `POST /api/bans` | Crear ban (body: `{"ip":"1.2.3.4","reason":"..."}`) |
| `DELETE /api/bans/<ip>` | Quitar ban |
| `GET /api/ports` | Puertos abiertos (exposición local) |
| `GET /api/vulns` | Hallazgos de vulnerabilidad de red |
| `GET /api/recommendations` | Recomendaciones de hardening |
| `GET /ws/events` | WebSocket: stream de eventos en vivo |
| `GET /` | UI del dashboard (SPA Vue) |

---

## Arquitectura

```
pewpew/
├── cmd/pewpew/           # main, CLI (start, version, help)
├── internal/
│   ├── app/             # Wiring: DB, tailer, GeoIP, HTTP
│   ├── domain/          # SecurityEvent, BanAction, AttackerStat, etc.
│   ├── usecase/         # recommendations
│   ├── infra/
│   │   ├── tail/        # SSHTailer (archivo + rotación), SSHJournalTailer
│   │   ├── parse/       # Parser SSH (regex)
│   │   ├── geoip/       # Resolver + caché
│   │   ├── store/       # SQLite (events, bans, findings)
│   │   ├── firewall/    # UFW backend
│   │   └── scan/        # Puertos, vulnerabilidades
│   └── transport/http/  # Handlers REST, WebSocket, servir UI embebida
├── static/              # embed.FS de web/dist
└── web/                 # Vue 3 + Vite (build → static/dist)
```

---

## Despliegue

### Solo local

```bash
./bin/pewpew start
```

### Detrás de Nginx (recomendado para acceso externo)

```nginx
server {
    server_name pewpew.tudominio.com;
    listen 443 ssl;
    # ... certificados ...

    location / {
        proxy_pass http://127.0.0.1:9090;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Sec-WebSocket-Key $http_sec_websocket_key;
        proxy_set_header Sec-WebSocket-Version $http_sec_websocket_version;
    }
}
```

### Systemd (cuando exista unit)

```bash
sudo systemctl enable --now pewpew
```

*(Falta aún: unit y script de instalación; ver Roadmap.)*

---

## Uso de RAM

- Proceso: ~5–10 MB.
- SQLite con retención de 30 días (purga automática): típicamente <50 MB para ~100k eventos/día.
- **Objetivo: <100 MB** en VPS pequeños (2 GB RAM).

---

## Roadmap

### Próximamente (cerrar MVP y distribución)

- [ ] **Script de instalación** (`install.sh`): descarga del release, chmod, usuario, systemd unit, arranque.
- [ ] **Unit systemd** para `pewpew.service`.
- [ ] **Releases** en GitHub: binarios linux-amd64 y linux-arm64 + checksums.
- [ ] **Config** opcional: `pewpew.yml` o flags para puerto, ruta del log, días de retención.
- [ ] **Autenticación** en el panel: login simple (usuario/contraseña en SQLite) + cookie/JWT.
- [ ] **GeoIP real**: integración con MaxMind GeoLite2 (país/ciudad/lat/lon).
- [ ] **README**: screenshot del dashboard y GIF del mapa en vivo.
- [ ] **Endpoint `/debug`** (opcional): backlog, eventos/s, tamaño DB.

### Fase 2 (Docker y Nginx)

- [ ] **Docker-aware**: detectar `/var/run/docker.sock`, listar contenedores y permitir “monitorear logs del contenedor X”.
- [ ] Regla de detección: MySQL “Access denied” en logs de contenedores.
- [ ] **Detección de escaneos Nginx**: rutas típicas (/.env, /wp-login.php, /phpmyadmin) con muestreo/agregación para no saturar.

### Fase 3 (Pro)

- [ ] **Auto-ban** tipo fail2ban: reglas configurables (umbral de intentos, ventana, acción).
- [ ] **Notificaciones**: Discord, Slack y/o Telegram.
- [ ] **FIM** (integridad de archivos) opcional.
- [ ] **Multi-nodo**: agente liviano que envía eventos a un central.

---

## Desarrollo

```bash
# Build completo (Vue + Go con UI embebida)
make build

# Solo frontend (dev server Vue)
make dev-frontend

# Solo backend (sin UI embebida; sirve API)
make dev-backend

# Tests
make test

# Limpiar
make clean
```

---

## Seguridad

- **Bind local:** Por defecto PewPew solo escucha en `127.0.0.1`. Para exponerlo, usar reverse proxy (Nginx) con HTTPS y, si quieres, autenticación (Basic Auth en Nginx o login en PewPew cuando exista).
- **Ban IP:** Se ejecuta `ufw deny from <ip>` / `ufw delete deny from <ip>`. Hace falta permisos suficientes (root o sudoers para esos comandos). Las IP se validan; no se concatena input a comandos sin sanitizar.
- **Sin auth en el panel (hoy):** Si expones el puerto, cualquiera con acceso puede ver el dashboard y ejecutar ban. No exponer sin proxy y/o auth.

---

## Licencia

MIT. Ver [LICENSE](LICENSE).
