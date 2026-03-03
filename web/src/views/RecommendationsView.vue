<template>
  <div class="panel">
    <div class="head">
      <h2>Hardening Recommendations</h2>
      <button class="btn" @click="store.fetchRecommendations()">Refresh</button>
    </div>
    <div v-for="r in store.state.recommendations" :key="r.id" class="rec">
      <div class="title">{{ r.severity.toUpperCase() }} - {{ r.title }}</div>
      <div>{{ r.details }}</div>
      <code v-if="r.command">{{ r.command }}</code>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useStore } from '../stores/store.js'
const store = useStore()
onMounted(() => store.fetchRecommendations())
</script>

<style scoped>
.panel{ background: var(--bg-secondary); border:1px solid #334155; border-radius:8px; padding:1rem;}
.head{ display:flex; justify-content:space-between; margin-bottom:1rem;}
.rec{ border:1px solid #334155; background:#0f172a; padding:.75rem; border-radius:4px; margin-bottom:.5rem; }
.title{ color:#38bdf8; font-weight:bold; margin-bottom:.25rem; }
code{ display:block; margin-top:.25rem; color:#f59e0b; }
.btn { background:#0f172a; color:#cbd5e1; border:1px solid #334155; border-radius:4px; padding:.4rem .8rem; }
</style>
