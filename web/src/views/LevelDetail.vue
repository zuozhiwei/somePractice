<template>
  <div class="page" v-loading="loading">
    <template v-if="level">
      <div class="page-title-row">
        <div>
          <nav class="crumb" aria-label="面包屑">
            <router-link class="crumb-link" to="/learn">学习地图</router-link>
            <span class="crumb-sep">/</span>
            <router-link v-if="level.mapId" class="crumb-link" :to="'/map/' + level.mapId">{{ level.mapName }}</router-link>
            <span class="crumb-sep">/</span>
            <span aria-current="page">{{ level.name }}</span>
          </nav>
          <div class="title-line">
            <h1 class="page-title">{{ level.name }}</h1>
            <el-tag :type="statusTagType" size="small" effect="light">{{ statusText(level.userStatus?.status) }}</el-tag>
            <el-tag v-if="level.tier" size="small" effect="plain">Lv.{{ level.tier }}</el-tag>
            <el-tag v-if="level.category" size="small" effect="plain" type="info">{{ level.category }}</el-tag>
            <span class="level-num-chip" v-if="levelNum">第 {{ levelNum }} 关</span>
          </div>
          <p v-if="level.prerequisiteNames?.length" class="prereq-hint">
            前置关卡 · {{ level.prerequisiteNames.join('、') }}
          </p>
        </div>
        <el-button @click="goBack" text>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" style="margin-right:4px"><path d="M19 12H5M12 19l-7-7 7-7"/></svg>
          返回
        </el-button>
      </div>

      <div class="level-summary">
        <div><span>任务进度</span><strong>{{ completedCount }} <small>/ {{ level.tasks?.length || 0 }}</small></strong></div>
        <el-progress :percentage="taskPercent" :stroke-width="6" :show-text="false" aria-label="已通过任务比例" />
      </div>

      <div class="card-grid cols-2 lesson-columns">
        <!-- ============ Tutorials Panel ============ -->
        <div class="section-card">
          <div class="section-header">
            <span>学习教程</span>
            <span class="count-badge">{{ level.tutorials?.length || 0 }} 篇</span>
          </div>
          <div class="section-body" style="padding:0">
            <div v-if="level.tutorials?.length">
              <div v-for="(t, idx) in level.tutorials" :key="t.id" class="tutorial-item">
                <button type="button" class="tutorial-header" :aria-expanded="!!openTutorials[idx]" @click="toggleTutorial(idx)">
                  <span class="tutorial-num">{{ idx + 1 }}</span>
                  <span class="tutorial-title">{{ t.title }}</span>
                  <svg class="tutorial-arrow" :class="{ open: openTutorials[idx] }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18"><polyline points="6 9 12 15 18 9"/></svg>
                </button>
                <div v-if="openTutorials[idx]" class="tutorial-body">
                  <TutorialReader :content="t.content" />
                </div>
              </div>
            </div>
            <el-empty v-else description="暂无教程" :image-size="80" />
          </div>
        </div>

        <!-- ============ Tasks Panel ============ -->
        <div class="section-card">
          <div class="section-header">
            <span>关卡任务</span>
            <span class="count-badge">
              {{ completedCount }} / {{ level.tasks?.length || 0 }}
            </span>
          </div>
          <div class="section-body" style="padding:0">
            <div v-for="(task, idx) in level.tasks" :key="task.id" :id="'task-' + task.id" class="task-item"
                 :class="{ expanded: expandedTask === task.id }">
              <button type="button" class="task-header" :aria-expanded="expandedTask === task.id" @click="expandedTask = expandedTask === task.id ? null : task.id">
                <span class="task-header-left">
                  <span class="task-num">
                    <span v-if="task.submissionStatus === 'APPROVED'" class="done-check">✓</span>
                    <span v-else>{{ idx + 1 }}</span>
                  </span>
                  <span>
                    <span class="task-title">{{ task.title }}</span>
                    <span class="task-tags">
                      <span class="status-tag" :class="task.submissionStatus.toLowerCase()">
                        {{ statusText(task.submissionStatus) }}
                      </span>
                      <span class="xp-tag">{{ task.xpReward }} XP</span>
                      <span v-if="task.acceptanceCriteria?.length" class="criteria-count">
                        {{ task.acceptanceCriteria.length }} 项验收标准
                      </span>
                    </span>
                  </span>
                </span>
                <svg class="task-arrow" :class="{ open: expandedTask === task.id }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16"><polyline points="6 9 12 15 18 9"/></svg>
              </button>
              <div v-if="expandedTask === task.id" class="task-body">
                <div class="task-description">
                  <div class="desc-label">任务说明</div>
                  <div class="desc-content">
                    <TutorialReader :content="task.description" />
                  </div>
                </div>
                <div v-if="task.acceptanceCriteria?.length" class="task-criteria">
                  <div class="desc-label">验收标准</div>
                  <div v-for="(c, i) in task.acceptanceCriteria" :key="i" class="criteria-item">
                    <span class="criteria-num">{{ i + 1 }}.</span>
                    <span>{{ c.criteria }}</span>
                    <span class="criteria-weight">权重 {{ c.weight }}%</span>
                  </div>
                </div>
                <div v-if="task.aiFeedback" class="ai-feedback-box">
                  <div class="ai-feedback-header">
                    <span>AI 评审结果</span>
                    <span class="ai-score">{{ extractScore(task.aiFeedback) }} 分</span>
                  </div>
                  <div class="ai-feedback-text">{{ extractFeedback(task.aiFeedback) }}</div>
                </div>
                <div class="task-actions">
                  <div class="task-status-info">
                    <span v-if="task.submissionStatus === 'APPROVED'" class="hint-text approved">你已完成此任务</span>
                    <span v-else-if="task.submissionStatus === 'PENDING'" class="hint-text pending">等待审核，可继续本关其他任务</span>
                    <span v-else-if="task.submissionStatus === 'REJECTED'" class="hint-text rejected">未通过，请重新提交</span>
                    <span v-else-if="task.submissionStatus === 'AI_REVIEWED'" class="hint-text ai-reviewed">AI 已评审，等待最终确认</span>

                  </div>
                  <div class="task-action-btns">
                    <el-button v-if="task.submissionStatus !== 'NONE'" size="default" round
                               @click.stop="openHistory(task)">
                      提交记录
                    </el-button>
                    <el-button type="primary" size="default" round
                               :disabled="task.submissionStatus === 'APPROVED' || level.userStatus?.status === 'LOCKED'"
                               @click.stop="openSubmit(task)">
                      {{ task.submissionStatus === 'REJECTED' ? '重新提交' : task.submissionStatus === 'APPROVED' ? '已完成' : task.submissionStatus === 'PENDING' ? '修改后重新提交' : '提交任务' }}
                    </el-button>
                  </div>
                </div>
              </div>
            </div>
            <div v-if="!level.tasks?.length" style="padding:40px 0">
              <el-empty description="暂无任务" :image-size="80" />
            </div>
          </div>
        </div>
      </div>

      <TaskSubmitDialog ref="submitDialog" :task="currentTask" @done="onTaskDone" />
      <SubmissionHistoryDialog ref="historyDialog" />
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive, computed, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { mapsApi } from '../api/maps'
import { useMapStore } from '../store/map'
import { ElMessage } from 'element-plus'
import TutorialReader from '../components/TutorialReader.vue'
import TaskSubmitDialog from '../components/TaskSubmitDialog.vue'
import SubmissionHistoryDialog from '../components/SubmissionHistoryDialog.vue'

const route = useRoute()
const router = useRouter()
const mapStore = useMapStore()
const level = ref(null)
const loading = ref(true)
const submitDialog = ref(null)
const currentTask = ref(null)
const historyDialog = ref(null)
const expandedTask = ref(null)
const openTutorials = reactive({})

const levelNum = computed(() => Number(route.query.n) || '')
const completedCount = computed(() => level.value?.tasks?.filter(task => task.submissionStatus === 'APPROVED').length || 0)
const taskPercent = computed(() => level.value?.tasks?.length ? Math.round(completedCount.value * 100 / level.value.tasks.length) : 0)

function goBack() {
  if (window.history.state?.back) router.back()
  else router.push(level.value?.mapId ? '/map/' + level.value.mapId : '/learn')
}
const statusTagType = computed(() => {
  const s = level.value?.userStatus?.status
  return { COMPLETED: 'success', LEARNING: 'primary', LOCKED: 'info' }[s] || 'info'
})

function statusText(s) {
  return { COMPLETED: '已通关', LEARNING: '挑战中', LOCKED: '未解锁', APPROVED: '已通过', REJECTED: '未通过', PENDING: '审核中', AI_REVIEWED: 'AI已评', NONE: '待提交' }[s] || '待提交'
}
function extractScore(fb) { const m = fb?.match(/AI评分:\s*(\d+)/); return m ? m[1] : '--' }
function extractFeedback(fb) { const idx = fb?.indexOf('\n'); return idx > -1 ? fb.substring(idx + 1).trim() : fb }
function toggleTutorial(idx) { openTutorials[idx] = !openTutorials[idx] }

function openSubmit(task) { currentTask.value = task; submitDialog.value.open() }
function openHistory(task) { historyDialog.value.open(task) }

async function onTaskDone() {
  const prevStatus = level.value?.userStatus?.status
  await fetchDetail()
  mapStore.refresh()
  const now = level.value?.userStatus?.status
  if (prevStatus !== 'COMPLETED' && now === 'COMPLETED') {
    ElMessage({
      message: '🎉 关卡完成！获得 ' + (level.value.tasks?.reduce((s, t) => s + (t.xpReward || 0), 0) || 0) + ' XP 奖励',
      type: 'success',
      duration: 4000
    })
  }
}

async function fetchDetail() {
  loading.value = true
  try {
    const res = await mapsApi.levelDetail(route.params.id)
    level.value = res.data.data
    if (level.value?.tutorials?.length) openTutorials[0] = true
  } catch (e) {
    const msg = e.response?.data?.message
    if (msg) ElMessage.warning(msg)
    router.push('/map')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await fetchDetail()
  const target = level.value?.tasks?.find(task => String(task.id) === route.query.task)
  if (target) {
    expandedTask.value = target.id
    await nextTick()
    document.getElementById('task-' + target.id)?.scrollIntoView({ block: 'nearest' })
  }
})
</script>

<style scoped>
.page { max-width: 1200px; }
.page-title-row { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }

.crumb { font-size: 13px; color: var(--color-text-secondary); margin-bottom: 8px; }
.crumb-link { cursor: pointer; }
.crumb-link:hover { color: #216354; }
.crumb-sep { margin: 0 6px; color: #bfcbbf; }

.title-line { display: flex; align-items: center; gap: 10px; }
.level-num-chip {
  font-size: 12px; font-weight: 600; color: #31634d;
  background: #f2f6ee; padding: 2px 10px; border-radius: 10px;
}
.prereq-hint { font-size: 13px; color: #936524; margin-top: 8px; }

.count-badge {
  font-size: 12px; color: var(--color-text-secondary);
  background: #edf0e9; padding: 2px 10px; border-radius: 10px;
}

/* ===== Tutorials ===== */
.tutorial-item { border-bottom: 1px solid var(--color-border); }
.tutorial-item:last-child { border-bottom: none; }
.tutorial-header {
  display: flex; align-items: center; gap: 12px;
  padding: 14px 20px;
  cursor: pointer;
  transition: background .15s;
  user-select: none;
}
.tutorial-header:hover { background: #f5f6f0; }
.tutorial-num {
  width: 26px; height: 26px; border-radius: 50%;
  background: linear-gradient(135deg, #216354, #397c67);
  color: #fff; font-size: 12px; font-weight: 700;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.tutorial-title { flex: 1; font-size: 14px; font-weight: 500; }
.tutorial-arrow { color: #667568; flex-shrink: 0; transition: transform .2s; }
.tutorial-arrow.open { transform: rotate(180deg); }
.tutorial-body { padding: 0 20px 20px; overflow-wrap: anywhere; }

/* ===== Tasks ===== */
.task-item { border-bottom: 1px solid var(--color-border); transition: background .15s; }
.task-item:last-child { border-bottom: none; }
.task-item:hover { background: #f7f7f2; }
.task-item.expanded { background: #f5f6f0; }
.task-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 14px 20px;
  cursor: pointer;
  user-select: none;
}
.task-header-left { display: flex; align-items: center; gap: 12px; flex: 1; min-width: 0; }
.task-num {
  width: 32px; height: 32px; border-radius: 8px;
  background: #edf0e9;
  display: flex; align-items: center; justify-content: center;
  font-size: 13px; font-weight: 700; color: #58685d;
  flex-shrink: 0;
}
.done-check { color: #fff; font-size: 16px; width: 100%; height: 100%; background: #3b825f; border-radius: 8px; display: flex; align-items: center; justify-content: center; }
.task-title { font-size: 14px; font-weight: 500; color: var(--color-text); }
.task-tags { display: flex; align-items: center; gap: 8px; margin-top: 4px; flex-wrap: wrap; }
.status-tag { font-size: 11px; font-weight: 500; padding: 2px 8px; border-radius: 4px; }
.status-tag.approved { background: #e0efe6; color: #30684b; }
.status-tag.rejected { background: #fee2e2; color: #dc2626; }
.status-tag.pending { background: #faf0d7; color: #895b19; }
.status-tag.ai_reviewed { background: #e0f2fe; color: #0284c7; }
.status-tag.none { background: #edf0e9; color: #667568; }
.xp-tag { font-size: 11px; font-weight: 600; color: #895b19; background: #faf0d7; padding: 2px 8px; border-radius: 4px; }
.criteria-count { font-size: 11px; color: #667568; }
.task-arrow { color: #667568; flex-shrink: 0; transition: transform .2s; }
.task-arrow.open { transform: rotate(180deg); }
.task-body { padding-top: 4px; overflow-wrap: anywhere; }
.task-description, .task-criteria { padding: 0 20px 16px; }
.desc-label { font-size: 13px; font-weight: 600; color: var(--color-text); margin-bottom: 8px; }
.desc-content { font-size: 14px; color: #3e5147; line-height: 1.8; }
.criteria-item { display: flex; align-items: baseline; gap: 6px; font-size: 13px; color: #3e5147; padding: 4px 0; }
.criteria-num { color: #216354; font-weight: 600; min-width: 20px; }
.criteria-weight { font-size: 11px; color: #667568; margin-left: auto; }
.task-actions {
  flex-wrap: wrap; gap: 10px;
  display: flex; align-items: center; justify-content: space-between;
  padding: 14px 20px;
  border-top: 1px solid var(--color-border);
  background: #fff;
}
.task-action-btns { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
.hint-text { font-size: 13px; color: #667568; }
.hint-text.approved { color: #30684b; font-weight: 500; }
.hint-text.pending { color: #895b19; font-weight: 500; }
.hint-text.rejected { color: #dc2626; font-weight: 500; }
.hint-text.ai-reviewed { color: #0284c7; font-weight: 500; }
.ai-feedback-box { margin: 0 20px 0; padding: 14px 16px; background: linear-gradient(135deg, #eff6ff, #f0f9ff); border: 1px solid #bae6fd; border-radius: 10px; }
.ai-feedback-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; font-size: 13px; font-weight: 600; color: #0369a1; }
.ai-score { font-size: 18px; font-weight: 700; color: #0284c7; }
.ai-feedback-text { font-size: 13px; color: #3e5147; line-height: 1.7; white-space: pre-wrap; }

.crumb-link { color: var(--color-text-secondary); text-decoration: none; }
.crumb-link:hover { text-decoration: underline; }
.page-title-row { gap: 10px; }
.title-line { flex-wrap: wrap; }
.title-line h1 { margin: 0; }
.lesson-columns { align-items: start; }
.lesson-columns > * { min-width: 0; }
.level-summary { padding: 16px 20px; border: 1px solid #cbdace; background: #edf3e9; border-radius: var(--radius); margin-bottom: 20px; }
.level-summary > div:first-child { display: flex; align-items: baseline; justify-content: space-between; margin-bottom: 10px; color: #2d5543; font-weight: 600; }
.level-summary strong { font-size: 20px; }
.level-summary small { font-size: 13px; font-weight: 500; }
.tutorial-header,.task-header { width: 100%; font: inherit; color: inherit; text-align: left; border: 0; background: transparent; }
.task-header-left > span:last-child { min-width: 0; overflow-wrap: anywhere; }
.task-title { display: block; }
.task-item { scroll-margin-top: 12px; }
.task-actions:has(.task-status-info:empty) { justify-content: flex-end; }
.criteria-weight { white-space: nowrap; }
@media (max-width: 800px) {
  .lesson-columns { grid-template-columns: minmax(0,1fr); }
  .task-action-btns { flex-wrap: wrap; }
  .tutorial-header,.task-header,.task-actions { padding: 14px; }
}
</style>
