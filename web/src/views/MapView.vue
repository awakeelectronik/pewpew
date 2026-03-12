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
import worldCountries from '../data/world-countries.json'

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
const activeAnimations = ref([])
const animationFrameId = ref(null)
const resizeObserver = ref(null)
const lastFrameTime = ref(0)
const loopRunning = ref(false)

const lastFiveEvents = computed(() => store.state.events.slice(0, 5))

function eventTypeClass(eventType) {
  if (!eventType) return 'event-type-other'
  if (eventType === 'failed_password' || eventType === 'invalid_user') return 'event-type-fail'
  if (eventType.includes('accepted')) return 'event-type-accepted'
  return 'event-type-other'
}

function getArcColor(eventType) {
  if (!eventType) return '#38bdf8'
  if (eventType === 'failed_password' || eventType === 'invalid_user') return '#ff5454'
  if (eventType.includes('accepted')) return '#facc15'
  return '#38bdf8'
}

function isAcceptedEvent(eventType) {
  return eventType && String(eventType).includes('accepted')
}

function formatRelative(timestamp) {
  if (!timestamp) return '—'
  const sec = (Date.now() - new Date(timestamp)) / 1000
  if (sec < 60) return `hace ${Math.round(sec)}s`
  if (sec < 3600) return `hace ${Math.round(sec / 60)}m`
  if (sec < 86400) return `hace ${Math.round(sec / 3600)}h`
  return `hace ${Math.round(sec / 86400)}d`
}

/**
 * Web Mercator projection: converts lat/lon to pixel coordinates
 * This is the standard projection used by Google Maps, OpenStreetMap, etc.
 */
function latLonToMercator(lat, lon, canvasW, canvasH) {
  // Clamp latitude to valid Mercator range
  const maxLat = 85.051129
  const clampedLat = Math.max(-maxLat, Math.min(maxLat, lat))

  // Convert to radians
  const latRad = (clampedLat * Math.PI) / 180
  const lonRad = (lon * Math.PI) / 180

  // Web Mercator formula
  const x = ((lon + 180) / 360) * canvasW
  const y = ((Math.PI - Math.log(Math.tan(Math.PI / 4 + latRad / 2))) / (2 * Math.PI)) * canvasH

  return { x, y }
}

/**
 * Draw country polygons on the map
 */
function drawCountries(ctx, width, height) {
  if (!worldCountries || !worldCountries.features) return

  ctx.fillStyle = '#1e293b'
  ctx.strokeStyle = '#334155'
  ctx.lineWidth = 0.5

  worldCountries.features.forEach((feature) => {
    if (feature.geometry.type === 'Polygon') {
      const rings = feature.geometry.coordinates
      drawPolygon(ctx, rings, width, height)
    }
  })
}

/**
 * Draw a single polygon with Mercator projection
 */
function drawPolygon(ctx, rings, width, height) {
  rings.forEach((ring, ringIdx) => {
    ctx.beginPath()
    let isFirst = true

    for (let i = 0; i < ring.length; i++) {
      const [lon, lat] = ring[i]
      const { x, y } = latLonToMercator(lat, lon, width, height)

      if (isFirst) {
        ctx.moveTo(x, y)
        isFirst = false
      } else {
        ctx.lineTo(x, y)
      }
    }

    ctx.closePath()

    if (ringIdx === 0) {
      ctx.fill()
    }
    ctx.stroke()
  })
}

/**
 * Cubic Bézier curve point calculation
 */
function cubicBezierPoint(t, x0, y0, x1, y1, x2, y2, x3, y3) {
  const u = 1 - t
  const u2 = u * u
  const u3 = u2 * u
  const t2 = t * t
  const t3 = t2 * t
  const x = u3 * x0 + 3 * u2 * t * x1 + 3 * u * t2 * x2 + t3 * x3
  const y = u3 * y0 + 3 * u2 * t * y1 + 3 * u * t2 * y2 + t3 * y3
  return { x, y }
}

/**
 * Sample a cubic Bézier curve into discrete points
 */
function sampleCubicBezier(x0, y0, x1, y1, x2, y2, x3, y3, n) {
  const points = []
  for (let i = 0; i <= n; i++) {
    const t = i / n
    points.push(cubicBezierPoint(t, x0, y0, x1, y1, x2, y2, x3, y3))
  }
  return points
}

/**
 * Draw arc from origin to center with progressive animation
 */
function drawArcProgressive(ctx, startX, startY, endX, endY, progress, color) {
  // Control points for the Bézier curve
  // The arc peaks above the start/end points
  const ctrlY = Math.min(startY, endY) - Math.min(80, Math.abs(startX - endX) * 0.25)
  const ctrl1X = startX + (endX - startX) * 0.25
  const ctrl1Y = ctrlY
  const ctrl2X = startX + (endX - startX) * 0.75
  const ctrl2Y = ctrlY

  const points = sampleCubicBezier(startX, startY, ctrl1X, ctrl1Y, ctrl2X, ctrl2Y, endX, endY, BEZIER_SAMPLES)
  const lastIdx = Math.floor(progress * BEZIER_SAMPLES)

  if (lastIdx <= 0) return

  ctx.strokeStyle = color
  ctx.lineWidth = 2.5
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  ctx.beginPath()
  ctx.moveTo(points[0].x, points[0].y)

  for (let i = 1; i <= Math.min(lastIdx, points.length - 1); i++) {
    ctx.lineTo(points[i].x, points[i].y)
  }

  ctx.stroke()
}

/**
 * Enqueue a new animation when an attack event arrives
 */
function enqueueAnimation(event) {
  const lat = event.latitude
  const lon = event.longitude

  if (lat == null || lon == null || Number.isNaN(lat) || Number.isNaN(lon)) return

  const w = canvasWidth.value
  const h = canvasHeight.value
  const start = latLonToMercator(lat, lon, w, h)

  if (!start) return

  // Center point of the map (server location)
  const endX = w / 2
  const endY = h / 2

  const list = activeAnimations.value

  // Evict oldest animation if we exceed max
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

  if (!loopRunning.value) {
    startLoop()
  }
}

/**
 * Main animation loop using requestAnimationFrame
 */
function startLoop() {
  loopRunning.value = true
  lastFrameTime.value = performance.now()

  function loop(now) {
    animationFrameId.value = requestAnimationFrame(loop)

    const ctx = canvasEl.value?.getContext('2d')
    if (!ctx || !canvasEl.value) return

    const w = canvasWidth.value
    const h = canvasHeight.value
    const delta = (now - lastFrameTime.value) / 1000
    lastFrameTime.value = now

    // Clear canvas
    ctx.fillStyle = '#0f172a'
    ctx.fillRect(0, 0, w, h)

    // Draw the world map
    drawCountries(ctx, w, h)

    // Draw center point (VPS location)
    ctx.fillStyle = '#38bdf8'
    ctx.beginPath()
    ctx.arc(w / 2, h / 2, 6, 0, Math.PI * 2)
    ctx.fill()
    ctx.strokeStyle = '#0f172a'
    ctx.lineWidth = 2
    ctx.stroke()

    const list = activeAnimations.value
    const nowMs = Date.now()

    // Update animation phases
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

    // Draw arcs
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

    // Draw ripple effects
    list.forEach((a) => {
      if (a.phase === 'ripple' && a.rippleStart != null) {
        const elapsed = nowMs - a.rippleStart
        const t = Math.min(1, elapsed / RIPPLE_DURATION_MS)
        const maxRadius = isAcceptedEvent(a.event_type) ? 28 : 14
        const radius = t * maxRadius
        const opacity = 1 - t

        ctx.globalAlpha = opacity
        ctx.strokeStyle = getArcColor(a.event_type)
        ctx.lineWidth = 2.5
        ctx.beginPath()
        ctx.arc(a.endX, a.endY, radius, 0, Math.PI * 2)
        ctx.stroke()
        ctx.globalAlpha = 1
      }
    })

    // Draw pulsing source dots
    list.forEach((a) => {
      const pulse = 3.5 + Math.sin(nowMs * 0.005) * 2.5
      ctx.fillStyle = getArcColor(a.event_type)
      ctx.globalAlpha = 0.8 + Math.sin(nowMs * 0.004) * 0.2
      ctx.beginPath()
      ctx.arc(a.startX, a.startY, pulse, 0, Math.PI * 2)
      ctx.fill()
      ctx.globalAlpha = 1
    })

    // Stop loop if no active animations
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

    // Debounce large batches
    if (added > 10) {
      prevEventsLength = newLen
      return
    }

    prevEventsLength = newLen

    // Enqueue animations for new events
    for (let i = 0; i < added; i++) {
      const ev = store.state.events[i]
      if (ev && (ev.latitude != null || ev.longitude != null)) {
        enqueueAnimation(ev)
      }
    }
  }
)

onMounted(async () => {
  await store.fetchEvents(500)
  prevEventsLength = store.state.events.length

  // Setup canvas sizing with ResizeObserver
  if (canvasWrapper.value && canvasEl.value) {
    const setSize = () => {
      if (!canvasWrapper.value) return

      const rect = canvasWrapper.value.getBoundingClientRect()
      const w = Math.max(400, Math.floor(rect.width))
      const h = Math.max(300, Math.floor(rect.height))

      canvasWidth.value = w
      canvasHeight.value = h

      if (activeAnimations.value.length > 0 && !loopRunning.value) {
        startLoop()
      }
    }

    setSize()

    resizeObserver.value = new ResizeObserver(setSize)
    resizeObserver.value.observe(canvasWrapper.value)
  }

  // Initial draw
  if (activeAnimations.value.length === 0) {
    const ctx = canvasEl.value?.getContext('2d')
    if (ctx && canvasEl.value) {
      const w = canvasWidth.value
      const h = canvasHeight.value

      ctx.fillStyle = '#0f172a'
      ctx.fillRect(0, 0, w, h)

      drawCountries(ctx, w, h)

      ctx.fillStyle = '#38bdf8'
      ctx.beginPath()
      ctx.arc(w / 2, h / 2, 6, 0, Math.PI * 2)
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
  font-size: 1.5rem;
  margin: 0;
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
  transition: all 0.2s ease;
}

.stat.connected {
  border-color: #22c55e;
  color: #22c55e;
}

.canvas-wrapper {
  width: 100%;
  min-height: 400px;
  margin-bottom: 1rem;
  border-radius: 4px;
  border: 1px solid #334155;
  overflow: hidden;
  background: #0f172a;
  display: flex;
  align-items: center;
  justify-content: center;
}

.attack-map {
  display: block;
  width: 100%;
  height: 100%;
  max-width: 100%;
}

.last-events {
  margin-top: 1.5rem;
}

.last-events h3 {
  font-size: 0.875rem;
  color: #94a3b8;
  margin: 0 0 0.75rem 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.event-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.event-row {
  display: grid;
  grid-template-columns: 1fr 1fr auto auto;
  gap: 1rem;
  align-items: center;
  padding: 0.5rem 0.75rem;
  font-size: 0.8125rem;
  font-family: 'Courier New', monospace;
  background: #0f172a;
  border-radius: 4px;
  border: 1px solid #334155;
  transition: all 0.2s ease;
}

.event-row:hover {
  border-color: #38bdf8;
  background: #1e293b;
}

.event-ip {
  color: #38bdf8;
  font-weight: 600;
}

.event-country {
  color: #22c55e;
  font-weight: 500;
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
  text-align: right;
}

.event-list-empty {
  color: #64748b;
  font-size: 0.875rem;
  margin: 0;
  padding: 1rem;
  text-align: center;
}
</style>
