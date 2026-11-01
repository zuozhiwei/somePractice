<template>
  <div class="page" v-loading="loading">
    <h1 class="page-title">{{ isExternal ? '我的提交记录' : '师徒关系' }}</h1>

    <!-- ========== Apply for mentor (compact, top; 外部成员不可拜师) ========== -->
    <div v-if="!isExternal" class="section-card" style="margin-bottom:20px">
      <div class="section-header">🔍 申请师傅</div>
      <div class="section-body">
        <div class="request-row">
          <el-select v-model="mentorId" filterable remote :remote-method="searchUsers"
                     :loading="searchLoading" clearable
                     placeholder="输入用户名或姓名搜索"
                     loading-text="搜索中..."
                     no-match-text="无匹配用户"
                     no-data-text="输入关键词开始搜索"
                     size="default" style="width:320px" value-key="id">
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
        <span>📤 我发出的申请</span>
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
          class="timeline-viewport"
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

    <!-- ========== Submissions tabs ========== -->
    <div class="section-card">
      <div class="section-body" style="padding:0">
        <el-tabs v-model="subTab" @tab-change="onSubTabChange">
          <el-tab-pane label="📝 我的提交" name="my" />
          <el-tab-pane v-if="apprentices.length" label="👨‍🎓 徒弟提交" name="apprentice" />
          <el-tab-pane v-if="isInternal" label="🌐 外部提交" name="external" />
        </el-tabs>
      </div>
    </div>

    <!-- My Submissions -->
    <div v-show="subTab === 'my'" class="section-card">
      <div class="section-body" style="padding:0">
        <el-table :data="mySubs" stripe empty-text="暂无提交记录">
          <el-table-column prop="taskTitle" label="任务" min-width="180" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <span class="status-tag" :class="row.status.toLowerCase()">{{ subStatusText(row.status) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="提交时间" width="160">
          <template #default="{ row }">{{ formatTime(row.submittedAt) }}</template>
        </el-table-column>
          <el-table-column prop="aiFeedback" label="AI 反馈" min-width="180">
            <template #default="{ row }">
              <span class="fb-text">{{ row.aiFeedback || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="reviewComment" label="评语" min-width="120">
            <template #default="{ row }">
              <span class="fb-text">{{ row.reviewComment || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" text @click="openMyDetail(row)">查看</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div style="display:flex;justify-content:center;padding:12px" v-if="myTotal > 10">
          <el-pagination v-model:current-page="myPage" :page-size="10"
                         layout="prev, pager, next" :total="myTotal"
                         @current-change="p => fetchMySubs(p - 1)" background small />
        </div>
      </div>
    </div>

    <!-- Apprentice Submissions (mentor review) -->
    <div v-show="subTab === 'apprentice'" class="section-card">
      <div class="section-body" style="padding:0">
        <el-table :data="apprenticeSubs" stripe empty-text="暂无徒弟提交">
          <el-table-column prop="id" label="ID" width="55" />
          <el-table-column label="用户" width="130">
            <template #default="{ row }">
              {{ row.name || row.username }}
            </template>
          </el-table-column>
          <el-table-column prop="taskTitle" label="任务" min-width="160" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <span class="status-tag" :class="row.status.toLowerCase()">{{ subStatusText(row.status) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="提交时间" width="160">
          <template #default="{ row }">{{ formatTime(row.submittedAt) }}</template>
        </el-table-column>
          <el-table-column label="操作" width="90" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" text @click="openReview(row)">审核</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div style="display:flex;justify-content:center;padding:12px" v-if="apprenticeTotal > 10">
          <el-pagination v-model:current-page="apprenticePage" :page-size="10"
                         layout="prev, pager, next" :total="apprenticeTotal"
                         @current-change="p => fetchApprenticeSubs(p - 1)" background small />
        </div>
      </div>
    </div>

    <!-- External Submissions (crowd review pool) -->
    <div v-show="subTab === 'external'" class="section-card">
      <div class="section-body" style="padding:0">
        <div class="external-tip">这里展示等级低于你的外部成员提交，任意一人审核即可通过/拒绝；已审核的提交仅审核者本人可见。</div>
        <el-table :data="externalSubs" stripe empty-text="暂无待审核的外部提交">
          <el-table-column prop="id" label="ID" width="55" />
          <el-table-column label="用户" width="130">
            <template #default="{ row }">
              {{ row.name || row.username }}
            </template>
          </el-table-column>
          <el-table-column prop="taskTitle" label="任务" min-width="160" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <span class="status-tag" :class="row.status.toLowerCase()">{{ subStatusText(row.status) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="提交时间" width="160">
            <template #default="{ row }">{{ formatTime(row.submittedAt) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="90" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" text @click="openReview(row)">审核</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div style="display:flex;justify-content:center;padding:12px" v-if="externalTotal > 10">
          <el-pagination v-model:current-page="externalPage" :page-size="10"
                         layout="prev, pager, next" :total="externalTotal"
                         @current-change="p => fetchExternalSubs(p - 1)" background small />
        </div>
      </div>
    </div>

    <!-- Review Dialog -->
    <el-dialog v-model="reviewVisible" title="审核提交" width="800px" top="3vh">
      <div v-if="reviewSub" class="review-detail">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="提交 ID">{{ reviewSub.id }}</el-descriptions-item>
          <el-descriptions-item label="用户">{{ reviewSub.name || reviewSub.username }}</el-descriptions-item>
          <el-descriptions-item label="任务">{{ reviewSub.taskTitle }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <span class="status-tag" :class="reviewSub.status.toLowerCase()">{{ subStatusText(reviewSub.status) }}</span>
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

        <div class="sub-section" v-if="!isOwnSubmission">
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
import { ref, onMounted, computed } from 'vue'
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
const mySubs = ref([])
const myPage = ref(1)
const myTotal = ref(0)
const apprenticeSubs = ref([])
const apprenticePage = ref(1)
const apprenticeTotal = ref(0)
const externalSubs = ref([])
const externalPage = ref(1)
const externalTotal = ref(0)

const isInternal = computed(() => ['INTERNAL', 'TEACHER'].includes(authStore.user?.role))
const isExternal = computed(() => authStore.user?.role === 'EXTERNAL')

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

function roleText(r) { return { TEACHER:'教师',INTERNAL:'内部',EXTERNAL:'游客' }[r]||r }
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
function subStatusText(s) { return { APPROVED:'已通过',REJECTED:'未通过',PENDING:'审核中',AI_REVIEWED:'AI已评' }[s]||s }

function formatTime(ts) {
  if (!ts) return '-'
  const d = new Date(ts)
  return d.getFullYear() + '-' + String(d.getMonth()+1).padStart(2,'0') + '-' + String(d.getDate()).padStart(2,'0') + ' ' + String(d.getHours()).padStart(2,'0') + ':' + String(d.getMinutes()).padStart(2,'0') + ':' + String(d.getSeconds()).padStart(2,'0')
}

function onSubTabChange(tab) {
  if (tab === 'apprentice') fetchApprenticeSubs(0)
  if (tab === 'external') fetchExternalSubs(0)
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

async function fetchMySubs(page = 0) {
  try {
    const res = await tasksApi.mySubmissions(page, 10)
    const d = res.data.data
    mySubs.value = d.content || []
    myTotal.value = d.total || 0
  } catch (e) { console.error(e) }
}

async function fetchApprenticeSubs(page = 0) {
  try {
    const res = await tasksApi.apprenticeSubmissions(page, 10)
    const d = res.data.data
    apprenticeSubs.value = d.content || []
    apprenticeTotal.value = d.total || 0
  } catch (e) { console.error(e) }
}

async function fetchExternalSubs(page = 0) {
  try {
    const res = await tasksApi.externalSubmissions(page, 10)
    const d = res.data.data
    externalSubs.value = d.content || []
    externalTotal.value = d.total || 0
  } catch (e) { console.error(e) }
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

/** 徒弟查看自己某次提交的完整内容 */
async function openMyDetail(row) {
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
    if (subTab.value === 'external') {
      fetchExternalSubs(externalPage.value - 1)
    } else {
      await fetchApprenticeSubs(apprenticePage.value - 1)
      await fetchData()
    }
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '审核失败')
  } finally { reviewing.value = false }
}

onMounted(async () => {
  await fetchData()
  fetchMySubs(0)
  // 检查是否有待审核的徒弟提交，默认打开徒弟提交标签
  if (apprentices.value.length > 0) {
    await fetchApprenticeSubs(0)
    const pendingCount = apprenticeSubs.value.filter(s => s.status === 'PENDING').length
    if (pendingCount > 0) {
      subTab.value = 'apprentice'
    }
  }
  if (isInternal.value) {
    fetchExternalSubs(0)
  }
})
</script>

<style scoped>
.page { max-width: 1000px; }
.count-badge { font-size: 12px; color: var(--color-text-secondary); background: #f1f5f9; padding: 2px 10px; border-radius: 10px; }

.person-row { display: flex; align-items: center; gap: 12px; padding: 10px 0; border-bottom: 1px solid var(--color-border); }
.person-row:last-child { border-bottom: none; }
.person-avatar { background: linear-gradient(135deg, #6366f1, #8b5cf6); color: #fff; font-weight: 600; font-size: 13px; }
.person-avatar.apprentice { background: linear-gradient(135deg, #22c55e, #16a34a); }
.person-info { flex: 1; }
.person-name { font-size: 14px; font-weight: 500; color: var(--color-text); }
.level-badge { font-size: 11px; font-weight: 600; color: #6366f1; background: #eef2ff; padding: 2px 6px; border-radius: 4px; margin-left: 6px; }

.request-row { display: flex; gap: 12px; align-items: center; }

.person-meta { font-size: 12px; color: #94a3b8; margin-top: 2px; }
.invite-tip { font-size: 12px; color: #6366f1; background: #eef2ff; padding: 8px 12px; border-radius: 8px; margin-bottom: 10px; }
.apprentice-list { max-height: 400px; overflow-y: auto; }

/* Apprentice progress timeline */
.timeline-header { gap: 16px; }
.section-hint { margin-top: 4px; font-size: 12px; font-weight: 400; color: #94a3b8; }
.timeline-body { padding-top: 20px; }
.timeline-viewport {
  overflow-x: auto; overflow-y: hidden; padding: 14px 20px 14px;
  cursor: grab; user-select: none; touch-action: pan-x;
  scrollbar-color: #c7d2fe #f8fafc; scrollbar-width: thin;
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
  color: #16a34a; font-size: 11px; font-weight: 600; white-space: nowrap;
}
.timeline-progress-marker span { overflow: hidden; text-overflow: ellipsis; }
.timeline-axis { position: relative; display: flex; height: 128px; align-items: flex-start; }
.timeline-axis-line {
  position: absolute; left: 66px; right: 66px; top: 27px; height: 4px;
  border-radius: 4px; background: linear-gradient(90deg, #c7d2fe, #ddd6fe, #c7d2fe);
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
  color: #6366f1; background: #fff; border: 3px solid #a5b4fc;
  box-shadow: 0 2px 7px rgba(99,102,241,.2); font-size: 11px; font-weight: 700;
}
.timeline-task-tick {
  width: 8px; height: 15px; display: inline-block; border-radius: 3px;
  background: #cbd5e1; box-shadow: inset 0 -2px 0 rgba(100,116,139,.18);
}
.timeline-task-item .timeline-task-tick { margin: 20px 0 20px; }
.timeline-task-tick.partial { background: #fbbf24; box-shadow: inset 0 -2px 0 rgba(180,83,9,.2); }
.timeline-task-tick.completed { background: #22c55e; box-shadow: inset 0 -2px 0 rgba(21,128,61,.25); }
.timeline-level-name {
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  padding: 0 8px; color: var(--color-text); font-size: 13px; font-weight: 600;
}
.timeline-task-index { color: #94a3b8; font-size: 10px; white-space: nowrap; }
.timeline-level-map { margin-top: 5px; color: #94a3b8; font-size: 11px; }
.timeline-legend {
  display: flex; justify-content: space-between; align-items: center; margin-top: 2px;
  color: #94a3b8; font-size: 12px;
}
.timeline-legend span:first-child { display: inline-flex; align-items: center; gap: 6px; color: #64748b; }
.legend-dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; }
.apprentice-dot { background: #22c55e; box-shadow: 0 0 0 3px #dcfce7; }
.timeline-task-legend { display: inline-flex; align-items: center; gap: 4px; }
.legend-task { width: 7px; height: 11px; border-radius: 2px; display: inline-block; margin-left: 3px; }
.pending-task { background: #cbd5e1; }
.partial-task { background: #fbbf24; }
.done-task { background: #22c55e; }

.status-tag { font-size: 11px; font-weight: 500; padding: 2px 8px; border-radius: 4px; white-space: nowrap; }
.status-tag.approved { background: #dcfce7; color: #16a34a; }
.status-tag.rejected { background: #fee2e2; color: #dc2626; }
.status-tag.pending { background: #fef3c7; color: #d97706; }
.status-tag.ai_reviewed { background: #e0f2fe; color: #0284c7; }
.fb-text { font-size: 13px; color: #64748b; }
.external-tip { font-size: 12px; color: #6366f1; background: #eef2ff; padding: 8px 12px; margin: 12px; border-radius: 8px; }

/* Review detail */
.review-detail { display: flex; flex-direction: column; gap: 16px; }
.sub-section { padding: 14px 16px; background: #f8fafc; border-radius: 10px; }
.sub-section-title { font-size: 13px; font-weight: 600; color: var(--color-text); margin-bottom: 10px; }
.sub-md-preview { font-size: 14px; line-height: 1.8; color: #334155; }
.sub-md-preview :deep(img) { max-width: 100%; border-radius: 6px; border: 1px solid #e2e8f0; }
.sub-md-preview :deep(pre) { background: #1e293b; color: #e2e8f0; padding: 12px; border-radius: 6px; overflow-x: auto; font-size: 13px; }
.sub-md-preview :deep(code) { font-size: 13px; background: #f1f5f9; padding: 1px 4px; border-radius: 3px; }
.sub-md-preview :deep(pre code) { background: none; padding: 0; color: inherit; }
.ai-feedback { background: #f0f9ff; border: 1px solid #bae6fd; }
.sub-ai-text { font-size: 14px; color: #0c4a6e; white-space: pre-wrap; line-height: 1.7; }
.sub-actions { display: flex; gap: 8px; align-items: center; }

/* Search select dropdown */
.search-option { display: flex; justify-content: space-between; align-items: center; width: 100%; }
.search-option-name { font-weight: 500; color: #1e293b; }
.search-option-meta { font-size: 12px; color: #94a3b8; flex-shrink: 0; margin-left: 12px; }
.search-empty { padding: 12px; text-align: center; color: #94a3b8; font-size: 13px; }
</style>
