<template>
  <div class="page map-board">
    <div class="page-title-row">
      <div>
        <h1 class="page-title">章节目录</h1>
        <p class="page-subtitle">{{ maps.length }} 个章节 · {{ store.totalLevels }} 个关卡</p>
      </div>
      <div class="xp-chip">

        <div>
          <div class="xp-value">{{ store.totalXp }}</div>
          <div class="xp-label">总经验 XP</div>
        </div>
      </div>
    </div>

    <!-- 全局进度 -->
    <div class="section-card global-progress">
      <div class="gp-row">
        <div class="gp-info">
          <span class="gp-label">项目进度</span>
          <span class="gp-value">{{ store.completedLevels }} / {{ store.totalLevels }} 关</span>
        </div>
        <div class="gp-current" v-if="store.currentMap">当前章节 · {{ store.currentMap }}</div>
        <el-button v-if="currentChapter" size="small" round class="gp-go" @click="continueLearning">继续学习</el-button>
      </div>
      <el-progress :percentage="store.overallPct" :stroke-width="10" :show-text="false" color="#216354" />
    </div>

    <!-- 章节时间轴 -->
    <div class="story-timeline" v-loading="loading">
      <div v-for="map in maps" :key="map.id" class="tl-chapter"
           :id="'chapter-' + map.id"
           :class="[{ current: map.status === 'IN_PROGRESS' }, map.status.toLowerCase()]">
        <div class="tl-node" :class="map.status.toLowerCase()">{{ chapterNodeIcon(map) }}</div>

        <div class="tl-chapter-card" :class="{ locked: map.status === 'LOCKED' }">
          <div class="tl-chapter-head">
            <button type="button" class="tl-chapter-title-line" :aria-expanded="expandedId === map.id" @click="toggleChapter(map)">
              <span class="tl-chapter-index">第 {{ map.chapterNumber }} 章</span>
              <span class="tl-chapter-name">{{ map.icon }} {{ map.name }}</span>
              <span class="tl-status" :class="map.status.toLowerCase()">
                {{ map.status === 'COMPLETED' ? '已通关' : map.status === 'LOCKED' ? '未解锁' : '进行中' }}
              </span>
              <span class="tl-count">{{ map.levelsCompleted }}/{{ map.totalLevels }} 关</span>
              <span class="chapter-chevron" :class="{ open: expandedId === map.id }" aria-hidden="true">⌄</span>
            </button>
            <div class="tl-chapter-desc">{{ map.description }}</div>
            <div v-if="expandedId === map.id && (map.story || map.status === 'LOCKED')" class="tl-story" :class="{ locked: map.status === 'LOCKED' }">
              {{ map.status === 'LOCKED' ? '完成前置关卡后解锁。' : map.story }}
            </div>
          </div>

          <!-- 关卡子项 -->
          <div v-if="expandedId === map.id" class="tl-levels" v-loading="loadingIds.has(map.id)">
            <div v-for="lv in levelsOf(map.id)" :key="lv.id" class="tl-level"
                 :class="[{ current: map.currentLevelId === lv.id }, lv.status.toLowerCase()]">
              <div class="tl-level-dot" :class="lv.status.toLowerCase()">
                {{ lv.status === 'COMPLETED' ? '✓' : lv.status === 'LEARNING' ? '⭐' : '🔒' }}
              </div>
              <div class="tl-level-main">
                <div class="tl-level-title-row">
                  <span class="tl-level-name">{{ lv.name }}</span>
                  <span class="tl-level-xp">+{{ lv.xpReward }} XP</span>
                  <span v-if="map.currentLevelId === lv.id" class="tl-level-now">当前关卡</span>
                </div>
                <div class="tl-level-meta">
                  <span class="tl-level-status" :class="lv.status.toLowerCase()">
                    {{ lv.status === 'COMPLETED' ? '已通关' : lv.status === 'LEARNING' ? '挑战中' : '未解锁' }}
                  </span>
                  <span class="tl-level-tasks">{{ lv.taskDone }}/{{ lv.taskTotal }} 任务</span>
                </div>
              </div>
              <div class="tl-level-actions">
                <template v-if="lv.status !== 'LOCKED'">
                  <el-button size="small" round @click.stop="submitForLevel(lv)">提交作业</el-button>
                  <el-button size="small" type="primary" round @click.stop="goLevel(lv)">进入关卡</el-button>
                </template>
                <el-button v-else size="small" round disabled>未解锁</el-button>
              </div>
            </div>
            <div v-if="errors.has(map.id)" class="tl-level-empty">关卡载入失败 <el-button text @click="loadLevels(map.id)">重试</el-button></div>
            <div v-else-if="levelsByMap.has(map.id) && !levelsOf(map.id).length" class="tl-level-empty">本章暂无关卡</div>
          </div>
        </div>
      </div>
      <div v-if="loadFailed" class="tl-level-empty">无法载入章节 <el-button text @click="load">重试</el-button></div>
      <el-empty v-else-if="!loading && !maps.length" description="暂无章节" />
    </div>

    <!-- 多任务时选择提交哪个 -->
    <el-dialog v-model="taskPickerVisible" title="选择要提交的任务" width="min(440px, calc(100% - 24px))">
      <div class="task-picker-list">
        <button type="button" v-for="t in pickableTasks" :key="t.id" class="task-picker-item" @click="pickTask(t)">
          <span class="task-picker-title">{{ t.title }}</span>
          <span class="task-picker-xp">+{{ t.xpReward }} XP</span>
          <span class="task-picker-arrow">›</span>
        </button>
        <el-empty v-if="!pickableTasks.length" description="该关卡任务均已通过，去下一关吧" :image-size="60" />
      </div>
    </el-dialog>

    <TaskSubmitDialog ref="submitDialog" :task="currentTask" @done="onSubmitted" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { mapsApi } from '../api/maps'
import { useMapStore } from '../store/map'
import { ElMessage } from 'element-plus'
import 'element-plus/es/components/message/style/css'
import TaskSubmitDialog from '../components/TaskSubmitDialog.vue'

const router = useRouter()
const store = useMapStore()

const maps = computed(() => store.progress?.maps || [])
const expandedId = ref(null)
const loadingIds = reactive(new Set())
const errors = reactive(new Set())
const loadFailed = ref(false)
const levelsByMap = ref(new Map())
const loading = ref(true)

const submitDialog = ref(null)
const currentTask = ref(null)
const taskPickerVisible = ref(false)
const pickableTasks = ref([])

const currentChapter = computed(() => maps.value.find(m => m.status === 'IN_PROGRESS') || null)

function levelsOf(mapId) {
  return levelsByMap.value.get(mapId) || []
}

function chapterNodeIcon(map) {
  if (map.status === 'COMPLETED') return '✓'
  if (map.status === 'LOCKED') return '🔒'
  return map.icon || '🗺️'
}

function scrollToChapter() {
  if (!currentChapter.value) return
  document.getElementById('chapter-' + currentChapter.value.id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

async function continueLearning() {
  const chapter = currentChapter.value
  expandedId.value = chapter.id
  await loadLevels(chapter.id)
  const level = levelsOf(chapter.id).find(item => item.id === chapter.currentLevelId && item.status !== 'LOCKED')
  if (level) goLevel(level)
  else scrollToChapter()
}

async function toggleChapter(map) {
  expandedId.value = expandedId.value === map.id ? null : map.id
  if (expandedId.value) await loadLevels(map.id)
}

async function loadLevels(id) {
  if (levelsByMap.value.has(id) || loadingIds.has(id)) return
  loadingIds.add(id)
  errors.delete(id)
  try {
    const response = await mapsApi.detail(id)
    levelsByMap.value.set(id, response.data.data?.levels || [])
  } catch {
    errors.add(id)
  } finally {
    loadingIds.delete(id)
  }
}

function goLevel(lv) {
  if (lv.status === 'LOCKED') {
    ElMessage.warning('通关前一关卡后解锁')
    return
  }
  router.push('/level/' + lv.id)
}

async function submitForLevel(lv) {
  try {
    const res = await mapsApi.levelDetail(lv.id)
    const tasks = (res.data.data?.tasks || []).filter(t => t.submissionStatus !== 'APPROVED')
    if (!tasks.length) {
      ElMessage.info('该关卡任务均已通过')
      return
    }
    if (tasks.length === 1) {
      currentTask.value = tasks[0]
      submitDialog.value?.open()
    } else {
      pickableTasks.value = tasks
      taskPickerVisible.value = true
    }
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '加载任务失败')
  }
}

function pickTask(t) {
  taskPickerVisible.value = false
  currentTask.value = t
  submitDialog.value?.open()
}

async function onSubmitted() {
  const id = expandedId.value
  if (id) levelsByMap.value.delete(id)
  await Promise.all([store.refresh(), id ? loadLevels(id) : null])
}

async function load() {
  loading.value = true
  const progress = await store.fetchProgress()
  loadFailed.value = !progress
  expandedId.value = currentChapter.value?.id ?? maps.value[0]?.id ?? null
  if (expandedId.value) await loadLevels(expandedId.value)
  loading.value = false
}

onMounted(load)
</script>

<style scoped>
.page { max-width: 1080px; }
.page-title-row { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px; }

.xp-chip {
  display: flex; align-items: center; gap: 10px;
  background: linear-gradient(135deg, #faf0d7, #fde68a);
  border-radius: 12px; padding: 10px 18px;
}
.xp-icon { font-size: 22px; }
.xp-value { font-size: 20px; font-weight: 700; color: #92400e; line-height: 1.1; }
.xp-label { font-size: 11px; color: #b45309; }

.global-progress {
  padding: 16px 24px;
  margin-bottom: 32px;
  background: linear-gradient(135deg, #edf5ee, #f2f6ee);
  border: 1px solid #dce7dc;
}
.gp-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; flex-wrap: wrap; gap: 8px; }
.gp-label { font-size: 13px; font-weight: 600; color: var(--color-text); }
.gp-value { font-size: 15px; font-weight: 700; color: #216354; margin-left: 10px; }
.gp-current { font-size: 13px; color: #58685d; }
.gp-go { margin-left: auto; }

/* ===== 时间轴 ===== */
.story-timeline {
  position: relative;
  padding-left: 44px;
}
.story-timeline::before {
  content: '';
  position: absolute;
  left: 16px; top: 8px; bottom: 8px;
  width: 3px;
  background: linear-gradient(180deg, #bed2bf, #dce2d7 30%, #dce2d7 70%, #a7f3d0);
  border-radius: 2px;
}

.tl-chapter { position: relative; margin-bottom: 14px; }

/* 章节节点（时间轴主圆点） */
.tl-node {
  position: absolute;
  left: -44px; top: 20px;
  width: 34px; height: 34px;
  border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  font-size: 15px; font-weight: 700;
  color: #fff;
  border: 4px solid #fff;
  box-shadow: 0 2px 8px rgba(0, 0, 0, .15);
  z-index: 2;
}
.tl-node.in_progress { background: linear-gradient(135deg, #bc8b3b, #d9a753); }
.tl-node.completed { background: linear-gradient(135deg, #30684b, #74a785); }
.tl-node.locked { background: #bfcbbf; color: #58685d; }

/* 章节卡片 */
.tl-chapter-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 16px;
  padding: 20px 22px 18px;
  transition: all .25s;
  position: relative;
  overflow: hidden;
}
.tl-chapter-card::before {
  content: '';
  position: absolute; top: 0; left: 0; right: 0; height: 4px;
  background: linear-gradient(90deg, #216354, #397c67);
}
.tl-chapter.completed .tl-chapter-card::before { background: linear-gradient(90deg, #3b825f, #74a785); }
.tl-chapter.locked .tl-chapter-card::before { background: #bfcbbf; }
.tl-chapter.current .tl-chapter-card {
  border-color: #a6bb9e;
  box-shadow: 0 4px 14px rgba(49,68,39,.06);
}
.tl-chapter-card.locked { background: #f5f5ee; }

.tl-chapter-head { padding-bottom: 14px; }
.tl-chapter-title-line { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; margin-bottom: 6px; }
.tl-chapter-index {
  font-size: 12px; font-weight: 700; letter-spacing: 1px;
  color: #216354; background: #edf5ee;
  padding: 3px 10px; border-radius: 20px;
}
.tl-chapter.completed .tl-chapter-index { color: #30684b; background: #e0efe6; }
.tl-chapter.locked .tl-chapter-index { color: #667568; background: #edf0e9; }
.tl-chapter-name { font-size: 17px; font-weight: 700; color: var(--color-text); }
.tl-status { font-size: 12px; font-weight: 600; padding: 3px 10px; border-radius: 20px; }
.tl-status.in_progress { background: #faf0d7; color: #b45309; }
.tl-status.completed { background: #e0efe6; color: #30684b; }
.tl-status.locked { background: #edf0e9; color: #667568; }
.tl-count { font-size: 12px; color: #667568; }

.tl-chapter-desc { font-size: 13px; color: var(--color-text-secondary); margin-bottom: 10px; }

.tl-story {
  font-size: 14px; line-height: 2;
  color: #3e5147;
  background: #fffbeb;
  border: 1px solid #fde68a;
  border-left: 4px solid #bc8b3b;
  border-radius: 0 12px 12px 12px;
  padding: 12px 16px;
}
.tl-story.locked {
  background: #f5f6f0;
  border-color: #dce2d7;
  border-left-color: #bfcbbf;
  color: #667568;

}

/* 关卡子项 */
.tl-levels { position: relative; margin-top: 6px; }
.tl-level {
  display: flex; align-items: center; gap: 12px;
  padding: 12px 14px;
  border-radius: 12px;
  transition: background .15s;
  border: 1px dashed transparent;
}
.tl-level:hover { background: #f5f6f0; border-color: #dce2d7; }
.tl-level.locked { color: var(--color-text-secondary); }
.tl-level.current { background: #fffbeb; border-color: #fde68a; }

.tl-level-dot {
  width: 30px; height: 30px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  font-size: 13px; flex-shrink: 0;
}
.tl-level-dot.completed { background: #e0efe6; color: #30684b; font-weight: 700; }
.tl-level-dot.learning { background: #faf0d7; color: #b45309; }
.tl-level-dot.locked { background: #edf0e9; color: #667568; }

.tl-level-main { flex: 1; min-width: 0; }
.tl-level-title-row { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.tl-level-name { font-size: 14px; font-weight: 600; color: var(--color-text); }
.tl-level-xp { font-size: 11px; font-weight: 600; color: #895b19; background: #faf0d7; padding: 2px 8px; border-radius: 4px; }
.tl-level-now { font-size: 11px; font-weight: 700; color: #b45309; background: #fde68a; padding: 2px 8px; border-radius: 4px; }
.tl-level-meta { display: flex; align-items: center; gap: 10px; margin-top: 3px; }
.tl-level-status { font-size: 12px; color: #667568; }
.tl-level-status.completed { color: #30684b; }
.tl-level-status.learning { color: #895b19; }
.tl-level-tasks { font-size: 12px; color: #667568; }

.tl-level-actions { display: flex; gap: 8px; flex-shrink: 0; }
.tl-level-empty { padding: 14px; text-align: center; font-size: 13px; color: #667568; }

/* 任务选择弹窗 */
.task-picker-list { display: flex; flex-direction: column; gap: 10px; }
.task-picker-item {
  display: flex; align-items: center; gap: 10px;
  padding: 12px 14px;
  border: 1px solid var(--color-border);
  border-radius: 10px;
  cursor: pointer;
  transition: all .15s;
}
.task-picker-item:hover { border-color: #bed2bf; background: #f2f6ee; transform: translateX(3px); }
.task-picker-title { flex: 1; font-size: 14px; font-weight: 500; color: var(--color-text); }
.task-picker-xp { font-size: 12px; font-weight: 600; color: #895b19; }
.task-picker-arrow { color: #667568; font-size: 18px; }

@media (max-width: 768px) {
  .story-timeline { padding-left: 36px; }
  .tl-node { left: -34px; width: 30px; height: 30px; font-size: 13px; }
  .tl-level { flex-wrap: wrap; }
  .tl-level-actions { width: 100%; justify-content: flex-end; }
}

.tl-chapter-title-line { width: 100%; text-align: left; font: inherit; border: 0; padding: 0; color: inherit; background: transparent; cursor: pointer; }
.tl-chapter-head { padding-bottom: 0; }
.chapter-chevron { margin-inline-start: auto; font-size: 20px; line-height: 1; color: #496a55; }
.chapter-chevron.open { transform: rotate(180deg); }
.task-picker-item { font: inherit; background: var(--color-surface); text-align: left; }
.tl-levels { min-height: 42px; }
@media(max-width: 600px) {
  .story-timeline { padding-left: 0; }
  .tl-node,.story-timeline::before { display: none; }
  .tl-chapter-card { padding: 14px; }
  .page-title-row { gap: 12px; align-items: center; }
}
</style>
