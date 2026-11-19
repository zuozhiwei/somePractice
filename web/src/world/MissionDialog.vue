<template>
  <div>
    <el-dialog v-model="doorOpen" title="推荐委托" width="min(94vw, 840px)" top="5vh" :close-on-click-modal="false">
      <div v-if="missionLoading" class="dialog-state" role="status">正在取出你的委托……</div>
      <div v-else-if="missionError" class="dialog-state" role="alert">委托暂时未能送达。<button @click="loadMission">重新领取</button></div>
      <template v-else-if="mission?.task">
        <div class="mission-heading"><span class="seal">当前任务</span><span class="status">{{ taskStatus }}</span></div>
        <h2 class="mission-title">{{ mission.task.title }}</h2>
        <p v-if="mission.state === 'WAITING_REVIEW'" class="waiting-note">你已提交当前可做的委托，正在等待审核。可以查看记录或补充成果，审核通过后再来领取新的委托。</p>
        <div class="mission-tabs" role="tablist" aria-label="委托内容">
          <button id="mission-task-tab" role="tab" :aria-selected="tab === 'task'" aria-controls="mission-task-panel" :class="{ active: tab === 'task' }" @click="tab = 'task'">委托内容</button>
          <button id="mission-guide-tab" role="tab" :aria-selected="tab === 'guide'" aria-controls="mission-guide-panel" :class="{ active: tab === 'guide' }" @click="tab = 'guide'">配套教程</button>
        </div>
        <div v-if="tab === 'task'" id="mission-task-panel" role="tabpanel" aria-labelledby="mission-task-tab" class="mission-body">
          <TutorialReader :content="mission.task.description || '请向老师了解本项任务的要求。'" />
          <div v-if="mission.task.acceptanceCriteria?.length" class="criteria"><h3>完成这份委托需要</h3><ul><li v-for="(item, index) in mission.task.acceptanceCriteria" :key="index">{{ item.criteria }}</li></ul></div>
          <p v-if="mission.task.aiFeedback" class="feedback">{{ mission.task.aiFeedback }}</p>
        </div>
        <div v-else id="mission-guide-panel" role="tabpanel" aria-labelledby="mission-guide-tab" class="mission-body">
          <details v-for="(tutorial, index) in mission.tutorials" :key="tutorial.id" :open="index === 0"><summary>{{ tutorial.title }}</summary><TutorialReader :content="tutorial.content" /></details>
          <p v-if="!mission.tutorials?.length">这份委托没有附带手册，遇到困难可以向老师求助。</p>
        </div>
        <div class="mission-actions"><button v-if="mission.task.submissionStatus !== 'NONE'" :disabled="opening" @click="openHistory">查看提交记录</button><button :disabled="missionLoading" @click="loadMission">刷新委托</button><button class="primary" :disabled="opening" @click="openSubmit">{{ opening ? '正在打开…' : mission.state === 'WAITING_REVIEW' ? '补充提交' : '提交成果' }}</button></div>
      </template>
      <div v-else class="dialog-state"><strong>{{ emptyTitle }}</strong><p>{{ emptyMessage }}</p><button @click="loadMission">再看看</button></div>
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
const emit = defineEmits(['submitted'])
const TutorialReader = defineAsyncComponent(() => import('../components/TutorialReader.vue'))
const mission = ref(null), missionLoading = ref(false), missionError = ref(false), tab = ref('task')
const submitComponent = shallowRef(null), historyComponent = shallowRef(null)
const submitDialog = ref(null), historyDialog = ref(null), submissionTask = ref(null), opening = ref(false)
let requestId = 0, disposed = false
const taskStatus = computed(() => ({ NONE: '尚未提交', PENDING: '等待审核', REJECTED: '可以修改后重试', AI_REVIEWED: '等待最终审核', APPROVED: '已完成' }[mission.value?.task?.submissionStatus] || ''))
const emptyTitle = computed(() => mission.value?.state === 'COMPLETE' ? '所有委托都完成了！' : '暂时没有新的委托')
const emptyMessage = computed(() => ({ COMPLETE: '这段旅程的委托已经全部通过，继续在岛上探索吧。', EMPTY: '老师还在准备新的内容，稍后再来看看。', BLOCKED: '当前学习进度需要老师确认，请联系老师后再来领取。' }[mission.value?.state] || '稍后再来看看。'))
async function loadMission() {
  const current = ++requestId
  missionLoading.value = true; missionError.value = false; mission.value = null
  try {
    const response = await adventureApi.mission()
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

watch(doorOpen, open => {
  if (open) { tab.value = 'task'; loadMission() }
  else { requestId++; missionLoading.value = false }
}, { immediate: true })
function onSubmitted() { loadMission(); emit('submitted') }
onBeforeUnmount(() => { disposed = true; requestId++ })
</script>

<style scoped>
.mission-heading{display:flex;justify-content:space-between;align-items:center;margin-top:22px;gap:12px}.seal{font-size:11px;color:#a1793e;letter-spacing:.12em;font-weight:700}.status{font-size:12px;color:#778674}.mission-title{font-size:22px;line-height:1.5;color:#344e3c;margin:8px 0 18px}.mission-tabs{display:flex;gap:10px;border-bottom:1px solid #e5e9df;padding-bottom:10px}.mission-tabs button{padding:8px 14px;border:0;border-radius:7px;background:none;color:#819078;cursor:pointer}.mission-tabs button.active{background:#edf2e6;color:#4a6747;font-weight:700}.mission-body{max-height:48vh;overflow:auto;padding:18px 3px;line-height:1.8}.mission-body summary{cursor:pointer;font-weight:700;padding:12px 0;color:#536b4f}.criteria{margin-top:20px;background:#f7f8f1;padding:14px 18px;border-radius:10px}.criteria h3{font-size:14px;margin:0}.criteria ul{padding-left:20px;font-size:13px}.feedback{white-space:pre-wrap;color:#7c725c;background:#faf5e9;padding:14px;border-radius:8px}.waiting-note{font-size:13px;line-height:1.8;color:#987334;background:#fff8e7;padding:12px;border-radius:8px}.mission-actions{display:flex;justify-content:flex-end;gap:8px;flex-wrap:wrap;padding-top:14px;border-top:1px solid #e5e9df}.mission-actions button{border:1px solid #d8e1d4;border-radius:8px;padding:10px 16px;background:#fff;color:#567150;cursor:pointer}.mission-actions button.primary{background:#436d50;color:#fff;border-color:#436d50}.mission-actions button:disabled{opacity:.5;cursor:wait}.dialog-state{padding:28px 6px;color:#7a856e;line-height:1.8}
.dialog-state button { color:var(--color-primary,#216552); border:none; background:none; padding:6px; cursor:pointer; }
@media(max-width:650px) { .mission-title{font-size:19px}.mission-body{max-height:44vh} }
</style>
