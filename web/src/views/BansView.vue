<template>
  <div class="bans-container">
    <h2>🚫 Active Bans</h2>
    
    <div v-if="store.state.bans.length === 0" class="empty">
      No active bans
    </div>
    
    <div v-else class="bans-grid">
      <div 
        v-for="ban in activeBans"
        :key="ban.ip"
        class="ban-card"
      >
        <div class="ban-ip">{{ ban.ip }}</div>
        <div class="ban-meta">
          <div><strong>Backend:</strong> {{ ban.backend }}</div>
          <div><strong>Reason:</strong> {{ ban.reason }}</div>
          <div><strong>Banned at:</strong> {{ formatTime(ban.timestamp) }}</div>
          <div v-if="ban.created_by"><strong>By:</strong> {{ ban.created_by }}</div>
        </div>
        <div class="ban-actions">
          <button 
            @click="unbanIP(ban.ip)"
            class="btn-unban"
          >
            Unban
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useStore } from '../stores/store.js'

const store = useStore()

const activeBans = computed(() => 
  store.state.bans.filter(ban => ban.active)
)

const formatTime = (timestamp) => {
  const date = new Date(timestamp)
  return date.toLocaleString()
}

const unbanIP = async (ip) => {
  if (confirm(`Unban IP ${ip}?`)) {
    const success = await store.unbanIP(ip)
    if (success) {
      alert('IP unbanned successfully')
    }
  }
}

onMounted(() => {
  store.fetchBans()
})
</script>

<style scoped>
.bans-container {
  background: var(--bg-secondary);
  border-radius: 8px;
  padding: 1.5rem;
  border: 1px solid #334155;
}

.bans-container h2 {
  color: #38bdf8;
  margin-bottom: 1rem;
}

.empty {
  text-align: center;
  color: #94a3b8;
  padding: 2rem;
}

.bans-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1rem;
}

.ban-card {
  background: #0f172a;
  border: 1px solid #334155;
  border-left: 3px solid #ff5454;
  border-radius: 4px;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.ban-ip {
  font-size: 1.125rem;
  font-weight: bold;
  color: #38bdf8;
  font-family: monospace;
}

.ban-meta {
  font-size: 0.875rem;
  color: #cbd5e1;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.ban-meta strong {
  color: #94a3b8;
}

.ban-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.btn-unban {
  flex: 1;
  background: transparent;
  border: 1px solid #ff5454;
  color: #ff5454;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.875rem;
  transition: all 0.2s;
}

.btn-unban:hover {
  background: #ff5454;
  color: white;
}
</style>
