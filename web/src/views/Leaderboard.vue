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

        <el-table v-if="rankRows.length || !leaders.length" :data="rankRows" stripe style="width:100%" empty-text="暂无排行数据">
          <el-table-column label="排名" width="80">
            <template #default="{ row }">
              <span class="rank-num">{{ row.rank }}</span>
            </template>
          </el-table-column>
          <el-table-column label="用户" min-width="116">
            <template #default="{ row }">
              {{ row.name || row.username }}
            </template>
          </el-table-column>
          <el-table-column label="等级" width="120">
            <template #default="{ row }">
              <span class="level-text">Lv.{{ row.level }}</span>
              <span class="level-title">{{ levelShort(row.level) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="经验值" width="120">
            <template #default="{ row }">
              <span class="xp-text">{{ row.totalXp }} XP</span>
            </template>
          </el-table-column>
          <el-table-column label="本页经验对比" min-width="150">
            <template #default="{ row }">
              <div class="xp-bar-wrap">
                <div class="xp-bar" :style="{ width: Math.min(row.totalXp / maxXp * 100, 100) + '%' }"></div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button v-if="canRequestMentor(row)" size="small" type="primary" text @click="openMentorDialog(row)">
                拜师
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <div style="display:flex;justify-content:center;margin-top:20px">
      <el-pagination v-model:current-page="page" :page-size="20"
                     layout="prev, pager, next" :total="total"
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
const maxXp = computed(() => Math.max(...list.value.map(l => l.totalXp), 1))

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

async function fetchData() {
  loading.value = true
  try {
    const res = await leaderboardApi.global(page.value - 1, 20)
    const data = res.data.data
    list.value = data.content || []
    total.value = data.total || 0
  } catch (e) { console.error(e) } finally { loading.value = false }
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
.xp-bar-wrap { height: 6px; background: #edf0e9; border-radius: 3px; overflow: hidden; }
.xp-bar { height: 100%; background: linear-gradient(90deg, #216354, #397c67); border-radius: 3px; transition: width .5s; }

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
