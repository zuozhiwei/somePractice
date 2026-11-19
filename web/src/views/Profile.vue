<template>
  <div class="page" v-loading="loading">
    <header class="profile-head">
      <div class="profile-monogram" v-if="profile" aria-hidden="true">{{ Array.from(profile.name || profile.username)[0] }}</div>
      <div>
        <h1 class="page-title">个人中心</h1>
        <p v-if="profile" class="profile-identity">{{ profile.name || profile.username }} · {{ roleText(profile.role) }}</p>
      </div>
    </header>

    <dl v-if="profile" class="profile-stats">
      <div><dt>累计经验</dt><dd>{{ profile.totalXp }} <small>XP</small></dd></div>
      <div><dt>已通关关卡</dt><dd>{{ profile.completedLevels }}</dd></div>
      <div><dt>已通过任务</dt><dd>{{ approvedTasks }}</dd></div>
      <div><dt>认证级别</dt><dd>{{ profile.levelTitle }} <small>Lv.{{ profile.level }}</small></dd></div>
    </dl>

    <!-- Profile info -->
    <div class="section-card" style="margin-bottom:24px" v-if="profile">
      <div class="section-header">
        <span>基本信息</span>
        <el-button size="small" text type="primary" @click="startEdit" v-if="!editingProfile">编辑</el-button>
        <el-button size="small" text @click="editingProfile = false" v-else>取消</el-button>
      </div>
      <div class="section-body">
        <dl v-if="!editingProfile" class="profile-fields">
          <div><dt>用户名</dt><dd>{{ profile.username }}</dd></div>
          <div><dt>用户 ID</dt><dd>{{ profile.id }}</dd></div>
          <div><dt>姓名</dt><dd>{{ profile.name || '未填写' }}</dd></div>
          <div><dt>角色</dt><dd>{{ roleText(profile.role) }}</dd></div>
          <div><dt>邮箱</dt><dd>{{ profile.email || '未绑定' }}</dd></div>
          <div><dt>注册时间</dt><dd>{{ formatTime(profile.createdAt) }}</dd></div>
        </dl>
        <el-form v-else label-position="top" style="max-width:400px">
          <el-form-item label="姓名">
            <el-input v-model="editForm.name" placeholder="你的姓名" />
          </el-form-item>
          <el-form-item label="邮箱">
            <el-input v-model="editForm.email" placeholder="用于找回密码" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="saveProfile" :loading="saving">保存</el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- Submission history -->
    <div class="section-card">
      <div class="section-header">
        <span>提交历史</span>
        <span class="count-badge">{{ subTotal }} 条</span>
      </div>
      <div class="section-body" style="padding:0">
        <el-table :data="submissions" stripe style="width:100%" empty-text="暂无提交记录">
          <el-table-column prop="taskTitle" label="任务" min-width="200" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <span class="status-tag" :class="row.status.toLowerCase()">{{ subStatusText(row.status) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="提交时间" width="170">
          <template #default="{ row }">{{ formatTime(row.submittedAt) }}</template>
        </el-table-column>
          <el-table-column prop="aiFeedback" label="AI 反馈" min-width="200">
            <template #default="{ row }">
              <span class="feedback-text">{{ row.aiFeedback || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="reviewComment" label="审核评语" min-width="150">
            <template #default="{ row }">
              <span class="feedback-text">{{ row.reviewComment || '-' }}</span>
            </template>
          </el-table-column>
        </el-table>
        <div style="display:flex;justify-content:center;padding:12px 0" v-if="subTotal > 10">
          <el-pagination v-model:current-page="subPage" :page-size="10"
                         layout="prev, pager, next" :total="subTotal"
                         @current-change="p => fetchSubmissions(p - 1)" background small />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { authApi } from '../api/auth'
import { tasksApi } from '../api/tasks'
import { ElMessage } from 'element-plus'

const profile = ref(null)
const submissions = ref([])
const subPage = ref(1)
const subTotal = ref(0)
const approvedTasks = ref(0)
const loading = ref(true)
const editingProfile = ref(false)
const saving = ref(false)
const editForm = reactive({ name: '', email: '' })

function roleText(r) { return { SUPERADMIN: '超级管理员', TEACHER:'教师',INTERNAL:'内部成员',EXTERNAL:'游客' }[r]||r }

function subStatusText(s) { return { APPROVED:'已通过',REJECTED:'未通过',PENDING:'审核中',AI_REVIEWED:'AI已评' }[s]||s }

function formatTime(ts) {
  if (!ts) return '-'
  const d = new Date(ts)
  return d.getFullYear() + '-' + String(d.getMonth()+1).padStart(2,'0') + '-' + String(d.getDate()).padStart(2,'0') + ' ' + String(d.getHours()).padStart(2,'0') + ':' + String(d.getMinutes()).padStart(2,'0') + ':' + String(d.getSeconds()).padStart(2,'0')
}

function startEdit() {
  editForm.name = profile.value?.name || ''
  editForm.email = profile.value?.email || ''
  editingProfile.value = true
}

async function saveProfile() {
  saving.value = true
  try {
    const res = await authApi.updateProfile({ name: editForm.name, email: editForm.email })
    profile.value = res.data.data
    editingProfile.value = false
    ElMessage.success('个人信息已更新')
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '保存失败')
  } finally { saving.value = false }
}

async function fetchSubmissions(page = 0) {
  try {
    const res = await tasksApi.mySubmissions(page, 10)
    const data = res.data.data
    submissions.value = data.content || []
    subTotal.value = data.total || 0
    approvedTasks.value = data.approvedTasks || 0
  } catch (e) { console.error(e) }
}

onMounted(async () => {
  try {
    const [p] = await Promise.all([authApi.profile(), fetchSubmissions(0)])
    profile.value = p.data.data
  } catch (e) { console.error(e) } finally { loading.value = false }
})
</script>

<style scoped>
.page { max-width: 960px; }
.count-badge {
  font-size: 12px; color: var(--color-text-secondary);
  background: #edf0e9; padding: 2px 10px; border-radius: 10px;
}
.status-tag {
  font-size: 11px; font-weight: 500; padding: 2px 8px; border-radius: 4px;
}
.status-tag.approved { background: #e0efe6; color: #30684b; }
.status-tag.rejected { background: #fee2e2; color: #dc2626; }
.status-tag.pending { background: #faf0d7; color: #895b19; }
.feedback-text { font-size: 13px; color: #58685d; }
.profile-head { display: flex; align-items: center; gap: 16px; padding: 6px 0 22px; }
.profile-head .page-title { margin: 0; }
.profile-monogram { display: grid; place-items: center; width: 54px; height: 54px; flex: 0 0 auto; border-radius: 17px; background: #1e5043; color: #fffaf0; font-size: 23px; font-weight: 700; }
.profile-identity { color: var(--color-text-secondary); font-size: 14px; margin: 5px 0 0; }
.profile-stats { display: grid; grid-template-columns: repeat(4,minmax(0,1fr)); border: 1px solid var(--color-border); border-radius: var(--radius); overflow: hidden; background: var(--color-surface); margin: 0 0 22px; }
.profile-stats > div { padding: 19px clamp(13px, 2vw, 28px); border-inline-end: 1px solid var(--color-border); }
.profile-stats > div:last-child { border: 0; }
.profile-stats dt,.profile-fields dt { font-size: 13px; color: var(--color-text-secondary); margin-bottom: 6px; }
.profile-stats dd,.profile-fields dd { margin: 0; color: var(--color-text); }
.profile-stats dd { font-weight: 700; font-size: clamp(21px,2.8vw,31px); }
.profile-stats dd small { color: var(--color-text-secondary); font-size: 12px; }
.profile-fields { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 18px; margin: 0; }
.profile-fields dd { overflow-wrap: anywhere; }
.status-tag.ai_reviewed { background: #deedf1; color: #28657a; }
@media(max-width: 680px) {
  .profile-stats { grid-template-columns: repeat(2,minmax(0,1fr)); }
  .profile-stats > div:nth-child(2) { border-inline-end: 0; }
  .profile-stats > div:nth-child(-n+2) { border-bottom: 1px solid var(--color-border); }
}
</style>
