<template>
  <div v-if="view === 'project'" class="learn-page project-page">
    <section class="project-hero" aria-labelledby="project-title">
      <div class="project-copy">
        <div class="project-label"><span></span> 全栈实战项目</div>
        <h1 id="project-title">校园树洞</h1>
        <p>从在电脑上建第一个文件夹开始，做出同学可以匿名投稿、评论和收到回复的校园树洞。最后，邀请同学打开你发布的网址。</p>
        <div class="project-actions">
          <RouterLink v-if="nextLevel" class="workshop-button warm" :to="'/level/' + nextLevel">
            {{ store.completedLevels ? '继续我的项目' : '开始第一关' }} <ArrowRight aria-hidden="true" />
          </RouterLink>
          <RouterLink v-else-if="store.progress && store.totalLevels && store.completedLevels === store.totalLevels"
                      class="workshop-button warm" :to="{ path: '/learn', query: { view: 'project' }, hash: '#stages' }">
            回顾项目 <ArrowRight aria-hidden="true" />
          </RouterLink>
          <button v-else-if="loadFailed" type="button" class="workshop-button warm" @click="load">重试加载进度</button>
          <RouterLink v-else class="workshop-button warm" to="/learn?view=story">
            查看章节 <ArrowRight aria-hidden="true" />
          </RouterLink>
        </div>
      </div>
      <div class="project-art"><StudioIllustration sizes="(max-width: 610px) 100vw, (max-width: 768px) 48vw, (max-width: 1690px) calc((100vw - 270px) / 2), 680px" priority="high" /></div>
    </section>

    <div v-if="store.progress && store.totalLevels" class="project-summary">
      <section class="resume-card" aria-labelledby="resume-title">
        <span class="resume-icon"><Flag aria-hidden="true" /></span>
        <div>
          <p>{{ current ? '下一步' : store.completedLevels === store.totalLevels ? '课程已完成' : '学习进度' }}</p>
          <h2 id="resume-title">{{ current?.currentLevelName || (store.completedLevels === store.totalLevels ? '全部关卡已通过' : '在章节目录查看解锁条件') }}</h2>
        </div>
        <RouterLink v-if="current" :to="'/map/' + current.id" class="resume-link" :aria-label="'查看章节 ' + current.name">
          {{ current.name }} <ArrowRight aria-hidden="true" />
        </RouterLink>
      </section>
      <section class="progress-card" aria-label="项目进度">
        <div><strong>已完成</strong><span><b>{{ store.completedLevels }}</b> / {{ store.totalLevels }} 关</span></div>
        <progress :value="store.completedLevels" :max="store.totalLevels" aria-label="已完成关卡"></progress>
      </section>
    </div>

    <section id="stages" class="learn-stages" aria-labelledby="stages-title">
      <div class="stages-heading">
        <div>
          <h2 id="stages-title">项目路线</h2>
          <p>在同一份代码上持续迭代，直到可以邀请同学试用。</p>
        </div>
        <RouterLink class="text-link" to="/learn?view=story">全部章节 <ArrowRight aria-hidden="true" /></RouterLink>
      </div>
      <p v-if="loadFailed" class="route-message" role="status">进度暂时无法加载，下面仍可查看项目路线。</p>
      <p v-else-if="store.progress && !chapters?.length" class="route-message">课程尚未发布，下面是项目路线预览。</p>
      <ProjectStages :chapters="chapters?.length ? chapters : undefined" :current-id="current?.id" />
    </section>
  </div>

  <div v-else class="learn-page">
    <component :is="currentComponent" :key="view" />
  </div>
</template>

<script setup>
import { computed, defineAsyncComponent, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowRight, Flag } from '@element-plus/icons-vue'
import { learningView } from '../workshop/learningViews'
import { useMapStore } from '../store/map'
import ProjectStages from '../workshop/ProjectStages.vue'
import StudioIllustration from '../workshop/StudioIllustration.vue'

const route = useRoute()
const store = useMapStore()
const loadFailed = ref(false)
const components = {
  world: defineAsyncComponent(() => import('./WorldMap.vue')),
  story: defineAsyncComponent(() => import('./MapBoard.vue')),
  tree: defineAsyncComponent(() => import('./SkillTree.vue'))
}
const view = computed(() => learningView(route.query.view))
const currentComponent = computed(() => components[view.value])
const chapters = computed(() => store.progress?.maps)
const current = computed(() => chapters.value?.find(chapter => chapter.currentLevelId))
const nextLevel = computed(() => current.value?.currentLevelId)

async function load() {
  loadFailed.value = false
  loadFailed.value = !(await store.fetchProgress())
}
watch(view, value => { if (value === 'project') load() }, { immediate: true })
</script>

<style scoped>
.learn-page { min-width: 0; }
.project-page { max-width: 1390px; margin-inline: auto; }
.project-hero { display: grid; grid-template-columns: minmax(0, 1.05fr) minmax(0, 1fr); min-height: 295px; color: #fff9e8; background: #1c4337; border-radius: 20px; overflow: hidden; }
.project-copy { display: flex; flex-direction: column; align-items: flex-start; justify-content: center; padding: clamp(25px, 3.6vw, 45px); }
.project-label { display: inline-flex; gap: 8px; align-items: center; color: #e4e8cb; font-size: 12px; font-weight: 650; letter-spacing: .11em; }
.project-label span { width: 7px; height: 7px; background: #f3d578; border-radius: 50%; }
.project-hero h1 { margin: 9px 0 12px; font-size: clamp(36px, 4.4vw, 62px); line-height: 1.15; letter-spacing: -.065em; }
.project-hero p { font-size: 15px; line-height: 1.75; color: #e2ebdc; max-width: 29em; }
.project-actions { display: flex; margin-top: 23px; }
.project-art { position: relative; background: #d6ac66; }
.project-art { --studio-position: center 53%; }
.project-summary { display: grid; grid-template-columns: minmax(0, 1.6fr) minmax(0, 1fr); gap: 12px; margin-top: 14px; }
.resume-card, .progress-card { background: var(--color-surface); border: 1px solid var(--color-border); border-radius: 14px; padding: 14px 16px; }
.resume-card { display: flex; gap: 12px; align-items: center; }
.resume-icon { width: 40px; height: 40px; border-radius: 11px; flex: none; display: grid; place-items: center; background: #e7edde; color: #325c4a; }
.resume-icon svg { width: 19px; height: 19px; }
.resume-card h2 { font-size: 15px; line-height: 1.45; }
.resume-card p { color: var(--color-text-secondary); font-size: 12px; margin-bottom: 3px; }
.resume-link { margin-left: auto; display: inline-flex; align-items: center; gap: 5px; color: var(--color-primary); min-height: 38px; text-decoration: none; font-size: 13px; text-align: end; }
.resume-link:hover { text-decoration: underline; }
.resume-link svg { width: 14px; height: 14px; flex: none; }
.progress-card { display: flex; flex-direction: column; justify-content: center; gap: 9px; }
.progress-card div { display: flex; justify-content: space-between; align-items: end; gap: 10px; font-size: 13px; }
.progress-card span { color: var(--color-text-secondary); font-variant-numeric: tabular-nums; }
.progress-card b { color: var(--color-text); font-size: 17px; }
.progress-card progress { width: 100%; height: 6px; accent-color: #3c6a50; }
.learn-stages { margin-top: 27px; scroll-margin-top: 75px; }
.stages-heading { display: flex; flex-wrap: wrap; align-items: flex-end; justify-content: space-between; gap: 5px 14px; margin-bottom: 14px; }
.stages-heading h2 { font-size: clamp(23px, 2.2vw, 30px); line-height: 1.2; letter-spacing: -.045em; }
.stages-heading p { color: var(--color-text-secondary); font-size: 14px; line-height: 1.5; margin-top: 6px; }
.route-message { margin-bottom: 12px; color: var(--color-text-secondary); font-size: 14px; }
@media (max-width: 810px) {
  .project-summary { grid-template-columns: 1fr; }
}
@media (max-width: 610px) {
  .project-hero { grid-template-columns: 1fr; border-radius: 16px; }
  .project-art { aspect-ratio: 16 / 7.8; }
  .project-copy { padding: 22px; }
}
</style>
