<template>
  <div class="page">
    <h1 class="page-title">排行榜</h1>
    <p class="page-subtitle">按累计经验排名 · {{ total }} 位成员</p>

    <div class="section-card" v-loading="loading">
      <div class="section-body" style="padding:0">
        <div class="podium" v-if="leaders.length">
          <article v-for="(user, i) in leaders" :key="user.userId" class="podium-item" :class="{ winner: i === 0 }">
            <span class="podium-place">第 {{ i + 1 }} 名</span>
            <el-avatar :size="44" class="podium-avatar">{{ (user.name || user.username)?.charAt(0) }}</el-avatar>
            <div class="podium-name">{{ user.name || user.username }}</div>
            <span class="podium-level">Lv.{{ user.level }} · {{ levelShort(user.level) }}</span>
            <strong class="podium-xp">{{ user.totalXp }} XP</strong>
            <el-button v-if="canRequestMentor(user)" size="small" round class="podium-btn" @click="openMentorDialog(user)">申请拜师</el-button>
          </article>
        </div>

        <table v-if="rankRows.length" class="rank-table" aria-label="成员排名">
          <thead><tr>
            <th scope="col">排名</th>
            <th scope="col">成员</th>
            <th scope="col">等级</th>
            <th scope="col" class="numeric">累计经验</th>
            <th scope="col" class="numeric">距榜首</th>
            <th v-if="showActions" scope="col">操作</th>
          </tr></thead>
          <tbody>
            <tr v-for="row in rankRows" :key="row.userId">
              <td class="rank-num"><span class="mobile-prefix">第 </span>{{ row.rank }}<span class="mobile-prefix"> 名</span></td>
              <td class="member-name">{{ row.name || row.username }}</td>
              <td class="member-level"><span class="level-text">Lv.{{ row.level }}</span><span class="level-title">{{ levelShort(row.level) }}</span></td>
              <td class="numeric xp-text">{{ row.totalXp }} XP</td>
              <td class="numeric xp-gap"><span v-if="row.totalXp < leaderXp"><span class="mobile-prefix">距榜首 </span>{{ leaderXp - row.totalXp }} XP</span><span v-else>与榜首持平</span></td>
              <td v-if="showActions" class="rank-action">
                <el-button v-if="canRequestMentor(row)" size="small" type="primary" text @click="openMentorDialog(row)">拜师</el-button>
              </td>
            </tr>
          </tbody>
        </table>
        <el-empty v-else-if="!loading && !leaders.length" description="暂无排行数据" :image-size="56" />
      </div>
    </div>

    <div v-if="total > 20" class="rank-pages">
      <el-pagination v-model:current-page="page" :page-size="20"
                     layout="prev, pager, next" :total="total" :pager-count="5"
                     @current-change="fetchData" background />
    </div>

    <!-- 拜师对话框 -->
    <el-dialog v-model="mentorDialogVisible" title="申请拜师" width="min(400px, calc(100% - 24px))" :close-on-click-modal="false">
      <div v-if="selectedUser" class="mentor-dialog-content">
        <div class="mentor-info-card">
          <el-avatar :size="60" class="mentor-avatar">{{ (selectedUser.name || selectedUser.username)?.charAt(0)?.toUpperCase() }}</el-avatar>
          <div class="mentor-detail">
            <div class="mentor-name">{{ selectedUser.name || selectedUser.username }}</div>
            <div class="mentor-level">Lv.{{ selectedUser.level }} {{ levelShort(selectedUser.level) }}</div>
            <div class="mentor-xp">{{ selectedUser.totalXp }} XP</div>
          </div>
        </div>
        <div class="mentor-tip">
          <el-icon><InfoFilled /></el-icon>
          <span>发送拜师申请后，等待对方通过即可成为师徒关系。</span>
        </div>
      </div>
      <template #footer>
        <el-button @click="mentorDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="sendMentorRequest" :loading="requestLoading">
          发送申请
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { leaderboardApi } from '../api/leaderboard'
import { mentorshipApi } from '../api/mentorship'
import { useAuthStore } from '../store/auth'
import { ElMessage } from 'element-plus'
import { InfoFilled } from '@element-plus/icons-vue'

const authStore = useAuthStore()
const list = ref([])
const loading = ref(false)
const page = ref(1)
const total = ref(0)
const leaderXp = ref(0)
// 当前用户已有的徒弟/师傅/申请关系（判断能否拜师）
const myApprenticeIds = ref([])
const myMentorIds = ref([])
const myPendingTargetIds = ref([])
const mySentTargetIds = ref([])

// 拜师对话框
const mentorDialogVisible = ref(false)
const selectedUser = ref(null)
const requestLoading = ref(false)

const leaders = computed(() => page.value === 1 ? list.value.slice(0, 3) : [])
const rankRows = computed(() => list.value.slice(leaders.value.length))
const showActions = computed(() => rankRows.value.some(canRequestMentor))

function levelShort(lv) {
  if (lv >= 70) return '👑王者'
  if (lv >= 60) return '🏆宗师'
  if (lv >= 50) return '⭐大师'
  if (lv >= 40) return '💎钻石'
  if (lv >= 30) return '🔮铂金'
  if (lv >= 20) return '🥇黄金'
  if (lv >= 10) return '🥈白银'
  return '🥉青铜'
}

let latestPageRequest = 0
async function fetchData() {
  const request = ++latestPageRequest
  loading.value = true
  try {
    const res = await leaderboardApi.global(page.value - 1, 20)
    if (request !== latestPageRequest) return
    const data = res.data.data
    leaderXp.value = data.leaderXp
    list.value = data.content || []
    total.value = data.total || 0
  } catch {
    if (request === latestPageRequest) {
      list.value = []
      ElMessage.error('排行榜加载失败，请稍后重试')
    }
  } finally {
    if (request === latestPageRequest) loading.value = false
  }
}

/** 拉取当前用户的师徒关系，用于判断"能否拜某人为师" */
async function fetchRelations() {
  try {
    const res = await mentorshipApi.my()
    const d = res.data.data
    myApprenticeIds.value = (d?.apprentices || []).map(a => a.userId)
    myMentorIds.value = (d?.mentors || []).map(m => m.userId)
    myPendingTargetIds.value = (d?.pending || []).map(p => p.userId)
    mySentTargetIds.value = (d?.sentRequests || []).map(s => s.userId)
  } catch (e) { console.error(e) }
}

/** 判断是否可以为 user 显示拜师按钮：本人 / 自己的徒弟 / 已有关系 / 外部成员 都不显示 */
function canRequestMentor(user) {
  const me = authStore.user
  if (!me) return false
  if (me.role === 'EXTERNAL') return false
  if (me.role !== 'INTERNAL' && me.role !== 'TEACHER' && me.role !== 'SUPERADMIN') return false
  const uid = user.userId ?? user.id
  if (uid === me.id) return false
  if (myApprenticeIds.value.includes(uid)) return false
  if (myMentorIds.value.includes(uid)) return false
  if (myPendingTargetIds.value.includes(uid)) return false
  if (mySentTargetIds.value.includes(uid)) return false
  return true
}

function openMentorDialog(user) {
  if (!canRequestMentor(user)) return
  selectedUser.value = user
  mentorDialogVisible.value = true
}

async function sendMentorRequest() {
  if (!selectedUser.value) return
  requestLoading.value = true
  try {
    await mentorshipApi.request(selectedUser.value.userId)
    ElMessage.success('拜师申请已发送，等待对方通过')
    mentorDialogVisible.value = false
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '申请失败')
  } finally { requestLoading.value = false }
}

onMounted(() => {
  fetchData()
  fetchRelations()
})
</script>

<style scoped>
.page { max-width: 900px; }

.podium { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 12px; padding: clamp(12px,3vw,24px); }
.podium-item { position: relative; min-width: 0; display: flex; flex-direction: column; align-items: flex-start; gap: 5px; padding: clamp(13px,2vw,22px); border: 1px solid var(--color-border); border-radius: 14px; background: #f6f7ef; }
.podium-item.winner { background: #234b3f; color: #fffaf0; border-color: #234b3f; }
.podium-place { font-size: 12px; margin-bottom: 10px; font-weight: 700; }
.podium-avatar { background: #dce7db; color: #315a44; font-weight: 700; }
.winner .podium-avatar { background: #edce8c; color: #4e411f; }
.podium-name { margin-top: 5px; word-break: break-word; font-size: 16px; font-weight: 650; }
.podium-level { font-size: 12px; color: #5d6a5e; }
.winner .podium-level { color: #c9dacd; }
.podium-xp { font-size: 18px; color: #755118; }
.winner .podium-xp { color: #f4d89e; }
.podium-btn { margin-top: 5px; }
@media(max-width: 600px) {
  .podium { display: flex; flex-direction: column; gap: 6px; }
  .podium-item { display: grid; grid-template-columns: auto minmax(0,1fr) auto; align-items: center; column-gap: 12px; }
  .podium-place { grid-row: 1; grid-column: 1; margin: 0; }
  .podium-avatar { display: none; }
  .podium-name { grid-row: 1; grid-column: 2; margin: 0; }
  .podium-level { grid-column: 2; }
  .podium-xp { grid-row: 1; grid-column: 3; font-size: 14px; }
  .podium-btn { grid-column: 3; grid-row: 2; margin: 0; }
}
.rank-num { font-weight: 600; color: #667568; }
.level-text { font-weight: 600; color: #216354; margin-right: 4px; }
.level-title { font-size: 12px; color: #667568; }
.xp-text { font-weight: 600; color: #895b19; }
.rank-table { width: 100%; border-collapse: collapse; font-size: 14px; text-align: left; }
.rank-table td, .rank-table th { padding: 12px; }
.rank-table th { font-weight: 600; color: var(--color-text-secondary); background: var(--el-fill-color-light); }
.rank-table tbody tr + tr { border-top: 1px solid var(--color-border); }
.rank-table td { overflow-wrap: anywhere; }
.numeric { text-align: right; font-variant-numeric: tabular-nums; }
.xp-gap { color: var(--color-text-secondary); }
.rank-table .numeric, .rank-num { white-space: nowrap; }
.rank-pages { display: flex; justify-content: center; margin-top: 20px; }
.mobile-prefix { display: none; }
.section-card { overflow: hidden; container-type: inline-size; }
@container (max-width: 620px) {
  .rank-table, .rank-table tbody { display: block; }
  .rank-table thead { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
  .rank-table tr { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 3px 10px; align-items: baseline; padding: 12px 16px; }
  .rank-table td { padding: 0; }
  .member-name { grid-area: 1 / 1; font-weight: 600; }
  .xp-text { grid-area: 1 / 2; }
  .rank-num { grid-area: 2 / 1; font-size: 12px; }
  .xp-gap { grid-area: 2 / 2; font-size: 12px; }
  .member-level { grid-area: 3 / 1; font-size: 12px; }
  .rank-action { grid-area: 3 / 2; text-align: right; }
  .mobile-prefix { display: inline; }
}

/* 拜师对话框样式 */
.mentor-dialog-content { display: flex; flex-direction: column; gap: 16px; }
.mentor-info-card {
  display: flex; align-items: center; gap: 16px;
  padding: 16px; background: #f5f6f0; border-radius: 12px;
}
.mentor-avatar {
  background: linear-gradient(135deg, #216354, #397c67);
  color: #fff; font-weight: 700; font-size: 20px;
}
.mentor-detail { flex: 1; }
.mentor-name { font-size: 16px; font-weight: 600; color: var(--color-text); }
.mentor-level { font-size: 14px; color: #216354; font-weight: 600; margin-top: 4px; }
.mentor-xp { font-size: 13px; color: #895b19; margin-top: 2px; }
.mentor-tip {
  display: flex; align-items: center; gap: 8px;
  font-size: 13px; color: #667568;
  padding: 12px; background: #edf5ee; border-radius: 8px;
}
</style>
