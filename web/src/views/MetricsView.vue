<template>
  <div class="metrics-container">
    <div class="metrics-header">
      <h2>📈 VPS Health</h2>
      <span class="refresh-badge">Auto-refresh 3s</span>
    </div>

    <div v-if="!metrics" class="loading">Loading metrics…</div>

    <div v-else class="metrics-grid">
      <!-- CPU -->
      <div class="metric-card">
        <div class="card-title">🖥️ CPU</div>
        <div class="bar-row">
          <div class="bar-track">
            <div class="bar-fill" :class="cpuColor" :style="{ width: metrics.cpu.usage_percent + '%' }"></div>
          </div>
          <span class="bar-label">{{ fmt(metrics.cpu.usage_percent) }}%</span>
        </div>
        <div class="sub-stats">
          <span>user {{ fmt(metrics.cpu.user_percent) }}%</span>
          <span>sys {{ fmt(metrics.cpu.sys_percent) }}%</span>
          <span :class="{ warn: metrics.cpu.iowait_percent > 5 }">iowait {{ fmt(metrics.cpu.iowait_percent) }}%</span>
          <span :class="{ danger: metrics.cpu.steal_percent > 5 }">steal {{ fmt(metrics.cpu.steal_percent) }}%</span>
        </div>
      </div>

      <!-- RAM -->
      <div class="metric-card">
        <div class="card-title">🧠 Memory</div>
        <div class="bar-row">
          <div class="bar-track">
            <div class="bar-fill" :class="memColor" :style="{ width: metrics.memory.usage_percent + '%' }"></div>
          </div>
          <span class="bar-label">{{ fmt(metrics.memory.usage_percent) }}%</span>
        </div>
        <div class="sub-stats">
          <span>used {{ metrics.memory.used_mb }} MB</span>
          <span>avail {{ metrics.memory.available_mb }} MB</span>
          <span>total {{ metrics.memory.total_mb }} MB</span>
        </div>
        <div v-if="metrics.memory.swap_total_mb > 0" class="swap-row">
          <span class="label-sm">Swap</span>
          <div class="bar-track small">
            <div class="bar-fill warn" :style="{ width: swapPercent + '%' }"></div>
          </div>
          <span class="bar-label">{{ metrics.memory.swap_used_mb }}/{{ metrics.memory.swap_total_mb }} MB</span>
        </div>
        <div v-else class="swap-row">
          <span class="label-sm no-swap">No swap configured</span>
        </div>
      </div>

      <!-- Load Average -->
      <div class="metric-card">
        <div class="card-title">⚡ Load Average</div>
        <div class="load-grid">
          <div class="load-item">
            <div class="load-value" :class="loadColor(metrics.load.load1)">{{ fmtLoad(metrics.load.load1) }}</div>
            <div class="load-label">1 min</div>
          </div>
          <div class="load-item">
            <div class="load-value" :class="loadColor(metrics.load.load5)">{{ fmtLoad(metrics.load.load5) }}</div>
            <div class="load-label">5 min</div>
          </div>
          <div class="load-item">
            <div class="load-value" :class="loadColor(metrics.load.load15)">{{ fmtLoad(metrics.load.load15) }}</div>
            <div class="load-label">15 min</div>
          </div>
        </div>
        <div class="sub-stats">
          <span>{{ metrics.load.procs_running }} running / {{ metrics.load.procs_total }} total processes</span>
        </div>
      </div>

      <!-- Network -->
      <div class="metric-card" v-for="iface in metrics.network" :key="iface.interface">
        <div class="card-title">🌐 Network — <code>{{ iface.interface }}</code></div>
        <div class="net-grid">
          <div class="net-item">
            <div class="net-label">RX</div>
            <div class="net-value">{{ fmtBytes(iface.rx_bytes_per_sec) }}/s</div>
          </div>
          <div class="net-item">
            <div class="net-label">TX</div>
            <div class="net-value">{{ fmtBytes(iface.tx_bytes_per_sec) }}/s</div>
          </div>
        </div>
        <div class="sub-stats">
          <span :class="{ danger: iface.rx_drops > 0 }">
            RX drops: {{ iface.rx_drops.toLocaleString() }}
          </span>
          <span :class="{ danger: iface.tx_drops > 0 }">
            TX drops: {{ iface.tx_drops.toLocaleString() }}
          </span>
        </div>
      </div>

      <!-- Disk -->
      <div class="metric-card" v-for="disk in metrics.disk" :key="disk.device">
        <div class="card-title">💾 Disk — <code>{{ disk.device }}</code></div>
        <div class="net-grid">
          <div class="net-item">
            <div class="net-label">Reads/s</div>
            <div class="net-value">{{ fmt(disk.reads_per_sec) }}</div>
          </div>
          <div class="net-item">
            <div class="net-label">Writes/s</div>
            <div class="net-value">{{ fmt(disk.writes_per_sec) }}</div>
          </div>
        </div>
        <div class="sub-stats">
          <span :class="{ warn: disk.io_util_percent > 60, danger: disk.io_util_percent > 85 }">
            I/O util: {{ fmt(disk.io_util_percent) }}%
          </span>
        </div>
      </div>
    </div>

    <div class="collected-at" v-if="metrics">
      Last collected: {{ collectedAt }}
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'

const metrics = ref(null)
let timer = null

const fetchMetrics = async () => {
  try {
    const res = await fetch('/api/metrics')
    metrics.value = await res.json()
  } catch (e) {
    console.error('metrics fetch failed', e)
  }
}

const fmt = (n) => (typeof n === 'number' ? n.toFixed(1) : '0.0')
const fmtLoad = (n) => (typeof n === 'number' ? n.toFixed(2) : '0.00')

const fmtBytes = (bytes) => {
  if (bytes < 1024) return bytes.toFixed(0) + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / 1024 / 1024).toFixed(1) + ' MB'
}

const cpuColor = computed(() => {
  if (!metrics.value) return ''
  const p = metrics.value.cpu.usage_percent
  if (p >= 85) return 'danger'
  if (p >= 60) return 'warn'
  return 'ok'
})

const memColor = computed(() => {
  if (!metrics.value) return ''
  const p = metrics.value.memory.usage_percent
  if (p >= 90) return 'danger'
  if (p >= 75) return 'warn'
  return 'ok'
})

const swapPercent = computed(() => {
  if (!metrics.value || metrics.value.memory.swap_total_mb === 0) return 0
  return (metrics.value.memory.swap_used_mb / metrics.value.memory.swap_total_mb) * 100
})

const loadColor = (val) => {
  if (val >= 2) return 'danger'
  if (val >= 1) return 'warn'
  return 'ok'
}

const collectedAt = computed(() => {
  if (!metrics.value) return ''
  return new Date(metrics.value.collected_at * 1000).toLocaleTimeString()
})

onMounted(() => {
  fetchMetrics()
  timer = setInterval(fetchMetrics, 3000)
})

onUnmounted(() => {
  clearInterval(timer)
})
</script>

<style scoped>
.metrics-container {
  background: var(--bg-secondary);
  border-radius: 8px;
  padding: 1.5rem;
  border: 1px solid #334155;
}

.metrics-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.metrics-header h2 {
  color: #38bdf8;
}

.refresh-badge {
  font-size: 0.7rem;
  color: #94a3b8;
  background: #1e293b;
  border: 1px solid #334155;
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
}

.loading {
  color: #94a3b8;
  font-style: italic;
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1rem;
}

.metric-card {
  background: #0f172a;
  border: 1px solid #334155;
  border-radius: 6px;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.card-title {
  font-size: 0.8rem;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: bold;
}

.card-title code {
  color: #38bdf8;
  font-size: 0.9em;
}

/* Bar */
.bar-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.bar-track {
  flex: 1;
  height: 10px;
  background: #1e293b;
  border-radius: 5px;
  overflow: hidden;
}

.bar-track.small {
  height: 6px;
}

.bar-fill {
  height: 100%;
  border-radius: 5px;
  transition: width 0.5s ease;
}

.bar-fill.ok    { background: #22c55e; }
.bar-fill.warn  { background: #f59e0b; }
.bar-fill.danger{ background: #ff5454; }

.bar-label {
  font-size: 0.875rem;
  font-weight: bold;
  color: #f1f5f9;
  font-family: monospace;
  min-width: 3.5rem;
  text-align: right;
}

/* Sub stats */
.sub-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  font-size: 0.78rem;
  color: #94a3b8;
  font-family: monospace;
}

.sub-stats .warn   { color: #f59e0b; }
.sub-stats .danger { color: #ff5454; }

/* Swap row */
.swap-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.label-sm {
  font-size: 0.72rem;
  color: #94a3b8;
  text-transform: uppercase;
  min-width: 3rem;
}

.label-sm.no-swap {
  color: #22c55e;
}

/* Load */
.load-grid {
  display: flex;
  gap: 1.5rem;
}

.load-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.2rem;
}

.load-value {
  font-size: 1.5rem;
  font-weight: bold;
  font-family: monospace;
}

.load-value.ok     { color: #22c55e; }
.load-value.warn   { color: #f59e0b; }
.load-value.danger { color: #ff5454; }

.load-label {
  font-size: 0.7rem;
  color: #94a3b8;
}

/* Network / Disk grid */
.net-grid {
  display: flex;
  gap: 1.5rem;
}

.net-item {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.net-label {
  font-size: 0.7rem;
  color: #94a3b8;
  text-transform: uppercase;
}

.net-value {
  font-size: 1.1rem;
  font-weight: bold;
  font-family: monospace;
  color: #38bdf8;
}

/* Footer timestamp */
.collected-at {
  margin-top: 1rem;
  font-size: 0.72rem;
  color: #475569;
  text-align: right;
}
</style>
