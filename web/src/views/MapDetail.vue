<template>
  <div class="page" v-loading="loading">
    <template v-if="map">
      <div class="page-title-row">
        <div>
          <div class="crumb">
            <span class="crumb-link" @click="$router.push('/map')">📖 冒险之旅</span>
            <span class="crumb-sep">/</span>
            <span>{{ map.icon }} {{ map.name }}</span>
          </div>
          <div class="title-line">
            <h1 class="page-title">{{ map.name }}</h1>
            <el-tag :type="map.status === 'COMPLETED' ? 'success' : map.status === 'LOCKED' ? 'info' : 'primary'"
                    size="small" effect="light">
              {{ map.status === 'COMPLETED' ? '✅ 已通关' : map.status === 'LOCKED' ? '🔒 未解锁' : '🚀 进行中' }}
            </el-tag>
            <el-tag size="small" effect="plain">{{ map.levelsCompleted }}/{{ map.totalLevels }} 关</el-tag>
          </div>
        </div>
        <el-button @click="$router.push('/map')" text>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" style="margin-right:4px"><path d="M19 12H5M12 19l-7-7 7-7"/></svg>
          返回地图
        </el-button>
      </div>

      <!-- Story banner -->
      <div class="story-banner">
        <div class="story-label">📜 区域故事</div>
        <div class="story-text">{{ map.story }}</div>
      </div>

      <!-- Mentor NPC -->
      <div class="mentor-card">
        <div class="mentor-avatar">🧙</div>
        <div class="mentor-body">
          <div class="mentor-name">领路人 · 师父</div>
          <div class="mentor-dialog" :class="{ typing: mentorLoading }">{{ mentorLoading ? '……' : mentor?.content }}</div>
        </div>
        <div class="mentor-actions" v-if="mentor">
          <span v-if="mentor.graduated" class="mentor-badge done">🎉 你已出师！</span>
          <span v-else-if="mentor.mapCompleted" class="mentor-badge done">✅ 本区域已通关</span>
          <span v-else-if="mentor.currentLevelName" class="mentor-badge">▶ 下一关：{{ mentor.currentLevelName }}</span>
          <el-button size="small" round @click="fetchMentor" :loading="mentorLoading">再说两句</el-button>
        </div>
      </div>

      <!-- Level list -->
      <div class="level-list" v-if="map.levels?.length">
        <div v-for="lv in map.levels" :key="lv.id" class="level-card"
             :class="{ locked: lv.status === 'LOCKED', completed: lv.status === 'COMPLETED' }"
             @click="goLevel(lv)">
          <div class="level-num" :class="{ done: lv.status === 'COMPLETED' }">
            <span v-if="lv.status === 'COMPLETED'">✓</span>
            <span v-else>{{ lv.orderIndex }}</span>
          </div>
          <div class="level-main">
            <div class="level-title-row">
              <div class="level-name">{{ lv.name }}</div>
              <span class="level-xp">+{{ lv.xpReward }} XP</span>
            </div>
            <div class="level-desc">{{ lv.description }}</div>
            <div class="level-bottom">
              <span class="level-status" :class="lv.status.toLowerCase()">
                <template v-if="lv.status === 'LOCKED'">🔒 未解锁</template>
                <template v-else-if="lv.status === 'COMPLETED'">✅ 已通关</template>
                <template v-else>🚀 挑战中</template>
              </span>
              <span class="level-tasks">{{ lv.taskDone }}/{{ lv.taskTotal }} 任务</span>
            </div>
            <div class="level-progress-bar">
              <div class="level-progress-fill" :style="{ width: lv.progress + '%' }"></div>
            </div>
          </div>
          <svg class="level-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18"><path d="M9 18l6-6-6-6"/></svg>
        </div>
      </div>
      <el-empty v-else description="本区域暂无关卡" />
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { mapsApi } from '../api/maps'
import { ElMessage } from 'element-plus'

const route = useRoute()
const router = useRouter()
const map = ref(null)
const loading = ref(true)
const mentor = ref(null)
const mentorLoading = ref(false)

function goLevel(lv) {
  if (lv.status === 'LOCKED') {
    ElMessage.warning('通关前一关卡后解锁')
    return
  }
  router.push(`/level/${lv.id}?n=${lv.orderIndex}`)
}

async function fetchMentor() {
  mentorLoading.value = true
  try {
    const res = await mapsApi.mentorDialog(map.value.id)
    mentor.value = res.data.data
  } catch (e) {
    console.error(e)
  } finally {
    mentorLoading.value = false
  }
}

async function fetchDetail() {
  loading.value = true
  try {
    const res = await mapsApi.detail(route.params.mapId)
    map.value = res.data.data
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '地图不存在')
    router.push('/map')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await fetchDetail()
  if (map.value) fetchMentor()
})
</script>

<style scoped>
.page { max-width: 1000px; }
.page-title-row { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px; }

.crumb { font-size: 13px; color: var(--color-text-secondary); margin-bottom: 8px; }
.crumb-link { cursor: pointer; }
.crumb-link:hover { color: #6366f1; }
.crumb-sep { margin: 0 6px; color: #cbd5e1; }

.title-line { display: flex; align-items: center; gap: 10px; }

.story-banner {
  background: linear-gradient(135deg, #1e1b4b, #312e81);
  border-radius: var(--radius);
  padding: 20px 24px;
  color: #e0e7ff;
  margin-bottom: 20px;
  box-shadow: var(--shadow-md);
}
.story-label { font-size: 12px; font-weight: 700; letter-spacing: 2px; color: #a5b4fc; margin-bottom: 8px; }
.story-text { font-size: 14px; line-height: 1.9; color: #c7d2fe; }

.mentor-card {
  display: flex; align-items: flex-start; gap: 14px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 16px 20px;
  margin-bottom: 24px;
  box-shadow: var(--shadow-sm);
}
.mentor-avatar {
  width: 48px; height: 48px; border-radius: 50%;
  background: linear-gradient(135deg, #fbbf24, #f59e0b);
  display: flex; align-items: center; justify-content: center;
  font-size: 26px; flex-shrink: 0;
}
.mentor-body { flex: 1; min-width: 0; }
.mentor-name { font-size: 12px; font-weight: 700; color: #b45309; margin-bottom: 4px; }
.mentor-dialog {
  font-size: 14px; color: #475569; line-height: 1.7;
  background: #fffbeb; border: 1px solid #fde68a;
  border-radius: 0 12px 12px 12px; padding: 10px 14px;
}
.mentor-dialog.typing { color: #a16207; }
.mentor-actions { display: flex; flex-direction: column; align-items: flex-end; gap: 8px; flex-shrink: 0; }
.mentor-badge { font-size: 12px; font-weight: 600; color: #6366f1; }
.mentor-badge.done { color: #16a34a; }

.level-list { display: flex; flex-direction: column; gap: 14px; }

.level-card {
  display: flex; align-items: center; gap: 16px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 18px 20px;
  cursor: pointer;
  transition: all .2s;
}
.level-card:hover { box-shadow: var(--shadow-md); transform: translateX(4px); }
.level-card.locked { opacity: .55; cursor: not-allowed; }
.level-card.locked:hover { transform: none; box-shadow: var(--shadow-sm); }
.level-card.completed { border-color: #86efac; background: linear-gradient(90deg, #f0fdf4, #ffffff); }

.level-num {
  width: 44px; height: 44px; border-radius: 12px;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: #fff; font-size: 17px; font-weight: 700;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.level-num.done { background: linear-gradient(135deg, #22c55e, #4ade80); }

.level-main { flex: 1; min-width: 0; }
.level-title-row { display: flex; align-items: center; gap: 10px; margin-bottom: 4px; }
.level-name { font-size: 15px; font-weight: 600; color: var(--color-text); }
.level-xp { font-size: 12px; font-weight: 600; color: #d97706; background: #fef3c7; padding: 1px 8px; border-radius: 10px; }
.level-desc {
  font-size: 13px; color: var(--color-text-secondary); line-height: 1.6;
  margin-bottom: 10px;
  display: -webkit-box; -webkit-line-clamp: 1; -webkit-box-orient: vertical; overflow: hidden;
}
.level-bottom { display: flex; gap: 14px; margin-bottom: 8px; }
.level-status { font-size: 12px; font-weight: 600; }
.level-status.locked { color: #94a3b8; }
.level-status.learning { color: #6366f1; }
.level-status.completed { color: #16a34a; }
.level-tasks { font-size: 12px; color: #94a3b8; }

.level-progress-bar { height: 5px; background: #f1f5f9; border-radius: 3px; overflow: hidden; }
.level-progress-fill { height: 100%; background: linear-gradient(90deg, #6366f1, #8b5cf6); border-radius: 3px; transition: width .4s; }
.level-card.completed .level-progress-fill { background: linear-gradient(90deg, #22c55e, #4ade80); }

.level-arrow { color: #cbd5e1; flex-shrink: 0; }
.level-card:hover .level-arrow { color: #6366f1; }
</style>
