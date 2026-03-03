<template>
  <div class="panel">
    <div class="head">
      <h2>Open Ports</h2>
      <button class="btn" :disabled="portsLoading" @click="onRefresh">Refresh</button>
    </div>
    <table class="tbl">
      <thead>
        <tr>
          <th>Proto</th><th>Address</th><th>Port</th><th>State</th><th>Risk</th><th>Reason</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="p in store.state.ports" :key="`${p.proto}-${p.address}-${p.port}`">
          <td>{{ p.proto }}</td>
          <td>{{ p.address }}</td>
          <td>{{ p.port }}</td>
          <td>{{ p.state }}</td>
          <td>{{ p.risk }}</td>
          <td>{{ p.reason }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useStore } from '../stores/store.js'
const store = useStore()
const portsLoading = ref(false)
const onRefresh = async () => {
  portsLoading.value = true
  try {
    await store.fetchPorts()
  } finally {
    portsLoading.value = false
  }
}
onMounted(() => store.fetchPorts())
</script>

<style scoped>
.panel { background: var(--bg-secondary); border: 1px solid #334155; border-radius: 8px; padding: 1rem; }
.head { display:flex; justify-content:space-between; margin-bottom:1rem; }
.tbl { width:100%; font-size:.85rem; border-collapse: collapse; }
.tbl th,.tbl td { border-bottom:1px solid #334155; padding:.5rem; text-align:left; }
.btn { background:#0f172a; color:#cbd5e1; border:1px solid #334155; border-radius:4px; padding:.4rem .8rem; cursor:pointer; }
.btn:disabled { opacity:.6; cursor:not-allowed; }
</style>
