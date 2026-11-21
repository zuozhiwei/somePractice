<template>
  <div class="chapter-page">
    <nav class="chapter-crumb" aria-label="课程位置">
      <RouterLink :to="{ path: '/learn', query: { place: 'hub:' + route.params.mapId } }"><el-icon><Back /></el-icon> 返回世界地图</RouterLink>
    </nav>

    <p v-if="loading" class="chapter-state" role="status">正在打开这一章…</p>
    <div v-else-if="loadError" class="chapter-state" role="alert">
      <p>{{ loadError }}</p>
      <el-button round @click="loadChapter(route.params.mapId)">再试一次</el-button>
    </div>

    <template v-else-if="chapter">
      <header class="chapter-hero">
        <p v-if="chapter.chapterNumber" class="chapter-number">第 {{ chapter.chapterNumber }} 章</p>
        <h1>{{ chapter.name }}</h1>
        <p v-if="chapter.story || chapter.description" class="chapter-goal">{{ chapter.story || chapter.description }}</p>
        <div class="chapter-actions">
          <RouterLink v-if="currentLevel" class="continue-link" :to="'/level/' + currentLevel.id">
            {{ currentLevel.progress ? '继续' : '开始' }} {{ currentLevel.name }} <el-icon aria-hidden="true"><Right /></el-icon>
          </RouterLink>
          <div v-if="chapter.totalLevels" class="chapter-progress">
            <progress :value="chapter.levelsCompleted" :max="chapter.totalLevels" aria-label="关卡验收进度" />
            <span>{{ chapter.levelsCompleted }} / {{ chapter.totalLevels }} 关完成</span>
          </div>
        </div>
      </header>

      <section class="chapter-content" aria-labelledby="chapter-milestones">
        <h2 id="chapter-milestones">本章关卡</h2>
        <ol v-if="chapter.levels?.length" class="chapter-list">
          <li v-for="level in chapter.levels" :key="level.id" :class="{ locked: level.status === 'LOCKED' }">
            <div class="chapter-level-number" :class="{ complete: level.status === 'COMPLETED' }">
              <span v-if="level.levelNumber" :aria-label="'第 ' + level.levelNumber + ' 关'">{{ level.levelNumber }}</span>
              <el-icon v-else-if="level.status === 'COMPLETED'"><Check /></el-icon>
              <el-icon v-else-if="level.status === 'LOCKED'"><Lock /></el-icon>
            </div>
            <div class="chapter-level-copy">
              <h3>
                <span v-if="level.status === 'LOCKED'">{{ level.name }}</span>
                <RouterLink v-else class="level-link" :to="'/level/' + level.id">{{ level.name }}</RouterLink>
              </h3>
              <p v-if="level.description">{{ level.description }}</p>
              <small v-if="level.status === 'LOCKED'">
                {{ level.prerequisiteNames?.length
                  ? '完成「' + level.prerequisiteNames.join('」「') + '」后解锁'
                  : '这一关暂未解锁' }}
              </small>
              <small v-else>{{ level.status === 'COMPLETED' ? '已完成' : '可开始' }} · {{ level.taskDone }} / {{ level.taskTotal }} 项任务通过</small>
            </div>
            <el-icon v-if="level.status !== 'LOCKED'" class="chapter-arrow" aria-hidden="true"><Right /></el-icon>
          </li>
        </ol>
        <p v-else class="chapter-state">这一章的关卡还在准备中。</p>
      </section>

      <details class="chapter-guide" @toggle="openGuide">
        <summary><el-icon><ChatDotRound /></el-icon> 导师提示</summary>
        <div class="chapter-guide-body">
          <p v-if="mentorLoading" role="status">正在加载提示…</p>
          <p v-else-if="mentorError" role="alert">提示暂时加载不了。</p>
          <p v-else-if="mentor?.content">{{ mentor.content }}</p>
          <div class="chapter-guide-actions">
            <RouterLink to="/mentorship">{{ auth.user?.role === 'EXTERNAL' ? '查看我的提交' : '找学长或导师' }}</RouterLink>
            <el-button v-if="mentorError" :disabled="mentorLoading" round @click="loadMentor">再试一次</el-button>
          </div>
        </div>
      </details>
    </template>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Back, ChatDotRound, Check, Lock, Right } from '@element-plus/icons-vue'
import { mapsApi } from '../api/maps'
import { useAuthStore } from '../store/auth'

const route = useRoute()
const auth = useAuthStore()
const chapter = ref(null)
const loading = ref(true)
const loadError = ref('')
const mentor = ref(null)
const mentorLoading = ref(false)
const mentorError = ref(false)
const currentLevel = computed(() => chapter.value?.levels?.find(level => level.status === 'LEARNING'))
let requestId = 0

async function loadChapter(id) {
  const request = ++requestId
  loading.value = true
  loadError.value = ''
  chapter.value = null
  mentor.value = null
  mentorLoading.value = false
  mentorError.value = false
  try {
    const { data } = await mapsApi.detail(id)
    if (request === requestId) chapter.value = data.data
  } catch (error) {
    if (request === requestId) loadError.value = error.response?.data?.message || '章节暂时加载不了。'
  } finally {
    if (request === requestId) loading.value = false
  }
}

async function loadMentor() {
  const request = requestId
  if (!chapter.value || mentorLoading.value) return
  mentorLoading.value = true
  mentorError.value = false
  try {
    const { data } = await mapsApi.mentorDialog(chapter.value.id)
    if (request === requestId) mentor.value = data.data
  } catch {
    if (request === requestId) mentorError.value = true
  } finally {
    if (request === requestId) mentorLoading.value = false
  }
}

function openGuide(event) {
  if (event.currentTarget.open && !mentor.value && !mentorError.value) loadMentor()
}

watch(() => route.params.mapId, id => {
  if (id) loadChapter(id)
}, { immediate: true })
onBeforeUnmount(() => { requestId++ })
</script>

<style scoped>
.chapter-page { max-width: 980px; margin-inline: auto; }
.chapter-crumb { display: flex; align-items: center; min-height: 40px; margin-bottom: 10px; }
.chapter-crumb a, .chapter-guide a { display: inline-flex; align-items: center; gap: 7px; color: var(--color-primary); font-weight: 600; font-size: 13px; text-decoration: none; }
.chapter-crumb a:hover, .chapter-guide a:hover { text-decoration: underline; text-underline-offset: 4px; }
.chapter-hero { padding: clamp(23px, 4vw, 46px); border-radius: 20px; background: #173d35; color: #fcf8eb; overflow-wrap: anywhere; }
.chapter-number { font-weight: 650; color: #f3d578; font-size: 13px; margin-bottom: 10px; }
.chapter-hero h1 { font-size: clamp(27px, 4.3vw, 45px); line-height: 1.25; letter-spacing: -.035em; }
.chapter-goal { margin-top: 12px; max-width: 60ch; white-space: pre-line; line-height: 1.75; color: #dfebe1; }
.chapter-actions { display: flex; align-items: center; flex-wrap: wrap; gap: 18px 26px; margin-top: 20px; }
.chapter-progress { min-width: 120px; color: #ecf3e8; font-size: 12px; }
.chapter-progress progress { width: min(150px, 100%); height: 6px; display: block; margin-bottom: 5px; appearance: none; border: 0; border-radius: 9px; overflow: hidden; color: #f3d578; background: #46665d; }
.chapter-progress progress::-webkit-progress-bar { background-color: #46665d; }
.chapter-progress progress::-webkit-progress-value { background-color: #f3d578; }
.chapter-progress progress::-moz-progress-bar { background-color: #f3d578; }
.continue-link { display: inline-flex; gap: 9px; min-height: 44px; align-items: center; padding: 9px 16px; border-radius: 8px; background: #f3d578; color: #263b26; text-decoration: none; font-weight: 700; font-size: 14px; }
.continue-link:hover { background: #fae5a4; }
.continue-link .el-icon { flex: none; }

.chapter-content { margin-top: 27px; }
.chapter-content > h2 { margin-bottom: 12px; font-size: clamp(19px, 2.4vw, 24px); }
.chapter-list { list-style: none; overflow-wrap: anywhere; }
.chapter-list li {
  position: relative; display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center;
  gap: 13px; margin-bottom: 8px; padding: 15px clamp(13px, 2.5vw, 24px); border: 1px solid var(--color-border);
  background: var(--color-surface); border-radius: 12px;
}
.chapter-list li:not(.locked):hover { border-color: #829e8b; background: #f0f4e9; }
.chapter-list .locked { background: #f1f0e8; grid-template-columns: auto minmax(0, 1fr); }
.chapter-level-number { width: 34px; height: 34px; display: grid; place-items: center; border-radius: 9px; background: #e3ede1; color: #245242; font-size: 14px; font-weight: 700; }
.chapter-level-number.complete { background: #2b7257; color: white; }
.locked .chapter-level-number { background: #e7e8de; color: #5a695d; }
.chapter-level-copy h3 { font-size: 16px; font-weight: 700; line-height: 1.5; }
.chapter-level-copy p { color: var(--color-text-secondary); font-size: 14px; line-height: 1.65; margin-top: 3px; }
.chapter-level-copy small { display: block; margin-top: 7px; color: #4d6258; font-size: 12px; line-height: 1.55; }
.level-link { color: inherit; text-decoration: none; }
.level-link::after { content: ''; position: absolute; inset: 0; border-radius: 12px; }
.level-link:focus-visible { outline: none; }
.level-link:focus-visible::after { outline: 3px solid #a87326; outline-offset: 2px; }
.chapter-arrow { color: var(--color-primary); }

.chapter-guide, .chapter-state { padding: 16px 20px; background: var(--color-surface); border: 1px solid var(--color-border); border-radius: 13px; }
.chapter-guide { margin-top: 20px; }
.chapter-guide summary { min-height: 28px; cursor: pointer; font-size: 14px; font-weight: 700; }
.chapter-guide summary .el-icon { vertical-align: middle; margin-inline: 5px; }
.chapter-guide-body { margin-top: 8px; padding-inline-start: 18px; color: var(--color-text-secondary); font-size: 14px; line-height: 1.7; }
.chapter-guide-actions { display: flex; gap: 12px; align-items: center; flex-wrap: wrap; margin-top: 10px; }
.chapter-state { color: var(--color-text-secondary); }
.chapter-state p { margin-bottom: 12px; }
</style>
