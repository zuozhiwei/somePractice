<template>
  <ol class="project-stages" aria-label="校园树洞的开发阶段">
    <li v-for="stage in stages" :key="stage.id" :class="{ 'current-stage': stage.isCurrent }">
      <component :is="stage.chapter ? RouterLink : 'article'" class="project-stage"
                 :to="stage.chapter ? '/map/' + stage.id : undefined">
        <div class="stage-top">
          <span class="stage-icon" :class="'tone-' + stage.tone"><component :is="icons[stage.icon]" aria-hidden="true" /></span>
          <span class="stage-status" :class="{ 'stage-status-current': stage.isCurrent }">
            <Check v-if="stage.chapter?.status === 'COMPLETED'" aria-hidden="true" />
            <Lock v-else-if="stage.chapter?.status === 'LOCKED'" aria-hidden="true" />
            {{ stage.isCurrent ? '进行中' : stage.chapter?.status === 'COMPLETED' ? '已完成' : stage.chapter?.status === 'LOCKED' ? '待解锁' : '第 ' + stage.order + ' 阶段' }}
          </span>
        </div>
        <h3>{{ stage.chapter?.name || stage.name }}</h3>
        <p>{{ stage.result }}</p>
        <ArrowRight v-if="stage.chapter" class="stage-arrow" aria-hidden="true" />
      </component>
    </li>
  </ol>
</template>

<script setup>
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { ArrowRight, Check, Checked, Coin, Collection, Connection, Flag, Lock, Monitor, TrendCharts, User } from '@element-plus/icons-vue'
import { projectStages } from './projectRoute'

const props = defineProps({ chapters: Array, currentId: Number })
const icons = { Checked, Coin, Collection, Connection, Flag, Monitor, TrendCharts, User }
const stages = computed(() => {
  if (!props.chapters) return projectStages.map((stage, index) => ({ ...stage, order: index + 1 }))
  return props.chapters.map((chapter, index) => {
    const stage = projectStages.find(item => item.id === chapter.id)
    return {
      id: chapter.id,
      name: chapter.name,
      chapter,
      result: chapter.description || stage?.result,
      icon: stage?.icon || 'Flag',
      tone: stage?.tone || 'mint',
      order: index + 1,
      isCurrent: chapter.id === props.currentId
    }
  })
})
</script>

<style scoped>
.project-stages { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; list-style: none; }
.project-stages li { min-width: 0; }
.project-stage { position: relative; display: block; height: 100%; padding: 17px 17px 19px; border: 1px solid var(--color-border); border-radius: 15px; background: var(--color-surface); color: var(--color-text); text-decoration: none; }
a.project-stage { transition: border-color .18s, background .18s; }
a.project-stage:hover { border-color: #88a799; background: #fffffb; }
.current-stage .project-stage { border-color: #7f9e8b; background: #f4f8ee; }
.stage-top { display: flex; align-items: center; justify-content: space-between; gap: 6px; margin-bottom: 15px; }
.stage-icon { display: grid; place-items: center; width: 37px; height: 37px; flex: none; border-radius: 10px; color: #244f40; }
.stage-icon svg { width: 18px; height: 18px; }
.tone-mint { background: #e2edd9; }
.tone-sand { background: #f7e8bc; color: #735117; }
.tone-lilac { background: #ece7f5; color: #584876; }
.tone-peach { background: #f6e6d8; color: #744b37; }
.stage-status { display: inline-flex; gap: 4px; align-items: center; color: var(--color-text-secondary); font-size: 11px; white-space: nowrap; }
.stage-status svg { width: 12px; height: 12px; }
.stage-status-current { color: #285949; font-weight: 750; background: #e4eedc; padding: 4px 7px; border-radius: 6px; }
.project-stage h3 { font-size: 15px; line-height: 1.4; margin-bottom: 5px; overflow-wrap: anywhere; }
.project-stage p { color: var(--color-text-secondary); font-size: 13px; line-height: 1.55; }
a.project-stage h3 { padding-right: 18px; }
.stage-arrow { width: 15px; height: 15px; position: absolute; top: 73px; right: 14px; color: #507364; }
@media (max-width: 1100px) { .project-stages { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 480px) {
  .project-stages { gap: 8px; }
  .project-stage { padding: 12px 10px 14px; }
  .stage-top { margin-bottom: 10px; }
  .stage-arrow { top: 62px; right: 8px; }
}
</style>
