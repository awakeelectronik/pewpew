<template>
  <div class="attackers-container">
    <h2>🎯 Top Attackers</h2>
    
    <div class="table-wrapper">
      <table class="attackers-table">
        <thead>
          <tr>
            <th>Rank</th>
            <th>IP Address</th>
            <th>Attempts</th>
            <th>Country</th>
            <th>Last Seen</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr 
            v-for="(attacker, idx) in store.state.topAttackers"
            :key="attacker.ip"
          >
            <td class="rank">{{ idx + 1 }}</td>
            <td class="ip-cell">{{ attacker.ip }}</td>
            <td class="count">
              <span class="badge-count">{{ attacker.count }}</span>
            </td>
            <td class="country">{{ attacker.country }}</td>
            <td class="timestamp">{{ formatTime(attacker.lastSeen) }}</td>
            <td class="status">
              <span v-if="isBanned(attacker.ip)" class="badge-banned">BANNED</span>
              <span v-else class="badge-active">ACTIVE</span>
            </td>
            <td class="actions">
              <button 
                v-if="!isBanned(attacker.ip)"
                @click="banIP(attacker.ip)"
                class="btn-small btn-ban"
                title="Ban this IP"
              >
                Ban
              </button>
              <button 
                v-else
                @click="unbanIP(attacker.ip)"
                class="btn-small btn-unban"
                title="Unban this IP"
              >
                Unban
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useStore } from '../stores/store.js'

const store = useStore()

const isBanned = (ip) => {
  return store.state.bans.some(ban => ban.ip === ip && ban.active)
}

const formatTime = (timestamp) => {
  const date = new Date(timestamp)
  return date.toLocaleTimeString()
}

const banIP = async (ip) => {
  if (confirm(`Ban IP ${ip}?`)) {
    const success = await store.banIP(ip, 'Banned from Top Attackers')
    if (success) {
      alert('IP banned successfully')
    }
  }
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
  store.fetchAttackers()
})
</script>

<style scoped>
.attackers-container {
  background: var(--bg-secondary);
  border-radius: 8px;
  padding: 1.5rem;
  border: 1px solid #334155;
}

.attackers-container h2 {
  color: #38bdf8;
  margin-bottom: 1rem;
}

.table-wrapper {
  overflow-x: auto;
}

.attackers-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
  font-family: monospace;
}

.attackers-table thead {
  background: #0f172a;
  border-bottom: 2px solid #334155;
}

.attackers-table th {
  color: #38bdf8;
  padding: 0.75rem;
  text-align: left;
  font-weight: 600;
}

.attackers-table td {
  color: #cbd5e1;
  padding: 0.75rem;
  border-bottom: 1px solid #334155;
}

.attackers-table tbody tr:hover {
  background: #1e293b;
}

.rank {
  text-align: center;
  font-weight: bold;
  color: #38bdf8;
}

.ip-cell {
  color: #38bdf8;
  font-weight: bold;
}

.count {
  text-align: center;
}

.badge-count {
  background: #ff5454;
  color: white;
  padding: 0.25rem 0.5rem;
  border-radius: 3px;
  font-size: 0.75rem;
  font-weight: bold;
}

.country {
  color: #22c55e;
}

.timestamp {
  color: #94a3b8;
  font-size: 0.75rem;
}

.status {
  text-align: center;
}

.badge-banned {
  background: #ff5454;
  color: white;
  padding: 0.25rem 0.5rem;
  border-radius: 3px;
  font-size: 0.75rem;
  font-weight: bold;
}

.badge-active {
  background: #22c55e;
  color: white;
  padding: 0.25rem 0.5rem;
  border-radius: 3px;
  font-size: 0.75rem;
}

.actions {
  text-align: center;
}

.btn-small {
  background: transparent;
  border: 1px solid currentColor;
  padding: 0.25rem 0.5rem;
  border-radius: 3px;
  cursor: pointer;
  font-size: 0.75rem;
  transition: all 0.2s;
}

.btn-ban {
  color: #ff5454;
}

.btn-ban:hover {
  background: #ff5454;
  color: white;
}

.btn-unban {
  color: #94a3b8;
}

.btn-unban:hover {
  background: #94a3b8;
  color: white;
}
</style>
