<template>
  <el-dialog v-model="visible" :title="`提交记录 · ${task?.title || ''}`" width="min(860px, calc(100% - 24px))" top="3vh">
    <div v-loading="loading" class="history-body">
      <div v-if="!loading && !submissions.length" class="empty-tip">
        <el-empty description="还没有提交记录" :image-size="70" />
      </div>

      <div v-for="(sub, idx) in submissions" :key="sub.id" class="history-item">
        <div class="history-head">
          <div class="history-left">
            <span class="attempt-badge">第 {{ submissions.length - idx }} 次提交</span>
            <span class="status-tag" :class="sub.status?.toLowerCase()">{{ submissionStatusText(sub.status) }}</span>
          </div>
          <span class="history-time">{{ formatTime(sub.submittedAt) }}</span>
        </div>

        <div class="history-section">
          <div class="history-section-title">提交内容</div>
          <div class="md-preview" v-html="renderMd(sub.submissionData)"></div>
        </div>

        <div v-if="sub.aiFeedback" class="history-section ai-box">
          <div class="history-section-title">AI 评语</div>
          <div class="plain-text">{{ sub.aiFeedback }}</div>
        </div>

        <div v-if="sub.reviewComment" class="history-section">
          <div class="history-section-title">审核意见</div>
          <div class="plain-text">{{ sub.reviewComment }}</div>
        </div>

        <div v-if="sub.reviewedAt" class="reviewed-line">
          审核时间：{{ formatTime(sub.reviewedAt) }}
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { submissionStatusText } from '../submissionStatus'
import { ref } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { tasksApi } from '../api/tasks'
import { ElMessage } from 'element-plus'

const visible = ref(false)
const loading = ref(false)
const task = ref(null)
const submissions = ref([])


function formatTime(ts) {
  if (!ts) return '-'
  const d = new Date(ts)
  return d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0') +
    ' ' + String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0') + ':' + String(d.getSeconds()).padStart(2, '0')
}

function renderMd(text) {
  if (!text) return ''
  return DOMPurify.sanitize(marked.parse(text))
}

async function open(t) {
  task.value = t
  visible.value = true
  loading.value = true
  submissions.value = []
  try {
    const res = await tasksApi.taskSubmissions(t.id)
    submissions.value = res.data.data || []
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '加载提交记录失败')
  } finally {
    loading.value = false
  }
}

defineExpose({ open })
</script>

<style scoped>
.history-body { display: flex; flex-direction: column; gap: 16px; min-height: 120px; }
.empty-tip { padding: 20px 0; }

.history-item {
  border: 1px solid var(--color-border);
  border-radius: 10px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.history-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.history-left { display: flex; align-items: center; gap: 10px; }
.attempt-badge {
  font-size: 12px; font-weight: 600; color: #216354;
  background: #edf5ee; padding: 3px 10px; border-radius: 6px;
}
.history-time { font-size: 12px; color: #667568; }

.status-tag { font-size: 11px; font-weight: 500; padding: 2px 8px; border-radius: 4px; }
.status-tag.approved { background: #e0efe6; color: #30684b; }
.status-tag.rejected { background: #fee2e2; color: #dc2626; }
.status-tag.pending { background: #faf0d7; color: #895b19; }
.status-tag.ai_reviewed { background: #e0f2fe; color: #0284c7; }

.history-section { padding: 12px 14px; background: #f5f6f0; border-radius: 8px; }
.history-section-title { font-size: 12px; font-weight: 600; color: #58685d; margin-bottom: 8px; }
.md-preview { font-size: 14px; line-height: 1.8; color: #334155; }
.md-preview :deep(img) { max-width: 100%; border-radius: 6px; border: 1px solid #dce2d7; }
.md-preview :deep(pre) { background: #263b31; color: #dce2d7; padding: 12px; border-radius: 6px; overflow-x: auto; font-size: 13px; }
.md-preview :deep(code) { font-size: 13px; background: #edf0e9; padding: 1px 4px; border-radius: 3px; }
.md-preview :deep(pre code) { background: none; padding: 0; color: inherit; }
.md-preview :deep(blockquote) { border-left: 3px solid #216354; padding-left: 12px; color: #58685d; margin: 8px 0; }
.md-preview :deep(ul), .md-preview :deep(ol) { padding-left: 20px; }

.ai-box { background: #f0f9ff; border: 1px solid #bae6fd; }
.plain-text { font-size: 13px; color: #3e5147; white-space: pre-wrap; line-height: 1.7; }
.reviewed-line { font-size: 12px; color: #667568; }
</style>
