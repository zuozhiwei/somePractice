<template>
  <div class="page" v-loading="loading">
    <h1 class="page-title">个人中心</h1>
    <p class="page-subtitle">查看学习进度与提交记录</p>

    <!-- Stats row -->
    <div class="card-grid cols-3" style="margin-bottom:24px" v-if="profile">
      <div class="stat-card">
        <div class="stat-icon" style="background:#eef2ff;color:#6366f1">⚡</div>
        <div>
          <div class="stat-value">{{ profile.totalXp }}</div>
          <div class="stat-label">经验值 XP</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background:#f0fdf4;color:#22c55e">🎯</div>
        <div>
          <div class="stat-value">{{ profile.completedLevels }}</div>
          <div class="stat-label">已通关关卡</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background:#fef3c7;color:#d97706">🏅</div>
        <div>
          <div class="stat-value">{{ submissions.filter(s => s.status === 'APPROVED').length }}</div>
          <div class="stat-label">通过任务</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" :style="levelIconStyle">{{ levelEmoji(profile.level) }}</div>
        <div>
          <div class="stat-value">{{ levelTitle(profile.level) }}</div>
          <div class="stat-label">认证级别</div>
        </div>
      </div>
    </div>

    <!-- Profile info -->
    <div class="section-card" style="margin-bottom:24px" v-if="profile">
      <div class="section-header">
        <span>基本信息</span>
        <el-button size="small" text type="primary" @click="startEdit" v-if="!editingProfile">编辑</el-button>
        <el-button size="small" text @click="editingProfile = false" v-else>取消</el-button>
      </div>
      <div class="section-body">
        <el-descriptions v-if="!editingProfile" :column="2" border>
          <el-descriptions-item label="ID">{{ profile.id }}</el-descriptions-item>
          <el-descriptions-item label="用户名">{{ profile.username }}</el-descriptions-item>
          <el-descriptions-item label="姓名">{{ profile.name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="角色">
            <el-tag size="small" effect="light">{{ roleText(profile.role) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="认证级别">
            <span class="profile-level">{{ levelTitle(profile.level) }} （Lv.{{ profile.level }}）</span>
          </el-descriptions-item>
          <el-descriptions-item label="邮箱">{{ profile.email || '-' }}</el-descriptions-item>
          <el-descriptions-item label="注册时间" :span="2">{{ formatTime(profile.createdAt) }}</el-descriptions-item>
        </el-descriptions>
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
                         @current-change="fetchSubmissions" background small />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive, computed } from 'vue'
import { authApi } from '../api/auth'
import { tasksApi } from '../api/tasks'
import { ElMessage } from 'element-plus'

const profile = ref(null)
const submissions = ref([])
const subPage = ref(1)
const subTotal = ref(0)
const loading = ref(true)
const editingProfile = ref(false)
const saving = ref(false)
const editForm = reactive({ name: '', email: '' })

function roleText(r) { return { TEACHER:'教师',INTERNAL:'内部成员',EXTERNAL:'游客' }[r]||r }

function levelTitle(lv) {
  if (lv >= 70) return '👑 王者'
  if (lv >= 60) return '🏆 宗师'
  if (lv >= 50) return '⭐ 大师'
  if (lv >= 40) return '💎 钻石'
  if (lv >= 30) return '🔮 铂金'
  if (lv >= 20) return '🥇 黄金'
  if (lv >= 10) return '🥈 白银'
  return '🥉 青铜'
}

function levelEmoji(lv) {
  if (lv >= 70) return '👑'
  if (lv >= 60) return '🏆'
  if (lv >= 50) return '⭐'
  if (lv >= 40) return '💎'
  if (lv >= 30) return '🔮'
  if (lv >= 20) return '🥇'
  if (lv >= 10) return '🥈'
  return '🥉'
}

const levelIconStyle = computed(() => {
  const lv = profile.value?.level || 0
  if (lv >= 70) return { background: '#fef3c7', color: '#92400e' }
  if (lv >= 50) return { background: '#fef2f2', color: '#dc2626' }
  if (lv >= 30) return { background: '#f5f3ff', color: '#7c3aed' }
  if (lv >= 10) return { background: '#f1f5f9', color: '#475569' }
  return { background: '#fdf2f8', color: '#be185d' }
})
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
  background: #f1f5f9; padding: 2px 10px; border-radius: 10px;
}
.status-tag {
  font-size: 11px; font-weight: 500; padding: 2px 8px; border-radius: 4px;
}
.status-tag.approved { background: #dcfce7; color: #16a34a; }
.status-tag.rejected { background: #fee2e2; color: #dc2626; }
.status-tag.pending { background: #fef3c7; color: #d97706; }
.feedback-text { font-size: 13px; color: #64748b; }
.profile-level { font-weight: 600; color: #6366f1; }
</style>
