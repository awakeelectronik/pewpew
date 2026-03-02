<template>
  <div class="feed-container">
    <div class="feed-header">
      <h2>📡 Live Feed</h2>
      <input 
        v-model="searchFilter"
        type="text"
        placeholder="Filter by IP, user, country..."
        class="search-input"
      />
    </div>

    <div class="events-list">
      <div 
        v-for="event in filteredEvents"
        :key="event.id"
        :class="['event-item', `type-${event.event_type}`]"
      >
        <div class="event-time">
          {{ formatTime(event.timestamp) }}
        </div>
        <div class="event-type">
          <span :class="`badge-${event.event_type}`">
            {{ event.event_type.replace(/_/g, ' ').toUpperCase() }}
          </span>
        </div>
        <div class="event-detail">
          <span class="ip">{{ event.ip }}</span>
          <span v-if="event.username" class="user">{{ event.username }}</span>
          <span class="country">{{ event.country }}</span>
        </div>
        <div class="event-actions">
          <button 
            @click="banIP(event.ip)"
            class="btn-ban"
            title="Ban this IP"
          >
            🚫
          </button>
        </div>
      </div>

      <div v-if="filteredEvents.length === 0" class="empty">
        No events found
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useStore } from '../stores/store.js'

const store = useStore()
const searchFilter = ref('')

const filteredEvents = computed(() => {
  if (!searchFilter.value) return store.state.events

  const query = searchFilter.value.toLowerCase()
  return store.state.events.filter(event => 
    event.ip.includes(query) ||
    (event.username && event.username.includes(query)) ||
    (event.country && event.country.toLowerCase().includes(query))
  )
})

const formatTime = (timestamp) => {
  const date = new Date(timestamp)
  return date.toLocaleTimeString()
}

const banIP = async (ip) => {
  if (confirm(`Ban IP ${ip}?`)) {
    const success = await store.banIP(ip)
    if (success) {
      alert('IP banned successfully')
    }
  }
}

onMounted(() => {
  store.fetchEvents(1000)
})
</script>

<style scoped>
.feed-container {
  background: var(--bg-secondary);
  border-radius: 8px;
  padding: 1.5rem;
  border: 1px solid #334155;
}

.feed-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
  gap: 1rem;
}

.feed-header h2 {
  color: #38bdf8;
}

.search-input {
  flex: 1;
  max-width: 300px;
  background: #0f172a;
  border: 1px solid #334155;
  color: #f1f5f9;
  padding: 0.5rem 0.75rem;
  border-radius: 4px;
  font-size: 0.875rem;
}

.search-input:focus {
  outline: none;
  border-color: #38bdf8;
}

.events-list {
  max-height: 600px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.event-item {
  background: #0f172a;
  border: 1px solid #334155;
  border-left: 3px solid #94a3b8;
  padding: 0.75rem;
  border-radius: 4px;
  display: flex;
  gap: 1rem;
  align-items: center;
  font-size: 0.875rem;
  font-family: monospace;
  transition: all 0.2s;
}

.event-item:hover {
  border-color: #38bdf8;
  background: #1e293b;
}

.event-item.type-failed_password {
  border-left-color: #ff5454;
}

.event-item.type-invalid_user {
  border-left-color: #ff9500;
}

.event-item.type-accepted {
  border-left-color: #22c55e;
}

.event-time {
  color: #94a3b8;
  min-width: 80px;
}

.event-type {
  min-width: 120px;
}

.badge-failed_password {
  background: #ff5454;
  color: white;
  padding: 0.25rem 0.5rem;
  border-radius: 3px;
  font-size: 0.75rem;
}

.badge-invalid_user {
  background: #ff9500;
  color: white;
  padding: 0.25rem 0.5rem;
  border-radius: 3px;
  font-size: 0.75rem;
}

.badge-accepted {
  background: #22c55e;
  color: white;
  padding: 0.25rem 0.5rem;
  border-radius: 3px;
  font-size: 0.75rem;
}

.event-detail {
  flex: 1;
  display: flex;
  gap: 0.75rem;
  align-items: center;
}

.ip {
  color: #38bdf8;
}

.user {
  color: #a78bfa;
}

.country {
  color: #22c55e;
}

.event-actions {
  display: flex;
  gap: 0.5rem;
}

.btn-ban {
  background: transparent;
  border: 1px solid #ff5454;
  color: #ff5454;
  padding: 0.25rem 0.5rem;
  border-radius: 3px;
  cursor: pointer;
  font-size: 0.75rem;
  transition: all 0.2s;
}

.btn-ban:hover {
  background: #ff5454;
  color: white;
}

.empty {
  text-align: center;
  color: #94a3b8;
  padding: 2rem;
}
</style>
