<template>
  <div class="app dark-mode">
    <header class="navbar">
      <div class="navbar-brand">
        <h1>🔫 PewPew</h1>
        <span class="version">v0.1.0-alpha</span>
      </div>
      <nav class="navbar-nav">
        <button 
          :class="['nav-btn', { active: activeTab === 'map' }]"
          @click="activeTab = 'map'"
        >
          Map
        </button>
        <button 
          :class="['nav-btn', { active: activeTab === 'feed' }]"
          @click="activeTab = 'feed'"
        >
          Feed
        </button>
        <button 
          :class="['nav-btn', { active: activeTab === 'attackers' }]"
          @click="activeTab = 'attackers'"
        >
          Top Attackers
        </button>
        <button 
          :class="['nav-btn', { active: activeTab === 'bans' }]"
          @click="activeTab = 'bans'"
        >
          Bans
        </button>
        <button 
          :class="['nav-btn', { active: activeTab === 'status' }]"
          @click="activeTab = 'status'"
        >
          Status
        </button>
      </nav>
    </header>

    <main class="container">
      <component :is="currentView" />
    </main>

    <footer class="footer">
      <p>PewPew Security Dashboard — Running on {{ hostname }}</p>
    </footer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import MapView from './views/MapView.vue'
import FeedView from './views/FeedView.vue'
import AttackersView from './views/AttackersView.vue'
import BansView from './views/BansView.vue'
import StatusView from './views/StatusView.vue'
import { useStore } from './stores/store.js'

const activeTab = ref('map')
const hostname = ref('localhost')
const store = useStore()

const currentView = computed(() => {
  const views = {
    map: MapView,
    feed: FeedView,
    attackers: AttackersView,
    bans: BansView,
    status: StatusView
  }
  return views[activeTab.value]
})

onMounted(() => {
  store.connectWebSocket()
  
  fetch('/api/status')
    .then(r => r.json())
    .then(data => {
      hostname.value = data.hostname || 'localhost'
    })
    .catch(err => console.error('Failed to fetch status:', err))
})

onUnmounted(() => {
  store.disconnectWebSocket()
})
</script>

<style scoped>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: #0f172a;
  color: #f1f5f9;
}

.dark-mode {
  --bg-primary: #0f172a;
  --bg-secondary: #1e293b;
  --text-primary: #f1f5f9;
  --text-secondary: #cbd5e1;
  --accent: #38bdf8;
  --danger: #ff5454;
  --success: #22c55e;
}

.navbar {
  background: #020617;
  border-bottom: 1px solid #334155;
  padding: 1rem 2rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}

.navbar-brand {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.navbar-brand h1 {
  font-size: 1.5rem;
  color: #38bdf8;
}

.version {
  font-size: 0.75rem;
  color: #94a3b8;
  background: #1e293b;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

.navbar-nav {
  display: flex;
  gap: 0.5rem;
}

.nav-btn {
  background: transparent;
  border: 1px solid #334155;
  color: #cbd5e1;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.875rem;
  transition: all 0.2s;
}

.nav-btn:hover {
  border-color: #38bdf8;
  color: #38bdf8;
}

.nav-btn.active {
  background: #38bdf8;
  color: #0f172a;
  border-color: #38bdf8;
}

.container {
  flex: 1;
  padding: 2rem;
  overflow-y: auto;
}

.footer {
  background: #020617;
  border-top: 1px solid #334155;
  padding: 1rem 2rem;
  text-align: center;
  font-size: 0.875rem;
  color: #94a3b8;
}
</style>
