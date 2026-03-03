import { reactive, ref } from 'vue'
import axios from 'axios'

const state = reactive({
  events: [],
  bans: [],
  topAttackers: [],
  status: null,
  ports: [],
  vulnerabilities: [],
  recommendations: [],
  wsConnected: false
})

let ws = null

export function useStore() {
  const connectWebSocket = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const wsUrl = `${protocol}://${window.location.host}/ws/events`
    
    ws = new WebSocket(wsUrl)
    
    ws.onopen = () => {
      state.wsConnected = true
      console.log('WebSocket connected')
    }
    
    ws.onmessage = (event) => {
      try {
        const event_data = JSON.parse(event.data)
        state.events.unshift(event_data)
        if (state.events.length > 1000) {
          state.events.pop()
        }
        updateTopAttackers()
      } catch (err) {
        console.error('Failed to parse WS event:', err)
      }
    }
    
    ws.onerror = (err) => {
      console.error('WebSocket error:', err)
      state.wsConnected = false
    }
    
    ws.onclose = () => {
      state.wsConnected = false
      console.log('WebSocket disconnected')
      setTimeout(connectWebSocket, 3000)
    }
  }

  const disconnectWebSocket = () => {
    if (ws) {
      ws.close()
    }
  }

  const fetchEvents = async (limit = 100) => {
    try {
      const { data } = await axios.get(`/api/events?limit=${limit}`)
      state.events = data || []
      updateTopAttackers()
    } catch (err) {
      console.error('Failed to fetch events:', err)
    }
  }

  const fetchBans = async () => {
    try {
      const { data } = await axios.get('/api/bans')
      state.bans = data || []
    } catch (err) {
      console.error('Failed to fetch bans:', err)
    }
  }

  const fetchAttackers = async () => {
    try {
      const { data } = await axios.get('/api/attackers')
      state.topAttackers = data || []
    } catch (err) {
      console.error('Failed to fetch attackers:', err)
      updateTopAttackers()
    }
  }

  const fetchStatus = async () => {
    try {
      const { data } = await axios.get('/api/status')
      state.status = data
    } catch (err) {
      console.error('Failed to fetch status:', err)
    }
  }

  const fetchPorts = async () => {
    try {
      const { data } = await axios.get('/api/ports')
      const list = Array.isArray(data) ? data : []
      state.ports.splice(0, state.ports.length, ...list)
    } catch (err) {
      console.error('Failed to fetch ports:', err)
    }
  }

  const fetchVulnerabilities = async () => {
    try {
      const { data } = await axios.get('/api/vulns')
      state.vulnerabilities = data || []
    } catch (err) {
      console.error('Failed to fetch vulnerabilities:', err)
    }
  }

  const fetchRecommendations = async () => {
    try {
      const { data } = await axios.get('/api/recommendations')
      state.recommendations = data || []
    } catch (err) {
      console.error('Failed to fetch recommendations:', err)
    }
  }

  const banIP = async (ip, reason = 'Manual ban via dashboard') => {
    try {
      await axios.post('/api/bans', { ip, reason })
      await fetchBans()
      return true
    } catch (err) {
      console.error('Failed to ban IP:', err)
      return false
    }
  }

  const unbanIP = async (ip) => {
    try {
      await axios.delete(`/api/bans/${ip}`)
      await fetchBans()
      return true
    } catch (err) {
      console.error('Failed to unban IP:', err)
      return false
    }
  }

  const updateTopAttackers = () => {
    const attackerMap = new Map()
    
    state.events.forEach(event => {
      if (!attackerMap.has(event.ip)) {
        attackerMap.set(event.ip, {
          ip: event.ip,
          count: 0,
          lastSeen: event.timestamp,
          country: event.country,
          city: event.city,
          latitude: event.latitude,
          longitude: event.longitude
        })
      }
      const attacker = attackerMap.get(event.ip)
      attacker.count++
      attacker.lastSeen = event.timestamp
    })
    
    state.topAttackers = Array.from(attackerMap.values())
      .sort((a, b) => b.count - a.count)
      .slice(0, 20)
  }

  return {
    state,
    connectWebSocket,
    disconnectWebSocket,
    fetchEvents,
    fetchBans,
    fetchAttackers,
    fetchStatus,
    fetchPorts,
    fetchVulnerabilities,
    fetchRecommendations,
    banIP,
    unbanIP
  }
}
