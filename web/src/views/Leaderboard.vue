<template>
  <div class="page">
    <h1 class="page-title">排行榜</h1>
    <p class="page-subtitle">看看谁是最强的技能掌握者</p>

    <div class="section-card">
      <div class="section-body" style="padding:0">
        <!-- Top 3 podium (page 1 only) -->
        <div class="podium" v-if="page === 1 && list.length >= 3">
          <div class="podium-item second" v-if="list[1]">
            <el-avatar :size="44" class="podium-avatar silver">{{ list[1].username?.charAt(0)?.toUpperCase() }}</el-avatar>
            <div class="podium-name">{{ list[1].name || list[1].username }}</div>
            <div class="podium-level">Lv.{{ list[1].level }} {{ levelShort(list[1].level) }}</div>
            <div class="podium-xp">{{ list[1].totalXp }} XP</div>
            <el-button v-if="canRequestMentor(list[1])" size="small" type="primary" round class="podium-btn" @click="openMentorDialog(list[1])">
              🙋 拜师
            </el-button>
            <div class="podium-rank silver">2</div>
          </div>
          <div class="podium-item first" v-if="list[0]">
            <div class="crown">👑</div>
            <el-avatar :size="52" class="podium-avatar gold">{{ list[0].username?.charAt(0)?.toUpperCase() }}</el-avatar>
            <div class="podium-name">{{ list[0].name || list[0].username }}</div>
            <div class="podium-level">Lv.{{ list[0].level }} {{ levelShort(list[0].level) }}</div>
            <div class="podium-xp">{{ list[0].totalXp }} XP</div>
            <el-button v-if="canRequestMentor(list[0])" size="small" type="primary" round class="podium-btn" @click="openMentorDialog(list[0])">
              🙋 拜师
            </el-button>
            <div class="podium-rank gold">1</div>
          </div>
          <div class="podium-item third" v-if="list[2]">
            <el-avatar :size="40" class="podium-avatar bronze">{{ list[2].username?.charAt(0)?.toUpperCase() }}</el-avatar>
            <div class="podium-name">{{ list[2].name || list[2].username }}</div>
            <div class="podium-level">Lv.{{ list[2].level }} {{ levelShort(list[2].level) }}</div>
            <div class="podium-xp">{{ list[2].totalXp }} XP</div>
            <el-button v-if="canRequestMentor(list[2])" size="small" type="primary" round class="podium-btn" @click="openMentorDialog(list[2])">
              🙋 拜师
            </el-button>
            <div class="podium-rank bronze">3</div>
          </div>
        </div>

        <el-table :data="page === 1 ? list.slice(3) : list" stripe style="width:100%" empty-text="暂无排行数据">
          <el-table-column label="排名" width="80">
            <template #default="{ row }">
              <span class="rank-num">{{ row.rank }}</span>
            </template>
          </el-table-column>
          <el-table-column label="用户">
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
          <el-table-column label="经验值" sortable width="120">
            <template #default="{ row }">
              <span class="xp-text">{{ row.totalXp }} XP</span>
            </template>
          </el-table-column>
          <el-table-column label="进度" min-width="150">
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
    <el-dialog v-model="mentorDialogVisible" title="申请拜师" width="400px" :close-on-click-modal="false">
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
  if (me.role !== 'INTERNAL' && me.role !== 'TEACHER') return false
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

.podium {
  display: flex; justify-content: center; align-items: flex-end;
  gap: 16px; padding: 40px 24px 32px;
  background: linear-gradient(180deg, #f8fafc 0%, #fff 100%);
}
.podium-item {
  display: flex; flex-direction: column; align-items: center;
  padding: 20px 28px; border-radius: 16px;
  background: #fff;
  box-shadow: var(--shadow-sm);
  text-align: center;
  transition: transform .2s;
}
.podium-item:hover { transform: translateY(-4px); }
.podium-item.first { order: 1; padding: 28px 36px; box-shadow: var(--shadow-md); }
.podium-item.second { order: 0; }
.podium-item.third { order: 2; }

.crown { font-size: 24px; margin-bottom: -8px; }
.podium-avatar { font-weight: 700; }
.podium-avatar.gold { background: linear-gradient(135deg, #fbbf24, #f59e0b); color: #fff; }
.podium-avatar.silver { background: linear-gradient(135deg, #94a3b8, #64748b); color: #fff; }
.podium-avatar.bronze { background: linear-gradient(135deg, #f97316, #ea580c); color: #fff; }

.podium-name { font-size: 14px; font-weight: 600; margin-top: 8px; color: var(--color-text); }
.podium-level { font-size: 12px; color: #6366f1; font-weight: 600; margin-top: 4px; }
.podium-xp { font-size: 12px; color: #d97706; font-weight: 600; margin-top: 2px; }
.podium-btn { margin-top: 8px; }
.podium-rank {
  width: 28px; height: 28px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  font-size: 13px; font-weight: 700; color: #fff; margin-top: 8px;
}
.podium-rank.gold { background: #f59e0b; }
.podium-rank.silver { background: #64748b; }
.podium-rank.bronze { background: #ea580c; }

.rank-num { font-weight: 600; color: #94a3b8; }
.level-text { font-weight: 600; color: #6366f1; margin-right: 4px; }
.level-title { font-size: 12px; color: #94a3b8; }
.xp-text { font-weight: 600; color: #d97706; }
.xp-bar-wrap { height: 6px; background: #f1f5f9; border-radius: 3px; overflow: hidden; }
.xp-bar { height: 100%; background: linear-gradient(90deg, #6366f1, #8b5cf6); border-radius: 3px; transition: width .5s; }

/* 拜师对话框样式 */
.mentor-dialog-content { display: flex; flex-direction: column; gap: 16px; }
.mentor-info-card {
  display: flex; align-items: center; gap: 16px;
  padding: 16px; background: #f8fafc; border-radius: 12px;
}
.mentor-avatar {
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: #fff; font-weight: 700; font-size: 20px;
}
.mentor-detail { flex: 1; }
.mentor-name { font-size: 16px; font-weight: 600; color: var(--color-text); }
.mentor-level { font-size: 14px; color: #6366f1; font-weight: 600; margin-top: 4px; }
.mentor-xp { font-size: 13px; color: #d97706; margin-top: 2px; }
.mentor-tip {
  display: flex; align-items: center; gap: 8px;
  font-size: 13px; color: #94a3b8;
  padding: 12px; background: #eef2ff; border-radius: 8px;
}
</style>
