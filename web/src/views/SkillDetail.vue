<template>
  <div class="page" v-loading="loading">
    <div class="page-title-row">
      <div>
        <h1 class="page-title">{{ skill?.name }}</h1>
        <p class="page-subtitle">
          <el-tag :type="levelTag(skill?.tier)" size="small" effect="light">{{ 'Lv.' + skill?.tier }}</el-tag>
          <span style="margin-left:8px;color:var(--color-text-secondary)">{{ skill?.category }}</span>
          <span v-if="skill?.prerequisiteNames?.length" style="margin-left:8px;font-size:13px;color:#e6a23c">
            前置：{{ skill?.prerequisiteNames.join('、') }}
          </span>
        </p>
      </div>
      <el-button @click="$router.back()" text>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" style="margin-right:4px"><path d="M19 12H5M12 19l-7-7 7-7"/></svg>
        返回技能树
      </el-button>
    </div>

    <div class="card-grid cols-2">
      <!-- ============ Tutorials Panel ============ -->
      <div class="section-card">
        <div class="section-header">
          <span>📖 教程</span>
          <span class="count-badge">{{ skill?.tutorials?.length || 0 }} 篇</span>
        </div>
        <div class="section-body" style="padding:0">
          <div v-if="skill?.tutorials?.length">
            <div v-for="(t, idx) in skill.tutorials" :key="t.id" class="tutorial-item">
              <div class="tutorial-header" @click="toggleTutorial(idx)">
                <div class="tutorial-num">{{ idx + 1 }}</div>
                <span class="tutorial-title">{{ t.title }}</span>
                <svg class="tutorial-arrow" :class="{ open: openTutorials[idx] }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18"><polyline points="6 9 12 15 18 9"/></svg>
              </div>
              <div class="tutorial-body" :class="{ open: openTutorials[idx] }">
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
          <span>📝 任务列表</span>
          <span class="count-badge">
            {{ skill?.tasks?.filter(t => t.submissionStatus === 'APPROVED').length || 0 }} / {{ skill?.tasks?.length || 0 }}
          </span>
        </div>
        <div class="section-body" style="padding:0">
          <div v-for="(task, idx) in skill?.tasks" :key="task.id" class="task-item"
               :class="{ expanded: expandedTask === task.id }">
            <div class="task-header" @click="expandedTask = expandedTask === task.id ? null : task.id">
              <div class="task-header-left">
                <span class="task-num">
                  <span v-if="task.submissionStatus === 'APPROVED'" class="done-check">✓</span>
                  <span v-else>{{ idx + 1 }}</span>
                </span>
                <div>
                  <div class="task-title">{{ task.title }}</div>
                  <div class="task-tags">
                    <span class="status-tag" :class="task.submissionStatus.toLowerCase()">
                      {{ statusText(task.submissionStatus) }}
                    </span>
                    <span class="xp-tag">{{ task.xpReward }} XP</span>
                    <span v-if="task.acceptanceCriteria?.length" class="criteria-count">
                      {{ task.acceptanceCriteria.length }} 项验收标准
                    </span>
                  </div>
                </div>
              </div>
              <svg class="task-arrow" :class="{ open: expandedTask === task.id }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16"><polyline points="6 9 12 15 18 9"/></svg>
            </div>
            <div class="task-body" :class="{ open: expandedTask === task.id }">
              <div class="task-description">
                <div class="desc-label">📋 任务说明</div>
                <div class="desc-content">{{ task.description }}</div>
              </div>
              <div v-if="task.acceptanceCriteria?.length" class="task-criteria">
                <div class="desc-label">✅ 验收标准</div>
                <div v-for="(c, i) in task.acceptanceCriteria" :key="i" class="criteria-item">
                  <span class="criteria-num">{{ i + 1 }}.</span>
                  <span>{{ c.criteria }}</span>
                  <span class="criteria-weight">权重 {{ c.weight }}%</span>
                </div>
              </div>
              <div v-if="task.aiFeedback" class="ai-feedback-box">
                <div class="ai-feedback-header">
                  <span>🤖 AI 评审结果</span>
                  <span class="ai-score">{{ extractScore(task.aiFeedback) }} 分</span>
                </div>
                <div class="ai-feedback-text">{{ extractFeedback(task.aiFeedback) }}</div>
              </div>
              <div class="task-actions">
                <div class="task-status-info">
                  <span v-if="task.submissionStatus === 'APPROVED'" class="hint-text approved">🎉 你已完成此任务</span>
                  <span v-else-if="task.submissionStatus === 'PENDING'" class="hint-text pending">⏳ 等待审核中...（可继续做技能点内其他任务）</span>
                  <span v-else-if="task.submissionStatus === 'REJECTED'" class="hint-text rejected">❌ 未通过，请重新提交</span>
                  <span v-else-if="task.submissionStatus === 'AI_REVIEWED'" class="hint-text ai-reviewed">🤖 AI 已评审，等待最终确认</span>
                  <span v-else class="hint-text">点击下方按钮提交你的完成成果</span>
                </div>
                <div class="task-action-btns">
                  <el-button v-if="task.submissionStatus !== 'NONE'" size="default" round
                             @click.stop="openHistory(task)">
                    📜 查看提交记录
                  </el-button>
                  <el-button type="primary" size="default" round
                             :disabled="task.submissionStatus === 'APPROVED' || skill?.userStatus?.status === 'LOCKED'"
                             @click.stop="openSubmit(task)">
                    {{ task.submissionStatus === 'REJECTED' ? '重新提交' : task.submissionStatus === 'APPROVED' ? '已完成' : task.submissionStatus === 'PENDING' ? '修改后重新提交' : '提交任务' }}
                  </el-button>
                </div>
              </div>
            </div>
          </div>
          <div v-if="!skill?.tasks?.length" style="padding:40px 0">
            <el-empty description="暂无任务" :image-size="80" />
          </div>
        </div>
      </div>
    </div>

    <TaskSubmitDialog ref="submitDialog" :task="currentTask" @done="fetchDetail" />
    <SubmissionHistoryDialog ref="historyDialog" />
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { useRoute } from 'vue-router'
import { skillsApi } from '../api/skills'
import TutorialReader from '../components/TutorialReader.vue'
import TaskSubmitDialog from '../components/TaskSubmitDialog.vue'
import SubmissionHistoryDialog from '../components/SubmissionHistoryDialog.vue'

const route = useRoute()
const skill = ref(null)
const loading = ref(true)
const submitDialog = ref(null)
const currentTask = ref(null)
const historyDialog = ref(null)
const expandedTask = ref(null)
const openTutorials = reactive({})

function levelTag(l) { const m = {1:'info',2:'',3:'warning',4:'danger',5:'danger'}; return m[l]||'info' }
function statusText(s) { return { APPROVED:'已通过',REJECTED:'未通过',PENDING:'审核中',AI_REVIEWED:'AI已评',NONE:'待提交'}[s]||'待提交' }
function extractScore(fb) { const m = fb?.match(/AI评分:\s*(\d+)/); return m ? m[1] : '--' }
function extractFeedback(fb) { const idx = fb?.indexOf('\n'); return idx > -1 ? fb.substring(idx + 1).trim() : fb }
function toggleTutorial(idx) { openTutorials[idx] = !openTutorials[idx] }

function openSubmit(task) { currentTask.value = task; submitDialog.value.open() }
function openHistory(task) { historyDialog.value.open(task) }

async function fetchDetail() {
  loading.value = true
  try {
    const res = await skillsApi.getDetail(route.params.id)
    skill.value = res.data.data
    // Expand first tutorial by default
    if (skill.value?.tutorials?.length) openTutorials[0] = true
  } catch (e) { console.error(e) } finally { loading.value = false }
}

onMounted(fetchDetail)
</script>

<style scoped>
.page { max-width: 1200px; }
.page-title-row { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }

.count-badge {
  font-size: 12px; color: var(--color-text-secondary);
  background: #f1f5f9; padding: 2px 10px; border-radius: 10px;
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
.tutorial-header:hover { background: #f8fafc; }

.tutorial-num {
  width: 26px; height: 26px; border-radius: 50%;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: #fff; font-size: 12px; font-weight: 700;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.tutorial-title { flex: 1; font-size: 14px; font-weight: 500; }
.tutorial-arrow { color: #94a3b8; flex-shrink: 0; transition: transform .2s; }
.tutorial-arrow.open { transform: rotate(180deg); }

.tutorial-body {
  max-height: 0; overflow: hidden;
  transition: max-height .3s ease;
}
.tutorial-body.open { max-height: 2000px; padding: 0 20px 20px; }

/* ===== Tasks ===== */
.task-item { border-bottom: 1px solid var(--color-border); transition: background .15s; }
.task-item:last-child { border-bottom: none; }
.task-item:hover { background: #fafbfc; }
.task-item.expanded { background: #f8fafc; }

.task-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 14px 20px;
  cursor: pointer;
  user-select: none;
}
.task-header-left { display: flex; align-items: center; gap: 12px; flex: 1; min-width: 0; }

.task-num {
  width: 32px; height: 32px; border-radius: 8px;
  background: #f1f5f9;
  display: flex; align-items: center; justify-content: center;
  font-size: 13px; font-weight: 700; color: #64748b;
  flex-shrink: 0;
}
.done-check { color: #fff; font-size: 16px; width: 100%; height: 100%; background: #22c55e; border-radius: 8px; display: flex; align-items: center; justify-content: center; }

.task-title { font-size: 14px; font-weight: 500; color: var(--color-text); }
.task-tags { display: flex; align-items: center; gap: 8px; margin-top: 4px; flex-wrap: wrap; }

.status-tag { font-size: 11px; font-weight: 500; padding: 2px 8px; border-radius: 4px; }
.status-tag.approved { background: #dcfce7; color: #16a34a; }
.status-tag.rejected { background: #fee2e2; color: #dc2626; }
.status-tag.pending { background: #fef3c7; color: #d97706; }
.status-tag.ai_reviewed { background: #e0f2fe; color: #0284c7; }
.status-tag.none { background: #f1f5f9; color: #94a3b8; }

.xp-tag { font-size: 11px; font-weight: 600; color: #d97706; background: #fef3c7; padding: 2px 8px; border-radius: 4px; }
.criteria-count { font-size: 11px; color: #94a3b8; }

.task-arrow { color: #94a3b8; flex-shrink: 0; transition: transform .2s; }
.task-arrow.open { transform: rotate(180deg); }

.task-body {
  max-height: 0; overflow: hidden;
  transition: max-height .3s ease;
}
.task-body.open {
  max-height: 60vh;
  overflow-y: auto;
}

.task-description, .task-criteria {
  padding: 0 20px 16px;
}
.desc-label { font-size: 13px; font-weight: 600; color: var(--color-text); margin-bottom: 8px; }
.desc-content { font-size: 14px; color: #475569; line-height: 1.8; white-space: pre-wrap; }

.criteria-item {
  display: flex; align-items: baseline; gap: 6px;
  font-size: 13px; color: #475569; padding: 4px 0;
}
.criteria-num { color: #6366f1; font-weight: 600; min-width: 20px; }
.criteria-weight { font-size: 11px; color: #94a3b8; margin-left: auto; }

.task-actions {
  position: sticky;
  bottom: 0;
  z-index: 1;
  display: flex; align-items: center; justify-content: space-between;
  padding: 14px 20px;
  border-top: 1px solid var(--color-border);
  background: #fff;
}
.task-action-btns { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
.hint-text { font-size: 13px; color: #94a3b8; }
.hint-text.approved { color: #16a34a; font-weight: 500; }
.hint-text.pending { color: #d97706; font-weight: 500; }
.hint-text.rejected { color: #dc2626; font-weight: 500; }
.hint-text.ai-reviewed { color: #0284c7; font-weight: 500; }

.ai-feedback-box {
  margin: 0 20px 0;
  padding: 14px 16px;
  background: linear-gradient(135deg, #eff6ff, #f0f9ff);
  border: 1px solid #bae6fd;
  border-radius: 10px;
}
.ai-feedback-header {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 8px; font-size: 13px; font-weight: 600; color: #0369a1;
}
.ai-score {
  font-size: 18px; font-weight: 700; color: #0284c7;
}
.ai-feedback-text {
  font-size: 13px; color: #475569; line-height: 1.7; white-space: pre-wrap;
}
</style>
