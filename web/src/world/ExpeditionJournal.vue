<template>
  <section class="album" aria-labelledby="album-title">
    <header>
      <div>
        <h2 id="album-title"><el-icon><Collection /></el-icon> 我的成果册</h2>
        <p>已探访 {{ visited.size }} / {{ chapters.length }} 座岛。点击印章可定位章节，完成全章后点亮。</p>
      </div>
      <button type="button" class="ledger-link" @click="open = true">
        任务成果 <b>{{ achievements.length }}</b><el-icon><Right /></el-icon>
      </button>
    </header>
    <ol class="stamps" aria-label="章节印章，点击可定位章节">
      <li v-for="(chapter, index) in chapters" :key="chapter.id">
        <button type="button" class="stamp" :class="{ earned: chapter.status === 'COMPLETED', started: chapter.levelsCompleted > 0 }"
                :aria-label="chapter.name + '，' + (chapter.status === 'COMPLETED' ? '已获得印章' : '已通关 ' + chapter.levelsCompleted + ' 关') + '，在地图查看'"
                @click="emit('locate', chapter.id)">
          <span class="imprint" aria-hidden="true">
            <el-icon><component :is="motifs[index % motifs.length]" /></el-icon>
            <i class="stamp-check" v-if="chapter.status === 'COMPLETED'">✓</i>
          </span>
          <strong>{{ chapter.name }}</strong>
          <small>{{ chapter.status === 'COMPLETED' ? '已获得' : chapter.levelsCompleted + ' / ' + chapter.totalLevels + ' 关' }}</small>
        </button>
      </li>
    </ol>

    <el-dialog v-model="open" title="我的任务成果" width="min(94vw, 600px)">
      <ul v-if="achievements.length" class="entries">
        <li v-for="entry in achievements" :key="entry.taskId">
          <RouterLink :to="{ path: '/level/' + entry.levelId, query: { task: entry.taskId } }">
            <span class="entry-icon"><el-icon><DocumentChecked /></el-icon></span>
            <span><strong>{{ entry.title }}</strong><small>{{ levelNames.get(entry.levelId) }} · 已通过{{ entry.reviewedAt ? ' · ' + date.format(entry.reviewedAt) : '' }}</small></span>
            <el-icon><Right /></el-icon>
          </RouterLink>
        </li>
      </ul>
      <div v-else class="empty-album">
        <el-icon><Notebook /></el-icon>
        <strong>第一份成果，从一次提交开始</strong>
        <p>提交后可以继续做其他可挑战任务。审核通过时，这里会收好你的任务和交付记录。</p>
        <button type="button" class="workshop-button" @click="open = false; emit('mission')">查看推荐任务</button>
      </div>
    </el-dialog>
  </section>
</template>

<script setup>
import { computed, ref } from 'vue'
import { Collection, Compass, Cpu, DocumentChecked, Flag, Key, MagicStick, MapLocation, Notebook, Promotion, Right, Sunrise } from '@element-plus/icons-vue'
const props = defineProps({ chapters: { type: Array, required: true }, achievements: { type: Array, required: true }, visited: { type: Set, required: true } })
const emit = defineEmits(['locate', 'mission'])
const open = ref(false)
const motifs = [Sunrise, Compass, Cpu, Key, MagicStick, MapLocation, Promotion, Flag]
const date = new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'numeric', day: 'numeric' })
const levelNames = computed(() => new Map(props.chapters.flatMap(chapter => chapter.levels.map(level => [level.id, level.name]))))
</script>

<style scoped>
.album { padding: 20px; border: 1px solid #dbdfcf; border-radius: 17px; background: #fffdf6; }
.album header { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 6px 16px; margin-bottom: 15px; }
.album h2 { display:flex; align-items:center; gap:8px; font-size: 20px; color:#244d3c; }
.album header p,.empty-album p { font-size:13px; color:#536657; line-height:1.65; margin-top:4px; }
.ledger-link { display:inline-flex; align-items:center; gap:6px; font-size:13px; color:#315e47; border:1px solid #d3ddc9; background:#f2f5e9; padding:8px 11px; border-radius:8px; }
.stamps { display:grid; list-style:none; grid-template-columns:repeat(8,minmax(0,1fr)); gap:7px; }
.stamp { width:100%; height:100%; display:grid; justify-items:center; align-content:start; gap:5px; padding:12px 3px 9px; background:#f6f4eb; border:1px dashed #d7d6c5; color:#4d5e51; border-radius:10px; }
.stamp:hover { background:#edf0df; border-color:#879d7f; }
.imprint { display:grid;place-items:center;position:relative;width:37px;height:37px;border-radius:50%;border:3px double #aeb5a0;color:#78856d; }
.imprint .el-icon {font-size:17px;}
.imprint .stamp-check { position:absolute; bottom:-4px;right:-4px;background:#f1d479;border:1px solid #967d37;border-radius:50%;font:700 10px/14px system-ui;width:15px;color:#354b36; }
.stamp.earned { background:#f2eed7;border:1px solid #baab6d;color:#3a513a; }
.earned .imprint { background:#315e45;border-color:#dac886;color:#ffedb4; }
.started:not(.earned) .imprint { color:#365d41;border-color:#829b77; }
.stamp strong { font-size:12px; line-height:1.4; overflow-wrap:anywhere; }
.stamp small { font-size:11px;color:#596957; }
.entries { display:grid;gap:5px;list-style:none;max-height:60dvh;overflow:auto; }
.entries a { display:flex;align-items:center;gap:10px;color:#294e3e;text-decoration:none;padding:10px 8px;border-bottom:1px solid #e7e9dc; }
.entries a:hover { background:#f4f6e9; }
.entries a > .el-icon:last-child { margin-left:auto;flex:none; }
.entries strong,.entries small { display:block;overflow-wrap:anywhere; }
.entries small { margin-top:3px;color:#5e6f61;font-size:12px; }
.entry-icon { width:32px;height:32px;display:grid;place-items:center;background:#e8efdb;border-radius:8px;flex:none; }
.empty-album { display:grid;justify-items:start;gap:7px; }
.empty-album > .el-icon { font-size:26px;color:#517b55; }
.empty-album .workshop-button { margin-top:10px; }
@media(max-width:1180px) { .stamps { grid-template-columns:repeat(4, minmax(0,1fr)); } }
@media(max-width:550px) { .album { padding:14px; } .stamps { gap:5px; } .stamp { padding:8px 3px; } .stamp strong {font-size:11px;} }
</style>
