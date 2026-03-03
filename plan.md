Actúa como un desarrollador Full-Stack experto en Go y Vue 3. Abajo pego el archivo plan.md con la arquitectura exacta de una nueva aplicación llamada PewPew (un security dashboard de un solo binario). Por favor, lee todo el plan. Como primer paso, genera la estructura de carpetas en Go (Clean Architecture) y escribe el código para el archivo main.go y el servicio (tailer) que lee en tiempo real el archivo /var/log/auth.log sin consumir más de 10MB de RAM.

# pewpew — plan.md

## 0) Objetivo del proyecto

Construir **pewpew**, un *security dashboard* “hacker/devtool” para VPS que sea:
- **Zero-config**: que funcione apenas lo ejecutes (con autodetección de logs y entorno).
- **Single-binary**: backend Go + frontend Vue embebido en el binario (sin Node en producción).
- **Ultra-liviano**: diseñado para VPS pequeños; prioridad explícita en **bajo consumo de RAM/CPU** (p. ej. VPS de 2GB). [cite:50]
- **Wow-factor para GitHub**: “pew pew live attack map”, UI moderna, instalación de 1 comando, y botones de acción (ban IP).

El MVP debe enfocarse en señales universales (SSH) y en una UI que sea compartible (GIF/video del mapa).

---

## 1) Principios de diseño (no negociables)

### 1.1 RAM-first
- El sistema debe funcionar en VPS modestos (2GB) sin competir con Nginx/MySQL/Apps. [cite:50]
- Evitar stacks pesados (ELK, etc.). Mantener:
  - Go (stdlib + librerías pequeñas)
  - SQLite local
  - Vue build estático embebido

### 1.2 “No dependas de lo que monitoreas”
- La base de datos de pewpew debe ser **SQLite local** para no depender de MySQL (que es uno de los objetivos a observar). [cite:62]

### 1.3 Autodetección > configuración
- Config por defecto debe ser “best effort”: detectar distro, rutas de logs, Docker presente, firewall presente, etc.
- Aun así, permitir override vía `pewpew.yml` o flags.

### 1.4 Seguridad por defecto
- Principio de mínimo privilegio.
- Nada de exponer panel a internet por defecto (bind a `127.0.0.1` y uso de Nginx reverse proxy para acceso externo).
- Ban IP debe ser explícito (botón) en MVP; autoban puede llegar luego.

---

## 2) Alcance por fases (prioridad para popularidad)

### Fase 1 (MVP “stars”)
1. **Parser + stream de eventos de SSH**:
   - Tail de auth log (Debian/Ubuntu: `/var/log/auth.log`, etc.).
   - Detección de:
     - `Failed password`
     - `Invalid user`
     - (bonus) `Accepted password` / `Accepted publickey` como evento crítico
2. **GeoIP local**:
   - Resolver IP → país/ciudad/lat/lon (offline).
   - Cachear resoluciones para no recalcular.
3. **Mapa “pew pew” en tiempo real**:
   - UI (Vue) con modo oscuro por defecto.
   - WebSocket/SSE para feed live.
4. **Top attackers**:
   - Tabla de IPs, intentos, último evento, país, ASN (si se puede offline o con lookup opcional).
5. **Acción manual: Ban IP**:
   - Integración con UFW o iptables/nft.
   - Registrar baneo/desbaneo y razón.
6. **Instalación sin fricción**:
   - Releases con binarios (linux-amd64, linux-arm64).
   - Script de instalación + systemd unit opcional.

### Fase 2 (para usuarios fieles)
7. **Docker-aware (selección de contenedores)**:
   - Detectar Docker y permitir elegir “monitorear logs del contenedor X”.
   - Regla inicial incluida: MySQL “Access denied” (muy útil si MySQL está expuesto por docker-proxy). [cite:62]
8. **Nginx scan detection (modo eficiente)**:
   - Detectar rutas típicas de escaneo (/.env, /wp-login.php, /phpmyadmin).
   - Enfatizar performance: muestreo, rate-limit, o agregación por ventana de tiempo.

### Fase 3 (pro)
9. **Auto-ban (fail2ban-like)** con reglas configurables.
10. **Notificaciones** (Discord/Slack/Telegram).
11. **FIM (integridad de archivos)** opcional.
12. **Multi-nodo** (agente liviano / envío a central).

---

## 3) Arquitectura técnica

### 3.1 Monolito liviano (recomendado)
Un solo binario Go que:
- Sirve HTTP (API + UI embebida)
- Ejecuta tailers/parsers
- Persiste en SQLite
- Publica eventos en tiempo real

Inspirarse en el enfoque “single executable + UI embebida” y stack Go + SQLite + dashboard, como referencia de patrón. [cite:4]

### 3.2 Módulos (Clean Architecture práctica)

Propuesta de carpetas:

pewpew/
cmd/pewpew/ # main (cobra o flags stdlib)
internal/
app/ # wiring: init DB, init tailers, init http
domain/
event/ # entidades: SecurityEvent, BanAction, Source
rule/ # reglas: matcher, thresholds
usecase/
ingest/ # ingesta de líneas -> eventos
query/ # consultas para UI
ban/ # ban/unban
infra/
tail/ # tailer (fsnotify o polling liviano)
parse/ # parsers por fuente (ssh, nginx, docker)
geoip/ # resolver offline + cache
store/sqlite/ # repos + migraciones
firewall/ # ufw/iptables/nft backends
docker/ # leer logs docker (opcional fase 2)
transport/http/ # handlers REST, WS/SSE
web/ # Vue app


---

## 4) Empaquetado del frontend (Vue dentro de Go)

### 4.1 Build Vue
- `web/` con Vue + Vite.
- `npm run build` genera `web/dist/`.

### 4.2 Embed en Go
- Usar `embed.FS` para incluir `web/dist/**` dentro del binario.
- Servir estáticos con `http.FileServer`.
- Si es SPA (Vue router history), agregar fallback a `index.html` para rutas no encontradas.

### 4.3 Resultado
- Producción: ejecutar `./pewpew` y acceder al dashboard en `http://127.0.0.1:9090`.
- Para internet: Nginx reverse proxy (ver sección 6).

---

## 5) Autodetección (core de “zero-config”)

### 5.1 Detección de OS / logs
- Detectar si systemd existe (`/run/systemd/system`).
- Rutas candidatas SSH:
  - Debian/Ubuntu: `/var/log/auth.log`
  - RHEL/CentOS: `/var/log/secure`
  - Si journald: `journalctl -u ssh` (modo futuro; MVP puede priorizar archivo)
- Validar que el archivo exista y que se pueda leer; si no, mostrar warning en UI con pasos.

### 5.2 Detección Docker (fase 2)
- Detectar socket `/var/run/docker.sock`.
- Listar contenedores y permitir “Enable monitoring”.

### 5.3 Detección firewall
- Si existe `ufw`, usarlo como backend preferido.
- Si no, intentar iptables/nft (según disponibilidad).
- En UI: mostrar “Firewall backend: UFW/iptables/none”.

---

## 6) Despliegue detrás de Nginx (cómo se accede “desde la web”)

### 6.1 Proceso recomendado
- pewpew escucha solo local: `127.0.0.1:9090`.
- Nginx recibe HTTPS (443) y hace proxy a pewpew.

### 6.2 Config Nginx ejemplo
- server_name: `pewpew.tudominio.com`
- proxy_pass a `http://127.0.0.1:9090`
- habilitar WebSocket si aplica (`/ws` o similar).

### 6.3 Seguridad
- Autenticación en pewpew:
  - MVP: login simple (usuario/password en SQLite) + cookie/JWT.
- Adicional: basic auth en Nginx opcional.

---

## 7) Data model (SQLite)

Tablas mínimas:
- `events`:
  - id, ts, source (ssh/nginx/docker), event_type, ip, username (opcional), port (opcional),
    message (raw o compact), country_code, city, lat, lon
- `ip_stats` (o vista agregada):
  - ip, count_total, last_seen, country_code, banned(bool)
- `bans`:
  - ip, ts, backend (ufw/iptables), reason, expires_at (opcional), created_by

Retención:
- Mantener eventos crudos por 7–30 días (configurable).
- Agregar “rollups” para histórico sin inflar DB.

---

## 8) Ingesta eficiente (tail + parsing)

### 8.1 Tail strategy
- Seguir el archivo (like tail -F):
  - tolerar rotación de logs
  - reabrir archivo al rotar
- No cargar archivos completos.
- Procesamiento en pipeline:
  - reader -> parser -> enrichment(geoip cache) -> store(batch) -> realtime bus

### 8.2 Parsing (SSH MVP)
- Regex/strings para extraer IP y user:
  - Failed password for (invalid user)? <user> from <ip> port <p>
  - Invalid user <user> from <ip>
  - Accepted password/publickey for <user> from <ip> port <p>  (evento crítico)

### 8.3 Backpressure
- Canal con buffer limitado.
- Si UI no consume WS, el sistema sigue almacenando, pero no se “infla” en RAM.
- Batching de inserts (p. ej. cada 250ms o cada N eventos).

---

## 9) Ban IP (acción) — diseño seguro

### 9.1 Requisitos
- Ban manual desde UI.
- Registrar toda acción (audit log).
- Mostrar estado: banned/unbanned.

### 9.2 Privilegios (importante)
Leer `/var/log/auth.log` y ejecutar firewall normalmente requiere privilegios elevados.
Opciones:
- (A) Ejecutar pewpew como root (más simple, menos seguro).
- (B) Ejecutar como usuario `pewpew` y permitir acciones específicas vía sudoers:
  - Permitir solo `ufw deny from X` y `ufw delete ...` o scripts internos con validación.
Recomendación: B por defecto; A como “quick start” opcional.

### 9.3 Validación
- Validar que IP sea IPv4/IPv6 válida.
- Prevenir command injection (nunca concatenar strings sin validar).

---

## 10) UI/UX (lo que vende en GitHub)

Pantallas MVP:
1. **Live Map** (dark mode, puntos/lineas, contador live).
2. **Feed en vivo** (últimos eventos, filtro por tipo).
3. **Top Attackers** (tabla + botón BAN + búsqueda).
4. **Bans** (lista de baneos, desbanear).
5. **Status** (qué logs detectó, permisos, backend firewall, versión).

Detalles “hacker/devtool”:
- Shortcuts de teclado (K para search, / para focus search).
- Paleta oscura con acento rojo/verde.
- Export CSV (opcional).

---

## 11) Entregables para “éxito en GitHub”

### 11.1 README que vende
- Encabezado con tagline + screenshot + GIF corto del mapa.
- “Install in 30 seconds”.
- “Runs in < XX MB RAM” (medido, no inventado).
- “How it works” 5 líneas.
- Security notes (bind local + proxy recommended).

### 11.2 Releases y distribución
- GitHub Releases con:
  - linux-amd64, linux-arm64
  - checksums
- Script `install.sh`:
  - descarga, chmod +x, crea usuario, crea systemd unit, arranca servicio.

### 11.3 Licencia
- MIT.

### 11.4 Issues y templates
- Bug report (OS, distro, log path)
- Feature request
- Security policy básica

---

## 12) Métricas y pruebas (para no romper VPS)

### 12.1 Tests
- Parsers: unit tests con líneas sample reales (ssh).
- Store: migraciones y queries.
- Firewall: tests con mock.

### 12.2 Bench
- Bench parser regex vs strings.Contains + split.
- Bench batch inserts.

### 12.3 Observabilidad interna
- Endpoint local `/debug` (opcional) con:
  - backlog de colas
  - eventos procesados/seg
  - latencia geoip
  - tamaño DB

---

## 13) Checklist final del MVP

- [ ] CLI `pewpew start` / `pewpew version`
- [ ] Autodetect SSH log path y lectura sin reventar RAM
- [ ] Parsing eventos SSH + extracción IP/user/port
- [ ] SQLite schema + migraciones + retención básica
- [ ] GeoIP offline + cache
- [ ] Web UI embebida (Vue build -> embed)
- [ ] WebSocket/SSE para feed live
- [ ] Live map + tabla top attackers + feed
- [ ] Ban/unban IP (UFW backend mínimo)
- [ ] Nginx reverse proxy doc
- [ ] systemd unit + install script
- [ ] README con GIF + screenshots + quickstart
- [ ] Release linux-amd64/arm64 + checksums

---

## 14) Notas de implementación inmediatas (orden sugerido)

1. Backend Go: HTTP server + `/api/events` + WS/SSE (sin UI aún).
2. Tail SSH + parse + guardar en SQLite + emitir evento live.
3. GeoIP + cache.
4. Vue: mapa + feed (mockeando API), luego integrar real.
5. Embed UI en Go (build pipeline).
6. Ban IP (UFW) + auditoría.
7. Autodetección + pantalla Status.
8. Empaquetado: releases + install script + systemd.

Fin.