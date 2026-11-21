<template>
  <el-dialog v-model="visible" title="提交交付记录" class="submission-workspace"
             width="min(1220px, calc(100% - 24px))" top="clamp(10px, 2dvh, 22px)"
             :show-close="!busy" :close-on-press-escape="!busy" :close-on-click-modal="false"
             @close="reset" @opened="focusEditor">
    <div v-if="task" class="submit-content">
      <!-- Task info -->
      <div class="task-header-info">
        <div class="task-title-row">
          <span class="task-dialog-title">{{ task.title }}</span>
          <span class="xp-badge">{{ task.xpReward }} XP</span>
        </div>
        <div class="status-row">
          <span class="status-tag" :class="task.submissionStatus.toLowerCase()">{{ submissionStatusText(task.submissionStatus) }}</span>
        </div>
      </div>

      <div class="editor-wrapper">
        <div class="editor-toolbar" role="group" aria-label="作业格式和附件">
          <div class="toolbar-format">
            <button v-for="btn in toolbarBtns" :key="btn.label" type="button"
                    class="tb-btn" :title="btn.label" :disabled="busy"
                    @click="insertMarkdown(btn)">
              {{ btn.label }}
            </button>
          </div>
          <button type="button" class="tb-btn upload-button" :disabled="busy || uploading" @click="fileInputRef?.click()">
            {{ uploading ? '上传中…' : '添加图片或附件' }}
          </button>
          <input ref="fileInputRef" type="file" multiple hidden :accept="acceptedFiles"
                 @change="onPickFiles" />
        </div>

        <div class="editor-panels">
          <section class="source-panel" aria-label="编辑记录">
            <div class="panel-heading"><strong>编辑记录</strong><span>Markdown</span></div>
            <textarea ref="textareaRef"
                      v-model="submissionData"
                      class="md-editor"
                      aria-label="作业内容"
                      :readonly="busy"
                      placeholder="写下这一关做了什么、怎么验证，再附上任务要求的截图或链接。"
                      @paste="onPaste"
            ></textarea>
          </section>
          <section class="preview-panel" aria-label="实时预览">
            <div class="panel-heading">
              <strong>实时预览</strong>
              <button v-if="signatureFailed && signatureKey" type="button" class="retry-signatures"
                      @click="requestPreviewSignatures">重试加载附件</button>
            </div>
            <div class="preview-pane" tabindex="0" role="region" aria-label="作业预览内容">
              <div v-if="submissionData.trim()" v-html="renderedMarkdown" class="preview-body"></div>
              <p v-else class="preview-empty">输入内容后，预览会在这里更新。</p>
            </div>
          </section>
        </div>
      </div>

      <div class="tips-box">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" style="flex-shrink:0"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
        <span>可粘贴图片或上传附件，单个文件上限 10 MB。</span>
      </div>
    </div>
    <template #footer>
      <div class="submit-actions">
        <el-button @click="visible = false" :disabled="busy" round>取消</el-button>
        <el-button v-if="task?.aiReviewEnabled" @click="submit(true)" :loading="aiSubmitting" :disabled="!canSubmit" round>
          提交并 AI 验收
        </el-button>
        <el-button type="primary" @click="submit()" :loading="submitting" :disabled="!canSubmit" round>
          提交人工审核
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { submissionStatusText } from '../submissionStatus'
import { ref, shallowRef, computed, nextTick, watch, onBeforeUnmount } from 'vue'
import { tasksApi } from '../api/tasks'
import { uploadApi } from '../api/upload'
import { filesApi } from '../api/files'
import { useAuthStore } from '../store/auth'
import { ElMessage } from 'element-plus'
import 'element-plus/es/components/message/style/css'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const props = defineProps({ task: Object })
const emit = defineEmits(['done'])
const auth = useAuthStore()

const visible = ref(false)
const submitting = ref(false)
const aiSubmitting = ref(false)
const submissionData = ref('')
const busy = computed(() => submitting.value || aiSubmitting.value)
const canSubmit = computed(() => props.task?.submissionStatus !== 'APPROVED' && submissionData.value.trim().length > 0 && !uploading.value && !busy.value)
let editorSession = 0
let contentRevision = 0
const textareaRef = ref(null)
const uploading = ref(false)  // 图片上传中标记

const fileInputRef = ref(null)

const MAX_FILE_SIZE = 10 * 1024 * 1024
const ALLOWED_FILE_EXTS = ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'md', 'zip', 'rar', '7z']
const IMAGE_EXTS = ['png', 'jpg', 'jpeg', 'gif', 'webp']
const acceptedFiles = [...IMAGE_EXTS, ...ALLOWED_FILE_EXTS].map(ext => '.' + ext).join(',')

function fileExt(name) {
  const i = name.lastIndexOf('.')
  return i < 0 ? '' : name.slice(i + 1).toLowerCase()
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
  { label: '标题', prefix: '## ' },
  { label: '粗体', prefix: '**', suffix: '**' },
  { label: '行内代码', prefix: '`', suffix: '`' },
  { label: '链接', prefix: '[', suffix: ']()', cursorAfterSelection: 2 },
  { label: '列表', prefix: '- ' },
  { label: '代码块', prefix: '\n```\n', suffix: '\n```\n' },
]

const attachmentElements = 'a[href], img[src]'
const signedUrls = shallowRef(new Map())
const failedUrls = shallowRef(new Set())
const signingUrls = new Set()

function attachmentKey(element) {
  const userId = auth.user?.id
  if (!userId) return null
  const isImage = element.tagName === 'IMG'
  const raw = element.getAttribute(isImage ? 'src' : 'href')
  if (!/^https?:\/\//i.test(raw || '')) return null
  let url
  try { url = new URL(raw) } catch { return null }
  if (!url.pathname.startsWith('/uploads/' + userId + '/') || url.searchParams.has('token')) return null
  if (isImage || IMAGE_EXTS.includes(fileExt(url.pathname))) return raw
  url.searchParams.set('attname', element.textContent.trim() || 'attachment')
  return url.toString()
}

const previewFragment = computed(() => DOMPurify.sanitize(marked.parse(submissionData.value), { RETURN_DOM_FRAGMENT: true }))
const signatureKey = computed(() => Array.from(new Set(
  Array.from(previewFragment.value.querySelectorAll(attachmentElements), attachmentKey).filter(Boolean)
)).sort().join('\n'))
const signatureFailed = computed(() => signatureKey.value.split('\n').some(
  url => failedUrls.value.has(url) && !signedUrls.value.has(url)
))

const renderedMarkdown = computed(() => {
  const preview = document.createElement('div')
  preview.append(previewFragment.value.cloneNode(true))
  for (const element of preview.querySelectorAll(attachmentElements)) {
    const isImage = element.tagName === 'IMG'
    const attribute = isImage ? 'src' : 'href'
    const key = attachmentKey(element)
    if (isImage) element.loading = 'lazy'
    if (key && !signedUrls.value.has(key) && !failedUrls.value.has(key) && isImage) {
      const placeholder = document.createElement('span')
      placeholder.className = 'preview-image-pending'
      placeholder.textContent = element.alt ? element.alt + ' · 加载中…' : '图片加载中…'
      element.replaceWith(placeholder)
      continue
    }
    const signed = signedUrls.value.get(key)
    if (typeof signed === 'string' && /^https?:\/\//i.test(signed)) element.setAttribute(attribute, signed)
    if (!isImage && /^https?:\/\//i.test(element.getAttribute('href') || '')) {
      element.target = '_blank'
      element.rel = 'noopener noreferrer'
    }
  }
  return preview.innerHTML
})

async function requestPreviewSignatures() {
  if (!visible.value || !signatureKey.value) return
  const session = editorSession
  const urls = signatureKey.value.split('\n').filter(url => !signedUrls.value.has(url) && !signingUrls.has(url))
  for (const url of urls) signingUrls.add(url)
  // sign-urls accepts up to 200 links per request.
  for (let cursor = 0; cursor < urls.length && editorSession === session; cursor += 200) {
    const batch = urls.slice(cursor, cursor + 200)
    try {
      const { data } = await filesApi.signUrls(batch)
      if (session !== editorSession) return
      const next = new Map(signedUrls.value)
      for (const url of batch) next.set(url, data.data?.[url] || url)
      signedUrls.value = next
    } catch {
      if (session !== editorSession) return
      failedUrls.value = new Set([...failedUrls.value, ...batch])
    } finally {
      if (session === editorSession) for (const url of batch) signingUrls.delete(url)
    }
  }
}

watch([visible, signatureKey], ([open, key], _, cleanup) => {
  if (!open || !key) return
  const timeout = setTimeout(requestPreviewSignatures, 320)
  cleanup(() => clearTimeout(timeout))
})
watch(submissionData, () => { contentRevision++ }, { flush: 'sync' })

function focusEditor() {
  textareaRef.value?.focus({ preventScroll: true })
}

function insertMarkdown(btn) {
  const el = textareaRef.value
  if (!el) return
  const start = el.selectionStart
  const end = el.selectionEnd
  const text = submissionData.value
  const before = text.substring(0, start)
  const selected = text.substring(start, end)
  const after = text.substring(end)
  submissionData.value = before + btn.prefix + selected + (btn.suffix || '') + after

  nextTick(() => {
    el.focus()
    const pos = start + btn.prefix.length + selected.length + (selected ? btn.cursorAfterSelection || 0 : 0)
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
  await insertFiles(files)
}

async function onPickFiles(e) {
  const files = Array.from(e.target.files || [])
  e.target.value = ''
  await insertFiles(files)
}

async function insertFiles(files) {
  if (!files.length || uploading.value || busy.value) return
  const session = editorSession
  uploading.value = true
  try {
    for (const file of files) {
      if (session !== editorSession) break
      if (validateFile(file)) await uploadAndInsert(file, session)
    }
  } finally {
    if (session === editorSession) uploading.value = false
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
async function uploadAndInsert(file, session) {
  const el = textareaRef.value
  const revision = contentRevision
  const cursor = el?.selectionStart ?? submissionData.value.length
  const isImage = IMAGE_EXTS.includes(fileExt(file.name))
  try {
    const url = await uploadFile(file)
    if (session !== editorSession) return
    const start = revision === contentRevision ? cursor : (el?.selectionStart ?? submissionData.value.length)
    const editorHadFocus = document.activeElement === el
    const label = file.name.replace(/[\\[\]]/g, '\\$&')
    const link = `${isImage ? '!' : ''}[${label}](${url})`
    const before = submissionData.value.slice(0, start)
    const after = submissionData.value.slice(start)
    // Keep attachments on their own Markdown block so adjacent headings and lists survive.
    const leftBreak = before && !before.endsWith('\n\n') ? (before.endsWith('\n') ? '\n' : '\n\n') : ''
    const rightBreak = after.startsWith('\n\n') ? '' : (after.startsWith('\n') ? '\n' : '\n\n')
    const inserted = leftBreak + link + rightBreak
    submissionData.value = before + inserted + after
    nextTick(() => {
      if (session === editorSession && el) {
        const pos = start + inserted.length
        el.setSelectionRange(pos, pos)
        if (editorHadFocus) focusEditor()
      }
    })
  } catch (err) {
    if (session !== editorSession) return
    ElMessage.error((isImage ? '图片' : '附件') + '上传失败：' + (err.message || '未知错误'))
  }
}

async function open() {
  const session = ++editorSession
  await nextTick()
  if (session !== editorSession) return
  submissionData.value = ''
  const revision = contentRevision
  visible.value = true
  // 回显该任务最近一次提交内容（修改后重新提交时保留原内容）
  if (props.task?.id && props.task.submissionStatus !== 'NONE') {
    try {
      const res = await tasksApi.taskSubmissions(props.task.id, 1)
      const list = res.data.data || []
      if (session === editorSession && revision === contentRevision && list[0]?.submissionData) {
        submissionData.value = list[0].submissionData
      }
    } catch (e) { /* 忽略：加载失败时保持空编辑器 */ }
  }
}

function reset() {
  editorSession++
  uploading.value = false
  submissionData.value = ''
  signedUrls.value = new Map()
  failedUrls.value = new Set()
  signingUrls.clear()
}

async function submit(withAI = false) {
  if (!props.task || !canSubmit.value || (withAI && !props.task.aiReviewEnabled)) return
  const pending = withAI ? aiSubmitting : submitting
  pending.value = true
  try {
    const res = await tasksApi.submit(props.task.id, { submissionData: submissionData.value })
    if (withAI) {
      try {
        await tasksApi.aiReview(res.data.data.id)
        ElMessage.success('已生成 AI 验收结果')
      } catch (error) {
        const reason = error.response?.data?.message || 'AI 验收暂时不可用'
        ElMessage.warning('记录已提交，仍可人工审核。' + reason)
      }
    } else {
      ElMessage.success('已提交，等待人工审核')
    }
    visible.value = false
    emit('done')
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '提交失败')
  } finally {
    pending.value = false
  }
}

onBeforeUnmount(reset)
defineExpose({ open })
</script>

<style scoped>
:global(.el-dialog.submission-workspace) {
  --el-dialog-padding-primary: clamp(14px, 2vw, 24px);
  display: flex; flex-direction: column; max-height: calc(100dvh - 2 * clamp(10px, 2dvh, 22px));
}
:global(.submission-workspace > .el-dialog__body) { min-height: 0; overflow: auto; }
:global(.submission-workspace > .el-dialog__header),
:global(.submission-workspace > .el-dialog__footer) { flex: none; }
.submit-content { display: flex; flex-direction: column; gap: 12px; }

/* --- Task header --- */

.task-title-row { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; margin-bottom: 4px; }
.task-dialog-title { font-size: 16px; font-weight: 600; color: var(--color-text); line-height: 1.4; overflow-wrap: anywhere; }
.xp-badge { font-size: 13px; font-weight: 700; color: #895b19; background: #faf0d7; padding: 3px 10px; border-radius: 6px; white-space: nowrap; }
.status-row { font-size: 13px; color: #667568; display: flex; align-items: center; gap: 6px; }
.status-tag { font-size: 11px; font-weight: 500; padding: 2px 8px; border-radius: 4px; }
.status-tag.approved { background: #e0efe6; color: #30684b; }
.status-tag.rejected { background: #fee2e2; color: #dc2626; }
.status-tag.pending { background: #faf0d7; color: #895b19; }
.status-tag.ai_reviewed { background: #e0f2fe; color: #0284c7; }
.status-tag.none { background: #edf0e9; color: #667568; }

/* --- Editor wrapper --- */
.editor-wrapper { min-width: 0; border: 1px solid var(--color-border); border-radius: 12px; overflow: hidden; }

/* --- Toolbar --- */
.editor-toolbar {
  display: flex; justify-content: space-between; align-items: center;
  padding: 8px 12px; background: #f3f6eb; border-bottom: 1px solid var(--color-border);
  flex-wrap: wrap; gap: 8px;
}
.toolbar-format { display: flex; gap: 5px; flex-wrap: wrap; }
.tb-btn {
  min-height: 34px; border: 1px solid #dce2d7; background: #fff; border-radius: 6px;
  padding: 4px 10px; font-size: 12px; font-weight: 500; color: #3e5147;
}
.tb-btn:hover { background: #edf0e9; border-color: #bfcbbf; }
.upload-button { color: var(--color-primary); border-color: #b5cbb7; font-weight: 600; }

.editor-panels {
  --panel-height: clamp(235px, 51dvh, 600px);
  display: grid; grid-template-columns: repeat(2, minmax(0, 1fr));
}
.source-panel, .preview-panel { min-width: 0; }
.preview-panel { border-inline-start: 1px solid var(--color-border); }
.panel-heading {
  display: flex; justify-content: space-between; align-items: center; gap: 8px;
  min-height: 42px; padding: 6px 15px;
  color: var(--color-text); background: #fafbf6; border-bottom: 1px solid var(--color-border);
}
.panel-heading strong { font-size: 13px; }
.panel-heading span { color: var(--color-text-secondary); font-size: 12px; }
.retry-signatures {
  color: var(--color-primary); padding: 4px 0; font-size: 12px; text-decoration: underline;
  text-underline-offset: 2px; background: none; border: 0;
}
.md-editor, .preview-pane {
  display: block; height: var(--panel-height); overflow: auto; overscroll-behavior: contain; outline-offset: -3px;
}
.md-editor {
  width: 100%; border: none; resize: none; padding: 16px;
  font-size: 15px; line-height: 1.75; color: #263b31; background: #fff;
}
.md-editor::placeholder { color: #6e786f; font-family: inherit; }

.preview-pane { background: #fffef9; overflow-wrap: anywhere; }
.preview-empty { padding: 16px 20px; color: var(--color-text-secondary); font-size: 14px; }
.preview-body :deep(.preview-image-pending) { color: var(--color-text-secondary); font-size: 13px; }
.preview-body { padding: 16px 20px; line-height: 1.8; font-size: 14px; color: #334155; }
.preview-body :deep(h1) { font-size: 22px; margin: 16px 0 8px; border-bottom: 1px solid #dce2d7; padding-bottom: 4px; }
.preview-body :deep(h2) { font-size: 18px; margin: 14px 0 6px; }
.preview-body :deep(h3) { font-size: 15px; margin: 12px 0 4px; }
.preview-body :deep(p) { margin: 8px 0; }
.preview-body :deep(img) { max-width: 100%; border-radius: 6px; margin: 8px 0; border: 1px solid #dce2d7; }
.preview-body :deep(pre) { background: #f6f8fa; padding: 12px; border-radius: 6px; overflow-x: auto; font-size: 13px; }
.preview-body :deep(code) { font-size: 13px; background: #edf0e9; padding: 1px 4px; border-radius: 3px; }
.preview-body :deep(pre code) { background: none; padding: 0; }
.preview-body :deep(blockquote) { border-left: 3px solid #216354; padding-left: 12px; color: #58685d; margin: 8px 0; }
.preview-body :deep(ul), .preview-body :deep(ol) { padding-left: 20px; }
.preview-body :deep(li) { margin: 4px 0; }
.preview-body :deep(hr) { border: none; border-top: 1px solid #dce2d7; margin: 16px 0; }
.preview-body :deep(a) { color: #216354; }
.preview-body :deep(table) { display: block; overflow-x: auto; border-collapse: collapse; }
.preview-body :deep(td), .preview-body :deep(th) { border: 1px solid var(--color-border); padding: 5px 8px; }

/* --- Tips --- */
.tips-box {
  display: flex; align-items: flex-start; gap: 8px;
  font-size: 12px; color: #667568; line-height: 1.6;
  padding: 10px 12px; background: #f5f6f0; border-radius: 8px;
}
.submit-actions { display: flex; justify-content: flex-end; flex-wrap: wrap; gap: 8px; }
.submit-actions .el-button { margin: 0; min-height: 40px; }
@media(max-width: 720px) {
  .editor-panels { grid-template-columns: minmax(0, 1fr); }
  .preview-panel { border-inline-start: 0; border-block-start: 1px solid var(--color-border); }
  .md-editor { height: clamp(185px, 38dvh, 300px); }
  .preview-pane { height: auto; min-height: 110px; max-height: clamp(180px, 42dvh, 360px); overscroll-behavior: auto; }
  .editor-toolbar { align-items: flex-start; padding: 8px; }
  .tb-btn { min-height: 36px; }
  .submit-actions { gap: 6px; }
  .submit-actions .el-button { padding-inline: 12px; }
}
</style>
