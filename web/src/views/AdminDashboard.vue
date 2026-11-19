<template>
  <div class="admin-page">
    <div class="admin-header">
      <div class="staff-identity">{{ isSuperAdmin ? '超级管理员' : '教师' }}</div>
      <div>
        <h1 class="admin-title">{{ isSuperAdmin ? '系统管理' : '作业审核' }}</h1>
      </div>
    </div>

    <div class="admin-tabs">
      <div class="section-body" style="padding:0">
        <el-tabs v-model="activeTab">
          <el-tab-pane v-if="isSuperAdmin" label="关卡" name="skills" />
          <el-tab-pane v-if="isSuperAdmin" label="任务" name="tasks" />
          <el-tab-pane v-if="isSuperAdmin" label="教程" name="tutorials" />
          <el-tab-pane v-if="isSuperAdmin" label="用户" name="users" />
          <el-tab-pane label="提交审核" name="submissions" />
          <el-tab-pane v-if="isSuperAdmin" label="邀请码" name="inviteCodes" />
          <el-tab-pane v-if="isSuperAdmin" label="师徒关系图" name="mentorGraph" />
        </el-tabs>
      </div>
    </div>

    <!-- ======================== Skills Tab ======================== -->
    <div class="admin-workspace" v-if="isSuperAdmin" v-show="activeTab === 'skills'">
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
    <div class="admin-workspace" v-if="isSuperAdmin" v-show="activeTab === 'tasks'">
      <div class="manage-toolbar">
        <span style="font-size:13px;color:var(--color-text-secondary)">按关卡筛选：</span>
        <el-select aria-label="按关卡筛选" v-model="taskSkillFilter" filterable clearable placeholder="选择关卡" size="small" style="width:240px" @change="fetchTasks">
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
            <el-switch :aria-label="row.title + ' AI 审核'" v-model="row.aiReviewEnabled" size="small"
                       @change="toggleAiReview(row)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button size="small" text aria-label="上移任务" @click="moveTask(row.id, 'up')">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14"><polyline points="18 15 12 9 6 15"/></svg>
            </el-button>
            <el-button size="small" text aria-label="下移任务" @click="moveTask(row.id, 'down')" style="margin-left:0">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14"><polyline points="6 9 12 15 18 9"/></svg>
            </el-button>
            <el-button size="small" text type="primary" @click="openTaskDialog(row)">编辑</el-button>
            <el-button size="small" text type="danger" @click="delTask(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- ======================== Users Tab ======================== -->
    <div class="admin-workspace" v-if="isSuperAdmin" v-show="activeTab === 'tutorials'">
      <div class="manage-toolbar">
        <el-select aria-label="教程所属关卡" v-model="tutorialLevelId" filterable placeholder="选择关卡" @change="changeTutorialLevel">
          <el-option v-for="level in skillsList" :key="level.id" :label="level.name" :value="level.id" />
        </el-select>
        <el-button type="primary" :disabled="!tutorialLevelId" @click="openTutorialDialog()">新增教程</el-button>
      </div>
      <el-table v-loading="tutorialsLoading" :data="skillTutorials" stripe :empty-text="tutorialLevelId ? '该关卡暂无教程' : '请选择关卡'">
        <el-table-column prop="title" label="标题" min-width="220" />
        <el-table-column prop="sortOrder" label="排序" width="85" />
        <el-table-column label="操作" width="160">
          <template #default="{ row }">
            <el-button size="small" type="primary" text @click="openTutorialDialog(row)">编辑</el-button>
            <el-button size="small" type="danger" text @click="deleteTutorial(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <div class="admin-workspace" v-if="isSuperAdmin" v-show="activeTab === 'users'">
      <el-form class="manage-toolbar" @submit.prevent="searchUsers">
        <el-input v-model="userSearch" aria-label="搜索用户名、昵称或邮箱" clearable @clear="searchUsers" placeholder="用户名、昵称或邮箱" />
        <el-button native-type="submit" type="primary" :loading="usersLoading">搜索用户</el-button>
      </el-form>
      <div v-loading="usersLoading" class="directory">
        <table class="directory-table" aria-label="用户列表">
          <thead><tr><th>用户</th><th>角色</th><th>级别</th><th>操作</th></tr></thead>
          <tbody>
            <tr v-for="row in usersList" :key="row.id">
              <td class="directory-identity">
                <strong>{{ row.name || row.username }}</strong>
                <small v-if="row.name">{{ row.username }}</small>
                <span>{{ row.email || '未绑定邮箱' }}</span>
                <small>#{{ row.id }} · {{ row.totalXp || 0 }} XP</small>
              </td>
              <td>
                <span class="cell-label">角色</span>
                <span v-if="row.role === 'SUPERADMIN'" class="role-tag superadmin">超级管理员</span>
                <el-select v-else :model-value="row.role" :disabled="row._savingRole" :aria-label="row.username + ' 的角色'"
                           size="small" @change="v => updateRole(row, v)">
                  <el-option label="教师" value="TEACHER" /><el-option label="内部" value="INTERNAL" /><el-option label="游客" value="EXTERNAL" />
                </el-select>
              </td>
              <td>
                <span class="cell-label">级别</span>
                <div class="directory-level">
                  <span class="level-badge" :class="'tier-'+getTierClass(row.level)">{{ levelTitle(row.level) }}</span>
                  <el-input-number v-if="row.role !== 'SUPERADMIN'" :model-value="row.level ?? 0" :disabled="row._savingLevel" :min="0" :max="99" size="small"
                                   :aria-label="row.username + ' 的级别'" controls-position="right" @change="v => setLevel(row, v)" />
                </div>
              </td>
              <td class="directory-actions">
                <span v-if="row.role === 'SUPERADMIN'">账号受保护</span>
                <template v-else>
                  <el-button size="small" type="primary" text @click="openUserEdit(row)">编辑</el-button>
                  <el-button size="small" type="primary" text @click="openRecovery(row)">重置密码</el-button>
                  <el-button size="small" type="danger" text @click="removeUser(row)">删除</el-button>
                </template>
              </td>
            </tr>
          </tbody>
        </table>
        <el-empty v-if="!usersList.length && !usersLoading" :description="appliedUserSearch ? '没有符合条件的用户' : '暂无用户'" />
      </div>
      <div class="directory-footer">
        <span aria-live="polite">共 {{ totalUsers }} 位用户</span>
        <el-pagination
          :pager-count="5" layout="prev, pager, next"
          :total="totalUsers"
          :page-size="userPageSize"
          :current-page="userPage"
          :disabled="usersLoading"
          @update:current-page="changeUserPage"
          background
        />
      </div>
    </div>

    <el-dialog v-model="recoveryOpen" title="帮助用户找回密码" width="min(460px, 94vw)" :close-on-click-modal="false"
               :show-close="!recoveryBusy" :close-on-press-escape="!recoveryBusy" @closed="clearRecovery">
      <div v-if="recoveryUser" class="recovery-options">
        <p>登录用户名 <strong>{{ recoveryUser.username }}</strong></p>
        <template v-if="!recoveryCode">
          <p>{{ recoveryUser.email ? '绑定邮箱 · ' + recoveryUser.email : '该用户尚未绑定邮箱。' }}</p>
          <el-button v-if="recoveryUser.email" :disabled="recoveryBusy" @click="sendRecoveryEmail">向绑定邮箱发送验证码</el-button>
          <div class="recovery-handoff">
            <p>没有绑定邮箱或收不到邮件时，可核对学生身份后，单独把恢复码交给本人。</p>
            <el-checkbox v-model="identityVerified" :disabled="recoveryBusy">我已核实该用户身份</el-checkbox>
            <el-button type="primary" :disabled="!identityVerified || recoveryBusy" @click="issueRecoveryCode">生成一次性恢复码</el-button>
          </div>
        </template>
        <template v-else>
          <label for="admin-recovery-code">一次性恢复码</label>
          <input id="admin-recovery-code" class="recovery-code" :value="recoveryCode" readonly @focus="$event.target.select()">
          <p>恢复码可在 15 分钟内使用一次。请让用户在登录页选择「找回密码 → 管理员恢复码」，输入上方用户名和恢复码，设置新密码。</p>
        </template>
      </div>
      <template #footer><el-button :disabled="recoveryBusy" @click="recoveryOpen = false">关闭</el-button></template>
    </el-dialog>

    <el-dialog v-model="userDialogVisible" title="编辑用户资料" width="min(460px, 94vw)">
      <el-form label-position="top" @submit.prevent="saveUserProfile">
        <el-form-item label="用户名">
          <el-input v-model="userForm.username" maxlength="80" />
        </el-form-item>
        <el-form-item label="昵称">
          <el-input v-model="userForm.name" maxlength="100" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="userForm.email" maxlength="254" />
        </el-form-item>
        <el-button native-type="submit" type="primary" :loading="savingUser">保存</el-button>
        <el-button @click="userDialogVisible = false">取消</el-button>
      </el-form>
    </el-dialog>

    <el-dialog v-model="tutorialDialogVisible" :title="editingTutorialId ? '编辑教程' : '新增教程'" width="min(850px, 94vw)">
      <el-form label-position="top" @submit.prevent="saveTutorial">
        <el-form-item label="所属关卡">
          <el-select v-model="tutorialForm.levelId" filterable>
            <el-option v-for="level in skillsList" :key="level.id" :label="level.name" :value="level.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="标题">
          <el-input v-model="tutorialForm.title" />
        </el-form-item>
        <el-form-item label="内容（Markdown）">
          <el-input v-model="tutorialForm.content" type="textarea" :rows="10" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="tutorialForm.sortOrder" :min="0" :precision="0" />
        </el-form-item>
        <el-button native-type="submit" type="primary" :loading="savingTutorial">保存</el-button>
        <el-button @click="tutorialDialogVisible = false">取消</el-button>
      </el-form>
    </el-dialog>

    <!-- ======================== Submissions Tab ======================== -->
    <div class="admin-workspace" v-show="activeTab === 'submissions'">
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
            <span class="sub-status-tag" :class="row.status?.toLowerCase()">{{ submissionStatusText(row.status) }}</span>
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
    <el-dialog v-model="subDetailVisible" title="提交详情" width="min(800px, 94vw)" top="3vh">
      <div v-if="subDetail" class="sub-detail">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="提交 ID">{{ subDetail.id }}</el-descriptions-item>
          <el-descriptions-item label="用户">{{ subDetail.name ? subDetail.name + ' (' + subDetail.username + ')' : subDetail.username }}</el-descriptions-item>
          <el-descriptions-item label="任务">{{ subDetail.taskTitle }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <span class="sub-status-tag" :class="subDetail.status?.toLowerCase()">{{ submissionStatusText(subDetail.status) }}</span>
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

        <div v-if="awaitingReview(subDetail.status) && subDetail.userId !== authStore.user?.id" class="sub-section">
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
    <div class="admin-workspace" v-if="isSuperAdmin" v-show="activeTab === 'inviteCodes'">
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
    <div class="admin-workspace" v-if="isSuperAdmin" v-show="activeTab === 'mentorGraph'">
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
    <el-dialog v-model="skillDialogVisible" :title="editingSkill?.id ? '编辑关卡' : '新增关卡'" width="min(900px, 94vw)" top="3vh" @close="editingSkill = null">
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

      </el-form>
      <template #footer>
        <el-button @click="skillDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveSkill" :loading="saving">保存</el-button>
      </template>
    </el-dialog>

    <!-- ==================== Task Edit Dialog ==================== -->
    <el-dialog v-model="taskDialogVisible" :title="editingTask?.id ? '编辑任务' : '新增任务'" width="min(900px, 94vw)" top="3vh" @close="editingTask = null">
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
import { submissionStatusText, awaitingReview } from '../submissionStatus'
import { ref, onMounted, reactive, computed, watch, nextTick, onBeforeUnmount } from 'vue'
import { adminApi } from '../api/admin'
import { tasksApi } from '../api/tasks'
import { useAuthStore } from '../store/auth'
import { ElMessage, ElMessageBox } from 'element-plus'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const authStore = useAuthStore()
const isSuperAdmin = computed(() => authStore.isSuperAdmin)
const activeTab = ref(isSuperAdmin.value ? 'skills' : 'submissions')
const skillsList = ref([])
const tasksList = ref([])
const usersList = ref([])
const subsList = ref([])
const inviteCodesList = ref([])
const taskSkillFilter = ref(null)
const saving = ref(false)
const totalUsers = ref(0)
const userPageSize = 20
const userPage = ref(1)
const usersLoading = ref(false)
const userSearch = ref('')
const appliedUserSearch = ref('')
const userDialogVisible = ref(false)
const editingUserId = ref(null)
const userForm = reactive({ username: '', name: '', email: '' })
const savingUser = ref(false)
const recoveryOpen = ref(false)
const recoveryBusy = ref(false)
const recoveryUser = ref(null)
const identityVerified = ref(false)
const recoveryCode = ref('')
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
const tutorialLevelId = ref(null)
const skillTutorials = ref([])
const tutorialsLoading = ref(false)
const tutorialDialogVisible = ref(false)
const editingTutorialId = ref(null)
const tutorialForm = reactive({ levelId: null, title: '', content: '', sortOrder: 0 })
const savingTutorial = ref(false)

let tutorialRequest = 0
async function fetchSkillTutorials(levelId = tutorialLevelId.value) {
  const request = ++tutorialRequest
  tutorialsLoading.value = true
  try {
    const res = await adminApi.listTutorials(levelId)
    if (request === tutorialRequest) skillTutorials.value = res.data.data
  } catch (error) {
    if (request === tutorialRequest) ElMessage.error(error.response?.data?.message || '加载教程失败')
  } finally {
    if (request === tutorialRequest) tutorialsLoading.value = false
  }
}

function changeTutorialLevel(levelId) {
  skillTutorials.value = []
  fetchSkillTutorials(levelId)
}

function openTutorialDialog(row) {
  editingTutorialId.value = row?.id ?? null
  const nextOrder = Math.max(0, ...skillTutorials.value.map(t => t.sortOrder)) + 1
  Object.assign(tutorialForm, {
    levelId: tutorialLevelId.value,
    title: row?.title ?? '',
    content: row?.content ?? '',
    sortOrder: row?.sortOrder ?? nextOrder
  })
  tutorialDialogVisible.value = true
}

async function saveTutorial() {
  if (!tutorialForm.levelId || !tutorialForm.title.trim() || !Number.isInteger(tutorialForm.sortOrder)) {
    ElMessage.warning('请填写关卡、标题和排序')
    return
  }
  savingTutorial.value = true
  try {
    const data = { ...tutorialForm }
    if (editingTutorialId.value) {
      await adminApi.updateTutorial(editingTutorialId.value, data)
    } else {
      await adminApi.createTutorial(data)
    }
    tutorialDialogVisible.value = false
    ElMessage.success('教程已保存')
    if (data.levelId !== tutorialLevelId.value) {
      tutorialLevelId.value = data.levelId
      skillTutorials.value = []
    }
    await fetchSkillTutorials()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '保存教程失败')
  } finally {
    savingTutorial.value = false
  }
}

async function deleteTutorial(id) {
  try {
    await ElMessageBox.confirm('确定删除该教程？', '确认', { type: 'warning' })
  } catch {
    return
  }
  try {
    await adminApi.deleteTutorial(id)
    ElMessage.success('已删除')
    await fetchSkillTutorials()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '删除教程失败')
  }
}

// --- Data fetching ---
function roleText(r) { return { SUPERADMIN:'超级管理员',TEACHER:'教师',INTERNAL:'内部',EXTERNAL:'游客' }[r]||r }

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

async function setLevel(row, level) {
  if (!Number.isInteger(level)) return
  row._savingLevel = true
  try {
    await adminApi.setLevel(row.id, level)
    row.level = level
    ElMessage.success('级别已更新')
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '更新失败')
    await fetchUsers()
  } finally {
    row._savingLevel = false
  }
}

// --- Submission detail ---
const subDetailVisible = ref(false)
const subDetail = ref(null)
const reviewing = ref(false)
const aiReviewing = ref(false)

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
function changeUserPage(page) {
  if (page !== userPage.value) fetchUsers(page)
}

function searchUsers() {
  appliedUserSearch.value = userSearch.value.trim()
  fetchUsers(1)
}

let userListRequest = 0
async function fetchUsers(page = userPage.value) {
  const request = ++userListRequest
  usersLoading.value = true
  try {
    let data
    for (;;) {
      const res = await adminApi.listUsers(page - 1, userPageSize, appliedUserSearch.value)
      if (request !== userListRequest) return
      data = res.data.data
      const lastPage = Math.max(1, Math.ceil(data.total / userPageSize))
      if (page <= lastPage) break
      page = lastPage
    }
    userPage.value = page
    totalUsers.value = data.total
    usersList.value = data.content
  } catch (error) {
    if (request === userListRequest) {
      ElMessage.error(error.response?.data?.message || '加载用户列表失败')
    }
  } finally {
    if (request === userListRequest) usersLoading.value = false
  }
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
    await fetchUsers()
  } catch (e) { if (e !== 'cancel' && e !== 'close') ElMessage.error('删除失败') }
}
function openUserEdit(row) {
  editingUserId.value = row.id
  Object.assign(userForm, { username: row.username, name: row.name || '', email: row.email || '' })
  userDialogVisible.value = true
}
async function saveUserProfile() {
  if (!userForm.username.trim()) {
    ElMessage.warning('请输入用户名')
    return
  }
  savingUser.value = true
  try {
    await adminApi.updateUser(editingUserId.value, { ...userForm })
    userDialogVisible.value = false
    ElMessage.success('用户资料已更新')
    await fetchUsers()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '保存用户资料失败')
  } finally {
    savingUser.value = false
  }
}
function clearRecovery() {
  recoveryCode.value = ''
  identityVerified.value = false
  recoveryUser.value = null
}
function openRecovery(row) {
  clearRecovery()
  recoveryUser.value = row
  recoveryOpen.value = true
}
async function sendRecoveryEmail() {
  if (recoveryBusy.value || !recoveryUser.value?.email) return
  recoveryBusy.value = true
  try {
    await adminApi.sendPasswordReset(recoveryUser.value.id)
    ElMessage.success('邮件服务已接收，请用户检查收件箱和垃圾邮件')
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '发送失败，请重试')
  } finally { recoveryBusy.value = false }
}
async function issueRecoveryCode() {
  if (recoveryBusy.value || !identityVerified.value) return
  recoveryBusy.value = true
  try {
    const res = await adminApi.issueRecoveryCode(recoveryUser.value.id)
    recoveryCode.value = res.data.data.recoveryCode
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '恢复码生成失败，请重试')
  } finally { recoveryBusy.value = false }
}
async function updateRole(row, role) {
  row._savingRole = true
  try {
    await adminApi.updateRole(row.id, role)
    row.role = role
    ElMessage.success('角色已更新')
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '更新角色失败')
  } finally {
    row._savingRole = false
  }
}
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
let echarts
let rawNodes = []
let rawEdges = []
const roleColor = { SUPERADMIN: '#ad792c', TEACHER: '#236451', INTERNAL: '#5983a1', EXTERNAL: '#7a8279' }
const roleLegends = Object.entries(roleColor).map(([role, color]) => ({ label: roleText(role), color }))
const graphEmpty = ref(false)

async function fetchMentorGraph() {
  if (graphLoading.value) return
  graphLoading.value = true
  try {
    const [res, charts] = await Promise.all([adminApi.getMentorshipGraph(), import('echarts')])
    echarts = charts
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
  if (!graphRef.value || !echarts || activeTab.value !== 'mentorGraph') return
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
  if (tab === 'mentorGraph') {
    if (!chart) fetchMentorGraph()
    else nextTick(resizeChart)
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', resizeChart)
  if (chart) { chart.dispose(); chart = null }
})

onMounted(async () => {
  const requests = [fetchSubs()]
  if (isSuperAdmin.value) requests.push(fetchSkills(), fetchUsers(), fetchInviteCodes())
  await Promise.all(requests)
})
</script>

<style scoped>
.recovery-options, .recovery-handoff { display: grid; justify-items: start; gap: 12px; line-height: 1.6; }
.recovery-handoff { border-top: 1px solid var(--color-border); padding-top: 16px; margin-top: 4px; }
.recovery-code { max-width: 100%; width: 100%; padding: 10px; color: var(--color-text); background: var(--el-fill-color-light); border: 1px solid var(--color-border); border-radius: 6px; font: 600 17px/1.4 monospace; }

.admin-page { max-width: 1400px; margin-inline: auto; container: staff / inline-size; }
.admin-header { display: flex; flex-direction: column; align-items: flex-start; gap: 7px; margin-bottom: 22px; }
.staff-identity { display: flex; align-items: center; gap: 7px; color: var(--color-text-secondary); font-size: 12px; font-weight: 600; }
.staff-identity::before { content: ''; width: 15px; height: 3px; background: var(--color-accent); border-radius: 2px; }
.admin-title { font-size: clamp(26px, 3vw, 36px); line-height: 1.2; letter-spacing: -.04em; }
.admin-tabs { margin-bottom: 14px; padding: 0 14px; border: 1px solid var(--color-border); border-radius: 10px; background: #edefe5; }
.admin-tabs :deep(.el-tabs__header) { margin: 0; }
.admin-tabs :deep(.el-tabs__nav-wrap::after) { height: 0; }
.admin-tabs :deep(.el-tabs__item) { height: 45px; }
.admin-tabs :deep(.el-tabs__content) { display: none; }
.admin-workspace { min-width: 0; padding: clamp(12px, 1.6vw, 24px); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius); box-shadow: var(--shadow-sm); }
.directory { overflow: hidden; border: 1px solid var(--color-border); border-radius: 9px; }
.directory-table { width: 100%; border-collapse: collapse; font-size: 13px; text-align: left; }
.directory-table th { padding: 10px 12px; background: #eff3e9; color: #485f50; font-weight: 600; }
.directory-table td { padding: 10px 12px; border-top: 1px solid var(--color-border); }
.directory-table tr:nth-child(even) td { background: #f8f9f3; }
.directory-table .el-select { width: 87px; }
.directory-table .el-input-number { width: 78px; }
.directory-identity :is(strong, small, span) { display: block; line-height: 1.55; overflow-wrap: anywhere; }
.directory-identity small, .directory-identity span, .directory-actions > span, .directory-footer { color: var(--color-text-secondary); }
.directory-level { display: flex; align-items: center; flex-wrap: wrap; gap: 5px; }
.directory-actions { white-space: nowrap; }
.directory-actions .el-button { margin: 0; min-height: 30px; padding-inline: 7px; }
.directory-footer { margin-top: 18px; display: flex; justify-content: space-between; align-items: center; gap: 5px 12px; flex-wrap: wrap; font-size: 13px; }
.cell-label { display: none; }
@container staff (max-width: 840px) {
  .directory { border: 0; border-radius: 0; }
  .directory-table, .directory-table tbody { display: block; }
  .directory-table thead { position: absolute; clip-path: inset(50%); width: 1px; height: 1px; overflow: hidden; }
  .directory-table tbody { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 250px), 1fr)); gap: 10px; }
  .directory-table tbody tr { display: grid; grid-template-columns: minmax(75px, .62fr) minmax(120px, 1fr); place-content: start; gap: 8px; border: 1px solid var(--color-border); border-radius: 10px; padding: 12px; }
  .directory-table td, .directory-table tr:nth-child(even) td { display: block; padding: 0; background: transparent; border: 0; }
  .directory-identity, .directory-actions { grid-column: 1 / -1; }
  .cell-label { display: block; font-size: 11px; margin: 3px 0; color: var(--color-text-secondary); }
  .directory-actions .el-button { min-height: 40px; padding-inline: 10px; }
  .admin-workspace :deep(.el-table-fixed-column--right) { position: static !important; }
  .admin-workspace :deep(.el-table-fixed-column--right::before) { display: none; }
}
@media (max-width: 480px) {
  .directory-footer { justify-content: center; }
  .admin-workspace :deep(.el-pagination) { --el-pagination-button-width: 26px; }
  .admin-workspace :deep(.el-pagination.is-background li) { margin-inline: 2px; }
}
.manage-toolbar { display: flex; flex-wrap: wrap; align-items: center; gap: 12px; margin-bottom: 16px; }
.manage-toolbar :is(.el-select, .el-input) { width: min(320px, 100%); }

.lv-badge { font-size: 11px; font-weight: 700; padding: 2px 8px; border-radius: 4px; white-space: nowrap; }
.lv-badge.lv-1, .lv-badge.lv-2 { background: #f1f5f9; color: #64748b; }
.lv-badge.lv-3 { background: #fef3c7; color: #915709; }
.lv-badge.lv-4, .lv-badge.lv-5 { background: #fee2e2; color: #dc2626; }
.role-tag { font-size: 12px; font-weight: 500; padding: 2px 8px; border-radius: 4px; white-space: nowrap; }
.role-tag.superadmin { background: #fef3c7; color: #92400e; }
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
.sub-status-tag.approved { background: #dcfce7; color: #19713e; }
.sub-status-tag.rejected { background: #fee2e2; color: #dc2626; }
.sub-status-tag.pending { background: #fef3c7; color: #915709; }
.sub-status-tag.ai_reviewed { background: #e0f2fe; color: #0b6391; }

/* --- Tutorial manager --- */

/* Level badges */
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
