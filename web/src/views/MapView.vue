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
        <span v-if="orphanCount > 0" class="stat" title="Eventos sin geolocalización">
          <strong>{{ orphanCount }}</strong> Sin geo
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
        @wheel.prevent="onWheel"
        @mousedown="onMouseDown"
        @mousemove="onMouseMove"
        @mouseup="onMouseUp"
        @mouseleave="onMouseLeave"
        :style="{ cursor: isDragging ? 'grabbing' : (tooltip.visible ? 'pointer' : 'grab') }"
      />
      <div v-if="tooltip.visible" class="map-tooltip" :style="tooltipStyle">
        <div class="tooltip-ip">{{ tooltip.ip }}</div>
        <div class="tooltip-row">
          <span class="tooltip-label">País</span>
          <span>{{ tooltip.country }}</span>
        </div>
        <div class="tooltip-row">
          <span class="tooltip-label">Tipo</span>
          <span :class="eventTypeClass(tooltip.eventType)">{{ tooltip.eventType }}</span>
        </div>
        <div class="tooltip-row">
          <span class="tooltip-label">Hora</span>
          <span>{{ tooltip.time }}</span>
        </div>
      </div>
      <div class="zoom-controls">
        <button @click="zoomIn" title="Acercar">+</button>
        <button @click="resetView" title="Restablecer vista">⟳</button>
        <button @click="zoomOut" title="Alejar">−</button>
      </div>
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
import { ref, reactive, computed, watch, onMounted, onUnmounted } from 'vue'
import { useStore } from '../stores/store.js'
import { feature } from 'topojson-client'
import topology from 'world-atlas/countries-110m.json'
import axios from 'axios'

const worldCountries = feature(topology, topology.objects.countries)

const BEZIER_SAMPLES = 50
const MAX_ANIMATIONS = 30
const ARC_DURATION_MS = 600
const RIPPLE_DURATION_MS = 400
const FADEOUT_AFTER_MS = 3500
const FADEOUT_DURATION_MS = 500
const MIN_SCALE = 1
const MAX_SCALE = 12

const store = useStore()
const vpsLat = ref(0)
const vpsLon = ref(0)
const canvasEl = ref(null)
const canvasWrapper = ref(null)
const canvasWidth = ref(1200)
const canvasHeight = ref(600)
const activeAnimations = ref([])
const animationFrameId = ref(null)
const resizeObserver = ref(null)
const lastFrameTime = ref(0)
const loopRunning = ref(false)
const orphanCount = ref(0)

const viewScale = ref(1)
const viewPanX = ref(0)
const viewPanY = ref(0)

const isDragging = ref(false)
const dragStartX = ref(0)
const dragStartY = ref(0)
const dragStartPanX = ref(0)
const dragStartPanY = ref(0)

const tooltip = reactive({
  visible: false,
  x: 0,
  y: 0,
  ip: '',
  country: '',
  eventType: '',
  time: ''
})

const pendingRedraw = ref(false)

const lastFiveEvents = computed(() => store.state.events.slice(0, 5))

const tooltipStyle = computed(() => ({
  left: tooltip.x + 'px',
  top: tooltip.y + 'px'
}))

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

function latLonToMercator(lat, lon, canvasW, canvasH) {
  const maxLat = 85.051129
  const clampedLat = Math.max(-maxLat, Math.min(maxLat, lat))
  const latRad = (clampedLat * Math.PI) / 180
  const x = ((lon + 180) / 360) * canvasW
  const y = ((Math.PI - Math.log(Math.tan(Math.PI / 4 + latRad / 2))) / (2 * Math.PI)) * canvasH
  return { x, y }
}

function vpsScreenPos(w, h) {
  return latLonToMercator(vpsLat.value, vpsLon.value, w, h)
}

// --- Mouse coordinate helpers ---

function getMousePos(e) {
  const canvas = canvasEl.value
  if (!canvas) return { canvasX: 0, canvasY: 0, worldX: 0, worldY: 0 }
  const rect = canvas.getBoundingClientRect()
  const sx = canvas.width / rect.width
  const sy = canvas.height / rect.height
  const canvasX = (e.clientX - rect.left) * sx
  const canvasY = (e.clientY - rect.top) * sy
  const worldX = (canvasX - viewPanX.value) / viewScale.value
  const worldY = (canvasY - viewPanY.value) / viewScale.value
  return { canvasX, canvasY, worldX, worldY }
}

function clampPan() {
  const w = canvasWidth.value
  const h = canvasHeight.value
  const s = viewScale.value
  viewPanX.value = Math.min(0, Math.max(w * (1 - s), viewPanX.value))
  viewPanY.value = Math.min(0, Math.max(h * (1 - s), viewPanY.value))
}

function requestRedraw() {
  if (!loopRunning.value && !pendingRedraw.value) {
    pendingRedraw.value = true
    requestAnimationFrame(() => {
      pendingRedraw.value = false
      if (!loopRunning.value) redrawStatic()
    })
  }
}

// --- Zoom & Pan ---

function onWheel(e) {
  const { canvasX, canvasY } = getMousePos(e)
  const factor = e.deltaY < 0 ? 1.15 : 1 / 1.15
  const newScale = Math.max(MIN_SCALE, Math.min(MAX_SCALE, viewScale.value * factor))
  const ratio = newScale / viewScale.value
  viewPanX.value = canvasX - (canvasX - viewPanX.value) * ratio
  viewPanY.value = canvasY - (canvasY - viewPanY.value) * ratio
  viewScale.value = newScale
  clampPan()
  requestRedraw()
}

function onMouseDown(e) {
  if (e.button !== 0) return
  isDragging.value = true
  const { canvasX, canvasY } = getMousePos(e)
  dragStartX.value = canvasX
  dragStartY.value = canvasY
  dragStartPanX.value = viewPanX.value
  dragStartPanY.value = viewPanY.value
  tooltip.visible = false
}

function onMouseMove(e) {
  if (isDragging.value) {
    const { canvasX, canvasY } = getMousePos(e)
    viewPanX.value = dragStartPanX.value + (canvasX - dragStartX.value)
    viewPanY.value = dragStartPanY.value + (canvasY - dragStartY.value)
    clampPan()
    requestRedraw()
    return
  }

  const { worldX, worldY } = getMousePos(e)
  const hitRadius = 10 / viewScale.value
  const hitRadiusSq = hitRadius * hitRadius
  let found = null

  for (const anim of activeAnimations.value) {
    const dx = worldX - anim.startX
    const dy = worldY - anim.startY
    if (dx * dx + dy * dy < hitRadiusSq) {
      found = anim
      break
    }
  }

  if (found && found.event) {
    const wrapperRect = canvasWrapper.value.getBoundingClientRect()
    const rawX = e.clientX - wrapperRect.left + 14
    const rawY = e.clientY - wrapperRect.top + 14
    tooltip.visible = true
    tooltip.x = Math.min(rawX, wrapperRect.width - 210)
    tooltip.y = Math.min(rawY, wrapperRect.height - 110)
    tooltip.ip = found.event.ip || '—'
    tooltip.country = found.event.country || '—'
    tooltip.eventType = found.event.event_type || '—'
    tooltip.time = formatRelative(found.event.timestamp)
  } else {
    tooltip.visible = false
  }
}

function onMouseUp() {
  isDragging.value = false
}

function onMouseLeave() {
  isDragging.value = false
  tooltip.visible = false
}

function zoomIn() {
  const cx = canvasWidth.value / 2
  const cy = canvasHeight.value / 2
  const newScale = Math.min(MAX_SCALE, viewScale.value * 1.5)
  const ratio = newScale / viewScale.value
  viewPanX.value = cx - (cx - viewPanX.value) * ratio
  viewPanY.value = cy - (cy - viewPanY.value) * ratio
  viewScale.value = newScale
  clampPan()
  requestRedraw()
}

function zoomOut() {
  const cx = canvasWidth.value / 2
  const cy = canvasHeight.value / 2
  const newScale = Math.max(MIN_SCALE, viewScale.value / 1.5)
  const ratio = newScale / viewScale.value
  viewPanX.value = cx - (cx - viewPanX.value) * ratio
  viewPanY.value = cy - (cy - viewPanY.value) * ratio
  viewScale.value = newScale
  clampPan()
  requestRedraw()
}

function resetView() {
  viewScale.value = 1
  viewPanX.value = 0
  viewPanY.value = 0
  requestRedraw()
}

// --- Drawing ---

function drawGraticule(ctx, w, h) {
  ctx.strokeStyle = 'rgba(51, 65, 85, 0.12)'
  ctx.lineWidth = 0.3

  for (let lat = -60; lat <= 80; lat += 30) {
    ctx.beginPath()
    for (let lon = -180; lon <= 180; lon += 2) {
      const { x, y } = latLonToMercator(lat, lon, w, h)
      if (lon === -180) ctx.moveTo(x, y)
      else ctx.lineTo(x, y)
    }
    ctx.stroke()
  }

  for (let lon = -150; lon <= 180; lon += 30) {
    ctx.beginPath()
    for (let lat = -80; lat <= 80; lat += 2) {
      const { x, y } = latLonToMercator(lat, lon, w, h)
      if (lat === -80) ctx.moveTo(x, y)
      else ctx.lineTo(x, y)
    }
    ctx.stroke()
  }
}

function drawCountries(ctx, width, height) {
  if (!worldCountries || !worldCountries.features) return

  ctx.fillStyle = '#1e293b'
  ctx.strokeStyle = '#334155'
  ctx.lineWidth = 0.5

  worldCountries.features.forEach((feat) => {
    if (feat.id === '010') return

    const { type, coordinates } = feat.geometry
    if (type === 'Polygon') {
      drawPolygon(ctx, coordinates, width, height)
    } else if (type === 'MultiPolygon') {
      coordinates.forEach((polygon) => drawPolygon(ctx, polygon, width, height))
    }
  })
}

function drawPolygon(ctx, rings, width, height) {
  rings.forEach((ring, ringIdx) => {
    ctx.beginPath()
    let prevLon = null

    for (let i = 0; i < ring.length; i++) {
      const [lon, lat] = ring[i]
      const { x, y } = latLonToMercator(lat, lon, width, height)

      if (i === 0) {
        ctx.moveTo(x, y)
      } else if (prevLon !== null && Math.abs(lon - prevLon) > 170) {
        ctx.moveTo(x, y)
      } else {
        ctx.lineTo(x, y)
      }
      prevLon = lon
    }

    ctx.closePath()

    if (ringIdx === 0) {
      ctx.fill()
    }
    ctx.stroke()
  })
}

function drawVPSMarker(ctx, x, y, nowMs) {
  const pulse = 8 + Math.sin((nowMs || Date.now()) * 0.003) * 3
  ctx.beginPath()
  ctx.arc(x, y, pulse, 0, Math.PI * 2)
  ctx.strokeStyle = 'rgba(56, 189, 248, 0.25)'
  ctx.lineWidth = 1.5
  ctx.stroke()

  ctx.beginPath()
  ctx.arc(x, y, 5, 0, Math.PI * 2)
  ctx.fillStyle = '#38bdf8'
  ctx.fill()
  ctx.strokeStyle = '#0f172a'
  ctx.lineWidth = 2
  ctx.stroke()

  ctx.fillStyle = 'rgba(56, 189, 248, 0.7)'
  ctx.font = '10px sans-serif'
  ctx.textAlign = 'center'
  ctx.fillText('VPS', x, y - 14)
}

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

function sampleCubicBezier(x0, y0, x1, y1, x2, y2, x3, y3, n) {
  const points = []
  for (let i = 0; i <= n; i++) {
    const t = i / n
    points.push(cubicBezierPoint(t, x0, y0, x1, y1, x2, y2, x3, y3))
  }
  return points
}

function drawArcProgressive(ctx, startX, startY, endX, endY, progress, color) {
  const ctrlY = Math.min(startY, endY) - Math.min(80, Math.abs(startX - endX) * 0.25)
  const ctrl1X = startX + (endX - startX) * 0.25
  const ctrl1Y = ctrlY
  const ctrl2X = startX + (endX - startX) * 0.75
  const ctrl2Y = ctrlY

  const points = sampleCubicBezier(
    startX, startY, ctrl1X, ctrl1Y, ctrl2X, ctrl2Y, endX, endY, BEZIER_SAMPLES
  )
  const lastIdx = Math.floor(progress * BEZIER_SAMPLES)

  if (lastIdx <= 0) return

  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'

  ctx.beginPath()
  ctx.moveTo(points[0].x, points[0].y)
  for (let i = 1; i <= Math.min(lastIdx, points.length - 1); i++) {
    ctx.lineTo(points[i].x, points[i].y)
  }

  const savedAlpha = ctx.globalAlpha

  ctx.strokeStyle = color
  ctx.lineWidth = 6
  ctx.globalAlpha = savedAlpha * 0.15
  ctx.stroke()

  ctx.globalAlpha = savedAlpha
  ctx.lineWidth = 2
  ctx.stroke()
}

// --- Static redraw (when no animations are running) ---

function redrawStatic() {
  const ctx = canvasEl.value?.getContext('2d')
  if (!ctx) return
  const w = canvasWidth.value
  const h = canvasHeight.value

  ctx.setTransform(1, 0, 0, 1, 0, 0)
  ctx.fillStyle = '#0f172a'
  ctx.fillRect(0, 0, w, h)

  ctx.save()
  ctx.translate(viewPanX.value, viewPanY.value)
  ctx.scale(viewScale.value, viewScale.value)

  drawGraticule(ctx, w, h)
  drawCountries(ctx, w, h)
  const vpsStatic = vpsScreenPos(w, h)
  drawVPSMarker(ctx, vpsStatic.x, vpsStatic.y)

  ctx.restore()
}

// --- Animations ---

function enqueueAnimation(event) {
  const lat = event.latitude
  const lon = event.longitude

  if (lat == null || lon == null || Number.isNaN(lat) || Number.isNaN(lon)) {
    orphanCount.value++
    return
  }

  const w = canvasWidth.value
  const h = canvasHeight.value
  const start = latLonToMercator(lat, lon, w, h)

  if (!start) return

  const vps = vpsScreenPos(w, h)
  const endX = vps.x
  const endY = vps.y

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

  if (!loopRunning.value) {
    startLoop()
  }
}

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

    ctx.setTransform(1, 0, 0, 1, 0, 0)
    ctx.fillStyle = '#0f172a'
    ctx.fillRect(0, 0, w, h)

    ctx.save()
    ctx.translate(viewPanX.value, viewPanY.value)
    ctx.scale(viewScale.value, viewScale.value)

    drawGraticule(ctx, w, h)
    drawCountries(ctx, w, h)

    const nowMs = Date.now()
    const vpsPos = vpsScreenPos(w, h)
    drawVPSMarker(ctx, vpsPos.x, vpsPos.y, nowMs)

    const list = activeAnimations.value

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
        if (nowMs - a.rippleStart > RIPPLE_DURATION_MS) {
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
      } else {
        const alpha = a.phase === 'fadeout' ? a.opacity : 1
        ctx.globalAlpha = alpha
        drawArcProgressive(ctx, a.startX, a.startY, a.endX, a.endY, 1, color)
        ctx.globalAlpha = 1
      }
    })

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

    list.forEach((a) => {
      const pulse = 3.5 + Math.sin(nowMs * 0.005) * 2.5
      ctx.fillStyle = getArcColor(a.event_type)
      ctx.globalAlpha = 0.8 + Math.sin(nowMs * 0.004) * 0.2
      ctx.beginPath()
      ctx.arc(a.startX, a.startY, pulse, 0, Math.PI * 2)
      ctx.fill()
      ctx.globalAlpha = 1
    })

    ctx.restore()

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
      if (ev) enqueueAnimation(ev)
    }
  }
)

function applyCanvasSize() {
  const el = canvasEl.value
  if (!el || !canvasWrapper.value) return

  const rect = canvasWrapper.value.getBoundingClientRect()
  const w = Math.max(400, Math.floor(rect.width))
  const h = Math.max(300, Math.floor(rect.height))

  if (w === canvasWidth.value && h === canvasHeight.value) return

  canvasWidth.value = w
  canvasHeight.value = h
  el.width = w
  el.height = h

  viewScale.value = 1
  viewPanX.value = 0
  viewPanY.value = 0

  if (activeAnimations.value.length > 0 && !loopRunning.value) {
    startLoop()
  } else {
    redrawStatic()
  }
}

async function fetchVPSLocation() {
  try {
    const { data } = await axios.get('/api/status')
    if (data && (data.vps_lat || data.vps_lon)) {
      vpsLat.value = data.vps_lat
      vpsLon.value = data.vps_lon
      redrawStatic()
    }
  } catch (_) { /* VPS location stays at 0,0 until available */ }
}

onMounted(async () => {
  if (canvasWrapper.value && canvasEl.value) {
    applyCanvasSize()

    resizeObserver.value = new ResizeObserver(applyCanvasSize)
    resizeObserver.value.observe(canvasWrapper.value)
  }

  await Promise.all([
    store.fetchEvents(500),
    fetchVPSLocation()
  ])
  prevEventsLength = store.state.events.length
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
  position: relative;
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

/* Tooltip */
.map-tooltip {
  position: absolute;
  background: rgba(15, 23, 42, 0.95);
  border: 1px solid #38bdf8;
  border-radius: 6px;
  padding: 0.6rem 0.8rem;
  pointer-events: none;
  z-index: 10;
  min-width: 160px;
  backdrop-filter: blur(6px);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
}

.tooltip-ip {
  font-family: 'Courier New', monospace;
  font-size: 0.875rem;
  font-weight: 700;
  color: #38bdf8;
  margin-bottom: 0.35rem;
  border-bottom: 1px solid #1e293b;
  padding-bottom: 0.3rem;
}

.tooltip-row {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  font-size: 0.75rem;
  color: #e2e8f0;
  padding: 0.15rem 0;
}

.tooltip-label {
  color: #64748b;
}

/* Zoom controls */
.zoom-controls {
  position: absolute;
  bottom: 12px;
  right: 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  z-index: 5;
}

.zoom-controls button {
  width: 32px;
  height: 32px;
  border: 1px solid #334155;
  border-radius: 4px;
  background: rgba(15, 23, 42, 0.85);
  color: #94a3b8;
  font-size: 1.1rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
  backdrop-filter: blur(4px);
  line-height: 1;
}

.zoom-controls button:hover {
  background: #1e293b;
  color: #e2e8f0;
  border-color: #38bdf8;
}

/* Event list */
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
