<template>
  <div class="panel">
    <div class="head">
      <h2>Vulnerability Findings</h2>
      <button class="btn" @click="store.fetchVulnerabilities()">Scan now</button>
    </div>
    <div v-if="store.state.vulnerabilities.length === 0" class="empty">No findings</div>
    <div v-for="v in store.state.vulnerabilities" :key="v.id" class="card">
      <div><strong>{{ v.severity.toUpperCase() }}</strong> - {{ v.category }}</div>
      <div>{{ v.target }}</div>
      <div>{{ v.evidence }}</div>
      <div>{{ v.recommendation }}</div>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useStore } from '../stores/store.js'
const store = useStore()
onMounted(() => store.fetchVulnerabilities())
</script>

<style scoped>
.panel{ background: var(--bg-secondary); border:1px solid #334155; border-radius:8px; padding:1rem;}
.head{ display:flex; justify-content:space-between; margin-bottom:1rem;}
.card{ border:1px solid #334155; border-left:3px solid #ff5454; background:#0f172a; padding:.75rem; border-radius:4px; margin-bottom:.5rem; }
.empty{ color:#94a3b8; }
.btn { background:#0f172a; color:#cbd5e1; border:1px solid #334155; border-radius:4px; padding:.4rem .8rem; }
</style>
