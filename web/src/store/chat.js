import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useAuthStore } from './auth'
import { chatApi } from '../api/chat'

export const useChatStore = defineStore('chat', () => {
  const authStore = useAuthStore()
  const connected = ref(false)
  const totalUnread = ref(0)
  const handlers = new Set()
  const delivered = new Set()
  let socket = null
  let reconnectTimer = null
  let pollTimer = null
  let stopped = true
  let delay = 2000

  async function refreshUnread() {
    try { totalUnread.value = (await chatApi.unread()).data.data || 0 } catch (_) { /* retry later */ }
  }
  async function fetchLatest(conversationId, messageId) {
    try {
      const res = await chatApi.messages(conversationId, 0, 50)
      const messages = (res.data.data?.content || []).slice().reverse()
      for (const msg of messages) {
        if (msg.id !== messageId || delivered.has(msg.id)) continue
        delivered.add(msg.id)
        if (delivered.size > 1000) delivered.delete(delivered.values().next().value)
        handlers.forEach(fn => fn(msg))
      }
      await refreshUnread()
    } catch (_) { /* authorization or connection changed */ }
  }
  function connect() {
    if (!authStore.token || (socket && socket.readyState < 2)) return
    stopped = false
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${proto}//${location.host}/ws?token=${encodeURIComponent(authStore.token)}`)
    socket = ws
    ws.onopen = () => { if (socket === ws) { connected.value = true; delay = 2000; refreshUnread() } }
    ws.onmessage = e => {
      try { const event = JSON.parse(e.data); if (event.type === 'changed') fetchLatest(event.conversationId, event.messageId) } catch (_) { /* malformed event */ }
    }
    ws.onclose = () => {
      if (socket !== ws) return
      connected.value = false
      socket = null
      if (!stopped && authStore.token) {
        reconnectTimer = setTimeout(connect, delay)
        delay = Math.min(delay * 2, 30000)
      }
    }
    if (!pollTimer) pollTimer = setInterval(refreshUnread, 30000)
  }
  function disconnect() {
    stopped = true
    clearTimeout(reconnectTimer); clearInterval(pollTimer)
    reconnectTimer = null; pollTimer = null
    if (socket) socket.close()
    socket = null; connected.value = false; totalUnread.value = 0
    handlers.clear(); delivered.clear()
  }
  function onMessage(fn) { handlers.add(fn); return () => handlers.delete(fn) }
  // The server sends only per-user invalidation events; payloads are fetched over authorized HTTP.
  function subscribeTopic() { return { unsubscribe() {} } }
  return { connected, totalUnread, connect, disconnect, onMessage, subscribeTopic, refreshUnread }
})
