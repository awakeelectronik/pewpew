<template>
  <div class="map-container">
    <div class="map-header">
      <h2>Live Attack Map</h2>
      <div class="stats">
        <span class="stat">
          <strong>{{ store.state.events.length }}</strong> Events
        </span>
        <span class="stat">
          <strong>{{ store.state.topAttackers.length }}</strong> Unique IPs
        </span>
        <span :class="['stat', { connected: store.state.wsConnected }]">
          {{ store.state.wsConnected ? 'Live' : 'Offline' }}
        </span>
      </div>
    </div>

    <div ref="canvasWrapper" class="canvas-wrapper">
      <canvas
        ref="canvasEl"
        class="attack-map"
        :width="canvasWidth"
        :height="canvasHeight"
      />
    </div>

    <div class="last-events">
      <h3>Last 5 events</h3>
      <ul v-if="lastFiveEvents.length" class="event-list">
        <li
          v-for="(ev, i) in lastFiveEvents"
          :key="ev.id || `${ev.timestamp}-${ev.ip}-${i}`"
          class="event-row"
        >
          <span class="event-ip">{{ ev.ip }}</span>
          <span class="event-country">{{ ev.country || '—' }}</span>
          <span :class="['event-type', eventTypeClass(ev.event_type)]">{{ ev.event_type }}</span>
          <span class="event-ago">{{ formatRelative(ev.timestamp) }}</span>
        </li>
      </ul>
      <p v-else class="event-list-empty">No events yet.</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useStore } from '../stores/store.js'

const BEZIER_SAMPLES = 50
const MAX_ANIMATIONS = 30
const ARC_DURATION_MS = 600
const RIPPLE_DURATION_MS = 400
const FADEOUT_AFTER_MS = 3500
const FADEOUT_DURATION_MS = 500

const store = useStore()
const canvasEl = ref(null)
const canvasWrapper = ref(null)
const canvasWidth = ref(1200)
const canvasHeight = ref(600)
const worldImage = ref(null)
const activeAnimations = ref([])
const animationFrameId = ref(null)
const resizeObserver = ref(null)
const lastFrameTime = ref(0)
const loopRunning = ref(false)

const lastFiveEvents = computed(() => store.state.events.slice(0, 5))

function eventTypeClass (eventType) {
  if (!eventType) return 'event-type-other'
  if (eventType === 'failed_password' || eventType === 'invalid_user') return 'event-type-fail'
  if (eventType.includes('accepted')) return 'event-type-accepted'
  return 'event-type-other'
}

function getArcColor (eventType) {
  if (!eventType) return '#38bdf8'
  if (eventType === 'failed_password' || eventType === 'invalid_user') return '#ff5454'
  if (eventType.includes('accepted')) return '#facc15'
  return '#38bdf8'
}

function isAcceptedEvent (eventType) {
  return eventType && String(eventType).includes('accepted')
}

function formatRelative (timestamp) {
  if (!timestamp) return '—'
  const sec = (Date.now() - new Date(timestamp)) / 1000
  if (sec < 60) return `hace ${Math.round(sec)}s`
  if (sec < 3600) return `hace ${Math.round(sec / 60)}m`
  if (sec < 86400) return `hace ${Math.round(sec / 3600)}h`
  return `hace ${Math.round(sec / 86400)}d`
}

function latLonToMercator (lat, lon, w, h) {
  if (lat == null || lon == null || Number.isNaN(lat) || Number.isNaN(lon)) return null
  const latClamp = Math.max(-85, Math.min(85, lat))
  const x = ((lon + 180) / 360) * w
  const latRad = (latClamp * Math.PI) / 180
  const y = (1 - (Math.log(Math.tan(latRad) + 1 / Math.cos(latRad)) / Math.PI)) / 2 * h
  return { x, y }
}

// Mapa mundial equirectangular: viewBox 0 0 360 180
// x = lon+180 (0=180°O, 360=180°E), y = 90-lat (0=polo N, 180=polo S)
function getWorldSvgString () {
  const fill = '#1e293b'
  const stroke = '#334155'
  const strokeW = '0.4'
  const paths = [
    // Norteamérica (Alaska a México)
    'M 12,28 L 48,22 L 72,25 L 95,32 L 108,42 L 118,55 L 115,68 L 95,78 L 72,82 L 52,78 L 38,68 L 28,52 L 22,38 Z',
    // Groenlandia
    'M 288,8 L 318,5 L 332,18 L 328,35 L 308,42 L 292,35 Z',
    // Sudamérica
    'M 98,95 L 118,88 L 138,92 L 148,108 L 152,128 L 148,148 L 132,158 L 112,155 L 98,138 L 95,115 Z',
    // Europa
    'M 158,52 L 178,48 L 198,52 L 212,58 L 218,68 L 212,78 L 195,82 L 175,78 L 162,68 Z',
    'M 332,48 L 348,45 L 358,52 L 358,65 L 348,72 L 335,68 L 328,58 Z',
    // África
    'M 162,55 L 192,52 L 218,58 L 232,72 L 235,95 L 228,118 L 212,132 L 192,135 L 175,128 L 168,108 L 168,82 Z',
    'M 338,72 L 352,68 L 358,82 L 358,108 L 348,122 L 332,118 L 328,95 Z',
    // Asia (Rusia + Siberia)
    'M 218,22 L 258,18 L 298,22 L 332,28 L 355,35 L 358,48 L 352,62 L 328,72 L 298,75 L 268,72 L 242,65 L 225,55 Z',
    'M 0,32 L 28,28 L 52,35 L 62,52 L 58,72 L 42,82 L 22,78 L 5,62 L 0,45 Z',
    // India
    'M 268,82 L 288,78 L 302,88 L 302,105 L 288,118 L 268,115 L 258,98 Z',
    // Sudeste asiático + Indonesia
    'M 268,108 L 288,105 L 298,115 L 292,128 L 275,132 L 262,122 Z',
    // Australia
    'M 293,122 L 318,118 L 332,128 L 335,142 L 322,152 L 298,155 L 282,148 L 278,132 Z',
    // Japón
    'M 272,48 L 288,45 L 298,52 L 295,62 L 282,68 L 272,62 Z',
    // Islas Británicas
    'M 328,52 L 338,50 L 342,58 L 338,65 L 328,62 Z'
  ]
  const d = paths.join(' ')
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 360 180"><path fill="${fill}" stroke="${stroke}" stroke-width="${strokeW}" d="${d}"/></svg>`
}

function loadWorldImage () {
  return new Promise((resolve) => {
    const svg = getWorldSvgString()
    const blob = new Blob([svg], { type: 'image/svg+xml;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const img = new Image()
    img.onload = () => {
      URL.revokeObjectURL(url)
      resolve(img)
    }
    img.onerror = () => {
      URL.revokeObjectURL(url)
      resolve(null)
    }
    img.src = url
  })
}

function cubicBezierPoint (t, x0, y0, x1, y1, x2, y2, x3, y3) {
  const u = 1 - t
  const u2 = u * u
  const u3 = u2 * u
  const t2 = t * t
  const t3 = t2 * t
  const x = u3 * x0 + 3 * u2 * t * x1 + 3 * u * t2 * x2 + t3 * x3
  const y = u3 * y0 + 3 * u2 * t * y1 + 3 * u * t2 * y2 + t3 * y3
  return { x, y }
}

function sampleCubicBezier (x0, y0, x1, y1, x2, y2, x3, y3, n) {
  const points = []
  for (let i = 0; i <= n; i++) {
    const t = i / n
    points.push(cubicBezierPoint(t, x0, y0, x1, y1, x2, y2, x3, y3))
  }
  return points
}

function drawArcProgressive (ctx, startX, startY, endX, endY, progress, color) {
  const ctrlY = Math.min(startY, endY) - Math.min(60, Math.abs(startX - endX) * 0.2)
  const ctrl1X = startX + (endX - startX) * 0.25
  const ctrl1Y = ctrlY
  const ctrl2X = startX + (endX - startX) * 0.75
  const ctrl2Y = ctrlY
  const points = sampleCubicBezier(startX, startY, ctrl1X, ctrl1Y, ctrl2X, ctrl2Y, endX, endY, BEZIER_SAMPLES)
  const lastIdx = Math.floor(progress * BEZIER_SAMPLES)
  if (lastIdx <= 0) return
  ctx.strokeStyle = color
  ctx.lineWidth = 2
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  ctx.beginPath()
  ctx.moveTo(points[0].x, points[0].y)
  for (let i = 1; i <= lastIdx; i++) {
    ctx.lineTo(points[i].x, points[i].y)
  }
  ctx.stroke()
}

function enqueueAnimation (event) {
  const lat = event.latitude
  const lon = event.longitude
  if (lat == null || lon == null || Number.isNaN(lat) || Number.isNaN(lon)) return
  const w = canvasWidth.value
  const h = canvasHeight.value
  const start = latLonToMercator(lat, lon, w, h)
  if (!start) return
  const endX = w / 2
  const endY = h / 2
  const list = activeAnimations.value
  if (list.length >= MAX_ANIMATIONS) {
    list.sort((a, b) => a.createdAt - b.createdAt)
    list.shift()
  }
  list.push({
    id: `${Date.now()}-${Math.random()}`,
    startX: start.x,
    startY: start.y,
    endX,
    endY,
    progress: 0,
    phase: 'arc',
    createdAt: Date.now(),
    opacity: 1,
    rippleStart: null,
    event_type: event.event_type,
    event
  })
  if (!loopRunning.value) startLoop()
}

function startLoop () {
  loopRunning.value = true
  lastFrameTime.value = performance.now()
  function loop (now) {
    animationFrameId.value = requestAnimationFrame(loop)
    const ctx = canvasEl.value?.getContext('2d')
    if (!ctx || !canvasEl.value) return
    const w = canvasWidth.value
    const h = canvasHeight.value
    const delta = (now - lastFrameTime.value) / 1000
    lastFrameTime.value = now

    ctx.fillStyle = '#0f172a'
    ctx.fillRect(0, 0, w, h)

    const img = worldImage.value
    if (img) {
      ctx.globalAlpha = 0.9
      ctx.drawImage(img, 0, 0, w, h)
      ctx.globalAlpha = 1
    }

    ctx.fillStyle = '#38bdf8'
    ctx.beginPath()
    ctx.arc(w / 2, h / 2, 5, 0, Math.PI * 2)
    ctx.fill()
    ctx.strokeStyle = '#0f172a'
    ctx.lineWidth = 2
    ctx.stroke()

    const list = activeAnimations.value
    const nowMs = Date.now()

    for (let i = list.length - 1; i >= 0; i--) {
      const a = list[i]
      const age = nowMs - a.createdAt
      if (age > FADEOUT_AFTER_MS) {
        list.splice(i, 1)
        continue
      }

      if (a.phase === 'arc') {
        a.progress += delta / (ARC_DURATION_MS / 1000)
        if (a.progress >= 1) {
          a.progress = 1
          a.phase = 'ripple'
          a.rippleStart = nowMs
        }
      } else if (a.phase === 'ripple') {
        const rippleElapsed = nowMs - a.rippleStart
        if (rippleElapsed > RIPPLE_DURATION_MS) {
          a.phase = 'fadeout'
        }
      } else if (a.phase === 'fadeout') {
        const fadeElapsed = age - (FADEOUT_AFTER_MS - FADEOUT_DURATION_MS)
        a.opacity = Math.max(0, 1 - fadeElapsed / FADEOUT_DURATION_MS)
        if (a.opacity <= 0) {
          list.splice(i, 1)
          continue
        }
      }
    }

    list.forEach((a) => {
      const color = getArcColor(a.event_type)
      if (a.phase === 'arc') {
        drawArcProgressive(ctx, a.startX, a.startY, a.endX, a.endY, a.progress, color)
      } else if (a.phase === 'fadeout' && a.opacity > 0) {
        ctx.globalAlpha = a.opacity
        drawArcProgressive(ctx, a.startX, a.startY, a.endX, a.endY, 1, color)
        ctx.globalAlpha = 1
      }
    })

    list.forEach((a) => {
      if (a.phase === 'ripple' && a.rippleStart != null) {
        const elapsed = nowMs - a.rippleStart
        const t = Math.min(1, elapsed / RIPPLE_DURATION_MS)
        const maxRadius = isAcceptedEvent(a.event_type) ? 24 : 12
        const radius = t * maxRadius
        const opacity = 1 - t
        ctx.globalAlpha = opacity
        ctx.strokeStyle = getArcColor(a.event_type)
        ctx.lineWidth = 2
        ctx.beginPath()
        ctx.arc(a.endX, a.endY, radius, 0, Math.PI * 2)
        ctx.stroke()
        ctx.globalAlpha = 1
      }
    })

    list.forEach((a) => {
      const pulse = 3 + Math.sin(nowMs * 0.005) * 2
      ctx.fillStyle = getArcColor(a.event_type)
      ctx.globalAlpha = 0.85 + Math.sin(nowMs * 0.004) * 0.15
      ctx.beginPath()
      ctx.arc(a.startX, a.startY, pulse, 0, Math.PI * 2)
      ctx.fill()
      ctx.globalAlpha = 1
    })

    if (list.length === 0) {
      loopRunning.value = false
      cancelAnimationFrame(animationFrameId.value)
      animationFrameId.value = null
    }
  }
  animationFrameId.value = requestAnimationFrame(loop)
}

let prevEventsLength = 0
watch(
  () => store.state.events.length,
  (newLen, oldLen) => {
    if (oldLen === undefined) {
      prevEventsLength = newLen
      return
    }
    if (newLen <= prevEventsLength) return
    const added = newLen - prevEventsLength
    if (added > 10) {
      prevEventsLength = newLen
      return
    }
    prevEventsLength = newLen
    for (let i = 0; i < added; i++) {
      const ev = store.state.events[i]
      if (ev && (ev.latitude != null || ev.longitude != null)) enqueueAnimation(ev)
    }
  }
)

onMounted(async () => {
  await store.fetchEvents(500)
  prevEventsLength = store.state.events.length
  const img = await loadWorldImage()
  worldImage.value = img

  if (canvasWrapper.value && canvasEl.value) {
    const setSize = () => {
      if (!canvasWrapper.value) return
      const rect = canvasWrapper.value.getBoundingClientRect()
      const w = Math.max(300, Math.floor(rect.width))
      const h = Math.max(200, Math.floor(rect.height))
      canvasWidth.value = w
      canvasHeight.value = h
      if (activeAnimations.value.length > 0 && !loopRunning.value) startLoop()
    }
    setSize()
    resizeObserver.value = new ResizeObserver(setSize)
    resizeObserver.value.observe(canvasWrapper.value)
  }

  if (activeAnimations.value.length === 0) {
    const ctx = canvasEl.value?.getContext('2d')
    if (ctx && canvasEl.value) {
      const w = canvasWidth.value
      const h = canvasHeight.value
      ctx.fillStyle = '#0f172a'
      ctx.fillRect(0, 0, w, h)
      if (worldImage.value) {
        ctx.globalAlpha = 0.9
        ctx.drawImage(worldImage.value, 0, 0, w, h)
        ctx.globalAlpha = 1
      }
      ctx.fillStyle = '#38bdf8'
      ctx.beginPath()
      ctx.arc(w / 2, h / 2, 5, 0, Math.PI * 2)
      ctx.fill()
      ctx.strokeStyle = '#0f172a'
      ctx.lineWidth = 2
      ctx.stroke()
    }
  }
})

onUnmounted(() => {
  if (animationFrameId.value != null) {
    cancelAnimationFrame(animationFrameId.value)
    animationFrameId.value = null
  }
  loopRunning.value = false
  if (resizeObserver.value && canvasWrapper.value) {
    resizeObserver.value.disconnect()
    resizeObserver.value = null
  }
})
</script>

<style scoped>
.map-container {
  background: var(--bg-secondary, #0f172a);
  border-radius: 8px;
  padding: 1.5rem;
  border: 1px solid #334155;
}

.map-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.map-header h2 {
  color: #38bdf8;
}

.stats {
  display: flex;
  gap: 1.5rem;
}

.stat {
  font-size: 0.875rem;
  color: #cbd5e1;
  background: #0f172a;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  border: 1px solid #334155;
}

.stat.connected {
  border-color: #22c55e;
}

.canvas-wrapper {
  width: 100%;
  min-height: 400px;
  margin-bottom: 1rem;
  border-radius: 4px;
  border: 1px solid #334155;
  overflow: hidden;
  background: #0f172a;
}

.attack-map {
  display: block;
  width: 100%;
  height: 100%;
  max-width: 100%;
}

.last-events {
  margin-top: 1rem;
}

.last-events h3 {
  font-size: 0.875rem;
  color: #94a3b8;
  margin-bottom: 0.5rem;
}

.event-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.event-row {
  display: grid;
  grid-template-columns: 1fr 1fr auto auto;
  gap: 1rem;
  align-items: center;
  padding: 0.5rem 0.75rem;
  font-size: 0.8125rem;
  font-family: monospace;
  background: #0f172a;
  border-radius: 4px;
  border: 1px solid #334155;
  margin-bottom: 0.35rem;
}

.event-ip {
  color: #38bdf8;
}

.event-country {
  color: #22c55e;
}

.event-type-fail {
  color: #ff5454;
  font-weight: bold;
}

.event-type-accepted {
  color: #facc15;
  font-weight: bold;
}

.event-type-other {
  color: #94a3b8;
}

.event-ago {
  color: #64748b;
  font-size: 0.75rem;
}

.event-list-empty {
  color: #64748b;
  font-size: 0.875rem;
  margin: 0;
}
</style>
