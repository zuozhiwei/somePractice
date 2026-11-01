<template>
  <el-dialog :model-value="visible" @update:model-value="onVisible" width="520px" top="10vh"
             :close-on-click-modal="false" class="mentor-dialog">
    <div class="mentor-head">
      <canvas ref="portraitRef" width="64" height="64" class="mentor-portrait"></canvas>
      <div class="mentor-info">
        <div class="mentor-name">{{ mentorName || '师父' }}</div>
        <div class="mentor-title">{{ title }}</div>
      </div>
    </div>

    <div class="mentor-text">
      <p v-for="(l, i) in lines" :key="i" class="mentor-line" :class="{ typing: i === typingIndex }">{{ l }}</p>
    </div>

    <div v-if="locked" class="mentor-locked">🔒 这关还没解锁，先去通过前置关卡吧</div>
    <div v-else-if="tasks.length" class="mentor-tasks">
      <div v-for="t in tasks" :key="t.id" class="mentor-task" :class="String(t.submissionStatus || '').toLowerCase()">
        <span class="mt-title">{{ t.title }}</span>
        <span class="mt-status">{{ statusText(t.submissionStatus) }}</span>
        <el-button size="small" type="primary" :disabled="t.submissionStatus === 'APPROVED' || locked"
                   @click="submitTask(t)">提交</el-button>
      </div>
    </div>

    <template #footer>
      <el-button round @click="close">继续探索</el-button>
      <el-button type="primary" round @click="emitGotoLevel">进入关卡</el-button>
    </template>

    <TaskSubmitDialog ref="submitDialog" :task="currentTask" @done="onTaskDone" />
  </el-dialog>
</template>

<script setup>
import { ref, nextTick } from 'vue'
import { skillsApi } from '../api/skills'
import { mentorLines } from '../world/data'
import { buildSprites } from '../world/sprites'
import TaskSubmitDialog from './TaskSubmitDialog.vue'

const emit = defineEmits(['update:visible', 'submitted', 'goto-level'])

const visible = ref(false)
const levelId = ref(null)
const levelName = ref('')
const zoneKey = ref('')
const mentorName = ref('')
const story = ref('')
const description = ref('')
const title = ref('')
const tasks = ref([])
const locked = ref(false)
const lines = ref([])
const typingIndex = ref(0)
const currentTask = ref(null)
const submitDialog = ref(null)
const portraitRef = ref(null)

let typeTimer = null
let spriteCache = null
function mentorsSprites() {
  if (!spriteCache) spriteCache = buildSprites()
  return spriteCache
}

function statusText(s) {
  return { NONE: '未提交', PENDING: '审核中', REJECTED: '未通过', AI_REVIEWED: 'AI 复核', APPROVED: '已通过' }[s] || '未提交'
}

function onVisible(v) {
  visible.value = v
  emit('update:visible', v)
}

function close() {
  onVisible(false)
}

function emitGotoLevel() {
  emit('goto-level', levelId.value)
}

function drawPortrait() {
  const cv = portraitRef.value
  if (!cv || !zoneKey.value) return
  const ctx = cv.getContext('2d')
  const portrait = mentorsSprites().mentors[zoneKey.value]?.portrait
  ctx.clearRect(0, 0, cv.width, cv.height)
  if (portrait) ctx.drawImage(portrait, 0, 0)
}

function typeNext() {
  if (typingIndex.value < lines.value.length) {
    typingIndex.value++
    if (typingIndex.value < lines.value.length) typeTimer = setTimeout(typeNext, 400)
  }
}

function startTyping() {
  if (typeTimer) clearTimeout(typeTimer)
  typingIndex.value = 0
  typeTimer = setTimeout(typeNext, 300)
}

function buildLines(ts) {
  return mentorLines({ story: story.value }, { name: levelName.value, description: description.value }, ts, mentorName.value || '师父')
}

async function loadDetail() {
  if (levelId.value == null) return
  try {
    const res = await skillsApi.levelDetail(levelId.value)
    const d = res.data?.data || {}
    tasks.value = d.tasks || []
    locked.value = false
    lines.value = buildLines(tasks.value)
    startTyping()
  } catch (e) {
    locked.value = true
    tasks.value = []
    lines.value = ['这关还没解锁，先去通过前置关卡吧']
  }
}

async function refresh() {
  try {
    if (levelId.value == null) return
    const res = await skillsApi.levelDetail(levelId.value)
    const d = res.data?.data || {}
    tasks.value = d.tasks || []
    locked.value = false
    lines.value = buildLines(tasks.value)
    startTyping()
  } catch (e) {
    /* 保持现状不崩 */
  } finally {
    emit('submitted')
  }
}

function submitTask(t) {
  currentTask.value = t
  submitDialog.value?.open()
}

function onTaskDone() {
  currentTask.value = null
  refresh()
}

function open(opts = {}) {
  levelId.value = opts.levelId ?? null
  levelName.value = opts.levelName || ''
  zoneKey.value = opts.zoneKey || ''
  mentorName.value = opts.mentorName || ''
  story.value = opts.story || ''
  description.value = opts.description || ''
  title.value = opts.title || ''
  tasks.value = []
  locked.value = false
  lines.value = []
  typingIndex.value = 0
  currentTask.value = null
  visible.value = true
  emit('update:visible', true)
  nextTick(drawPortrait)
  loadDetail()
}

defineExpose({ open })
</script>

<style scoped>
.mentor-dialog {
  border-radius: 12px;
}

.mentor-head {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}

.mentor-portrait {
  width: 64px;
  height: 64px;
  border: 2px solid #475569;
  border-radius: 8px;
  background: #1e293b;
  image-rendering: pixelated;
  flex-shrink: 0;
}

.mentor-info {
  min-width: 0;
}

.mentor-name {
  font-size: 18px;
  font-weight: 700;
  color: #1e293b;
  font-family: Georgia, 'Times New Roman', serif;
}

.mentor-title {
  margin-top: 4px;
  font-size: 12px;
  color: #64748b;
}

.mentor-text {
  background: #0f172a;
  border-radius: 10px;
  padding: 14px 16px;
  margin-bottom: 14px;
  max-height: 220px;
  overflow-y: auto;
}

.mentor-line {
  margin: 0 0 8px;
  color: #e2e8f0;
  font-size: 14px;
  line-height: 1.7;
  font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', Menlo, monospace;
}

.mentor-line:last-child {
  margin-bottom: 0;
}

.mentor-line.typing::after {
  content: '▌';
  color: #f1c40f;
  animation: mentor-blink 0.8s steps(1) infinite;
}

@keyframes mentor-blink {
  50% {
    opacity: 0;
  }
}

.mentor-locked {
  padding: 12px 16px;
  border-radius: 8px;
  background: #fef3c7;
  color: #b45309;
  font-size: 13px;
}

.mentor-tasks {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 200px;
  overflow-y: auto;
}

.mentor-task {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: #f8fafc;
}

.mt-title {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mt-status {
  font-size: 11px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 4px;
  white-space: nowrap;
}

.mentor-task.approved .mt-status { background: #dcfce7; color: #16a34a; }
.mentor-task.rejected .mt-status { background: #fee2e2; color: #dc2626; }
.mentor-task.pending .mt-status { background: #fef3c7; color: #d97706; }
.mentor-task.ai_reviewed .mt-status { background: #e0f2fe; color: #0284c7; }
.mentor-task.none .mt-status { background: #f1f5f9; color: #94a3b8; }

.mentor-dialog :deep(.el-dialog__header) {
  padding-bottom: 0;
}

.mentor-dialog :deep(.el-dialog__body) {
  padding-top: 12px;
}
</style>
