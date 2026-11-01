<template>
  <el-dialog v-model="visible" title="提交任务" width="900px" @close="reset" :close-on-click-modal="false" top="3vh">
    <div v-if="task" class="submit-content">
      <!-- Task info -->
      <div class="task-header-info">
        <div class="task-title-row">
          <span class="task-dialog-title">{{ task.title }}</span>
          <span class="xp-badge">{{ task.xpReward }} XP</span>
        </div>
        <div class="status-row">
          当前状态：
          <span class="status-tag" :class="task.submissionStatus.toLowerCase()">{{ statusText(task.submissionStatus) }}</span>
        </div>
      </div>

      <div class="editor-wrapper">
        <!-- Toolbar -->
        <div class="editor-toolbar">
          <div class="toolbar-left">
            <button v-for="btn in toolbarBtns" :key="btn.label"
                    class="tb-btn" :title="btn.label"
                    @click="insertMarkdown(btn)">
              {{ btn.label }}
            </button>
          </div>
          <div class="toolbar-right">
            <button class="tb-btn tip-hint" title="选择附件上传（pdf/doc/xls/ppt/txt/md/zip/rar/7z，≤10MB）"
                    @click="fileInputRef?.click()">📎 附件</button>
            <button class="tb-btn tip-hint" :class="{ 'tb-uploading': uploading }" title="支持直接粘贴图片或文件 (Ctrl+V)">
              {{ uploading ? '⏳ 上传中...' : '🖼 粘贴图片/文件' }}
            </button>
          </div>
          <input ref="fileInputRef" type="file" multiple hidden
                 accept=".pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.md,.zip,.rar,.7z"
                 @change="onPickFiles" />
        </div>

        <!-- Tabs -->
        <div class="editor-tabs">
          <button :class="{ active: mode === 'edit' }" @click="mode = 'edit'">📝 编辑</button>
          <button :class="{ active: mode === 'preview' }" @click="mode = 'preview'">👁 预览</button>
        </div>

        <!-- Edit area -->
        <div v-show="mode === 'edit'" class="editor-pane">
          <textarea ref="textareaRef"
                    v-model="submissionData"
                    class="md-editor"
                    placeholder="请描述你的完成过程和结果，支持 Markdown 语法。

可直接粘贴图片或文件 (Ctrl+V)，也可点工具栏「📎 附件」上传，文件自动存入七牛云。

示例：
# 实现步骤
1. 配置了 xxx 环境
2. 编写了 xxx 代码

## 运行截图
![截图](粘贴图片到这里)

## 核心代码
` ` `java
public class Demo { ... }
` ` `
"
                    @paste="onPaste"
          ></textarea>
        </div>

        <!-- Preview area -->
        <div v-show="mode === 'preview'" class="preview-pane">
          <div v-if="submissionData.trim()" v-html="renderedMarkdown" class="preview-body"></div>
          <el-empty v-else description="暂无内容" :image-size="60" />
        </div>
      </div>

      <div class="tips-box">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" style="flex-shrink:0"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
        <span>提交后可由导师人工审核或 AI 自动验收（AI 验收每任务限 3 次）。提交内容越详细，通过率越高。</span>
      </div>
    </div>
    <template #footer>
      <el-button @click="visible = false" round>取消</el-button>
      <el-button type="primary" @click="submit" :loading="submitting" round>
        提交人工审核
      </el-button>
      <el-tooltip :content="task?.aiReviewEnabled ? 'AI 将自动评估你的提交' : '此任务未开启 AI 审核'" placement="top">
        <el-button type="success" @click="submitWithAI" :loading="aiSubmitting" round
                   :disabled="!task?.aiReviewEnabled">
          🤖 提交并 AI 验收
        </el-button>
      </el-tooltip>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, nextTick, watch } from 'vue'
import { tasksApi } from '../api/tasks'
import { uploadApi } from '../api/upload'
import { filesApi } from '../api/files'
import { ElMessage } from 'element-plus'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const props = defineProps({ task: Object })
const emit = defineEmits(['done'])

const visible = ref(false)
const submitting = ref(false)
const aiSubmitting = ref(false)
const submissionData = ref('')
const mode = ref('edit')
const textareaRef = ref(null)
const uploading = ref(false)  // 图片上传中标记

const fileInputRef = ref(null)

const MAX_FILE_SIZE = 10 * 1024 * 1024
const ALLOWED_FILE_EXTS = ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'md', 'zip', 'rar', '7z']
const IMAGE_EXTS = ['png', 'jpg', 'jpeg', 'gif', 'webp']

function fileExt(name) {
  const i = name.lastIndexOf('.')
  return i < 0 ? '' : name.slice(i + 1).toLowerCase()
}

function isImageUrl(url) {
  const clean = url.toLowerCase().split('?')[0]
  return IMAGE_EXTS.some(ext => clean.endsWith('.' + ext))
}

function validateFile(file) {
  const ext = fileExt(file.name)
  if (!IMAGE_EXTS.includes(ext) && !ALLOWED_FILE_EXTS.includes(ext)) {
    ElMessage.error(`不支持的文件类型：${file.name}`)
    return false
  }
  if (file.size > MAX_FILE_SIZE) {
    ElMessage.error(`文件过大：${file.name}（上限 10MB）`)
    return false
  }
  return true
}

const toolbarBtns = [
  { label: 'H1', syntax: '# ', block: true },
  { label: 'H2', syntax: '## ', block: true },
  { label: 'H3', syntax: '### ', block: true },
  { label: 'B', syntax: '****', cursor: 2 },
  { label: 'I', syntax: '**', cursor: 1 },
  { label: '`', syntax: '``', cursor: 1 },
  { label: '---', syntax: '\n---\n', block: true },
  { label: '链接', syntax: '[]()', cursor: 1 },
  { label: '列表', syntax: '- ', block: true },
  { label: '代码块', syntax: '\n```\n\n```\n', cursor: 5 },
]

const previewMarkdown = ref('')
let signSeq = 0  // 竞态防护：只采纳最新一次签名的结果

// 编辑器内预览：私有空间图片 URL 需先向后端换签名链接（未签名会被七牛拒 403）
async function signPreview() {
  const seq = ++signSeq
  const md = submissionData.value || ''
  // 同时收集图片与普通链接；非图片链接附带 attname（链接文字作下载文件名）
  const found = []
  for (const m of md.matchAll(/(!?)\[([^\]]*)\]\((https?:\/\/[^)\s"']+)\)/g)) {
    const isImage = m[1] === '!'
    found.push(isImage || isImageUrl(m[3])
      ? m[3]
      : `${m[3]}?attname=${encodeURIComponent(m[2] || 'attachment')}`)
  }
  const unique = [...new Set(found)]
  if (!unique.length) {
    if (seq === signSeq) previewMarkdown.value = md
    return
  }
  try {
    const res = await filesApi.signUrls(unique)
    if (seq !== signSeq) return  // 已有更新的输入，丢弃本次结果
    const map = res.data.data || {}
    let out = md
    for (const [u, signed] of Object.entries(map)) {
      if (signed) out = out.replaceAll(u.split('?attname=')[0], signed)
    }
    previewMarkdown.value = out
  } catch {
    if (seq === signSeq) previewMarkdown.value = md  // 签名失败时原样渲染（非七牛图不受影响）
  }
}
watch(submissionData, signPreview, { immediate: true })

// Build submission payload: markdown 里已是七牛图片 URL，直接提交
function buildPayload() {
  return { submissionData: submissionData.value }
}

const renderedMarkdown = computed(() => {
  if (!previewMarkdown.value) return ''
  return DOMPurify.sanitize(marked.parse(previewMarkdown.value))
})

function statusText(s) {
  return { APPROVED: '已通过', REJECTED: '未通过', PENDING: '审核中', AI_REVIEWED: 'AI已评', NONE: '待提交' }[s] || '待提交'
}

function insertMarkdown(btn) {
  const el = textareaRef.value
  if (!el) return
  const start = el.selectionStart
  const end = el.selectionEnd
  const text = submissionData.value
  const before = text.substring(0, start)
  const sel = text.substring(start, end) || ''
  const after = text.substring(end)

  if (sel && !btn.block) {
    // Wrap selection
    submissionData.value = before + btn.syntax[0] + sel + btn.syntax[btn.syntax.length - 1] + after
  } else {
    submissionData.value = before + btn.syntax + after
  }

  nextTick(() => {
    el.focus()
    const pos = start + (btn.cursor != null ? btn.cursor : btn.syntax.length)
    el.setSelectionRange(pos, pos)
  })
}

// 从剪贴板提取可上传的文件（图片或白名单附件）；纯文本粘贴不受影响
function extractClipboardFiles(items) {
  return Array.from(items)
    .map(item => item.getAsFile())
    .filter(f => f && (f.type.startsWith('image/') || ALLOWED_FILE_EXTS.includes(fileExt(f.name))))
}

async function onPaste(e) {
  const items = e.clipboardData?.items
  if (!items) return
  const files = extractClipboardFiles(items)
  if (!files.length) return
  e.preventDefault()
  uploading.value = true
  try {
    for (const file of files) {
      if (validateFile(file)) await uploadAndInsert(file)
    }
  } finally {
    uploading.value = false
  }
}

async function onPickFiles(e) {
  const files = Array.from(e.target.files || [])
  e.target.value = ''  // 允许重复选择同一文件
  if (!files.length) return
  uploading.value = true
  try {
    for (const file of files) {
      if (validateFile(file)) await uploadAndInsert(file)
    }
  } finally {
    uploading.value = false
  }
}

// 拿七牛上传凭证 → 浏览器直传 → 返回访问 URL（私有空间，读取时由后端签名）
async function uploadOnce(file) {
  const res = await uploadApi.token()
  const { token, uploadHost, domain, keyPrefix } = res.data.data
  // key 必须以 keyPrefix 开头（七牛 isPrefixalScope 前缀限制）；随机文件名 + 原扩展名，防覆盖
  const ext = fileExt(file.name).replace(/[^a-z0-9]/g, '') || 'bin'
  const key = `${keyPrefix}${Date.now()}-${Math.random().toString(36).slice(2, 10)}.${ext}`
  const form = new FormData()
  form.append('token', token)
  form.append('key', key)
  form.append('file', file)
  const upRes = await fetch(uploadHost, { method: 'POST', body: form })
  if (!upRes.ok) {
    throw new Error(`七牛上传失败(HTTP ${upRes.status})`)
  }
  const body = await upRes.json()
  if (!body.key) {
    throw new Error(body.error || '七牛返回异常')
  }
  return `${domain}/${body.key}`
}

// 凭证有效期 600s 可能过期，失败时重取凭证重试一次
async function uploadFile(file) {
  try {
    return await uploadOnce(file)
  } catch {
    return await uploadOnce(file)
  }
}

// 上传单个文件并在光标处插入 Markdown（图片用 ![]()，其他用 [文件名](url)）
async function uploadAndInsert(file) {
  const el = textareaRef.value
  const start = el ? el.selectionStart : submissionData.value.length
  const isImage = IMAGE_EXTS.includes(fileExt(file.name))
  try {
    const url = await uploadFile(file)
    const md = isImage ? `![](${url})` : `[${file.name}](${url})`
    submissionData.value =
      submissionData.value.substring(0, start) + md + submissionData.value.substring(start)
    nextTick(() => {
      if (el) {
        const pos = start + md.length
        el.focus()
        el.setSelectionRange(pos, pos)
      }
    })
  } catch (err) {
    ElMessage.error((isImage ? '图片' : '附件') + '上传失败：' + (err.message || '未知错误'))
  }
}

async function open() {
  visible.value = true
  submissionData.value = ''
  mode.value = 'edit'
  // 回显该任务最近一次提交内容（修改后重新提交时保留原内容）
  if (props.task?.id) {
    try {
      const res = await tasksApi.taskSubmissions(props.task.id)
      const list = res.data.data || []
      if (list.length && list[0].submissionData) {
        submissionData.value = list[0].submissionData
      }
    } catch (e) { /* 忽略：加载失败时保持空编辑器 */ }
  }
}

function reset() {
  submissionData.value = ''
  mode.value = 'edit'
}

async function submit() {
  if (!props.task) return
  if (uploading.value) {
    ElMessage.warning('文件正在上传，请稍候再提交')
    return
  }
  submitting.value = true
  try {
    await tasksApi.submit(props.task.id, buildPayload())
    ElMessage.success('提交成功，等待导师审核')
    visible.value = false
    emit('done')
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '提交失败')
  } finally {
    submitting.value = false
  }
}

async function submitWithAI() {
  if (!props.task) return
  if (uploading.value) {
    ElMessage.warning('文件正在上传，请稍候再提交')
    return
  }
  aiSubmitting.value = true
  try {
    const res = await tasksApi.submit(props.task.id, buildPayload())
    await tasksApi.aiReview(res.data.data.id)
    ElMessage.success('AI 验收完成')
    visible.value = false
    emit('done')
  } catch (e) {
    ElMessage.error(e.response?.data?.message || 'AI 验收失败')
  } finally {
    aiSubmitting.value = false
  }
}

defineExpose({ open })
</script>

<style scoped>
.submit-content { display: flex; flex-direction: column; gap: 12px; }

/* --- Task header --- */
.task-header-info { }
.task-title-row { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; margin-bottom: 4px; }
.task-dialog-title { font-size: 16px; font-weight: 600; color: var(--color-text); line-height: 1.4; }
.xp-badge { font-size: 13px; font-weight: 700; color: #d97706; background: #fef3c7; padding: 3px 10px; border-radius: 6px; white-space: nowrap; }
.status-row { font-size: 13px; color: #94a3b8; display: flex; align-items: center; gap: 6px; }
.status-tag { font-size: 11px; font-weight: 500; padding: 2px 8px; border-radius: 4px; }
.status-tag.approved { background: #dcfce7; color: #16a34a; }
.status-tag.rejected { background: #fee2e2; color: #dc2626; }
.status-tag.pending { background: #fef3c7; color: #d97706; }
.status-tag.ai_reviewed { background: #e0f2fe; color: #0284c7; }
.status-tag.none { background: #f1f5f9; color: #94a3b8; }

/* --- Editor wrapper --- */
.editor-wrapper { border: 1px solid var(--color-border); border-radius: 10px; overflow: hidden; }

/* --- Toolbar --- */
.editor-toolbar {
  display: flex; justify-content: space-between; align-items: center;
  padding: 8px 12px; background: #fafbfc; border-bottom: 1px solid var(--color-border);
  flex-wrap: wrap; gap: 6px;
}
.toolbar-left { display: flex; gap: 4px; flex-wrap: wrap; }
.toolbar-right { display: flex; gap: 4px; }
.tb-btn {
  border: 1px solid #e2e8f0; background: #fff; border-radius: 5px;
  padding: 4px 10px; font-size: 12px; font-weight: 500; color: #475569;
  cursor: pointer; transition: all .15s;
}
.tb-btn:hover { background: #f1f5f9; border-color: #cbd5e1; }
.tip-hint { color: #6366f1; border-color: #ddd6fe; background: #f5f3ff; font-size: 11px; }
.tb-uploading { color: #16a34a; border-color: #bbf7d0; background: #f0fdf4; }

/* --- Tabs --- */
.editor-tabs {
  display: flex; gap: 0; background: #fafbfc; border-bottom: 1px solid var(--color-border);
}
.editor-tabs button {
  flex: 1; border: none; background: none; padding: 10px 16px;
  font-size: 13px; font-weight: 500; color: #94a3b8; cursor: pointer;
  transition: all .15s; border-bottom: 2px solid transparent;
}
.editor-tabs button.active { color: #6366f1; border-bottom-color: #6366f1; background: #fff; }
.editor-tabs button:hover:not(.active) { color: #475569; }

/* --- Edit pane --- */
.editor-pane { }
.md-editor {
  width: 100%; min-height: 380px; border: none; outline: none; resize: vertical;
  padding: 16px; font-size: 14px; font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', 'Menlo', monospace;
  line-height: 1.7; color: #1e293b; background: #fff;
  box-sizing: border-box;
}
.md-editor::placeholder { color: #cbd5e1; font-family: inherit; }

/* --- Preview pane --- */
.preview-pane { min-height: 380px; background: #fff; }
.preview-body { padding: 16px 20px; line-height: 1.8; font-size: 14px; color: #334155; }
.preview-body :deep(h1) { font-size: 22px; margin: 16px 0 8px; border-bottom: 1px solid #e2e8f0; padding-bottom: 4px; }
.preview-body :deep(h2) { font-size: 18px; margin: 14px 0 6px; }
.preview-body :deep(h3) { font-size: 15px; margin: 12px 0 4px; }
.preview-body :deep(p) { margin: 8px 0; }
.preview-body :deep(img) { max-width: 100%; border-radius: 6px; margin: 8px 0; border: 1px solid #e2e8f0; }
.preview-body :deep(pre) { background: #f6f8fa; padding: 12px; border-radius: 6px; overflow-x: auto; font-size: 13px; }
.preview-body :deep(code) { font-size: 13px; background: #f1f5f9; padding: 1px 4px; border-radius: 3px; }
.preview-body :deep(pre code) { background: none; padding: 0; }
.preview-body :deep(blockquote) { border-left: 3px solid #6366f1; padding-left: 12px; color: #64748b; margin: 8px 0; }
.preview-body :deep(ul), .preview-body :deep(ol) { padding-left: 20px; }
.preview-body :deep(li) { margin: 4px 0; }
.preview-body :deep(hr) { border: none; border-top: 1px solid #e2e8f0; margin: 16px 0; }
.preview-body :deep(a) { color: #6366f1; }

/* --- Tips --- */
.tips-box {
  display: flex; align-items: flex-start; gap: 8px;
  font-size: 12px; color: #94a3b8; line-height: 1.6;
  padding: 10px 12px; background: #f8fafc; border-radius: 8px;
}
</style>
