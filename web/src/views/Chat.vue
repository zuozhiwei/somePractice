<template>
  <div class="chat-page" v-loading="loadingConversations">
    <div class="chat-layout">
      <!-- 会话列表 -->
      <div class="conv-list">
        <div class="conv-header">
          <span>消息</span>
          <span class="conv-count">{{ conversations.length }}</span>
        </div>
        <div v-if="!conversations.length && !loadingConversations" class="conv-empty">
          <el-empty description="暂无会话" :image-size="60" />
        </div>
        <div v-for="conv in conversations" :key="conv.id"
             class="conv-item" :class="{ active: activeId === conv.id }"
             @click="selectConversation(conv)">
          <div class="conv-avatar" :class="{ group: conv.type === 'GROUP' }">
            {{ conv.type === 'GROUP' ? '群' : (conv.name || '?').charAt(0) }}
          </div>
          <div class="conv-info">
            <div class="conv-name-row">
              <span class="conv-name">{{ conv.name }}</span>
              <span v-if="conv.unreadCount > 0" class="conv-unread">{{ conv.unreadCount > 99 ? '99+' : conv.unreadCount }}</span>
            </div>
            <div class="conv-last">
              <span class="conv-last-text">{{ conv.lastMessage ? (conv.lastSenderName + '：' + conv.lastMessage) : '暂无消息' }}</span>
              <span v-if="conv.lastMessageAt" class="conv-time">{{ shortTime(conv.lastMessageAt) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 消息区 -->
      <div class="msg-panel">
        <div class="msg-header">
          <span>{{ activeConv?.name || '选择会话' }}</span>
          <span v-if="activeConv" class="msg-header-sub">
            {{ activeConv.type === 'GROUP' ? '师门群聊，师傅与所有徒弟可见' : '师徒私聊' }}
          </span>
        </div>

        <div ref="msgListEl" class="msg-list" @scroll="onScroll">
          <div v-if="hasMore" class="load-more" @click="loadOlder">
            <el-button size="small" text :loading="loadingOlder">加载更早消息</el-button>
          </div>
          <div v-for="msg in messages" :key="msg.id"
               class="msg-row" :class="{ mine: msg.senderId === myId }">
            <div class="msg-bubble-wrap">
              <div class="msg-sender" v-if="activeConv?.type === 'GROUP' && msg.senderId !== myId">
                {{ msg.senderName }}<span class="msg-role" v-if="msg.senderRole === 'TEACHER'">（师傅）</span>
              </div>
              <div class="msg-bubble">{{ msg.content }}</div>
              <div class="msg-time">{{ formatTime(msg.createdAt) }}</div>
            </div>
          </div>
          <div v-if="!messages.length && activeId" class="msg-empty">
            <el-empty description="还没有消息，打个招呼吧" :image-size="60" />
          </div>
        </div>

        <div class="msg-input">
          <el-input v-model="draft" type="textarea" :rows="3" resize="none"
                    placeholder="输入消息，Enter 发送，Shift+Enter 换行"
                    @keydown.enter.exact.prevent="send"
                    :disabled="!activeId" />
          <div class="input-footer">
            <span class="input-tip">{{ activeId ? '' : '先选择一个会话' }}</span>
            <el-button type="primary" round :loading="sending" :disabled="!activeId || !draft.trim()" @click="send">
              发送
            </el-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, nextTick, computed, watch } from 'vue'
import { chatApi } from '../api/chat'
import { useAuthStore } from '../store/auth'
import { useChatStore } from '../store/chat'
import { ElMessage } from 'element-plus'

const authStore = useAuthStore()
const chatStore = useChatStore()

const myId = computed(() => authStore.user?.id)

const loadingConversations = ref(false)
const conversations = ref([])
const activeId = ref(null)
const activeConv = computed(() => conversations.value.find(c => c.id === activeId.value) || null)

const messages = ref([])
const page = ref(0)
const hasMore = ref(false)
const loadingOlder = ref(false)
const draft = ref('')
const sending = ref(false)
const msgListEl = ref(null)
let stickBottom = true
const topicSubs = new Map()
let lastSyncedId = 0
let catchingUp = false
let catchUpTimer = null

/** 订阅所有会话主题（会话有权限访问才会出现在列表里） */
function resubscribeTopics() {
  topicSubs.forEach(sub => { try { sub.unsubscribe() } catch (e) { /* ignore */ } })
  topicSubs.clear()
  if (!chatStore.connected) return
  for (const conv of conversations.value) {
    const sub = chatStore.subscribeTopic(`/topic/chat/${conv.id}`, dto => handleIncoming(dto))
    if (sub) topicSubs.set(conv.id, sub)
  }
}

watch(() => chatStore.connected, connected => {
  if (connected) {
    resubscribeTopics()
    fetchConversations()
    catchUp()
    chatStore.refreshUnread()
  }
})

watch(conversations, () => resubscribeTopics(), { deep: true })

async function fetchConversations() {
  loadingConversations.value = true
  try {
    const res = await chatApi.conversations()
    conversations.value = res.data.data || []
    if (conversations.value.length && !activeId.value) {
      selectConversation(conversations.value[0])
    }
    chatStore.refreshUnread()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '加载会话失败')
  } finally {
    loadingConversations.value = false
  }
}

async function selectConversation(conv) {
  activeId.value = conv.id
  lastSyncedId = 0
  messages.value = []
  page.value = 0
  hasMore.value = false
  stickBottom = true
  await loadMessages()
  chatApi.markRead(conv.id).catch(() => {})
  conv.unreadCount = 0
  chatStore.refreshUnread()
}

async function loadMessages() {
  if (!activeId.value) return
  try {
    const conversationId = activeId.value
    const currentPage = page.value
    const res = await chatApi.messages(conversationId, currentPage, 50)
    if (activeId.value !== conversationId) return
    const d = res.data.data
    const list = (d.content || []).slice().reverse()
    messages.value = currentPage === 0 ? list : [...list, ...messages.value]
    if (currentPage === 0) lastSyncedId = list.reduce((max, msg) => Math.max(max, msg.id), 0)
    hasMore.value = messages.value.length < (d.total || 0)
    await nextTick()
    if (page.value === 0) scrollToBottom()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '加载消息失败')
  }
}

async function catchUp() {
  if (!activeId.value || catchingUp || document.hidden) return
  const conversationId = activeId.value
  catchingUp = true
  try {
    let received = false
    while (activeId.value === conversationId) {
      const res = await chatApi.messagesAfter(conversationId, lastSyncedId, 100)
      if (activeId.value !== conversationId) return
      const batch = res.data.data?.content || []
      for (const msg of batch) {
        lastSyncedId = Math.max(lastSyncedId, msg.id)
        applyIncoming(msg, false)
        received = true
      }
      if (batch.length < 100) break
    }
    if (received && activeId.value === conversationId) {
      chatApi.markRead(conversationId).catch(() => {})
      chatStore.refreshUnread()
    }
  } catch (_) { /* retry on the next poll or reconnect */ }
  finally { catchingUp = false }
}

async function loadOlder() {
  if (!activeId.value || loadingOlder.value) return
  loadingOlder.value = true
  const oldHeight = msgListEl.value?.scrollHeight || 0
  page.value += 1
  try {
    await loadMessages()
    await nextTick()
    if (msgListEl.value) {
      msgListEl.value.scrollTop = msgListEl.value.scrollHeight - oldHeight
    }
  } finally {
    loadingOlder.value = false
  }
}

function onScroll() {
  const el = msgListEl.value
  if (!el) return
  stickBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 60
}

function scrollToBottom() {
  const el = msgListEl.value
  if (el) el.scrollTop = el.scrollHeight
}

async function send() {
  const text = draft.value.trim()
  if (!text || !activeId.value) return
  sending.value = true
  try {
    const res = await chatApi.send(activeId.value, text)
    const dto = res.data.data
    const existing = messages.value.find(m => m.id === dto.id)
    if (!existing) {
      messages.value.push(dto)
      stickBottom = true
      await nextTick()
      scrollToBottom()
    }
    draft.value = ''
    const conv = conversations.value.find(c => c.id === activeId.value)
    if (conv) {
      conv.lastMessage = dto.content
      conv.lastSenderName = dto.senderName
      conv.lastMessageAt = dto.createdAt
      conv.unreadCount = 0
    }
    chatApi.markRead(activeId.value).catch(() => {})
    chatStore.refreshUnread()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '发送失败')
  } finally {
    sending.value = false
  }
}

function handleIncoming(dto) {
  if (dto?.conversationId === activeId.value) { catchUp(); return }
  applyIncoming(dto)
}

function applyIncoming(dto, refresh = true) {
  if (!dto?.conversationId) return
  const conv = conversations.value.find(c => c.id === dto.conversationId)
  if (!conv) {
    fetchConversations()
    return
  }
  const exists = messages.value.some(m => m.id === dto.id)
  if (activeId.value === dto.conversationId && !exists) {
    messages.value.push(dto)
    nextTick(() => {
      if (stickBottom) scrollToBottom()
    })
    if (refresh) chatApi.markRead(dto.conversationId).catch(() => {})
    conv.unreadCount = 0
  } else if (activeId.value !== dto.conversationId && !exists) {
    conv.unreadCount = (conv.unreadCount || 0) + 1
  }
  conv.lastMessage = dto.content
  conv.lastSenderName = dto.senderName
  conv.lastMessageAt = dto.createdAt
  if (refresh) chatStore.refreshUnread()
}

function shortTime(ts) {
  const d = new Date(ts)
  const now = new Date()
  if (d.toDateString() === now.toDateString()) {
    return String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0')
  }
  return (d.getMonth() + 1) + '-' + d.getDate()
}

function formatTime(ts) {
  const d = new Date(ts)
  return (d.getMonth() + 1) + '月' + d.getDate() + '日 ' +
    String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0')
}

let unsub = null

onMounted(async () => {
  chatStore.connect()
  unsub = chatStore.onMessage(handleIncoming)
  await fetchConversations()
  catchUpTimer = setInterval(catchUp, 20000)
  document.addEventListener('visibilitychange', catchUp)
})

onBeforeUnmount(() => {
  clearInterval(catchUpTimer)
  document.removeEventListener('visibilitychange', catchUp)
  if (unsub) unsub()
  topicSubs.forEach(sub => { try { sub.unsubscribe() } catch (e) { /* ignore */ } })
  topicSubs.clear()
})
</script>

<style scoped>
.chat-page { height: calc(100vh - 64px); max-width: 1200px; }
.chat-layout {
  display: flex;
  height: 100%;
  background: var(--color-surface);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
  overflow: hidden;
}

/* 会话列表 */
.conv-list {
  width: 300px;
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  overflow-y: auto;
}
.conv-header {
  padding: 16px 18px;
  border-bottom: 1px solid var(--color-border);
  font-size: 15px; font-weight: 700;
  display: flex; align-items: center; gap: 8px;
}
.conv-count {
  font-size: 12px; color: var(--color-text-secondary);
  background: #f1f5f9; padding: 2px 8px; border-radius: 10px;
}
.conv-empty { padding: 30px 10px; }
.conv-item {
  display: flex; gap: 12px;
  padding: 14px 18px;
  cursor: pointer;
  border-bottom: 1px solid #f1f5f9;
  transition: background .15s;
}
.conv-item:hover { background: #f8fafc; }
.conv-item.active { background: #eef2ff; }

.conv-avatar {
  width: 40px; height: 40px; border-radius: 50%;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: #fff; font-size: 15px; font-weight: 600;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.conv-avatar.group { background: linear-gradient(135deg, #22c55e, #16a34a); }

.conv-info { flex: 1; min-width: 0; }
.conv-name-row { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.conv-name { font-size: 14px; font-weight: 600; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.conv-unread {
  min-width: 18px; height: 18px; padding: 0 5px;
  border-radius: 9px; background: #ef4444; color: #fff;
  font-size: 11px; font-weight: 600;
  display: inline-flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.conv-last {
  display: flex; justify-content: space-between; gap: 8px;
  margin-top: 4px; font-size: 12px; color: #94a3b8;
}
.conv-last-text { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.conv-time { flex-shrink: 0; }

/* 消息区 */
.msg-panel { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.msg-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--color-border);
  display: flex; align-items: baseline; gap: 10px;
}
.msg-header > span:first-child { font-size: 15px; font-weight: 700; }
.msg-header-sub { font-size: 12px; color: #94a3b8; }

.msg-list {
  flex: 1;
  overflow-y: auto;
  padding: 18px 20px;
  display: flex; flex-direction: column; gap: 14px;
  background: #fafbfc;
}
.load-more { text-align: center; }
.msg-empty { margin: auto; }

.msg-row { display: flex; }
.msg-row.mine { justify-content: flex-end; }
.msg-bubble-wrap { max-width: 70%; }
.msg-sender { font-size: 12px; color: #94a3b8; margin-bottom: 3px; }
.msg-role { color: #d97706; }
.msg-bubble {
  padding: 10px 14px;
  border-radius: 12px;
  background: #fff;
  border: 1px solid var(--color-border);
  font-size: 14px; line-height: 1.6; color: #1e293b;
  white-space: pre-wrap; word-break: break-word;
}
.msg-row.mine .msg-bubble {
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: #fff; border: none;
}
.msg-time { font-size: 11px; color: #cbd5e1; margin-top: 3px; }
.msg-row.mine .msg-time { text-align: right; }

.msg-input {
  border-top: 1px solid var(--color-border);
  padding: 12px 16px;
  display: flex; flex-direction: column; gap: 8px;
}
.input-footer { display: flex; justify-content: space-between; align-items: center; }
.input-tip { font-size: 12px; color: #cbd5e1; }
</style>
