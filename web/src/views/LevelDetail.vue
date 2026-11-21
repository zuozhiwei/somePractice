<template>
  <div class="lesson-page">
    <nav class="lesson-crumb" aria-label="课程位置">
      <RouterLink :to="{ path: '/learn', query: { place: 'level:' + route.params.id } }"><el-icon><Back /></el-icon> 返回世界地图</RouterLink>
      <template v-if="level?.mapId">
        <el-icon aria-hidden="true"><ArrowRight /></el-icon>
        <RouterLink :to="'/map/' + level.mapId">{{ level.mapName }}</RouterLink>
      </template>
    </nav>

    <p v-if="loading && !level" class="lesson-state" role="status">正在打开这一关…</p>
    <div v-else-if="loadError && !level" class="lesson-state" role="alert">
      <p>{{ loadError }}</p>
      <el-button round @click="loadLevel(route.params.id)">再试一次</el-button>
    </div>

    <template v-if="level">
      <header class="lesson-hero">
        <div>
          <div class="lesson-meta">
            <span v-if="level.levelNumber">第 {{ level.levelNumber }} 关</span>
            <span v-if="level.category">{{ level.category }}</span>
          </div>
          <h1>{{ level.name }}</h1>
          <p v-if="level.description" class="lesson-goal">{{ level.description }}</p>
        </div>
        <div v-if="level.tasks?.length" class="lesson-progress">
          <div>
            <span>任务验收进度</span>
            <strong>{{ completedCount }} <span>/ {{ level.tasks.length }}</span></strong>
          </div>
          <progress :value="completedCount" :max="level.tasks.length" aria-label="任务验收进度" />
          <p v-if="level.userStatus?.status === 'COMPLETED'"><el-icon><CircleCheck /></el-icon> 这一关已完成</p>
        </div>
      </header>

      <RouterLink v-if="level.nextLevel" class="next-lesson" :to="'/level/' + level.nextLevel.id">
        <span><small>已解锁下一关</small><strong>{{ level.nextLevel.name }}</strong></span>
        <el-icon aria-hidden="true"><Right /></el-icon>
      </RouterLink>

      <div v-if="loadError" class="lesson-refresh" role="alert">
        <span>{{ loadError }}</span>
        <el-button text @click="loadLevel(route.params.id)">刷新进度</el-button>
      </div>

      <div class="lesson-layout">
        <nav class="lesson-index" aria-label="本关内容">
          <template v-if="level.tasks?.length">
            <h2><el-icon><List /></el-icon> 任务清单</h2>
            <button v-for="(task, index) in level.tasks" :key="task.id" type="button" class="index-item"
                    :aria-current="selection === taskKey(task) ? 'true' : undefined"
                    @click="selectSection(taskKey(task))">
              <span v-if="task.submissionStatus === 'APPROVED'" class="index-icon done" aria-label="已通过"><el-icon><Check /></el-icon></span>
              <span v-else class="index-icon" aria-hidden="true">{{ index + 1 }}</span>
              {{ task.title }}
            </button>
          </template>
          <div v-if="guides.length" class="reference-list">
            <button v-if="guides.length === 1" type="button" class="index-item"
                    :aria-current="activeGuide ? 'true' : undefined" @click="selectSection(guideKey(guides[0]))">
              <el-icon class="index-guide"><Reading /></el-icon> 本关参考
            </button>
            <details v-else :open="Boolean(activeGuide)">
              <summary>本关参考</summary>
              <button v-for="guide in guides" :key="guide.id" type="button" class="index-item"
                      :aria-current="selection === guideKey(guide) ? 'true' : undefined"
                      @click="selectSection(guideKey(guide))">{{ guide.title }}</button>
            </details>
          </div>
          <RouterLink class="help-link" to="/mentorship"><el-icon><ChatDotRound /></el-icon> {{ helpLabel }}</RouterLink>
        </nav>

        <div v-if="level.tasks?.length || level.tutorials?.length" class="lesson-select">
          <label for="lesson-section">本关内容</label>
          <select id="lesson-section" :value="selection" @change="selectSection($event.target.value)">
            <optgroup v-if="level.tasks?.length" label="任务清单">
              <option v-for="(task, index) in level.tasks" :key="task.id" :value="taskKey(task)">
                {{ index + 1 }}. {{ task.title }} · {{ submissionStatusText(task.submissionStatus) }}
              </option>
            </optgroup>
            <optgroup v-if="guides.length" label="参考资料">
              <option v-for="guide in guides" :key="guide.id" :value="guideKey(guide)">
                {{ guides.length === 1 ? '本关参考' : guide.title }}
              </option>
            </optgroup>
          </select>
        </div>

        <section v-if="activeTask" :key="selection" class="work-document" aria-labelledby="work-title">
          <header class="document-head">
            <div class="document-meta">
              <span>任务 {{ tasks.indexOf(activeTask) + 1 }} / {{ tasks.length }}</span>
              <span class="task-status" :class="activeTask.submissionStatus?.toLowerCase()">
                {{ submissionStatusText(activeTask.submissionStatus) }}
              </span>
              <span v-if="activeTask.xpReward">{{ activeTask.xpReward }} XP</span>
            </div>
            <h2 id="work-title" ref="documentHeading" tabindex="-1">{{ activeTask.title }}</h2>
          </header>

          <div class="document-body">
            <TutorialReader :content="activeTask.description" />
            <section v-if="activeTask.acceptanceCriteria?.length" class="criteria" aria-labelledby="criteria-title">
              <h3 id="criteria-title"><el-icon><Finished /></el-icon> 提交前对照</h3>
              <ol>
                <li v-for="(item, index) in activeTask.acceptanceCriteria" :key="index">
                  <span>{{ item.criteria }}</span>
                  <small v-if="item.weight != null">{{ item.weight }}%</small>
                </li>
              </ol>
            </section>
            <section v-if="activeTask.reviewComment || activeTask.aiFeedback" class="task-feedback" aria-label="最近反馈">
              <article v-if="activeTask.reviewComment">
                <h3>审核意见</h3><p>{{ activeTask.reviewComment }}</p>
              </article>
              <article v-if="activeTask.aiFeedback">
                <h3>AI 评语</h3><p>{{ activeTask.aiFeedback }}</p>
              </article>
            </section>
          </div>

          <footer class="document-actions">
            <p v-if="reviewHint" role="status">{{ reviewHint }}</p>
            <div>
              <el-button v-if="activeTask.submissionStatus !== 'NONE'" round @click="openDialog(activeTask, 'history')">
                提交记录
              </el-button>
              <el-button v-if="activeTask.submissionStatus !== 'APPROVED'" :type="canContinue ? 'default' : 'primary'" round
                         :disabled="level.userStatus?.status === 'LOCKED'" @click="openDialog(activeTask, 'submit')">
                {{ submissionActionText(activeTask.submissionStatus) }}
              </el-button>
              <el-button v-if="canContinue" type="primary" round @click="showTask(remainingTask, true)">
                继续任务 <el-icon class="el-icon--right"><Right /></el-icon>
              </el-button>
            </div>
          </footer>
        </section>

        <section v-else-if="activeGuide" :key="selection" class="work-document" aria-labelledby="work-title">
          <header class="document-head">
            <div class="document-meta"><span><el-icon><Reading /></el-icon> 本关参考</span></div>
            <h2 id="work-title" ref="documentHeading" tabindex="-1">{{ activeGuide.title }}</h2>
            <button v-if="returnTask" class="reference-link" type="button" @click="showTask(returnTask, true)">
              <el-icon><component :is="taskBeforeGuide ? Back : Right" /></el-icon> {{ guideAction }}
            </button>
          </header>
          <div class="document-body"><TutorialReader :content="activeGuide.content" :section-anchor="route.hash" show-outline /></div>
          <footer v-if="returnTask" class="document-actions">
            <el-button type="primary" round @click="showTask(returnTask, true)">{{ guideAction }} <el-icon class="el-icon--right"><Right /></el-icon></el-button>
          </footer>
        </section>

        <p v-else class="lesson-state">这一关的内容还在准备中。</p>
        <RouterLink class="help-link mobile-help" to="/mentorship"><el-icon><ChatDotRound /></el-icon> {{ helpLabel }}</RouterLink>
      </div>

      <component :is="submitView" v-if="submitView" ref="submitDialog" :task="submissionTask" @done="onTaskDone" />
      <component :is="historyView" v-if="historyView" ref="historyDialog" />
    </template>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import 'element-plus/es/components/message/style/css'
import { ArrowRight, Back, ChatDotRound, Check, CircleCheck, Finished, List, Reading, Right } from '@element-plus/icons-vue'
import { mapsApi } from '../api/maps'
import { useAuthStore } from '../store/auth'
import { useMapStore } from '../store/map'
import { awaitingReview, submissionActionText, submissionStatusText } from '../submissionStatus'
import TutorialReader from '../components/TutorialReader.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const helpLabel = computed(() => auth.user?.role === 'EXTERNAL' ? '查看我的提交' : '找学长或导师聊聊')
const mapStore = useMapStore()
const level = ref(null)
const loading = ref(true)
const loadError = ref('')
const selection = ref('')
const taskBeforeGuide = ref(null)
const documentHeading = ref(null)
const submitView = shallowRef(null)
const historyView = shallowRef(null)
const submitDialog = ref(null)
const submissionTask = ref(null)
const historyDialog = ref(null)
let requestId = 0, levelETag = ''

const taskKey = task => 'task:' + task.id
const guideKey = guide => 'guide:' + guide.id
const tasks = computed(() => level.value?.tasks || [])
const guides = computed(() => level.value?.tutorials || [])
const activeTask = computed(() => tasks.value.find(task => taskKey(task) === selection.value))
const activeGuide = computed(() => guides.value.find(guide => guideKey(guide) === selection.value))
const remainingTask = computed(() => tasks.value.find(task => ['NONE', 'REJECTED'].includes(task.submissionStatus)))
const canContinue = computed(() => remainingTask.value && (
  activeTask.value?.submissionStatus === 'APPROVED' || awaitingReview(activeTask.value?.submissionStatus)
))
const defaultTask = computed(() => remainingTask.value || tasks.value.find(task => task.submissionStatus !== 'APPROVED') || tasks.value[0])
const returnTask = computed(() => tasks.value.find(task => task.id === taskBeforeGuide.value) || defaultTask.value)
const guideAction = computed(() => (taskBeforeGuide.value ? '回到任务 · ' : '打开任务 · ') + (returnTask.value?.title || ''))
const completedCount = computed(() => tasks.value.filter(task => task.submissionStatus === 'APPROVED').length)
const reviewHint = computed(() => ({
  APPROVED: '已通过验收，交付记录已收进世界地图的成果册。',
  PENDING: '已提交，可以继续本关的其他任务。',
  REJECTED: '对照审核意见修改后，可以重新提交。',
  AI_REVIEWED: 'AI 已给出反馈，等待人工审核。'
}[activeTask.value?.submissionStatus] || ''))

function chooseDefault() {
  const requested = tasks.value.find(task => String(task.id) === route.query.task)
  const requestedGuide = guides.value.find(guide => String(guide.id) === route.query.guideId)
    || (route.query.guide === '1' && guides.value[0])
  if (requested) selection.value = taskKey(requested)
  else if (requestedGuide) selection.value = guideKey(requestedGuide)
  else if (defaultTask.value) selection.value = taskKey(defaultTask.value)
  else if (guides.value.length) selection.value = guideKey(guides.value[0])
}

function selectSection(key) {
  const query = { ...route.query }
  delete query.task
  delete query.guide
  delete query.guideId
  const [kind, id] = key.split(':')
  if (kind === 'task') query.task = id
  else if (guides.value[0] && key === guideKey(guides.value[0])) query.guide = '1'
  else query.guideId = id
  selection.value = key
  router.replace({ path: route.path, query })
}

function showTask(task, focus = false) {
  selectSection(taskKey(task))
  if (focus) focusDocument()
}

async function openDialog(task, kind) {
  const history = kind === 'history'
  const view = history ? historyView : submitView
  try {
    if (!view.value) {
      const module = await (history
        ? import('../components/SubmissionHistoryDialog.vue')
        : import('../components/TaskSubmitDialog.vue'))
      view.value = module.default
    }
  } catch {
    ElMessage.error('窗口暂时加载不了，请重试。')
    return
  }
  if (activeTask.value?.id !== task.id) return
  if (!history) submissionTask.value = task
  await nextTick()
  if (history) historyDialog.value?.open(task)
  else submitDialog.value?.open()
}

async function focusDocument() {
  await nextTick()
  documentHeading.value?.focus()
}

async function loadLevel(id, { reset = false, checkVersion = false } = {}) {
  const request = ++requestId
  if (reset) {
    levelETag = ''
    level.value = null
    selection.value = ''
    taskBeforeGuide.value = null
    submissionTask.value = null
  }
  loadError.value = ''
  loading.value = Boolean(id && auth.token)
  if (!loading.value) return
  try {
    const { data, headers, status } = await mapsApi.levelDetail(id, checkVersion ? levelETag : undefined)
    if (request !== requestId || status === 304) return
    level.value = data.data
    levelETag = headers.etag || ''
    if (!activeTask.value && !activeGuide.value) chooseDefault()
  } catch (error) {
    if (request === requestId) {
      loadError.value = error.response?.data?.message || '关卡暂时加载不了，请稍后再试。'
    }
  } finally {
    if (request === requestId) loading.value = false
  }
}

async function onTaskDone() {
  const id = route.params.id
  const token = auth.token
  const wasComplete = level.value?.userStatus?.status === 'COMPLETED'
  await loadLevel(id)
  if (id !== route.params.id || token !== auth.token) return
  mapStore.refresh()
  if (!wasComplete && level.value?.userStatus?.status === 'COMPLETED') {
    ElMessage.success('这一关的任务都通过了。')
  }
}

watch([() => route.params.id, () => auth.token], ([id]) => {
  loadLevel(id, { reset: true })
}, { immediate: true, flush: 'sync' })
watch([() => route.query.task, () => route.query.guide, () => route.query.guideId], () => {
  if (String(level.value?.id) !== String(route.params.id)) return
  const previous = selection.value
  chooseDefault()
  if (!route.hash && selection.value !== previous) focusDocument()
})
watch(selection, (_, previous) => {
  if (activeGuide.value) {
    const task = tasks.value.find(item => taskKey(item) === previous)
    if (task) taskBeforeGuide.value = task.id
  }
})
let reviewTimer
async function refreshPending() {
  if (document.hidden || loading.value || !tasks.value.some(task => awaitingReview(task.submissionStatus))) return
  const previous = completedCount.value
  const id = route.params.id
  const token = auth.token
  await loadLevel(id, { checkVersion: true })
  if (route.params.id === id && auth.token === token && completedCount.value > previous) {
    ElMessage.success('有任务通过验收，已收进成果册。')
    mapStore.refresh()
  }
}
onMounted(() => {
  document.addEventListener('visibilitychange', refreshPending)
  reviewTimer = window.setInterval(refreshPending, 30000)
})
onBeforeUnmount(() => {
  requestId++
  document.removeEventListener('visibilitychange', refreshPending)
  window.clearInterval(reviewTimer)
})
</script>

<style scoped>
.lesson-page { max-width: 1180px; margin-inline: auto; }
.lesson-crumb {
  display: flex; align-items: center; gap: 10px; flex-wrap: wrap; min-height: 40px;
  color: var(--color-text-secondary); font-size: 13px; margin-bottom: 10px;
}
.lesson-crumb a, .help-link {
  display: inline-flex; gap: 7px; align-items: center; color: var(--color-primary); text-decoration: none;
}
.lesson-crumb a:hover, .help-link:hover { text-decoration: underline; text-underline-offset: 4px; }
.lesson-hero {
  display: grid; grid-template-columns: minmax(0, 1fr) minmax(160px, 234px); align-items: end;
  gap: 24px 36px; padding: clamp(23px, 4vw, 46px); margin-bottom: 24px; border-radius: 20px;
  background: #173d35; color: #fcf8eb; overflow-wrap: anywhere;
}
.lesson-meta { display: flex; align-items: center; gap: 8px 16px; flex-wrap: wrap; color: #d6e3d9; font-size: 13px; }
.lesson-meta span:first-child { color: #f3d578; font-weight: 650; }
.lesson-hero h1 { font-size: clamp(26px, 4.2vw, 43px); line-height: 1.24; margin: 9px 0 0; letter-spacing: -.035em; }
.lesson-goal { color: #dfebe1; line-height: 1.75; font-size: clamp(14px, 1.5vw, 16px); margin-top: 12px; max-width: 54ch; }
.lesson-progress { align-self: end; border: 1px solid #ffffff40; border-radius: 13px; padding: 14px 16px; }
.lesson-progress div { display: flex; align-items: center; justify-content: space-between; gap: 8px; color: #e2e9db; font-size: 12px; }
.lesson-progress strong { text-align: end; color: #f3d578; font-size: 19px; font-variant-numeric: tabular-nums; }
.lesson-progress strong span { color: #e2e9db; font-weight: 500; font-size: 13px; }
.lesson-progress progress { width: 100%; height: 6px; display: block; margin-top: 11px; appearance: none; border: 0; border-radius: 9px; overflow: hidden; color: #f3d578; background: #46665d; }
.lesson-progress progress::-webkit-progress-bar { background-color: #46665d; }
.lesson-progress progress::-webkit-progress-value { background-color: #f3d578; }
.lesson-progress progress::-moz-progress-bar { background-color: #f3d578; }
.lesson-progress p { color: #f0dfaa; font-size: 12px; margin-top: 10px; display: flex; gap: 6px; align-items: center; }

.next-lesson {
  display: flex; justify-content: space-between; gap: 12px; align-items: center;
  padding: 13px 20px; margin-bottom: 22px; border: 1px solid #d5b455; border-radius: 14px;
  background: #f7e6ab; color: #263b26; text-decoration: none;
}
.next-lesson:hover { background: #f3d578; }
.next-lesson span { display: grid; gap: 3px; }
.next-lesson small { font-size: 12px; }
.next-lesson strong { font-size: 15px; }
.lesson-layout { display: grid; grid-template-columns: minmax(205px, 26%) minmax(0, 1fr); gap: 20px; align-items: start; }
.lesson-index, .work-document, .lesson-state {
  min-width: 0; border: 1px solid var(--color-border); background: var(--color-surface); border-radius: 17px;
}
.lesson-index { padding: 13px 11px; position: sticky; top: 18px; max-height: calc(100dvh - 36px); overflow-y: auto; }
.lesson-index h2 {
  display: flex; align-items: center; gap: 7px; margin: 7px 10px 6px;
  font-size: 13px; color: var(--color-text-secondary); font-weight: 650;
}
.lesson-index h2:not(:first-child) { margin-top: 16px; padding-top: 13px; border-top: 1px solid var(--color-border); }
.index-item {
  display: flex; align-items: baseline; gap: 10px; width: 100%; min-height: 44px; padding: 9px 10px;
  border: 0; border-radius: 8px; background: transparent; color: var(--color-text);
  font-size: 13px; line-height: 1.65; text-align: start; overflow-wrap: anywhere;
}
.index-item:hover { background: #f1f3e8; }
.index-item[aria-current] { background: #e6eee2; color: #164438; font-weight: 700; }
.index-icon {
  width: 19px; height: 19px; flex: none; display: inline-grid; place-content: center;
  border: 1px solid #a4b6a6; border-radius: 50%; align-self: flex-start; margin-top: 1px;
  color: #42614c; font-size: 11px; line-height: 1; font-variant-numeric: tabular-nums;
}
.index-icon.done { border: 0; background: #287359; color: white; font-size: 10px; }
.reference-list { margin-top: 10px; padding-top: 6px; border-top: 1px solid var(--color-border); }
.reference-list summary { cursor: pointer; padding: 10px; color: var(--color-primary); font-size: 13px; }
.index-guide { flex: none; color: #527464; }
.help-link { font-size: 13px; min-height: 44px; padding-inline: 10px; }
.lesson-index .help-link { border-top: 1px solid var(--color-border); margin-top: 10px; padding-top: 6px; width: 100%; }
.lesson-select, .mobile-help { display: none; }

.document-head, .document-body, .document-actions { padding-inline: clamp(20px, 3.3vw, 39px); }
.document-head { padding-block: clamp(20px, 3vw, 32px) 15px; }
.document-meta { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; color: var(--color-text-secondary); font-size: 12px; }
.document-meta > span { display: inline-flex; align-items: center; gap: 5px; }
.document-head h2 { margin-top: 8px; font-size: clamp(19px, 2.6vw, 28px); line-height: 1.38; overflow-wrap: anywhere; scroll-margin-top: 90px; }
.document-head h2:focus { outline: none; }
.reference-link {
  background: transparent; color: var(--color-primary); border: none; padding: 0; margin-top: 4px;
  min-height: 44px; display: inline-flex; align-items: center; gap: 6px; font-size: 13px; font-weight: 650;
}
.reference-link:hover { text-decoration: underline; text-underline-offset: 4px; }
.task-status { padding: 3px 9px; border-radius: 6px; font-weight: 650; background: #ebeee6; color: #4c6456; }
.task-status.approved { background: #e1eee4; color: #24624a; }
.task-status.pending, .task-status.ai_reviewed { background: #f8edd0; color: #77530f; }
.task-status.rejected { background: #f9e7e2; color: #a13f32; }

.document-body { padding-bottom: 24px; min-width: 0; }
.criteria { background: #f3f5ec; border-radius: 12px; margin-top: 24px; padding: 14px 18px; }
.criteria h3, .task-feedback h3 { font-size: 14px; display: flex; align-items: center; gap: 7px; }
.criteria ol { margin-top: 9px; padding-inline-start: 20px; }
.criteria li { padding: 5px 0; font-size: 14px; line-height: 1.6; overflow-wrap: anywhere; }
.criteria li::marker { color: var(--color-primary); font-weight: 650; }
.criteria small { color: var(--color-text-secondary); margin-inline-start: 8px; }
.task-feedback { display: grid; gap: 10px; margin-top: 19px; }
.task-feedback article { padding: 13px 15px; background: #faf1dd; border-radius: 10px; }
.task-feedback p { margin-top: 6px; line-height: 1.65; white-space: pre-wrap; overflow-wrap: anywhere; font-size: 14px; }
.document-actions {
  display: flex; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: 12px;
  border-top: 1px solid var(--color-border); padding-block: 15px;
}
.document-actions p { flex: 1 0 min(100%, 190px); font-size: 13px; line-height: 1.5; color: var(--color-text-secondary); }
.document-actions > div { display: flex; gap: 8px; flex-wrap: wrap; }
.document-actions .el-button { margin: 0; min-height: 42px; }
.lesson-state { padding: 26px; color: var(--color-text-secondary); }
.lesson-state button { margin-top: 12px; }
.lesson-refresh { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; margin-bottom: 14px; color: #77530f; font-size: 14px; }

@media (max-width: 860px) {
  .lesson-layout { grid-template-columns: minmax(0, 1fr); gap: 12px; }
  .lesson-index { display: none; }
  .lesson-select { display: flex; align-items: center; gap: 8px; font-size: 13px; font-weight: 650; }
  .lesson-select label { flex: none; }
  .lesson-select select {
    width: 100%; min-width: 0; min-height: 44px; border: 1px solid var(--color-border); padding: 7px;
    border-radius: 10px; background: var(--color-surface); color: var(--color-text); color-scheme: light;
  }
  .mobile-help { display: inline-flex; width: max-content; max-width: 100%; }
}

@media (max-width: 550px) {
  .lesson-hero { grid-template-columns: minmax(0, 1fr); gap: 18px; margin-bottom: 16px; }
  .lesson-progress { padding: 11px 13px; }
  .next-lesson { margin-bottom: 16px; }
}
</style>
