<template>
  <div>
    <el-dialog v-model="doorOpen" title="推荐任务" width="min(94vw, 840px)" top="5vh" :close-on-click-modal="false">
      <div v-if="missionLoading" class="dialog-state" role="status">正在加载推荐任务…</div>
      <div v-else-if="missionError" class="dialog-state" role="alert">暂时无法加载任务。<button @click="loadMission">重试</button></div>
      <template v-else-if="mission?.task">
        <div class="mission-heading">
          <span v-if="mission.level" class="level-context">
            所属关卡
            <RouterLink :to="{ path: '/level/' + mission.level.id, query: { task: mission.task.id, n: mission.level.orderIndex } }">
              {{ mission.level.chapterName }} › {{ mission.level.name }}
            </RouterLink>
          </span>
          <span class="status">{{ taskStatus }}</span>
        </div>
        <h2 class="mission-title">{{ mission.task.title }}</h2>
        <p v-if="mission.state === 'WAITING_REVIEW'" class="waiting-note">这项任务已提交，正在等待审核。你可以查看提交记录，也可以补充作业。</p>
        <div class="mission-tabs" role="tablist" aria-label="任务资料">
          <button id="mission-task-tab" role="tab" :aria-selected="tab === 'task'" aria-controls="mission-task-panel" :class="{ active: tab === 'task' }" @click="tab = 'task'">任务要求</button>
          <button id="mission-guide-tab" role="tab" :aria-selected="tab === 'guide'" aria-controls="mission-guide-panel" :class="{ active: tab === 'guide' }" @click="tab = 'guide'">配套教程</button>
        </div>
        <div v-if="tab === 'task'" id="mission-task-panel" role="tabpanel" aria-labelledby="mission-task-tab" class="mission-body">
          <TutorialReader :content="mission.task.description || '请向老师了解本项任务的要求。'" />
          <div v-if="mission.task.acceptanceCriteria?.length" class="criteria"><h3>验收标准</h3><ul><li v-for="(item, index) in mission.task.acceptanceCriteria" :key="index">{{ item.criteria }}</li></ul></div>
          <div v-if="mission.task.reviewComment" class="feedback"><strong>最近审核意见</strong><p>{{ mission.task.reviewComment }}</p></div>
          <p v-if="mission.task.aiFeedback" class="feedback">{{ mission.task.aiFeedback }}</p>
        </div>
        <div v-else id="mission-guide-panel" role="tabpanel" aria-labelledby="mission-guide-tab" class="mission-body">
          <details v-for="(tutorial, index) in mission.tutorials" :key="tutorial.id" :open="index === 0"><summary>{{ tutorial.title }}</summary><TutorialReader :content="tutorial.content" /></details>
          <p v-if="!mission.tutorials?.length">这个关卡还没有配套教程。</p>
        </div>
        <div class="mission-actions"><button v-if="mission.task.submissionStatus !== 'NONE'" :disabled="opening" @click="openHistory">查看提交记录</button><button :disabled="missionLoading" @click="loadMission">刷新任务</button><button class="primary" :disabled="opening" @click="openSubmit">{{ opening ? '正在打开…' : mission.state === 'WAITING_REVIEW' ? '补充作业' : '提交作业' }}</button></div>
      </template>
      <div v-else class="dialog-state"><strong>{{ emptyTitle }}</strong><p>{{ emptyMessage }}</p><button @click="loadMission">刷新任务</button></div>
    </el-dialog>
    <component :is="submitComponent" v-if="submitComponent" ref="submitDialog" :task="submissionTask" @done="onSubmitted" />
    <component :is="historyComponent" v-if="historyComponent" ref="historyDialog" />
  </div>
</template>

<script setup>
import { ref, shallowRef, computed, watch, defineAsyncComponent, onBeforeUnmount, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { adventureApi } from '../api/adventure'

const doorOpen = defineModel({ type: Boolean })
const props = defineProps({ chapterId: Number })
const emit = defineEmits(['submitted'])
const TutorialReader = defineAsyncComponent(() => import('../components/TutorialReader.vue'))
const mission = ref(null), missionLoading = ref(false), missionError = ref(false), tab = ref('task')
const submitComponent = shallowRef(null), historyComponent = shallowRef(null)
const submitDialog = ref(null), historyDialog = ref(null), submissionTask = ref(null), opening = ref(false)
let requestId = 0, disposed = false
const taskStatus = computed(() => ({ NONE: '尚未提交', PENDING: '等待审核', REJECTED: '可以修改后重试', AI_REVIEWED: '待人工审核', APPROVED: '已完成' }[mission.value?.task?.submissionStatus] || ''))
const emptyTitle = computed(() => mission.value?.state === 'COMPLETE' ? (props.chapterId ? '本章已通关' : '全部关卡已通关') : '暂无推荐任务')
const emptyMessage = computed(() => ({ COMPLETE: '可以回到地图，复习已完成的关卡。', EMPTY: '新的学习任务发布后，会出现在这里。', BLOCKED: '可在地图上查看各关卡的解锁条件。' }[mission.value?.state] || '请稍后再试。'))
async function loadMission() {
  const current = ++requestId
  missionLoading.value = true; missionError.value = false; mission.value = null
  try {
    const response = await adventureApi.mission(props.chapterId)
    if (current === requestId && !disposed) mission.value = response.data.data
  } catch { if (current === requestId && !disposed) missionError.value = true }
  finally { if (current === requestId && !disposed) missionLoading.value = false }
}
async function openSubmit() {
  if (!mission.value?.task || opening.value) return
  opening.value = true
  try {
    submissionTask.value = mission.value.task
    if (!submitComponent.value) submitComponent.value = (await import('../components/TaskSubmitDialog.vue')).default
    if (disposed || !doorOpen.value) return
    await nextTick(); submitDialog.value?.open()
  } catch { ElMessage.error('提交窗口加载失败，请重试。') } finally { opening.value = false }
}
async function openHistory() {
  if (!mission.value?.task || opening.value) return
  const task = mission.value.task; opening.value = true
  try {
    if (!historyComponent.value) historyComponent.value = (await import('../components/SubmissionHistoryDialog.vue')).default
    if (disposed || !doorOpen.value) return
    await nextTick(); historyDialog.value?.open(task)
  } catch { ElMessage.error('提交记录加载失败，请重试。') } finally { opening.value = false }
}

watch([doorOpen, () => props.chapterId], ([open]) => {
  if (open) { tab.value = 'task'; loadMission() }
  else { requestId++; missionLoading.value = false }
}, { immediate: true })
function onSubmitted() { loadMission(); emit('submitted') }
onBeforeUnmount(() => { disposed = true; requestId++ })
</script>

<style scoped>
.level-context { color:var(--color-text-secondary); font-size:13px; line-height:1.6; }
.level-context a { color:var(--color-primary); text-underline-offset:3px; margin-inline-start:5px; }
.mission-heading{display:flex;justify-content:space-between;align-items:center;margin-top:22px;gap:12px}.seal{font-size:11px;color:#a1793e;letter-spacing:.12em;font-weight:700}.status{font-size:12px;color:#778674}.mission-title{font-size:22px;line-height:1.5;color:#344e3c;margin:8px 0 18px}.mission-tabs{display:flex;gap:10px;border-bottom:1px solid #e5e9df;padding-bottom:10px}.mission-tabs button{padding:8px 14px;border:0;border-radius:7px;background:none;color:#819078;cursor:pointer}.mission-tabs button.active{background:#edf2e6;color:#4a6747;font-weight:700}.mission-body{max-height:48vh;overflow:auto;padding:18px 3px;line-height:1.8}.mission-body summary{cursor:pointer;font-weight:700;padding:12px 0;color:#536b4f}.criteria{margin-top:20px;background:#f7f8f1;padding:14px 18px;border-radius:10px}.criteria h3{font-size:14px;margin:0}.criteria ul{padding-left:20px;font-size:13px}.feedback{white-space:pre-wrap;color:#7c725c;background:#faf5e9;padding:14px;border-radius:8px}.waiting-note{font-size:13px;line-height:1.8;color:#987334;background:#fff8e7;padding:12px;border-radius:8px}.mission-actions{display:flex;justify-content:flex-end;gap:8px;flex-wrap:wrap;padding-top:14px;border-top:1px solid #e5e9df}.mission-actions button{border:1px solid #d8e1d4;border-radius:8px;padding:10px 16px;background:#fff;color:#567150;cursor:pointer}.mission-actions button.primary{background:#436d50;color:#fff;border-color:#436d50}.mission-actions button:disabled{opacity:.5;cursor:wait}.dialog-state{padding:28px 6px;color:#7a856e;line-height:1.8}
.dialog-state button { color:var(--color-primary,#216552); border:none; background:none; padding:6px; cursor:pointer; }
@media(max-width:650px) { .mission-title{font-size:19px}.mission-body{max-height:44vh} }
</style>
