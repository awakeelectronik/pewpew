<template>
  <div class="status-container">
    <h2>📊 System Status</h2>
    
    <div v-if="store.state.status" class="status-grid">
      <div class="status-card">
        <div class="label">Hostname</div>
        <div class="value">{{ store.state.status.hostname }}</div>
      </div>
      <div class="status-card">
        <div class="label">SSH Log Path</div>
        <div class="value">{{ store.state.status.ssh_log_path }}</div>
      </div>
      <div class="status-card">
        <div class="label">Firewall Backend</div>
        <div :class="['value', store.state.status.firewall_backend.toLowerCase()]">
          {{ store.state.status.firewall_backend }}
        </div>
      </div>
      <div class="status-card">
        <div class="label">DB Size</div>
        <div class="value">{{ formatBytes(store.state.status.db_size) }}</div>
      </div>
      <div class="status-card">
        <div class="label">Total Events</div>
        <div class="value">{{ store.state.status.total_events }}</div>
      </div>
      <div class="status-card">
        <div class="label">Active Bans</div>
        <div class="value">{{ store.state.status.active_bans }}</div>
      </div>
      <div class="status-card">
        <div class="label">Version</div>
        <div class="value">{{ store.state.status.version }}</div>
      </div>
      <div class="status-card">
        <div class="label">Uptime</div>
        <div class="value">{{ formatUptime(store.state.status.uptime_seconds) }}</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useStore } from '../stores/store.js'

const store = useStore()

const formatBytes = (bytes) => {
  if (bytes === 0) return '0 Bytes'
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

const formatUptime = (seconds) => {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  return `${days}d ${hours}h ${minutes}m`
}

onMounted(() => {
  store.fetchStatus()
})
</script>

<style scoped>
.status-container {
  background: var(--bg-secondary);
  border-radius: 8px;
  padding: 1.5rem;
  border: 1px solid #334155;
}

.status-container h2 {
  color: #38bdf8;
  margin-bottom: 1rem;
}

.status-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 1rem;
}

.status-card {
  background: #0f172a;
  border: 1px solid #334155;
  border-radius: 4px;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.label {
  font-size: 0.75rem;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.value {
  font-size: 1.125rem;
  font-weight: bold;
  color: #38bdf8;
  font-family: monospace;
}

.value.ufw {
  color: #22c55e;
}

.value.iptables {
  color: #a78bfa;
}

.value.none {
  color: #ff9500;
}
</style>
