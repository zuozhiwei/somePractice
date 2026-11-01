<template>
  <div class="page map-board">
    <div class="page-title-row">
      <div>
        <h1 class="page-title">📖 冒险之旅</h1>
        <p class="page-subtitle">七大章节 · 从「初出茅庐」到「独当一面」，一页一页翻开你的成长故事</p>
      </div>
      <div class="xp-chip">
        <span class="xp-icon">⚡</span>
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
          <span class="gp-label">冒险进度</span>
          <span class="gp-value">{{ store.completedLevels }} / {{ store.totalLevels }} 关</span>
        </div>
        <div class="gp-current" v-if="store.currentMap">📍 当前章节：{{ store.currentMap }}</div>
        <el-button v-if="currentChapter" size="small" round class="gp-go" @click="continueLearning">▶ 继续学习</el-button>
      </div>
      <el-progress :percentage="store.overallPct" :stroke-width="10" :show-text="false" color="#6366f1" />
    </div>

    <!-- 章节时间轴 -->
    <div class="story-timeline" v-loading="loading">
      <div v-for="(map, mi) in maps" :key="map.id" class="tl-chapter"
           :id="'chapter-' + map.id"
           :class="[{ current: map.status === 'IN_PROGRESS' }, map.status.toLowerCase()]">
        <div class="tl-node" :class="map.status.toLowerCase()">{{ chapterNodeIcon(map) }}</div>

        <div class="tl-chapter-card" :class="{ locked: map.status === 'LOCKED' }">
          <div class="tl-chapter-head">
            <div class="tl-chapter-title-line">
              <span class="tl-chapter-index">第 {{ mi + 1 }} 章</span>
              <span class="tl-chapter-name">{{ map.icon }} {{ map.name }}</span>
              <span class="tl-status" :class="map.status.toLowerCase()">
                {{ map.status === 'COMPLETED' ? '✅ 已通关' : map.status === 'LOCKED' ? '🔒 未解锁' : '🚀 进行中' }}
              </span>
              <span class="tl-count">{{ map.levelsCompleted }}/{{ map.totalLevels }} 关</span>
            </div>
            <div class="tl-chapter-desc">{{ map.description }}</div>
            <div class="tl-story" :class="{ locked: map.status === 'LOCKED' }">
              {{ map.status === 'LOCKED' ? '这段故事还藏在迷雾中……通关前一章后，你才能翻开它。' : map.story }}
            </div>
          </div>

          <!-- 关卡子项 -->
          <div class="tl-levels">
            <div v-for="lv in levelsOf(map.id)" :key="lv.id" class="tl-level"
                 :class="[{ current: map.currentLevelId === lv.id }, lv.status.toLowerCase()]">
              <div class="tl-level-dot" :class="lv.status.toLowerCase()">
                {{ lv.status === 'COMPLETED' ? '✓' : lv.status === 'LEARNING' ? '⭐' : '🔒' }}
              </div>
              <div class="tl-level-main">
                <div class="tl-level-title-row">
                  <span class="tl-level-name">{{ lv.name }}</span>
                  <span class="tl-level-xp">+{{ lv.xpReward }} XP</span>
                  <span v-if="map.currentLevelId === lv.id" class="tl-level-now">▶ 当前</span>
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
                  <el-button size="small" round @click.stop="submitForLevel(map, lv)">📤 提交</el-button>
                  <el-button size="small" type="primary" round @click.stop="goLevel(lv)">进入关卡</el-button>
                </template>
                <el-button v-else size="small" round disabled>🔒 未解锁</el-button>
              </div>
            </div>
            <div v-if="!levelsOf(map.id).length" class="tl-level-empty">本章暂无内容</div>
          </div>
        </div>
      </div>
      <el-empty v-if="!loading && !maps.length" description="暂无冒险章节" />
    </div>

    <!-- 多任务时选择提交哪个 -->
    <el-dialog v-model="taskPickerVisible" title="选择要提交的任务" width="440px">
      <div class="task-picker-list">
        <div v-for="t in pickableTasks" :key="t.id" class="task-picker-item" @click="pickTask(t)">
          <span class="task-picker-title">{{ t.title }}</span>
          <span class="task-picker-xp">+{{ t.xpReward }} XP</span>
          <span class="task-picker-arrow">›</span>
        </div>
        <el-empty v-if="!pickableTasks.length" description="该关卡任务均已通过，去下一关吧" :image-size="60" />
      </div>
    </el-dialog>

    <TaskSubmitDialog ref="submitDialog" :task="currentTask" @done="onSubmitted" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { mapsApi } from '../api/maps'
import { useMapStore } from '../store/map'
import { ElMessage } from 'element-plus'
import TaskSubmitDialog from '../components/TaskSubmitDialog.vue'

const router = useRouter()
const store = useMapStore()

const maps = ref([])
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

function continueLearning() {
  const level = levelsOf(currentChapter.value.id).find(item => item.id === currentChapter.value.currentLevelId && item.status !== 'LOCKED')
  if (level) goLevel(level)
  else scrollToChapter()
}

function goLevel(lv) {
  if (lv.status === 'LOCKED') {
    ElMessage.warning('通关前一关卡后解锁')
    return
  }
  router.push(`/level/${lv.id}?n=${lv.orderIndex}`)
}

async function submitForLevel(map, lv) {
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

function onSubmitted() {
  load()
}

async function load() {
  loading.value = true
  try {
    const res = await mapsApi.list()
    maps.value = res.data.data || []
    const byMap = new Map()
    await Promise.all(maps.value.map(async m => {
      try {
        const d = await mapsApi.detail(m.id)
        byMap.set(m.id, d.data?.data?.levels || [])
      } catch (e) {
        byMap.set(m.id, [])
      }
    }))
    levelsByMap.value = byMap
    store.refresh()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.page { max-width: 1080px; }
.page-title-row { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px; }

.xp-chip {
  display: flex; align-items: center; gap: 10px;
  background: linear-gradient(135deg, #fef3c7, #fde68a);
  border-radius: 12px; padding: 10px 18px;
}
.xp-icon { font-size: 22px; }
.xp-value { font-size: 20px; font-weight: 700; color: #92400e; line-height: 1.1; }
.xp-label { font-size: 11px; color: #b45309; }

.global-progress {
  padding: 16px 24px;
  margin-bottom: 32px;
  background: linear-gradient(135deg, #eef2ff, #f5f3ff);
  border: 1px solid #e0e7ff;
}
.gp-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; flex-wrap: wrap; gap: 8px; }
.gp-label { font-size: 13px; font-weight: 600; color: var(--color-text); }
.gp-value { font-size: 15px; font-weight: 700; color: #6366f1; margin-left: 10px; }
.gp-current { font-size: 13px; color: #64748b; }
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
  background: linear-gradient(180deg, #c7d2fe, #e2e8f0 30%, #e2e8f0 70%, #a7f3d0);
  border-radius: 2px;
}

.tl-chapter { position: relative; margin-bottom: 30px; }

/* 章节节点（时间轴主圆点） */
.tl-node {
  position: absolute;
  left: -44px; top: 26px;
  width: 34px; height: 34px;
  border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  font-size: 15px; font-weight: 700;
  color: #fff;
  border: 4px solid #fff;
  box-shadow: 0 2px 8px rgba(0, 0, 0, .15);
  z-index: 2;
}
.tl-node.in_progress { background: linear-gradient(135deg, #f59e0b, #fbbf24); }
.tl-node.completed { background: linear-gradient(135deg, #16a34a, #4ade80); }
.tl-node.locked { background: #cbd5e1; color: #64748b; }

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
  background: linear-gradient(90deg, #6366f1, #8b5cf6);
}
.tl-chapter.completed .tl-chapter-card::before { background: linear-gradient(90deg, #22c55e, #4ade80); }
.tl-chapter.locked .tl-chapter-card::before { background: #cbd5e1; }
.tl-chapter.current .tl-chapter-card {
  border-color: #fcd34d;
  box-shadow: 0 8px 24px rgba(245, 158, 11, .18);
}
.tl-chapter-card.locked { opacity: .6; }

.tl-chapter-head { padding-bottom: 14px; }
.tl-chapter-title-line { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; margin-bottom: 6px; }
.tl-chapter-index {
  font-size: 12px; font-weight: 700; letter-spacing: 1px;
  color: #6366f1; background: #eef2ff;
  padding: 3px 10px; border-radius: 20px;
}
.tl-chapter.completed .tl-chapter-index { color: #16a34a; background: #dcfce7; }
.tl-chapter.locked .tl-chapter-index { color: #94a3b8; background: #f1f5f9; }
.tl-chapter-name { font-size: 17px; font-weight: 700; color: var(--color-text); }
.tl-status { font-size: 12px; font-weight: 600; padding: 3px 10px; border-radius: 20px; }
.tl-status.in_progress { background: #fef3c7; color: #b45309; }
.tl-status.completed { background: #dcfce7; color: #16a34a; }
.tl-status.locked { background: #f1f5f9; color: #94a3b8; }
.tl-count { font-size: 12px; color: #94a3b8; }

.tl-chapter-desc { font-size: 13px; color: var(--color-text-secondary); margin-bottom: 10px; }

.tl-story {
  font-size: 14px; line-height: 2;
  color: #475569;
  background: #fffbeb;
  border: 1px solid #fde68a;
  border-left: 4px solid #f59e0b;
  border-radius: 0 12px 12px 12px;
  padding: 12px 16px;
}
.tl-story.locked {
  background: #f8fafc;
  border-color: #e2e8f0;
  border-left-color: #cbd5e1;
  color: #94a3b8;
  font-style: italic;
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
.tl-level:hover { background: #f8fafc; border-color: #e2e8f0; }
.tl-level.locked { opacity: .55; }
.tl-level.current { background: #fffbeb; border-color: #fde68a; }

.tl-level-dot {
  width: 30px; height: 30px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  font-size: 13px; flex-shrink: 0;
}
.tl-level-dot.completed { background: #dcfce7; color: #16a34a; font-weight: 700; }
.tl-level-dot.learning { background: #fef3c7; color: #b45309; }
.tl-level-dot.locked { background: #f1f5f9; color: #94a3b8; }

.tl-level-main { flex: 1; min-width: 0; }
.tl-level-title-row { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.tl-level-name { font-size: 14px; font-weight: 600; color: var(--color-text); }
.tl-level-xp { font-size: 11px; font-weight: 600; color: #d97706; background: #fef3c7; padding: 2px 8px; border-radius: 4px; }
.tl-level-now { font-size: 11px; font-weight: 700; color: #b45309; background: #fde68a; padding: 2px 8px; border-radius: 4px; }
.tl-level-meta { display: flex; align-items: center; gap: 10px; margin-top: 3px; }
.tl-level-status { font-size: 12px; color: #94a3b8; }
.tl-level-status.completed { color: #16a34a; }
.tl-level-status.learning { color: #d97706; }
.tl-level-tasks { font-size: 12px; color: #94a3b8; }

.tl-level-actions { display: flex; gap: 8px; flex-shrink: 0; }
.tl-level-empty { padding: 14px; text-align: center; font-size: 13px; color: #94a3b8; }

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
.task-picker-item:hover { border-color: #c7d2fe; background: #f5f3ff; transform: translateX(3px); }
.task-picker-title { flex: 1; font-size: 14px; font-weight: 500; color: var(--color-text); }
.task-picker-xp { font-size: 12px; font-weight: 600; color: #d97706; }
.task-picker-arrow { color: #94a3b8; font-size: 18px; }

@media (max-width: 768px) {
  .story-timeline { padding-left: 36px; }
  .tl-node { left: -34px; width: 30px; height: 30px; font-size: 13px; }
  .tl-level { flex-wrap: wrap; }
  .tl-level-actions { width: 100%; justify-content: flex-end; }
}
</style>
