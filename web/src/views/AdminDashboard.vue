<template>
  <div class="admin-page">
    <div class="admin-header">
      <div>
        <h1 class="admin-title">管理后台</h1>
        <p class="admin-sub">管理技能点、任务、用户与提交审核</p>
      </div>
    </div>

    <div class="section-card" style="margin-bottom:20px">
      <div class="section-body" style="padding:0">
        <el-tabs v-model="activeTab">
          <el-tab-pane label="关卡" name="skills" />
          <el-tab-pane label="任务" name="tasks" />
          <el-tab-pane label="用户" name="users" />
          <el-tab-pane label="提交审核" name="submissions" />
          <el-tab-pane label="邀请码" name="inviteCodes" />
          <el-tab-pane label="师徒关系图" name="mentorGraph" />
        </el-tabs>
      </div>
    </div>

    <!-- ======================== Skills Tab ======================== -->
    <div v-show="activeTab === 'skills'">
      <div style="margin-bottom:16px">
        <el-button type="primary" @click="openSkillDialog()">新增关卡</el-button>
      </div>
      <el-table :data="skillsList" stripe empty-text="暂无关卡">
        <el-table-column prop="id" label="ID" width="55" />
        <el-table-column prop="name" label="名称" min-width="200" />
        <el-table-column prop="category" label="分类" width="80" />
        <el-table-column label="等级" width="70">
          <template #default="{ row }">
            <span class="lv-badge" :class="'lv-'+row.tier">Lv.{{ row.tier }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="orderIndex" label="关卡序" width="75" align="center" />
        <el-table-column prop="xpReward" label="通关XP" width="80" align="center" />
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button size="small" text type="primary" @click="openSkillDialog(row)">编辑</el-button>
            <el-button size="small" text type="danger" @click="delSkill(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- ======================== Tasks Tab ======================== -->
    <div v-show="activeTab === 'tasks'">
      <div style="display:flex;gap:12px;margin-bottom:16px;align-items:center">
        <span style="font-size:13px;color:var(--color-text-secondary)">按关卡筛选：</span>
        <el-select v-model="taskSkillFilter" filterable clearable placeholder="选择关卡" size="small" style="width:240px" @change="fetchTasks">
          <el-option v-for="s in skillsList" :key="s.id" :label="s.name" :value="s.id" />
        </el-select>
        <el-button type="primary" size="small" @click="openTaskDialog()" style="margin-left:auto">新增任务</el-button>
      </div>
      <el-table :data="tasksList" stripe empty-text="暂无任务，请选择关卡筛选">
        <el-table-column prop="id" label="ID" width="55" />
        <el-table-column prop="sortOrder" label="排序" width="65" align="center" />
        <el-table-column prop="title" label="标题" min-width="280" />
        <el-table-column prop="levelId" label="关卡ID" width="80" />
        <el-table-column prop="xpReward" label="XP" width="70" align="center" />
        <el-table-column label="AI审核" width="80" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.aiReviewEnabled" size="small"
                       @change="toggleAiReview(row)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button size="small" text @click="moveTask(row.id, 'up')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14"><polyline points="18 15 12 9 6 15"/></svg>
            </el-button>
            <el-button size="small" text @click="moveTask(row.id, 'down')" style="margin-left:0">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14"><polyline points="6 9 12 15 18 9"/></svg>
            </el-button>
            <el-button size="small" text type="primary" @click="openTaskDialog(row)">编辑</el-button>
            <el-button size="small" text type="danger" @click="delTask(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- ======================== Users Tab ======================== -->
    <div v-show="activeTab === 'users'">
      <el-table :data="usersList" stripe empty-text="暂无用户">
        <el-table-column prop="id" label="ID" width="55" />
        <el-table-column label="用户" width="170">
          <template #default="{ row }">
            {{ row.name ? row.name + ' (' + row.username + ')' : row.username }}
          </template>
        </el-table-column>
        <el-table-column label="角色" width="90">
          <template #default="{ row }">
            <span class="role-tag" :class="row.role.toLowerCase()">{{ roleText(row.role) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="级别" width="260">
          <template #default="{ row }">
            <div class="level-cell">
              <span class="level-badge" :class="'tier-'+getTierClass(row.level)">{{ levelTitle(row.level) }}</span>
              <el-input-number v-model="row._level" :min="0" :max="99" size="small" style="width:100px;margin-left:8px" controls-position="right" @change="v => setLevel(row.id, v)" />
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="totalXp" label="XP" width="80" align="center" />
        <el-table-column label="操作" min-width="300">
          <template #default="{ row }">
            <el-select v-model="row._newRole" size="small" style="width:85px" @change="v => updateRole(row.id, v)">
              <el-option label="教师" value="TEACHER" />
              <el-option label="内部" value="INTERNAL" />
              <el-option label="游客" value="EXTERNAL" />
            </el-select>
            <el-button size="small" type="danger" text style="margin-left:6px" @click="removeUser(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div style="display:flex;justify-content:center;margin-top:16px">
        <el-pagination layout="prev, pager, next" :total="totalUsers" @current-change="p => fetchUsers(p - 1)" background />
      </div>
    </div>

    <!-- ======================== Submissions Tab ======================== -->
    <div v-show="activeTab === 'submissions'">
      <el-table :data="subsList" stripe empty-text="暂无提交记录">
        <el-table-column prop="id" label="ID" width="55" />
        <el-table-column label="用户" width="140">
          <template #default="{ row }">
            {{ row.name ? row.name + ' (' + row.username + ')' : row.username }}
          </template>
        </el-table-column>
        <el-table-column prop="taskTitle" label="任务" min-width="180" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <span class="sub-status-tag" :class="row.status.toLowerCase()">{{ subStatusText(row.status) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="提交时间" width="170">
          <template #default="{ row }">{{ formatTime(row.submittedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" text @click="openSubDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- ==================== Submission Detail Dialog ==================== -->
    <el-dialog v-model="subDetailVisible" title="提交详情" width="800px" top="3vh">
      <div v-if="subDetail" class="sub-detail">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="提交 ID">{{ subDetail.id }}</el-descriptions-item>
          <el-descriptions-item label="用户">{{ subDetail.name ? subDetail.name + ' (' + subDetail.username + ')' : subDetail.username }}</el-descriptions-item>
          <el-descriptions-item label="任务">{{ subDetail.taskTitle }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <span class="sub-status-tag" :class="subDetail.status.toLowerCase()">{{ subStatusText(subDetail.status) }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="提交时间">{{ formatTime(subDetail.submittedAt) }}</el-descriptions-item>
          <el-descriptions-item label="审核时间">{{ subDetail.reviewedAt ? formatTime(subDetail.reviewedAt) : '-' }}</el-descriptions-item>
        </el-descriptions>

        <div class="sub-section">
          <div class="sub-section-title">提交内容</div>
          <div class="sub-md-preview" v-html="subMdHtml"></div>
        </div>

        <div v-if="subDetail.aiFeedback" class="sub-section ai-feedback">
          <div class="sub-section-title">AI 评审意见</div>
          <div class="sub-ai-text">{{ subDetail.aiFeedback }}</div>
        </div>

        <div v-if="subDetail.reviewComment" class="sub-section">
          <div class="sub-section-title">审核评语</div>
          <div class="sub-comment-text">{{ subDetail.reviewComment }}</div>
        </div>

        <div class="sub-section">
          <div class="sub-section-title">审核操作</div>
          <div class="sub-actions">
            <el-input v-model="subDetail._comment" placeholder="输入评语" size="small" style="width:200px" />
            <el-button type="success" size="small" @click="doReview('approved')" :loading="reviewing">通过</el-button>
            <el-button type="danger" size="small" @click="doReview('rejected')" :loading="reviewing">拒绝</el-button>
            <el-button type="warning" size="small" @click="doAiReview()" :loading="aiReviewing">🤖 AI 评审</el-button>
          </div>
        </div>
      </div>
    </el-dialog>


    <!-- ======================== Invite Codes Tab ======================== -->
    <div v-show="activeTab === 'inviteCodes'">
      <div style="display:flex;gap:12px;margin-bottom:16px;align-items:center;flex-wrap:wrap">
        <el-input v-model="newInviteCode" placeholder="新邀请码" size="small" style="width:160px" />
        <el-input v-model="newInviteDesc" placeholder="描述（可选）" size="small" style="width:180px" />
        <el-input-number v-model="newInviteDays" :min="1" :max="365" size="small" style="width:130px" />
        <span style="font-size:12px;color:#94a3b8">天有效期（默认 7 天）</span>
        <el-button type="primary" size="small" @click="createInviteCode" :loading="saving">生成邀请码</el-button>
      </div>
      <el-table :data="inviteCodesList" stripe empty-text="暂无邀请码">
        <el-table-column prop="id" label="ID" width="55" />
        <el-table-column prop="code" label="邀请码" width="180" />
        <el-table-column prop="description" label="描述" min-width="160" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.active ? 'success' : 'info'" size="small">{{ row.active ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="170">
          <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="过期时间" width="170">
          <template #default="{ row }">{{ formatTime(row.expiresAt) }}</template>
        </el-table-column>
        <el-table-column label="剩余" width="90">
          <template #default="{ row }">
            <span v-if="row.expiresAt" :style="{ color: daysLeft(row.expiresAt) < 3 ? '#dc2626' : '#16a34a' }">{{ daysLeft(row.expiresAt) }} 天</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80">
          <template #default="{ row }">
            <el-button size="small" text type="danger" @click="delInviteCode(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- ======================== Mentor Graph Tab ======================== -->
    <div v-show="activeTab === 'mentorGraph'">
      <div style="display:flex;gap:12px;margin-bottom:16px;align-items:center;flex-wrap:wrap">
        <el-input v-model="mentorSearch" placeholder="搜索姓名或用户名，回车高亮其师承链" clearable size="small"
                  style="width:280px" @keyup.enter="highlightLineage" @clear="highlightLineage" />
        <span v-for="lg in roleLegends" :key="lg.label" style="display:inline-flex;align-items:center;gap:4px;font-size:12px;color:var(--color-text-secondary)">
          <span :style="{ width:'10px', height:'10px', borderRadius:'50%', background: lg.color, display:'inline-block' }" />{{ lg.label }}
        </span>
      </div>
      <div class="section-card">
        <div class="section-body" v-loading="graphLoading" style="height:640px;position:relative">
          <el-empty v-if="!graphLoading && graphEmpty" description="暂无师徒关系" style="padding-top:80px" />
          <div ref="graphRef" style="width:100%;height:100%" />
        </div>
      </div>
    </div>

    <!-- ==================== Skill Edit Dialog ==================== -->
    <el-dialog v-model="skillDialogVisible" :title="editingSkill?.id ? '编辑关卡' : '新增关卡'" width="900px" top="3vh" @close="editingSkill = null; skillTutorials = []">
      <el-form v-if="skillForm" label-position="top">
        <el-form-item label="名称">
          <el-input v-model="skillForm.name" placeholder="如：认识你的工具箱" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="skillForm.description" type="textarea" :rows="2" placeholder="关卡描述" />
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="分类">
              <el-select v-model="skillForm.category" style="width:100%">
                <el-option v-for="c in categories" :key="c" :label="c" :value="c" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="等级">
              <el-input-number v-model="skillForm.tier" :min="1" :max="5" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="所属地图 ID">
              <el-input-number v-model="skillForm.mapId" :min="1" :max="7" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="关卡序号">
              <el-input-number v-model="skillForm.orderIndex" :min="1" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="通关 XP">
              <el-input-number v-model="skillForm.xpReward" :min="10" :max="200" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="前置关卡（可多选）">
          <el-select v-model="skillForm.prerequisiteLevelIds" multiple style="width:100%" placeholder="选择前置关卡">
            <el-option v-for="s in skillsList" :key="s.id" :label="s.name" :value="s.id"
                       :disabled="s.id === editingSkill?.id" />
          </el-select>
        </el-form-item>

        <!-- Tutorials -->
        <div v-if="editingSkill?.id" class="tutorial-mgr">
          <div class="tutorial-mgr-header">
            <span style="font-weight:600;font-size:14px">📖 教程管理</span>
            <el-button size="small" type="primary" @click="openTutorialDialog()">新增教程</el-button>
          </div>
          <div v-if="!skillTutorials.length" style="color:#94a3b8;font-size:13px;padding:12px 0">暂无教程</div>
          <div v-for="(t, idx) in skillTutorials" :key="t.id" class="tutorial-row">
            <div class="tutorial-row-header">
              <span class="tutorial-idx">{{ idx + 1 }}</span>
              <span class="tutorial-title-text">{{ t.title }}</span>
              <span class="tutorial-order">排序: {{ t.sortOrder }}</span>
              <el-button size="small" text type="primary" @click="editTutorial(t)">编辑</el-button>
              <el-button size="small" text type="danger" @click="deleteTutorial(t.id)">删除</el-button>
            </div>
            <div v-if="editingTutorialId === t.id" class="tutorial-edit-area">
              <el-input v-model="tutorialForm.title" placeholder="教程标题" style="margin-bottom:8px" />
              <el-input v-model="tutorialForm.content" type="textarea" :rows="8" placeholder="教程内容（支持 Markdown）" />
              <div style="margin-top:8px;display:flex;gap:8px">
                <el-button size="small" type="primary" @click="saveTutorial(t.id)" :loading="savingTutorial">保存教程</el-button>
                <el-button size="small" @click="cancelTutorialEdit">取消</el-button>
              </div>
            </div>
          </div>
        </div>
        <div v-else style="color:#94a3b8;font-size:13px;padding:8px 0">保存关卡后可管理教程</div>
      </el-form>
      <template #footer>
        <el-button @click="skillDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveSkill" :loading="saving">保存</el-button>
      </template>
    </el-dialog>

    <!-- ==================== Task Edit Dialog ==================== -->
    <el-dialog v-model="taskDialogVisible" :title="editingTask?.id ? '编辑任务' : '新增任务'" width="900px" top="3vh" @close="editingTask = null">
      <el-form v-if="taskForm" label-position="top">
        <el-form-item label="所属关卡">
          <el-select v-model="taskForm.levelId" style="width:100%" placeholder="选择关卡">
            <el-option v-for="s in skillsList" :key="s.id" :label="s.name" :value="s.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="标题">
          <el-input v-model="taskForm.title" placeholder="任务标题" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="taskForm.description" type="textarea" :rows="8" placeholder="任务详细描述" />
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="经验值">
              <el-input-number v-model="taskForm.xpReward" :min="5" :max="200" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="排序">
              <el-input-number v-model="taskForm.sortOrder" :min="1" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="AI 自动审核">
          <el-switch v-model="taskForm.aiReviewEnabled" active-text="开启" inactive-text="关闭" />
        </el-form-item>
        <el-form-item label="验收标准（JSON 格式）">
          <el-input v-model="taskForm.acceptanceCriteriaJson" type="textarea" :rows="8"
                    placeholder='[{"criteria":"完成要求","weight":50}]' />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="taskDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveTask" :loading="saving">保存</el-button>
      </template>
    </el-dialog>

  </div>
</template>

<script setup>
import { ref, onMounted, reactive, computed, watch, nextTick, onBeforeUnmount } from 'vue'
import * as echarts from 'echarts'
import { adminApi } from '../api/admin'
import { tasksApi } from '../api/tasks'
import { ElMessage, ElMessageBox } from 'element-plus'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const activeTab = ref('skills')
const skillsList = ref([])
const tasksList = ref([])
const usersList = ref([])
const subsList = ref([])
const inviteCodesList = ref([])
const taskSkillFilter = ref(null)
const saving = ref(false)
const totalUsers = ref(0)
const newInviteCode = ref('')
const newInviteDesc = ref('')
const newInviteDays = ref(7)

function formatTime(ts) {
  if (!ts) return '-'
  const d = new Date(ts)
  return d.getFullYear() + '-' + String(d.getMonth()+1).padStart(2,'0') + '-' + String(d.getDate()).padStart(2,'0') + ' ' + String(d.getHours()).padStart(2,'0') + ':' + String(d.getMinutes()).padStart(2,'0') + ':' + String(d.getSeconds()).padStart(2,'0')
}

const categories = ['前端', '后端', '数据库', '运维', '工具', '综合', '软技能']

// --- Skill dialog ---
const skillDialogVisible = ref(false)
const editingSkill = ref(null)
const skillForm = ref(null)

function openSkillDialog(row) {
  editingSkill.value = row || null
  skillForm.value = row
    ? reactive({
        name: row.name,
        description: row.description || '',
        category: row.category,
        tier: row.tier,
        mapId: row.mapId,
        orderIndex: row.orderIndex,
        xpReward: row.xpReward,
        prerequisiteLevelIds: [...(row.prerequisiteLevelIds || row.prerequisiteIds || [])]
      })
    : reactive({
        name: '',
        description: '',
        category: '前端',
        tier: 1,
        mapId: 1,
        orderIndex: 1,
        xpReward: 50,
        prerequisiteLevelIds: []
      })
  // 加载教程列表
  if (row?.id) {
    fetchSkillTutorials(row.id)
  } else {
    skillTutorials.value = []
  }
  skillDialogVisible.value = true
}

async function saveSkill() {
  saving.value = true
  try {
    const data = {
      name: skillForm.value.name,
      description: skillForm.value.description,
      category: skillForm.value.category,
      tier: skillForm.value.tier,
      mapId: skillForm.value.mapId,
      orderIndex: skillForm.value.orderIndex,
      xpReward: skillForm.value.xpReward,
      prerequisiteLevelIds: skillForm.value.prerequisiteLevelIds,
      isBuiltin: false,
      createdBy: 1
    }
    if (editingSkill.value?.id) {
      await adminApi.updateLevel(editingSkill.value.id, data)
      ElMessage.success('关卡已更新')
    } else {
      await adminApi.createLevel(data)
      ElMessage.success('关卡已创建')
    }
    skillDialogVisible.value = false
    editingSkill.value = null
    await fetchSkills()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '保存失败')
  } finally { saving.value = false }
}

// --- Task dialog ---
const taskDialogVisible = ref(false)
const editingTask = ref(null)
const taskForm = ref(null)

function openTaskDialog(row) {
  editingTask.value = row || null
  taskForm.value = row
    ? reactive({
        levelId: row.levelId,
        title: row.title,
        description: row.description || '',
        xpReward: row.xpReward,
        sortOrder: row.sortOrder,
        aiReviewEnabled: row.aiReviewEnabled || false,
        acceptanceCriteriaJson: row.acceptanceCriteria ? JSON.stringify(row.acceptanceCriteria, null, 2) : '[]'
      })
    : reactive({
        levelId: null,
        title: '',
        description: '',
        xpReward: 20,
        sortOrder: null,
        aiReviewEnabled: false,
        acceptanceCriteriaJson: '[{"criteria":"正确完成任务要求","weight":50},{"criteria":"提交内容完整清晰","weight":30},{"criteria":"代码/操作规范","weight":20}]'
      })
  taskDialogVisible.value = true
}

async function saveTask() {
  saving.value = true
  try {
    let criteria
    try { criteria = JSON.parse(taskForm.value.acceptanceCriteriaJson) } catch { criteria = [] }
    const data = {
      title: taskForm.value.title,
      description: taskForm.value.description,
      levelId: taskForm.value.levelId,
      xpReward: taskForm.value.xpReward,
      sortOrder: taskForm.value.sortOrder,
      aiReviewEnabled: taskForm.value.aiReviewEnabled,
      acceptanceCriteria: criteria,
      isBuiltin: false,
      createdBy: 1
    }
    if (editingTask.value?.id) {
      await adminApi.updateTask(editingTask.value.id, data)
      ElMessage.success('任务已更新')
    } else {
      await adminApi.createTask(data)
      ElMessage.success('任务已创建')
    }
    taskDialogVisible.value = false
    editingTask.value = null
    await fetchTasks()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '保存失败')
  } finally { saving.value = false }
}

// --- Tutorial management ---
const skillTutorials = ref([])
const editingTutorialId = ref(null)
const tutorialForm = reactive({ title: '', content: '' })
const savingTutorial = ref(false)

async function fetchSkillTutorials(skillId) {
  try {
    const res = await adminApi.listTutorials(skillId)
    skillTutorials.value = res.data.data || []
  } catch (e) { /* ignore */ }
}

function openTutorialDialog() {
  editingTutorialId.value = 'new'
  tutorialForm.title = ''
  tutorialForm.content = ''
}

function editTutorial(t) {
  editingTutorialId.value = t.id
  tutorialForm.title = t.title
  tutorialForm.content = t.content
}

function cancelTutorialEdit() {
  editingTutorialId.value = null
  tutorialForm.title = ''
  tutorialForm.content = ''
}

async function saveTutorial(tutorialId) {
  if (!tutorialForm.title.trim()) {
    ElMessage.warning('请输入教程标题')
    return
  }
  savingTutorial.value = true
  try {
    const data = {
      levelId: editingSkill.value.id,
      title: tutorialForm.title,
      content: tutorialForm.content,
      sortOrder: skillTutorials.value.length + 1
    }
    if (tutorialId !== 'new') {
      await adminApi.updateTutorial(tutorialId, data)
      ElMessage.success('教程已更新')
    } else {
      await adminApi.createTutorial(data)
      ElMessage.success('教程已创建')
    }
    cancelTutorialEdit()
    await fetchSkillTutorials(editingSkill.value.id)
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '保存失败')
  } finally { savingTutorial.value = false }
}

async function deleteTutorial(id) {
  try {
    await ElMessageBox.confirm('确定删除该教程？', '确认', { type: 'warning' })
    await adminApi.deleteTutorial(id)
    ElMessage.success('已删除')
    await fetchSkillTutorials(editingSkill.value.id)
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('删除失败')
  }
}

// --- Data fetching ---
function roleText(r) { return { TEACHER:'教师',INTERNAL:'内部',EXTERNAL:'游客' }[r]||r }

function levelTitle(lv) {
  if (lv >= 70) return '👑 王者 Lv.' + lv
  if (lv >= 60) return '🏆 宗师 Lv.' + lv
  if (lv >= 50) return '⭐ 大师 Lv.' + lv
  if (lv >= 40) return '💎 钻石 Lv.' + lv
  if (lv >= 30) return '🔮 铂金 Lv.' + lv
  if (lv >= 20) return '🥇 黄金 Lv.' + lv
  if (lv >= 10) return '🥈 白银 Lv.' + lv
  return '🥉 青铜 Lv.' + lv
}

function getTierClass(lv) {
  if (lv >= 70) return 'king'
  if (lv >= 60) return 'grandmaster'
  if (lv >= 50) return 'master'
  if (lv >= 40) return 'diamond'
  if (lv >= 30) return 'platinum'
  if (lv >= 20) return 'gold'
  if (lv >= 10) return 'silver'
  return 'bronze'
}

async function setLevel(id, level) {
  try { await adminApi.setLevel(id, level); ElMessage.success('级别已更新') } catch (e) { ElMessage.error(e.response?.data?.message || '更新失败'); fetchUsers() }
}

// --- Submission detail ---
const subDetailVisible = ref(false)
const subDetail = ref(null)
const reviewing = ref(false)
const aiReviewing = ref(false)

function subStatusText(s) { return { APPROVED:'已通过',REJECTED:'未通过',PENDING:'审核中',AI_REVIEWED:'AI已评' }[s]||s }
const subMdHtml = computed(() => {
  if (!subDetail.value?.submissionData) return ''
  return DOMPurify.sanitize(marked.parse(subDetail.value.submissionData))
})

function openSubDetail(row) {
  subDetail.value = { ...row, _comment: row._comment || '' }
  subDetailVisible.value = true
}

async function doReview(status) {
  reviewing.value = true
  try {
    await tasksApi.review(subDetail.value.id, { status, reviewComment: subDetail.value._comment || undefined })
    ElMessage.success(status === 'approved' ? '已通过' : '已拒绝')
    subDetailVisible.value = false
    fetchSubs()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '审核失败')
  } finally { reviewing.value = false }
}

async function doAiReview() {
  aiReviewing.value = true
  try {
    await tasksApi.aiReview(subDetail.value.id)
    ElMessage.success('AI 评审完成')
    subDetailVisible.value = false
    fetchSubs()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || 'AI 评审失败')
  } finally { aiReviewing.value = false }
}


async function fetchSkills() {
  const res = await adminApi.listLevels()
  skillsList.value = res.data.data || []
}
async function fetchTasks() {
  const id = taskSkillFilter.value
  if (!id) { tasksList.value = []; return }
  const res = await tasksApi.list(id).catch(() => ({ data: { data: [] } }))
  tasksList.value = res.data.data || []
}
async function fetchUsers(page = 0) {
  const res = await adminApi.listUsers(page, 20)
  const data = res.data.data || {}
  usersList.value = (data.content || []).map(u => ({ ...u, _newRole: u.role, _level: u.level || 0 }))
  totalUsers.value = data.total || 0
}
async function fetchSubs() {
  const res = await adminApi.listSubmissions()
  subsList.value = (res.data.data || []).map(s => ({ ...s, _comment: '' }))
}

async function removeUser(row) {
  try {
    await ElMessageBox.confirm(`确定删除用户「${row.name || row.username}」？其提交、师徒关系、进度等数据将一并删除。`, '确认', { type: 'warning' })
    await adminApi.deleteUser(row.id)
    ElMessage.success('已删除')
    fetchUsers()
  } catch (e) { if (e !== 'cancel') ElMessage.error('删除失败') }
}
async function updateRole(id, role) { try { await adminApi.updateRole(id, role); ElMessage.success('已更新') } catch (e) { ElMessage.error('失败') } }
async function delSkill(id) {
  try { await ElMessageBox.confirm('确定删除该关卡？', '确认', { type: 'warning' }); await adminApi.deleteLevel(id); ElMessage.success('已删除'); fetchSkills() } catch (e) { if (e !== 'cancel') ElMessage.error(e.response?.data?.message || '删除失败') }
}
async function toggleAiReview(row) {
  try {
    await adminApi.updateTask(row.id, { ...row, aiReviewEnabled: row.aiReviewEnabled })
    ElMessage.success(row.aiReviewEnabled ? 'AI审核已开启' : 'AI审核已关闭')
  } catch (e) { ElMessage.error('切换失败'); fetchTasks() }
}
async function moveTask(id, direction) {
  try { await adminApi.moveTask(id, direction); fetchTasks() } catch (e) { ElMessage.error(e.response?.data?.message || '移动失败') }
}
async function delTask(id) {
  try { await ElMessageBox.confirm('确定删除该任务？', '确认', { type: 'warning' }); await adminApi.deleteTask(id); ElMessage.success('已删除'); fetchTasks() } catch (e) { if (e !== 'cancel') ElMessage.error(e.response?.data?.message || '删除失败') }
}

// --- Invite codes ---
async function fetchInviteCodes() {
  try { const res = await adminApi.listInviteCodes(); inviteCodesList.value = res.data.data || [] } catch (e) { /* ignore */ }
}
async function createInviteCode() {
  if (!newInviteCode.value.trim()) return
  saving.value = true
  try {
    const expiresAt = new Date(Date.now() + newInviteDays.value * 24 * 3600 * 1000).toISOString()
    await adminApi.createInviteCode({ code: newInviteCode.value.trim(), description: newInviteDesc.value.trim(), expiresAt })
    ElMessage.success(`邀请码已创建（${newInviteDays.value} 天后过期）`)
    newInviteCode.value = ''
    newInviteDesc.value = ''
    fetchInviteCodes()
  } catch (e) { ElMessage.error(e.response?.data?.message || '创建失败') } finally { saving.value = false }
}
function daysLeft(ts) {
  if (!ts) return 0
  return Math.max(0, Math.ceil((new Date(ts) - Date.now()) / 86400000))
}
async function delInviteCode(id) {
  try { await ElMessageBox.confirm('确定删除该邀请码？', '确认', { type: 'warning' }); await adminApi.deleteInviteCode(id); ElMessage.success('已删除'); fetchInviteCodes() } catch (e) { if (e !== 'cancel') ElMessage.error('删除失败') }
}

// --- Mentorship graph ---
const graphRef = ref(null)
const graphLoading = ref(false)
const mentorSearch = ref('')
let chart = null
let rawNodes = []
let rawEdges = []
const roleLegends = [
  { label: '教师', color: '#f56c6c' },
  { label: '内部', color: '#409eff' },
  { label: '游客', color: '#909399' }
]
const roleColor = { TEACHER: '#f56c6c', INTERNAL: '#409eff', EXTERNAL: '#909399' }
const graphEmpty = ref(false)

async function fetchMentorGraph() {
  graphLoading.value = true
  try {
    const res = await adminApi.getMentorshipGraph()
    const data = res.data.data || { nodes: [], edges: [] }
    rawNodes = data.nodes || []
    rawEdges = data.edges || []
    graphEmpty.value = rawNodes.length === 0
    await nextTick()
    renderGraph()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '师徒关系图加载失败')
  } finally { graphLoading.value = false }
}

function apprenticeCount(id) {
  return rawEdges.filter(e => e.mentorId === id).length
}

function escapeHtml(value) {
  return String(value ?? '').replace(/[&<>"']/g, ch => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[ch]))
}

function renderGraph() {
  if (!graphRef.value) return
  if (!chart) {
    chart = echarts.init(graphRef.value)
    window.addEventListener('resize', resizeChart)
  }
  const highlightSet = computeLineage(mentorSearch.value.trim())
  const nodes = rawNodes.map(n => ({
    id: String(n.id),
    name: n.name,
    symbolSize: (() => { const c = apprenticeCount(n.id); return c >= 3 ? 44 : c >= 1 ? 32 : 20 })(),
    itemStyle: {
      color: roleColor[n.role] || '#909399',
      opacity: highlightSet ? (highlightSet.has(String(n.id)) ? 1 : 0.15) : 1
    },
    label: { show: true, position: 'bottom', fontSize: 11 }
  }))
  const links = rawEdges.map(e => {
    const dim = highlightSet && !(highlightSet.has(String(e.mentorId)) && highlightSet.has(String(e.apprenticeId)))
    return {
      source: String(e.mentorId),
      target: String(e.apprenticeId),
      lineStyle: { opacity: dim ? 0.06 : 0.6, curveness: 0.15 }
    }
  })
  chart.setOption({
    tooltip: {
      formatter: p => {
        if (p.dataType !== 'node') return ''
        const n = rawNodes.find(x => String(x.id) === p.data.id) || {}
        return `${escapeHtml(n.name)}（${escapeHtml(n.username)}）<br/>角色：${escapeHtml(roleText(n.role))} · Lv.${escapeHtml(n.level)}<br/>徒弟数：${apprenticeCount(n.id)}`
      }
    },
    series: [{
      type: 'graph',
      layout: 'force',
      roam: true,
      draggable: true,
      data: nodes,
      links,
      edgeSymbol: ['none', 'arrow'],
      edgeSymbolSize: 8,
      force: { repulsion: 260, edgeLength: [70, 140], gravity: 0.08 },
      emphasis: { focus: 'adjacency' },
      label: { color: '#606266' }
    }]
  }, true)
}

/** 返回目标节点及其向上师傅链的 id 集合；无搜索词时返回 null 表示不高亮 */
function computeLineage(keyword) {
  if (!keyword) return null
  const k = keyword.toLowerCase()
  const start = rawNodes.find(n =>
    (n.name || '').toLowerCase().includes(k) || (n.username || '').toLowerCase().includes(k))
  if (!start) return null
  const mentorOf = {}
  rawEdges.forEach(e => { mentorOf[e.apprenticeId] = e.mentorId })
  const set = new Set([String(start.id)])
  let cur = start.id
  const seen = new Set([cur])
  while (mentorOf[cur] != null && !seen.has(mentorOf[cur])) {
    cur = mentorOf[cur]
    seen.add(cur)
    set.add(String(cur))
  }
  return set
}

function highlightLineage() { renderGraph() }

function resizeChart() { chart && chart.resize() }

watch(activeTab, tab => {
  if (tab === 'mentorGraph' && !chart) fetchMentorGraph()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', resizeChart)
  if (chart) { chart.dispose(); chart = null }
})

onMounted(async () => { await Promise.all([fetchSkills(), fetchTasks(), fetchUsers(), fetchSubs(), fetchInviteCodes()]) })
</script>

<style scoped>
.admin-page { max-width: 1300px; }
.admin-header { margin-bottom: 24px; }
.admin-title { font-size: 24px; font-weight: 700; color: var(--color-text); margin: 0 0 4px; }
.admin-sub { font-size: 14px; color: var(--color-text-secondary); margin: 0; }

.lv-badge { font-size: 11px; font-weight: 700; padding: 2px 8px; border-radius: 4px; white-space: nowrap; }
.lv-badge.lv-1, .lv-badge.lv-2 { background: #f1f5f9; color: #64748b; }
.lv-badge.lv-3 { background: #fef3c7; color: #d97706; }
.lv-badge.lv-4, .lv-badge.lv-5 { background: #fee2e2; color: #dc2626; }
.role-tag { font-size: 12px; font-weight: 500; padding: 2px 8px; border-radius: 4px; white-space: nowrap; }
.role-tag.teacher { background: #ede9fe; color: #7c3aed; }
.role-tag.internal { background: #eef2ff; color: #6366f1; }
.role-tag.external { background: #f1f5f9; color: #64748b; }
.preview-text { font-size: 13px; color: #64748b; }

/* --- Submission detail --- */
.sub-detail { display: flex; flex-direction: column; gap: 16px; }
.sub-section {
  padding: 14px 16px; background: #f8fafc; border-radius: 10px;
}
.sub-section-title { font-size: 13px; font-weight: 600; color: var(--color-text); margin-bottom: 10px; }
.sub-md-preview {
  font-size: 14px; line-height: 1.8; color: #334155;
}
.sub-md-preview :deep(h1) { font-size: 20px; margin: 12px 0 6px; }
.sub-md-preview :deep(h2) { font-size: 17px; margin: 10px 0 4px; }
.sub-md-preview :deep(h3) { font-size: 15px; margin: 8px 0 4px; }
.sub-md-preview :deep(img) { max-width: 100%; border-radius: 6px; border: 1px solid #e2e8f0; }
.sub-md-preview :deep(pre) { background: #1e293b; color: #e2e8f0; padding: 12px; border-radius: 6px; overflow-x: auto; font-size: 13px; }
.sub-md-preview :deep(code) { font-size: 13px; background: #f1f5f9; padding: 1px 4px; border-radius: 3px; }
.sub-md-preview :deep(pre code) { background: none; padding: 0; color: inherit; }
.sub-md-preview :deep(blockquote) { border-left: 3px solid #6366f1; padding-left: 12px; color: #64748b; margin: 8px 0; }
.sub-md-preview :deep(ul), .sub-md-preview :deep(ol) { padding-left: 20px; }
.sub-ai-text { font-size: 14px; color: #0c4a6e; white-space: pre-wrap; line-height: 1.7; }
.ai-feedback { background: #f0f9ff; border: 1px solid #bae6fd; }
.sub-comment-text { font-size: 14px; color: #475569; white-space: pre-wrap; line-height: 1.7; }
.sub-actions { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.sub-status-tag { font-size: 12px; font-weight: 500; padding: 2px 10px; border-radius: 4px; white-space: nowrap; }
.sub-status-tag.approved { background: #dcfce7; color: #16a34a; }
.sub-status-tag.rejected { background: #fee2e2; color: #dc2626; }
.sub-status-tag.pending { background: #fef3c7; color: #d97706; }
.sub-status-tag.ai_reviewed { background: #e0f2fe; color: #0284c7; }

/* --- Tutorial manager --- */
.tutorial-mgr {
  margin-top: 16px; padding: 16px; background: #f8fafc; border-radius: 10px; border: 1px solid var(--color-border);
}
.tutorial-mgr-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.tutorial-row { border-bottom: 1px solid #e2e8f0; padding: 8px 0; }
.tutorial-row:last-child { border-bottom: none; }
.tutorial-row-header { display: flex; align-items: center; gap: 8px; font-size: 13px; }
.tutorial-idx { width: 22px; height: 22px; border-radius: 50%; background: #6366f1; color: #fff; font-size: 11px; font-weight: 700; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.tutorial-title-text { flex: 1; font-weight: 500; }
.tutorial-order { font-size: 12px; color: #94a3b8; }
.tutorial-edit-area { margin-top: 8px; padding: 8px 0 4px 30px; }

/* Level badges */
.level-cell { display: flex; align-items: center; }
.level-badge { font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: 4px; white-space: nowrap; }
.level-badge.tier-bronze { background: #fdf2f8; color: #be185d; }
.level-badge.tier-silver { background: #f1f5f9; color: #475569; }
.level-badge.tier-gold { background: #fefce8; color: #b45309; }
.level-badge.tier-platinum { background: #f5f3ff; color: #7c3aed; }
.level-badge.tier-diamond { background: #ecfeff; color: #0e7490; }
.level-badge.tier-master { background: #fef2f2; color: #dc2626; }
.level-badge.tier-grandmaster { background: #fff7ed; color: #c2410c; }
.level-badge.tier-king { background: linear-gradient(135deg, #fef3c7, #fde68a); color: #92400e; }
</style>
