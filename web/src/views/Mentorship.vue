<template>
  <div class="page" v-loading="loading">
    <h1 class="page-title">{{ isExternal ? '我的提交记录' : '师徒关系' }}</h1>

    <!-- ========== Apply for mentor (compact, top; 外部成员不可拜师) ========== -->
    <div v-if="!isExternal" class="section-card" style="margin-bottom:20px">
      <div class="section-header">申请师傅</div>
      <div class="section-body">
        <div class="request-row">
          <el-select v-model="mentorId" filterable remote :remote-method="searchUsers"
                     :loading="searchLoading" clearable
                     placeholder="输入用户名或姓名搜索"
                     loading-text="搜索中..."
                     no-match-text="无匹配用户"
                     no-data-text="输入关键词开始搜索"
                     size="default" class="mentor-search" value-key="id">
            <el-option v-for="u in searchResults" :key="u.id"
                       :label="`${u.name || u.username}（${levelShort(u.level)} · ${roleText(u.role)} · ${u.totalXp} XP）`"
                       :value="u.id">
              <div class="search-option">
                <span class="search-option-name">{{ u.name || u.username }}</span>
                <span class="search-option-meta">{{ levelShort(u.level) }} · {{ roleText(u.role) }} · {{ u.totalXp }} XP</span>
              </div>
            </el-option>
            <template #empty>
              <div class="search-empty">暂无匹配用户，试试输入完整关键词</div>
            </template>
          </el-select>
          <el-button type="primary" @click="requestMentor" :loading="requestLoading">发送申请</el-button>
        </div>
      </div>
    </div>

    <!-- ========== Sent requests (as apprentice; 游客不可见) ========== -->
    <div v-if="!isExternal && sentRequests.length" class="section-card" style="margin-bottom:20px">
      <div class="section-header">
        <span>我发出的申请</span>
        <span class="count-badge">{{ sentRequests.length }} 条</span>
      </div>
      <div class="section-body">
        <div v-for="s in sentRequests" :key="s.id" class="person-row">
          <el-avatar :size="36" class="person-avatar">{{ (s.name || s.username)?.charAt(0)?.toUpperCase() }}</el-avatar>
          <div class="person-info">
            <div class="person-name">{{ s.name || s.username }}</div>
            <div class="person-meta">{{ formatTime(s.createdAt) }} · 等待对方通过</div>
          </div>
          <el-button size="small" type="warning" @click="cancelRequest(s.id)" :loading="cancelling === s.id">撤回</el-button>
        </div>
      </div>
    </div>

    <!-- ========== Pending requests (as mentor; 游客不可见) ========== -->
    <div v-if="!isExternal && pending.length" class="section-card" style="margin-bottom:20px">
      <div class="section-header">
        <span>📩 待处理的申请</span>
        <span class="count-badge">{{ pending.length }} 条</span>
      </div>
      <div class="section-body">
        <div v-for="p in pending" :key="p.id" class="person-row">
          <el-avatar :size="36" class="person-avatar apprentice">{{ p.username?.charAt(0)?.toUpperCase() }}</el-avatar>
          <div class="person-info">
            <div class="person-name">{{ p.name || p.username }}</div>
            <div class="person-meta">{{ formatTime(p.createdAt) }}</div>
          </div>
          <el-button size="small" type="success" @click="approveRequest(p.id)">通过</el-button>
          <el-button size="small" type="danger" @click="rejectRequest(p.id)">拒绝</el-button>
        </div>
      </div>
    </div>

    <!-- ========== Apprentice progress timeline ========== -->
    <div v-if="!isExternal" class="section-card mentorship-timeline-card" style="margin-bottom:20px">
      <div class="section-header timeline-header">
        <div>
          <div>徒弟成长时间轴</div>
          <div class="section-hint">横向拖动查看全部关卡与任务，头像所在任务位置就是徒弟的进度</div>
        </div>
        <span class="count-badge">{{ levels.length }} 个关卡 · {{ timelineTaskCount }} 个任务 · {{ apprentices.length }} 位徒弟</span>
      </div>
      <div v-if="levels.length" class="section-body timeline-body">
        <div
          ref="timelineViewport"
          class="timeline-viewport" tabindex="0" role="region" aria-label="徒弟关卡时间线，可左右滚动"
          :class="{ dragging: timelineDragging }"
          @pointerdown="startTimelineDrag"
          @pointermove="moveTimelineDrag"
          @pointerup="stopTimelineDrag"
          @pointercancel="stopTimelineDrag"
        >
          <div class="timeline-track" :style="{ minWidth: `${timelineWidth}px` }">
            <div
              class="timeline-apprentice-lanes"
              :style="{ minHeight: `${timelineApprenticeHeight}px` }"
            >
              <div
                v-for="row in apprenticeRows"
                :key="row.key"
                class="timeline-apprentice-lane"
                :style="{ width: `${timelineWidth}px`, height: `${row.height}px` }"
              >
                <div
                  v-for="group in row.groups"
                  :key="group.key"
                  class="timeline-progress-marker-stack"
                  :style="{ left: `${group.left}px` }"
                >
                  <el-tooltip
                    v-for="position in group.positions"
                    :key="position.userId"
                    :content="`${position.name || position.username} · 已完成 ${position.completedTaskCount}/${position.totalTaskCount} 个任务${position.currentTaskName ? ` · 当前：${position.currentTaskName}` : ''}`"
                    placement="top"
                  >
                    <div class="timeline-progress-marker">
                      <el-avatar :size="28" class="person-avatar apprentice">
                        {{ (position.name || position.username)?.charAt(0)?.toUpperCase() }}
                      </el-avatar>
                      <span>{{ position.name || position.username }}</span>
                    </div>
                  </el-tooltip>
                </div>
              </div>
            </div>

            <div class="timeline-axis" :style="{ width: `${timelineWidth}px` }">
              <div class="timeline-axis-line"></div>
              <div
                v-for="item in timelineLayout"
                :key="item.key"
                class="timeline-item"
                :class="item.type === 'level' ? 'timeline-level-item' : 'timeline-task-item'"
              >
                <template v-if="item.type === 'level'">
                  <div class="timeline-dot">{{ item.levelNumber }}</div>
                  <div class="timeline-level-name" :title="item.level.name">{{ item.level.name }}</div>
                  <div class="timeline-level-map">地图 {{ item.level.mapId || '—' }}</div>
                </template>
                <template v-else>
                  <el-tooltip :content="taskTooltip(item.task)" placement="top">
                    <span class="timeline-task-tick" :class="taskTickClass(item.task.id)"></span>
                  </el-tooltip>
                  <div class="timeline-task-index">任务 {{ item.taskNumber }}</div>
                </template>
              </div>
            </div>
          </div>
        </div>
        <div class="timeline-legend">
          <span><i class="legend-dot apprentice-dot"></i>徒弟当前位置</span>
          <span class="timeline-task-legend">任务：<i class="legend-task pending-task"></i>未完成 <i class="legend-task partial-task"></i>部分完成 <i class="legend-task done-task"></i>全部完成</span>
          <span>共 {{ levels.length }} 个关卡</span>
        </div>
      </div>
      <el-empty v-else description="暂无关卡数据" :image-size="60" />
    </div>

    <!-- ========== Mentors / Apprentices summary (游客不可见) ========== -->
    <div v-if="!isExternal" class="card-grid cols-2" style="margin-bottom:20px">
      <div class="section-card">
        <div class="section-header">
          <span>我的师傅</span>
          <span class="count-badge">{{ mentors.length }} 位</span>
        </div>
        <div class="section-body">
          <div v-for="m in mentors" :key="m.id" class="person-row">
            <el-avatar :size="32" class="person-avatar">{{ (m.name || m.username)?.charAt(0)?.toUpperCase() }}</el-avatar>
            <div class="person-info">
              <div class="person-name">{{ m.name || m.username }} <span class="level-badge">Lv.{{ m.level }} {{ levelShort(m.level) }}</span></div>
              <div class="person-meta">拜师于 {{ formatTime(m.createdAt) }}</div>
            </div>
            <el-button size="small" type="danger" plain @click="quitMentor(m)">退出师门</el-button>
          </div>
          <el-empty v-if="!mentors.length" description="还没有师傅" :image-size="50" />
        </div>
      </div>
      <div class="section-card">
        <div class="section-header">
          <span>我的徒弟</span>
          <span class="count-badge">{{ apprentices.length }} 位</span>
          <el-button size="small" type="primary" plain @click="copyInviteLink">🔗 复制邀请链接</el-button>
        </div>
        <div class="section-body apprentice-list">
          <div class="invite-tip" v-if="inviteLink">把链接发给新人，注册后自动成为你的徒弟</div>
          <div v-for="a in apprentices" :key="a.id" class="person-row">
            <el-avatar :size="32" class="person-avatar apprentice">{{ (a.name || a.username)?.charAt(0)?.toUpperCase() }}</el-avatar>
            <div class="person-info">
              <div class="person-name">{{ a.name || a.username }} <span class="level-badge">Lv.{{ a.level }} {{ levelShort(a.level) }}</span></div>
              <div class="person-meta">入门于 {{ formatTime(a.createdAt) }}</div>
            </div>
          </div>
          <el-empty v-if="!apprentices.length" description="还没有徒弟，复制邀请链接拉新人入门" :image-size="50" />
        </div>
      </div>
    </div>

    <section class="section-card submissions-card" aria-label="提交记录">
      <el-tabs v-model="subTab" @tab-change="onSubTabChange">
        <el-tab-pane v-for="group in visibleLists" :key="group.key" :label="group.label" :name="group.key">
          <p v-if="group.key === 'external'" class="external-tip">你可以审核等级低于你的外部成员提交的作业。审核完成后，仅审核人可以在这里查看记录。</p>
          <div v-loading="group.loading" class="submissions-panel" :aria-busy="group.loading">
            <p v-if="group.failed" class="submissions-empty" role="alert">
              提交记录加载失败。<el-button text type="primary" @click="loadSubmissions(group)">重试</el-button>
            </p>
            <template v-else-if="group.rows.length">
              <table class="submissions-table" :aria-label="group.label">
                <thead><tr>
                  <th scope="col">任务</th>
                  <th v-if="group.key !== 'my'" scope="col">提交人</th>
                  <th scope="col">状态</th>
                  <th scope="col">提交时间</th>
                  <th v-if="group.key === 'my'" scope="col">反馈</th>
                  <th scope="col">操作</th>
                </tr></thead>
                <tbody>
                  <tr v-for="row in group.rows" :key="row.id">
                    <td class="submission-title">{{ row.taskTitle }}</td>
                    <td v-if="group.key !== 'my'" class="submission-author">{{ row.name || row.username }} <small>提交 #{{ row.id }}</small></td>
                    <td class="submission-status"><span class="status-tag" :class="row.status?.toLowerCase()">{{ submissionStatusText(row.status) }}</span></td>
                    <td class="submission-date">{{ formatTime(row.submittedAt) }}</td>
                    <td v-if="group.key === 'my'" class="submission-feedback" :class="{ 'has-feedback': row.reviewComment || row.aiFeedback }">
                      <p v-if="row.reviewComment">审核评语：{{ row.reviewComment }}</p>
                      <p v-if="row.aiFeedback">AI 反馈：{{ row.aiFeedback }}</p>
                      <span v-if="!row.reviewComment && !row.aiFeedback">—</span>
                    </td>
                    <td class="submission-action">
                      <el-button size="small" type="primary" text @click="openReview(row)">{{ group.key !== 'my' && awaitingReview(row.status) ? '审核' : '查看' }}</el-button>
                    </td>
                  </tr>
                </tbody>
              </table>
              <el-pagination v-if="group.total > 10" class="submission-pages" :current-page="group.page"
                             :page-size="10" :total="group.total" :pager-count="5" layout="prev, pager, next"
                             @current-change="page => loadSubmissions(group, page)" background small />
            </template>
            <p v-else-if="!group.loading" class="submissions-empty">{{ group.empty }}</p>
          </div>
        </el-tab-pane>
      </el-tabs>
    </section>

    <!-- Review Dialog -->
    <el-dialog v-model="reviewVisible" :title="awaitingReview(reviewSub?.status) && !isOwnSubmission && subTab !== 'my' ? '审核提交' : '提交详情'" width="min(800px, calc(100% - 24px))" top="3vh">
      <div v-if="reviewSub" class="review-detail">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="提交 ID">{{ reviewSub.id }}</el-descriptions-item>
          <el-descriptions-item label="用户">{{ reviewSub.name || reviewSub.username }}</el-descriptions-item>
          <el-descriptions-item label="任务">{{ reviewSub.taskTitle }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <span class="status-tag" :class="reviewSub.status?.toLowerCase()">{{ submissionStatusText(reviewSub.status) }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="提交时间">{{ formatTime(reviewSub.submittedAt) }}</el-descriptions-item>
          <el-descriptions-item label="审核时间">{{ reviewSub.reviewedAt ? formatTime(reviewSub.reviewedAt) : '-' }}</el-descriptions-item>
        </el-descriptions>

        <div class="sub-section">
          <div class="sub-section-title">提交内容</div>
          <div class="sub-md-preview" v-html="reviewMdHtml"></div>
        </div>

        <div v-if="reviewSub.aiFeedback" class="sub-section ai-feedback">
          <div class="sub-section-title">AI 评审意见</div>
          <div class="sub-ai-text">{{ reviewSub.aiFeedback }}</div>
        </div>

        <div v-if="reviewSub.reviewComment" class="sub-section">
          <div class="sub-section-title">审核评语</div>
          <p>{{ reviewSub.reviewComment }}</p>
        </div>
        <div class="sub-section" v-if="!isOwnSubmission && awaitingReview(reviewSub.status)">
          <div class="sub-section-title">审核操作</div>
          <div class="sub-actions">
            <el-input v-model="reviewComment" placeholder="评语" size="small" style="width:200px" />
            <el-button type="success" size="small" @click="doReview('approved')" :loading="reviewing">通过</el-button>
            <el-button type="danger" size="small" @click="doReview('rejected')" :loading="reviewing">拒绝</el-button>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { submissionStatusText, awaitingReview } from '../submissionStatus'
import { ref, reactive, onMounted, computed } from 'vue'
import { mentorshipApi } from '../api/mentorship'
import { skillsApi } from '../api/skills'
import { tasksApi } from '../api/tasks'
import { useAuthStore } from '../store/auth'
import { ElMessage, ElMessageBox } from 'element-plus'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const authStore = useAuthStore()
const loading = ref(true)
const mentors = ref([])
const apprentices = ref([])
const pending = ref([])
const sentRequests = ref([])
const inviteLink = ref('')
const quitting = ref(null)
const levels = ref([])
const timelineViewport = ref(null)
const timelineDragging = ref(false)
const timelineDragStartX = ref(0)
const timelineScrollStart = ref(0)

// Apply for mentor
const mentorId = ref(null)
const requestLoading = ref(false)
const cancelling = ref(null)
const searchLoading = ref(false)
const searchResults = ref([])

// Submissions
const subTab = ref('my')
const isInternal = computed(() => ['INTERNAL', 'TEACHER', 'SUPERADMIN'].includes(authStore.user?.role))
const isExternal = computed(() => authStore.user?.role === 'EXTERNAL')
function submissionList(key, label, empty, fetch) {
  return reactive({ key, label, empty, fetch, rows: [], page: 1, total: 0, loading: false, failed: false, request: 0 })
}
const submissionLists = {
  my: submissionList('my', '我的提交', '暂无提交记录', tasksApi.mySubmissions),
  apprentice: submissionList('apprentice', '徒弟提交', '暂无徒弟提交', tasksApi.apprenticeSubmissions),
  external: submissionList('external', '外部成员提交', '暂无外部成员提交', tasksApi.externalSubmissions)
}
const visibleLists = computed(() => {
  const lists = [submissionLists.my]
  if (apprentices.value.length) lists.push(submissionLists.apprentice)
  if (isInternal.value) lists.push(submissionLists.external)
  return lists
})

const timelineLevels = computed(() => [...levels.value].sort((a, b) => {
  const mapDiff = (a.mapId ?? Number.MAX_SAFE_INTEGER) - (b.mapId ?? Number.MAX_SAFE_INTEGER)
  return mapDiff || (a.orderIndex ?? 0) - (b.orderIndex ?? 0) || a.id - b.id
}))

const timelineLayout = computed(() => {
  const result = []
  let left = 20
  let taskNumber = 0
  timelineLevels.value.forEach((level, levelIndex) => {
    const width = 132
    result.push({
      key: `level-${level.id}`, type: 'level', level, levelNumber: levelIndex + 1,
      left, center: left + width / 2, width
    })
    left += width
    ;(level.tasks || []).forEach(task => {
      const taskWidth = 76
      taskNumber += 1
      result.push({
        key: `task-${task.id}`, type: 'task', task, taskNumber,
        left, center: left + taskWidth / 2, width: taskWidth
      })
      left += taskWidth
    })
  })
  return result
})

const timelineWidth = computed(() => {
  const last = timelineLayout.value[timelineLayout.value.length - 1]
  return Math.max((last?.left || 0) + (last?.width || 0) + 20, 760)
})

const timelineTaskCount = computed(() =>
  timelineLevels.value.reduce((total, level) => total + (level.tasks?.length || 0), 0)
)

const apprenticePositions = computed(() => apprentices.value.map(apprentice => {
  const completedTaskIds = new Set(
    (apprentice.taskProgress || [])
      .filter(progress => progress.completed)
      .map(progress => progress.taskId)
  )
  const taskItems = timelineLayout.value.filter(item => item.type === 'task')
  const completedTaskItems = taskItems.filter(item => completedTaskIds.has(item.task.id))
  const lastCompleted = completedTaskItems[completedTaskItems.length - 1]
  const firstLevel = timelineLayout.value.find(item => item.type === 'level')
  const currentTask = taskItems.find(item => !completedTaskIds.has(item.task.id))
  const marker = lastCompleted || firstLevel
  return {
    ...apprentice,
    left: marker?.center || 20,
    markerKey: marker?.key || 'level-start',
    completedTaskCount: completedTaskIds.size,
    totalTaskCount: taskItems.length,
    currentTaskName: currentTask?.task?.title || ''
  }
}))

const apprenticeRows = computed(() => {
  const grouped = new Map()
  apprenticePositions.value.forEach(position => {
    const row = grouped.get(position.markerKey) || {
      key: position.markerKey,
      left: position.left,
      positions: []
    }
    row.positions.push(position)
    grouped.set(position.markerKey, row)
  })
  const groups = [...grouped.values()]
  return groups.length ? [{
    key: 'apprentices',
    groups,
    height: Math.max(42, ...groups.map(group => group.positions.length * 36))
  }] : []
})

const timelineApprenticeHeight = computed(() =>
  Math.max(apprenticeRows.value.reduce((total, row) => total + row.height, 0), 42)
)

function isTaskCompleted(apprentice, taskId) {
  return apprentice.taskProgress?.some(progress =>
    progress.taskId === taskId && progress.completed
  ) || false
}

function taskCompletionCount(taskId) {
  return apprentices.value.filter(apprentice => isTaskCompleted(apprentice, taskId)).length
}

function taskTickClass(taskId) {
  const completed = taskCompletionCount(taskId)
  return {
    completed: apprentices.value.length > 0 && completed === apprentices.value.length,
    partial: completed > 0 && completed < apprentices.value.length
  }
}

function taskTooltip(task) {
  if (!apprentices.value.length) return `${task.title} · 暂无徒弟进度`
  const statuses = apprentices.value.map(apprentice =>
    `${apprentice.name || apprentice.username}：${isTaskCompleted(apprentice, task.id) ? '已完成' : '未完成'}`
  )
  return `${task.title} · ${statuses.join(' / ')}`
}

// Review
const reviewVisible = ref(false)
const reviewSub = ref(null)
const reviewComment = ref('')
const reviewing = ref(false)

const reviewMdHtml = computed(() => {
  if (!reviewSub.value?.submissionData) return ''
  return DOMPurify.sanitize(marked.parse(reviewSub.value.submissionData))
})

/** 徒弟自己查看自己的提交时，隐藏审核操作 */
const isOwnSubmission = computed(() =>
  !!authStore.user?.id && reviewSub.value?.userId === authStore.user.id
)

function roleText(r) { return { SUPERADMIN: '超级管理员', TEACHER:'教师',INTERNAL:'内部',EXTERNAL:'游客' }[r]||r }
function levelShort(lv) {
  if (lv >= 70) return '👑王者'
  if (lv >= 60) return '🏆宗师'
  if (lv >= 50) return '⭐大师'
  if (lv >= 40) return '💎钻石'
  if (lv >= 30) return '🔮铂金'
  if (lv >= 20) return '🥇黄金'
  if (lv >= 10) return '🥈白银'
  return '🥉青铜'
}

function formatTime(ts) {
  if (!ts) return '-'
  const d = new Date(ts)
  return d.getFullYear() + '-' + String(d.getMonth()+1).padStart(2,'0') + '-' + String(d.getDate()).padStart(2,'0') + ' ' + String(d.getHours()).padStart(2,'0') + ':' + String(d.getMinutes()).padStart(2,'0') + ':' + String(d.getSeconds()).padStart(2,'0')
}

function onSubTabChange(key) {
  loadSubmissions(submissionLists[key])
}

function startTimelineDrag(event) {
  if (!timelineViewport.value) return
  timelineDragging.value = true
  timelineDragStartX.value = event.clientX
  timelineScrollStart.value = timelineViewport.value.scrollLeft
  timelineViewport.value.setPointerCapture?.(event.pointerId)
}

function moveTimelineDrag(event) {
  if (!timelineDragging.value || !timelineViewport.value) return
  timelineViewport.value.scrollLeft = timelineScrollStart.value - (event.clientX - timelineDragStartX.value)
}

function stopTimelineDrag(event) {
  if (!timelineDragging.value) return
  timelineDragging.value = false
  if (event?.pointerId != null) timelineViewport.value?.releasePointerCapture?.(event.pointerId)
}

// ---- Search users ----
let searchTimer = null
function searchUsers(query) {
  if (!query) { searchResults.value = []; return }
  clearTimeout(searchTimer)
  searchTimer = setTimeout(async () => {
    searchLoading.value = true
    try {
      const res = await mentorshipApi.searchUsers(query)
      const me = authStore.user
      const apprenticeIds = new Set(apprentices.value.map(a => a.userId))
      const mentorIds = new Set(mentors.value.map(m => m.userId))
      const sentTargetIds = new Set(sentRequests.value.map(s => s.userId))
      const pendingTargetIds = new Set(pending.value.map(p => p.userId))
      // 过滤：自己、自己的徒弟、已是自己师傅的人、已发过申请/对方待审批的人，不显示拜师按钮
      searchResults.value = (res.data.data || []).filter(u => {
        if (me && u.id === me.id) return false
        if (apprenticeIds.has(u.id)) return false
        if (mentorIds.has(u.id)) return false
        if (sentTargetIds.has(u.id)) return false
        if (pendingTargetIds.has(u.id)) return false
        return true
      })
    } catch (e) { searchResults.value = [] } finally { searchLoading.value = false }
  }, 200)
}

// ---- Fetch ----
async function fetchData() {
  loading.value = true
  try {
    const res = await mentorshipApi.my()
    mentors.value = res.data.data?.mentors || []
    apprentices.value = res.data.data?.apprentices || []
    pending.value = res.data.data?.pending || []
    sentRequests.value = res.data.data?.sentRequests || []
    if (!isExternal.value) await fetchTimeline()
  } catch (e) { console.error(e) } finally { loading.value = false }
}

async function fetchTimeline() {
  try {
    const res = await skillsApi.getTree()
    levels.value = res.data.data || []
  } catch (e) {
    levels.value = []
    console.error(e)
  }
}

async function loadSubmissions(group, page = group.page) {
  const request = ++group.request
  group.page = page
  group.loading = true
  group.failed = false
  try {
    const res = await group.fetch(page - 1, 10)
    if (request !== group.request) return
    const { content, total } = res.data.data
    const lastPage = Math.max(1, Math.ceil(total / 10))
    if (page > lastPage) return loadSubmissions(group, lastPage)
    group.rows = content
    group.total = total
  } catch {
    if (request === group.request) group.failed = true
  } finally {
    if (request === group.request) group.loading = false
  }
}

// ---- Mentor request ----
async function requestMentor() {
  if (!mentorId.value) return
  requestLoading.value = true
  try {
    await mentorshipApi.request(mentorId.value)
    ElMessage.success('申请已发送')
    mentorId.value = null
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '申请失败')
  } finally { requestLoading.value = false }
}

async function approveRequest(id) {
  try { await mentorshipApi.approve(id); ElMessage.success('已通过'); fetchData() } catch (e) { ElMessage.error('操作失败') }
}
async function rejectRequest(id) {
  try { await mentorshipApi.reject(id); ElMessage.success('已拒绝'); fetchData() } catch (e) { ElMessage.error('操作失败') }
}
async function cancelRequest(id) {
  cancelling.value = id
  try {
    await mentorshipApi.cancel(id)
    ElMessage.success('已撤回')
    await fetchData()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '撤回失败')
  } finally { cancelling.value = null }
}

// ---- Quit mentorship (apprentice) ----
async function quitMentor(m) {
  try {
    await ElMessageBox.confirm(`确定要退出 ${m.name || m.username} 的师门吗？退出后将解除师徒关系。`, '退出师门', {
      type: 'warning', confirmButtonText: '退出', cancelButtonText: '取消'
    })
  } catch (e) { return }
  quitting.value = m.id
  try {
    await mentorshipApi.quit(m.id)
    ElMessage.success('已退出师门')
    await fetchData()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '操作失败')
  } finally { quitting.value = null }
}

// ---- Invite link (mentor) ----
async function copyInviteLink() {
  let token
  try {
    const res = await mentorshipApi.inviteLink()
    token = res.data.data
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '生成邀请链接失败')
    return
  }
  const link = `${location.origin}/register?invite=${token}`
  const ok = () => {
    inviteLink.value = link
    ElMessage.success('邀请链接已复制，7 天内有效')
  }
  if (navigator.clipboard?.writeText) {
    navigator.clipboard.writeText(link).then(ok).catch(() => fallbackCopy(link, ok))
  } else {
    fallbackCopy(link, ok)
  }
}

function fallbackCopy(text, ok) {
  const ta = document.createElement('textarea')
  ta.value = text
  ta.style.position = 'fixed'
  ta.style.opacity = '0'
  document.body.appendChild(ta)
  ta.select()
  try { document.execCommand('copy'); ok() } catch (e) { ElMessage.error('复制失败，请手动复制') }
  document.body.removeChild(ta)
}

// ---- Review ----
async function openReview(row) {
  reviewVisible.value = true
  reviewSub.value = null
  reviewComment.value = ''
  try {
    const res = await tasksApi.submissionDetail(row.id)
    reviewSub.value = res.data.data
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '加载详情失败')
  }
}

async function doReview(status) {
  reviewing.value = true
  try {
    await tasksApi.review(reviewSub.value.id, { status, reviewComment: reviewComment.value || undefined })
    ElMessage.success(status === 'approved' ? '已通过' : '已拒绝')
    reviewVisible.value = false
    await loadSubmissions(submissionLists[subTab.value])
    if (subTab.value === 'apprentice') await fetchData()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '审核失败')
  } finally { reviewing.value = false }
}

onMounted(async () => {
  await Promise.all([fetchData(), loadSubmissions(submissionLists.my)])
  if (apprentices.value.length) {
    await loadSubmissions(submissionLists.apprentice)
    if (subTab.value === 'my' && submissionLists.apprentice.rows.some(row => awaitingReview(row.status))) {
      subTab.value = 'apprentice'
    }
  }
})
</script>

<style scoped>
.page { max-width: 1000px; }
.count-badge { font-size: 12px; color: var(--color-text-secondary); background: #edf0e9; padding: 2px 10px; border-radius: 10px; }

.person-row { display: flex; align-items: center; gap: 12px; padding: 10px 0; border-bottom: 1px solid var(--color-border); }
.person-row:last-child { border-bottom: none; }
.person-avatar { background: linear-gradient(135deg, #216354, #397c67); color: #fff; font-weight: 600; font-size: 13px; }
.person-avatar.apprentice { background: linear-gradient(135deg, #3b825f, #30684b); }
.person-info { flex: 1; }
.person-name { font-size: 14px; font-weight: 500; color: var(--color-text); }
.level-badge { font-size: 11px; font-weight: 600; color: #216354; background: #edf5ee; padding: 2px 6px; border-radius: 4px; margin-left: 6px; }

.request-row { display: flex; gap: 12px; align-items: center; }

.person-meta { font-size: 12px; color: #667568; margin-top: 2px; }
.invite-tip { font-size: 12px; color: #216354; background: #edf5ee; padding: 8px 12px; border-radius: 8px; margin-bottom: 10px; }
.apprentice-list { max-height: 400px; overflow-y: auto; }

/* Apprentice progress timeline */
.timeline-header { gap: 16px; }
.section-hint { margin-top: 4px; font-size: 12px; font-weight: 400; color: #667568; }
.timeline-body { padding-top: 20px; }
.timeline-viewport {
  overflow-x: auto; overflow-y: hidden; padding: 14px 20px 14px;
  cursor: grab; user-select: none; touch-action: pan-x;
  scrollbar-color: #bed2bf #f5f6f0; scrollbar-width: thin;
}
.timeline-viewport.dragging { cursor: grabbing; }
.timeline-track { position: relative; }
.timeline-apprentice-lanes { position: relative; }
.timeline-apprentice-lane { position: relative; }
.timeline-progress-marker-stack {
  position: absolute; top: 4px; transform: translateX(-50%);
  display: flex; flex-direction: column; align-items: flex-start; gap: 4px;
}
.timeline-progress-marker {
  display: flex; align-items: center; gap: 4px; max-width: 144px;
  color: #30684b; font-size: 11px; font-weight: 600; white-space: nowrap;
}
.timeline-progress-marker span { overflow: hidden; text-overflow: ellipsis; }
.timeline-axis { position: relative; display: flex; height: 128px; align-items: flex-start; }
.timeline-axis-line {
  position: absolute; left: 66px; right: 66px; top: 27px; height: 4px;
  border-radius: 4px; background: linear-gradient(90deg, #bed2bf, #cfddc9, #bed2bf);
}
.timeline-item {
  position: relative; flex: 0 0 auto; height: 128px;
  display: flex; flex-direction: column; align-items: center; text-align: center;
}
.timeline-level-item { width: 132px; }
.timeline-task-item { width: 76px; }
.timeline-dot {
  position: relative; z-index: 1; width: 30px; height: 30px; margin: 12px auto 14px;
  display: flex; align-items: center; justify-content: center; border-radius: 50%;
  color: #216354; background: #fff; border: 3px solid #a0bfac;
  box-shadow: 0 2px 7px rgba(29,82,61,.14); font-size: 11px; font-weight: 700;
}
.timeline-task-tick {
  width: 8px; height: 15px; display: inline-block; border-radius: 3px;
  background: #bfcbbf; box-shadow: inset 0 -2px 0 rgba(100,116,139,.18);
}
.timeline-task-item .timeline-task-tick { margin: 20px 0 20px; }
.timeline-task-tick.partial { background: #d9a753; box-shadow: inset 0 -2px 0 rgba(180,83,9,.2); }
.timeline-task-tick.completed { background: #3b825f; box-shadow: inset 0 -2px 0 rgba(21,128,61,.25); }
.timeline-level-name {
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  padding: 0 8px; color: var(--color-text); font-size: 13px; font-weight: 600;
}
.timeline-task-index { color: #667568; font-size: 10px; white-space: nowrap; }
.timeline-level-map { margin-top: 5px; color: #667568; font-size: 11px; }
.timeline-legend {
  display: flex; justify-content: space-between; align-items: center; margin-top: 2px;
  color: #667568; font-size: 12px;
}
.timeline-legend span:first-child { display: inline-flex; align-items: center; gap: 6px; color: #58685d; }
.legend-dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; }
.apprentice-dot { background: #3b825f; box-shadow: 0 0 0 3px #e0efe6; }
.timeline-task-legend { display: inline-flex; align-items: center; gap: 4px; }
.legend-task { width: 7px; height: 11px; border-radius: 2px; display: inline-block; margin-left: 3px; }
.pending-task { background: #bfcbbf; }
.partial-task { background: #d9a753; }
.done-task { background: #3b825f; }

.status-tag { font-size: 11px; font-weight: 500; padding: 2px 8px; border-radius: 4px; white-space: nowrap; }
.status-tag.approved { background: #e0efe6; color: #30684b; }
.status-tag.rejected { background: #fee2e2; color: #dc2626; }
.status-tag.pending { background: #faf0d7; color: #895b19; }
.status-tag.ai_reviewed { background: #e0f2fe; color: #0284c7; }
.submissions-card { padding: 4px 22px 18px; min-width: 0; }
.submissions-card :deep(.el-tabs__header) { margin-bottom: 12px; }
.submissions-card :deep(.el-tabs__item) { height: 48px; }
.submissions-panel { min-height: 76px; }
.submissions-table { width: 100%; border-collapse: collapse; font-size: 14px; text-align: left; overflow-wrap: anywhere; }
.submissions-table th, .submissions-table td { padding: 12px 10px; }
.submissions-table th { background: var(--el-fill-color-light); font-size: 13px; font-weight: 600; color: var(--color-text-secondary); }
.submissions-table td { border-bottom: 1px solid var(--color-border); vertical-align: middle; }
.submission-title { width: 27%; font-weight: 600; }
.submission-author small { display: block; color: var(--color-text-secondary); }
.submission-status, .submission-date, .submission-action { white-space: nowrap; }
.submission-feedback, .submission-date { font-size: 13px; color: var(--color-text-secondary); }
.submission-feedback { min-width: 8em; }
.submission-feedback p { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow: hidden; }
.submission-action .el-button { min-width: 44px; }
.submission-pages { justify-content: center; margin-top: 14px; }
.submissions-empty { padding: 28px 0; text-align: center; color: var(--color-text-secondary); font-size: 14px; }
.external-tip { color: var(--color-primary); font-size: 13px; padding: 8px 12px; margin-bottom: 12px; background: #edf5ee; border-radius: 8px; }
@media (max-width: 720px) {
  .submissions-card { padding: 2px 16px 8px; }
  .submissions-card :deep(.el-tabs__header) { margin-bottom: 4px; }
  .submissions-card :deep(.el-tabs__item) { padding-inline: 12px; }
  .submissions-card :deep(.el-tabs__item:first-child) { padding-left: 0; }
  .submissions-table, .submissions-table tbody { display: block; }
  .submissions-table thead { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
  .submissions-table tr { display: grid; grid-template-columns: minmax(0,1fr) auto; align-items: center; gap: 6px 10px; padding: 12px 0; border-bottom: 1px solid var(--color-border); }
  .submissions-table td { padding: 0; border: none; width: auto; }
  .submission-title { grid-area: 1 / 1; }
  .submission-status { grid-area: 1 / 2; justify-self: end; }
  .submission-author { grid-column: 1; font-size: 13px; }
  .submission-author small { display: inline; }
  .submission-feedback { grid-column: 1 / -1; min-width: 0; }
  .submission-feedback:not(.has-feedback) { display: none; }
  .submission-action { grid-area: 2 / 2; justify-self: end; }
  .submission-action .el-button { min-height: 34px; }
  .submission-pages { margin-block: 12px 6px; }
}

/* Review detail */
.review-detail { display: flex; flex-direction: column; gap: 16px; }
.sub-section { padding: 14px 16px; background: #f5f6f0; border-radius: 10px; }
.sub-section-title { font-size: 13px; font-weight: 600; color: var(--color-text); margin-bottom: 10px; }
.sub-md-preview { font-size: 14px; line-height: 1.8; color: #334155; }
.sub-md-preview :deep(img) { max-width: 100%; border-radius: 6px; border: 1px solid #dce2d7; }
.sub-md-preview :deep(pre) { background: #263b31; color: #dce2d7; padding: 12px; border-radius: 6px; overflow-x: auto; font-size: 13px; }
.sub-md-preview :deep(code) { font-size: 13px; background: #edf0e9; padding: 1px 4px; border-radius: 3px; }
.sub-md-preview :deep(pre code) { background: none; padding: 0; color: inherit; }
.ai-feedback { background: #f0f9ff; border: 1px solid #bae6fd; }
.sub-ai-text { font-size: 14px; color: #0c4a6e; white-space: pre-wrap; line-height: 1.7; }
.sub-actions { display: flex; gap: 8px; align-items: center; }

/* Search select dropdown */
.search-option { display: flex; justify-content: space-between; align-items: center; width: 100%; }
.search-option-name { font-weight: 500; color: #263b31; }
.search-option-meta { font-size: 12px; color: #667568; flex-shrink: 0; margin-left: 12px; }
.search-empty { padding: 12px; text-align: center; color: #667568; font-size: 13px; }

.mentor-search { width: min(320px,100%); }
.timeline-viewport:focus-visible { outline: 2px solid var(--el-color-primary); }
@media(max-width:720px) {
  .request-row,.sub-actions,.timeline-legend { flex-wrap: wrap; }
  .request-row .mentor-search { width: 100%; }
  .person-info { min-width: 0; overflow-wrap: anywhere; }
  .timeline-viewport { padding-inline: 8px; }
  .section-header { flex-wrap: wrap; gap: 8px; }
}
</style>
