<template>
  <div class="scene-frame">
    <div ref="viewport" class="viewport" aria-label="关卡地图，方向键沿路行走，靠近据点按 E 进入" tabindex="0" @keydown="onKey" @keyup="rest">
      <div ref="board" class="board" :class="{ fitted: overview }">
        <canvas ref="terrain" class="terrain" aria-hidden="true" @click="walkOnRoad" />
        <span class="compass" aria-hidden="true"><b>N</b><i></i></span>
        <button class="guide-marker" type="button" :style="position(177, 251)" title="拜访驿站领取推荐委托" aria-label="拜访驿站领取推荐委托" @click="visitGuide">
          <span>!</span>
        </button>
        <button v-for="stop in stops" :key="stop.id" type="button" class="outpost"
                :class="[stop.status.toLowerCase(), { selected: stop.id === selectedId }]"
                :style="position(stop.x, stop.y)"
                :aria-label="`第 ${stop.number} 关 ${stop.name}，${STATUS[stop.status]}`"
                :aria-pressed="stop.id === selectedId" @click="choose(stop)">
          <span class="pin"><el-icon v-if="stop.status === 'LOCKED'"><Lock /></el-icon>
            <el-icon v-else-if="stop.status === 'COMPLETED'"><Check /></el-icon><strong v-else>{{ stop.number }}</strong></span>
          <span class="caption">{{ stop.name }}</span>
        </button>
        <div v-if="actor" class="actor" :style="position(actor.x - 11, actor.y - 2)" aria-hidden="true">
          <span class="actor-shadow"></span><span class="sprite" :style="spriteStyle"></span>
        </div>
      </div>
    </div>
    <div class="map-tools">
      <button type="button" title="定位已选关卡" aria-label="定位已选关卡" @click="center(selectedStop)"><el-icon><Aim /></el-icon></button>
      <button class="zoom" type="button" :title="overview ? '放大地图' : '全景地图'" :aria-label="overview ? '放大地图' : '全景地图'" :aria-pressed="overview" @click="toggleOverview"><el-icon><component :is="overview ? ZoomIn : ZoomOut" /></el-icon></button>
    </div>
    <span v-if="hint" class="scene-hint" role="status">{{ hint }}</span>
    <span class="legend"><i class="open"></i>可挑战<i class="cleared"></i>已通关<i class="closed"></i>未解锁</span>
  </div>
</template>

<script setup>
import { computed, ref, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { Aim, Check, Lock, ZoomIn, ZoomOut } from '@element-plus/icons-vue'
import { HEIGHT, WIDTH, locate, roadHit, ROAD_LENGTH, STATUS } from './expedition'
import { paintScene } from './coast'
import traveler from './assets/traveler.webp'

const props = defineProps({
  stops: { type: Array, required: true },
  selectedId: Number, startingId: Number, themeIndex: Number, seed: Number,
})
const emit = defineEmits(['select', 'enter', 'mission'])
const terrain = ref(null), board = ref(null), viewport = ref(null), overview = ref(false)
const selectedStop = computed(() => props.stops.find(stop => stop.id === props.selectedId))
const distance = ref(null), frame = ref(0), facing = ref(2)
const hint = ref('')
const actor = computed(() => distance.value == null ? null : locate(distance.value))
const spriteStyle = computed(() => ({
  backgroundImage: `url("${traveler}")`,
  backgroundPosition: `${frame.value * 50}% ${facing.value * 100 / 3}%`,
}))
let animation = 0, resize, motionPreference, hintTimer
const reduceMotion = () => motionPreference?.matches === true
const position = (x,y) => ({ left: `${x / WIDTH * 100}%`, top: `${y / HEIGHT * 100}%` })

function draw() {
  if (!board.value || !terrain.value) return
  const scale = Math.min(2, window.devicePixelRatio || 1)
  terrain.value.width = Math.max(1, Math.round(board.value.clientWidth * scale))
  terrain.value.height = Math.max(1, Math.round(board.value.clientHeight * scale))
  paintScene(terrain.value, props.themeIndex, props.seed)
}
function center(stop, smooth = true) {
  if (!stop || !viewport.value || !board.value) return
  const left = board.value.clientWidth * stop.x / WIDTH - viewport.value.clientWidth / 2
  viewport.value.scrollTo({ left, behavior: smooth && !reduceMotion() ? 'smooth' : 'instant' })
}
async function toggleOverview() {
  overview.value = !overview.value
  await nextTick()
  center(selectedStop.value, false)
}
function stopAnimation() {
  cancelAnimationFrame(animation)
  frame.value = 0
}
function walkTo(targetDistance, arrived) {
  stopAnimation()
  const start = distance.value ?? 0
  const segment = targetDistance - start
  facing.value = segment < 0 ? 1 : 2
  if (!segment || reduceMotion()) { distance.value = targetDistance; arrived?.(); return }
  const duration = Math.min(2200, Math.max(380, Math.abs(segment) * 4))
  const began = performance.now()
  function step(now) {
    const t = Math.min(1, (now - began) / duration)
    distance.value = start + segment * t
    frame.value = t === 1 ? 0 : 1 + Math.floor((now - began) / 145) % 2
    if (t < 1) animation = requestAnimationFrame(step)
    else arrived?.()
  }
  animation = requestAnimationFrame(step)
}
function choose(stop) {
  emit('select', stop)
  walkTo(stop.distance)
  center(stop)
}
function tell(message) {
  hint.value = message
  clearTimeout(hintTimer)
  hintTimer = setTimeout(() => { hint.value = '' }, 2200)
}
function visitGuide() {
  const harbor = roadHit(174,302,2)
  walkTo(harbor.distance, () => emit('mission'))
}
function nearby() {
  return props.stops.find(stop => Math.abs(stop.distance - distance.value) < 27)
}
function walkOnRoad(event) {
  const rect = terrain.value.getBoundingClientRect()
  const x = (event.clientX - rect.left) * WIDTH / rect.width
  const y = (event.clientY - rect.top) * HEIGHT / rect.height
  const hit = roadHit(x, y, Math.max(18, 16 * WIDTH / rect.width))
  if (!hit) { tell('点击浅色小路或据点，旅人会沿路前往。'); return }
  viewport.value.focus({ preventScroll: true })
  walkTo(hit.distance, () => { const stop = nearby(); if (stop) emit('select', stop) })
}
function onKey(event) {
  if (event.altKey || event.ctrlKey || event.metaKey) return
  // Arrow keys navigate focused checkpoint buttons. On the map they walk along the trail.
  const index = props.stops.findIndex(stop => stop.id === props.selectedId)
  const key = event.key.toLowerCase()
  const direction = { arrowleft:-1, arrowup:1, arrowright:1, arrowdown:-1, w:1, a:-1, s:-1, d:1 }[key]
  if (direction) {
    event.preventDefault()
    if (key.startsWith('arrow') && event.target.closest('.outpost')) {
      const next = props.stops[index + direction]
      if (next) {
        choose(next)
        board.value.querySelectorAll('.outpost')[index + direction]?.focus({ preventScroll: true })
      }
      return
    }
    stopAnimation()
    distance.value = Math.max(0, Math.min(ROAD_LENGTH, (distance.value ?? 0) + direction * 12))
    facing.value = direction < 0 ? 1 : 2
    frame.value = reduceMotion() ? 0 : 1 + Math.floor(distance.value / 12) % 2
    center(actor.value, false)
    const stop = nearby()
    if (stop) emit('select', stop)
  } else if (key === 'e' || (key === 'enter' && event.target === viewport.value)) {
    event.preventDefault()
    const stop = nearby()
    if (!stop) { tell('靠近据点后按 E 进入关卡。'); return }
    emit('select', stop)
    if (stop.status !== 'LOCKED') emit('enter', stop)
    else tell('这个据点尚未解锁，解锁条件见下方关卡说明。')
  }
}
function rest(event) {
  if (event.key.toLowerCase().startsWith('arrow') || ['w','a','s','d'].includes(event.key.toLowerCase())) frame.value = 0
}
function settle() {
  stopAnimation()
  const start = props.stops.find(stop => stop.id === props.startingId)
  distance.value = start?.distance ?? props.stops[0]?.distance ?? 0
  center(selectedStop.value, false)
}
watch(() => props.stops, () => { settle(); draw() }, { flush:'post' })
onMounted(() => {
  motionPreference = matchMedia('(prefers-reduced-motion: reduce)')
  resize = new ResizeObserver(draw)
  resize.observe(board.value)
  settle()
})
onBeforeUnmount(() => { stopAnimation(); clearTimeout(hintTimer); resize?.disconnect() })
</script>

<style scoped>
.scene-frame { position:relative; background:#add1c2; isolation:isolate; }
.viewport { width:100%; overflow-x:auto; scrollbar-color:#659d90 #aacfbd; scrollbar-width:thin; overscroll-behavior-x:contain; }
.board { position:relative; width:100%; min-width:640px; aspect-ratio:1000 / 456; container-type:inline-size; }
.board.fitted { min-width:0; }
.terrain { position:absolute; inset:0; display:block; width:100%; height:100%; }
.guide-marker { position:absolute; transform:translate(-50%,-100%); width:40px; height:38px; display:grid; align-items:start; justify-items:center; padding:0; background:none; border:0; cursor:pointer; }
.guide-marker span { padding:2px 6px 3px; border:1px solid #8c6a32; border-radius:7px 7px 7px 0; background:#ffe7a2; color:#775016; font:bold 14px Georgia,serif; box-shadow:0 2px 4px #3343223d; }
.scene-hint { position:absolute; left:50%; top:10px; transform:translateX(-50%); padding:7px 11px; border-radius:6px; background:#223f34ed; color:#faf9e9; font-size:12px; text-align:center; pointer-events:none; }
.outpost { position:absolute; z-index:2; transform:translate(-50%,-50%); display:grid; place-items:center; width:44px; height:44px; border:0; padding:0; background:transparent; color:#fffef4; cursor:pointer; }
.pin { display:grid; place-items:center; width:clamp(20px,2.35cqw,32px); aspect-ratio:1; border:2px solid #f9e4af; border-radius:50%; background:#bc7b32; box-shadow:0 2px 0 #886a43,0 4px 6px #253f3352; font-size:clamp(12px,1.4cqw,16px); }
.outpost.completed .pin { background:#306954; }
.outpost.locked .pin { color:#f5f1dc; border-color:#d0d3b7; background:#62746d; }
.outpost.selected .pin { outline:2px solid #fff4ce; outline-offset:3px; }
.outpost:focus-visible { outline:2px solid #172f25; border-radius:50%; }
.caption { pointer-events:none; position:absolute; top:calc(50% + clamp(15px,1.7cqw,22px)); max-width:clamp(68px,11cqw,150px); overflow:hidden; text-overflow:ellipsis; white-space:nowrap; background:#304e43ed; border:1px solid #d8d8b470; border-radius:5px; box-shadow:0 2px 4px #38574730; padding:3px 7px; font-weight:600; font-size:clamp(10px,1.04cqw,13px); line-height:1.35; }
.outpost.selected .caption { color:#3b412c; background:#fff0c8; border-color:#bda36c; }
.actor { pointer-events:none; position:absolute; z-index:3; width:clamp(30px,4.8cqw,56px); aspect-ratio:96 / 112; transform:translate(-70%,-98%); }
.actor-shadow { position:absolute; bottom:1%; left:27%; right:21%; height:10%; border-radius:50%; background:#233d3840; }
.sprite { position:absolute; inset:0; background-size:300% 400%; background-repeat:no-repeat; }
.compass { position:absolute; top:5%; left:3%; width:28px; height:45px; display:grid; justify-items:center; align-content:start; color:#365d50; opacity:.82; pointer-events:none; }
.compass b { font:700 10px Georgia,serif; }
.compass i { display:block; width:15px; height:27px; background:linear-gradient(90deg,#496b58 50%,#e5e6c5 50%); clip-path:polygon(50% 0,100% 85%,50% 66%,0 85%); }
.map-tools { position:absolute; top:12px; right:12px; display:flex; gap:5px; }
.map-tools button { width:35px; height:35px; display:grid; place-items:center; border:1px solid #739e86; border-radius:8px; background:#f8f8e7e6; color:#355547; font-size:17px; cursor:pointer; }
.map-tools { display:none; }
.legend { position:absolute; bottom:12px; right:12px; display:flex; align-items:center; gap:5px; padding:6px 9px; border:1px solid #b5c7ae; background:#f4f6e6ed; color:#405c4d; border-radius:6px; font-size:11px; pointer-events:none; }
.legend i { width:7px; height:7px; border-radius:50%; }
.legend i:not(:first-child) { margin-left:4px; }
.open { background:#ad722e; }.cleared { background:#306954; }.closed { background:#667670; }
@media(max-width:760px) { .map-tools {display:flex;} .legend {bottom:8px;right:8px;} .fitted .caption {display:none;} }
@media(max-width:430px) { .map-tools {top:8px;right:8px;} .fitted .pin {width:19px;font-size:10px;} .legend {font-size:10px;padding:4px 6px;} }
</style>
