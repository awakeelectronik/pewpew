<template>
  <div class="map-container">
    <div class="map-header">
      <h2>🌍 Live Attack Map</h2>
      <div class="stats">
        <span class="stat">
          <strong>{{ store.state.events.length }}</strong> Events
        </span>
        <span class="stat">
          <strong>{{ store.state.topAttackers.length }}</strong> Unique IPs
        </span>
        <span :class="['stat', { connected: store.state.wsConnected }]">
          {{ store.state.wsConnected ? '🟢' : '🔴' }} 
          {{ store.state.wsConnected ? 'Live' : 'Offline' }}
        </span>
      </div>
    </div>

    <canvas 
      id="attack-canvas" 
      class="attack-map"
      width="1200"
      height="600"
    ></canvas>

    <div class="counter">
      <div v-if="recentEvent" class="recent-event">
        <span class="event-time">{{ formatTime(recentEvent.timestamp) }}</span>
        <span class="event-type">{{ recentEvent.event_type }}</span>
        <span class="event-ip">{{ recentEvent.ip }}</span>
        <span class="event-country">{{ recentEvent.country }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useStore } from '../stores/store.js'

const store = useStore()

const recentEvent = computed(() => store.state.events[0])

const formatTime = (timestamp) => {
  const date = new Date(timestamp)
  return date.toLocaleTimeString()
}

onMounted(() => {
  store.fetchEvents(500)
  drawMap()
})

const drawMap = () => {
  const canvas = document.getElementById('attack-canvas')
  if (!canvas) return

  const ctx = canvas.getContext('2d')
  
  ctx.fillStyle = '#0f172a'
  ctx.fillRect(0, 0, canvas.width, canvas.height)
  
  ctx.strokeStyle = '#1e293b'
  ctx.lineWidth = 1
  for (let i = 0; i < canvas.width; i += 100) {
    ctx.beginPath()
    ctx.moveTo(i, 0)
    ctx.lineTo(i, canvas.height)
    ctx.stroke()
  }
  
  const recentAttacks = store.state.events.slice(0, 50)
  recentAttacks.forEach((event, idx) => {
    const x = (Math.sin(idx) * canvas.width / 2) + canvas.width / 2
    const y = (Math.cos(idx) * canvas.height / 2) + canvas.height / 2
    
    ctx.fillStyle = event.event_type === 'accepted' ? '#22c55e' : '#ff5454'
    ctx.beginPath()
    ctx.arc(x, y, 3, 0, Math.PI * 2)
    ctx.fill()
  })
  
  ctx.fillStyle = '#38bdf8'
  ctx.font = '12px monospace'
  ctx.fillText(`${store.state.events.length} events`, 20, 30)
  ctx.fillText(`${store.state.topAttackers.length} attackers`, 20, 50)
}
</script>

<style scoped>
.map-container {
  background: var(--bg-secondary);
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

.attack-map {
  width: 100%;
  border-radius: 4px;
  border: 1px solid #334155;
  background: #0f172a;
  margin-bottom: 1rem;
}

.counter {
  text-align: center;
}

.recent-event {
  display: flex;
  gap: 1rem;
  justify-content: center;
  padding: 0.75rem;
  background: #0f172a;
  border-radius: 4px;
  font-size: 0.875rem;
  font-family: monospace;
}

.event-type {
  color: #ff5454;
  font-weight: bold;
}

.event-ip {
  color: #38bdf8;
}

.event-country {
  color: #22c55e;
}
</style>
