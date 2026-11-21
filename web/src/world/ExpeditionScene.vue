<template>
  <div class="atlas-frame">
    <div class="map-stage" :style="{ '--map-ratio': atlas.width + ' / ' + atlas.height }">
      <div ref="viewport" class="viewport" :class="{ dragging }" role="group" tabindex="0"
           aria-label="可探索的世界地图" aria-describedby="atlas-help"
           @pointerdown="camera.start" @pointermove="camera.move" @pointerup="camera.end"
           @pointercancel="camera.end" @pointerleave="pointerLeave" @click.capture="camera.cancelClick"
           @keydown="onKey">
        <div class="world-plane" :style="boardStyle" :class="{ detailed }">
          <canvas ref="terrain" class="terrain" :style="terrainStyle" aria-hidden="true" @click="terrainClick" />
          <button v-for="region in atlas.regions" :key="region.chapterId" type="button" class="region-name"
                  :class="{ active: region.chapterId === selectedNode?.chapterId, visited: visited.has(region.chapterId), earned: hub(region).chapter.status === 'COMPLETED' }"
                  :style="position(region.labelX,region.labelY)" :aria-label="'探索第 ' + (hub(region).chapter.chapterNumber ?? region.index + 1) + ' 章 ' + hub(region).name + (visited.has(region.chapterId) ? '，已探访' : '，尚未探访')"
                  @focus="revealFocus($event,{x:region.labelX,y:region.labelY})" @click="visitRegion(region)">
            <span>{{ hub(region).chapter.status === 'COMPLETED' ? '✓' : region.index + 1 }}</span>{{ hub(region).name }}
            <el-icon v-if="visited.has(region.chapterId) && hub(region).chapter.status !== 'COMPLETED'" aria-hidden="true"><Flag /></el-icon>
          </button>
          <template v-for="node in atlas.nodes" :key="node.id">
            <button v-if="node.kind === 'level' && detailed" type="button" class="outpost"
                    :class="[places.get(node.id).status.toLowerCase(), { selected: node.id === selectedId }]"
                    :style="position(node.x,node.y)" :aria-pressed="node.id === selectedId"
                    :aria-label="accessibleName(node)" :title="accessibleName(node)"
                    @focus="revealFocus($event,node)" @click="choose(node)">
              <span class="marker"><el-icon v-if="places.get(node.id).status === 'LOCKED'"><Lock /></el-icon><el-icon v-else-if="places.get(node.id).status === 'COMPLETED'"><Check /></el-icon><b v-else>{{ node.number }}</b></span>
              <span class="node-label">{{ places.get(node.id).name }}</span>
            </button>
            <i v-else-if="node.kind === 'level'" class="survey-dot" :class="[places.get(node.id).status.toLowerCase(), { beacon: node.id === nextId }]" :style="position(node.x,node.y)" aria-hidden="true"></i>
            <button v-else-if="detailed" type="button" class="camp" :style="position(node.x,node.y)"
                    :aria-label="'拜访' + places.get(node.id).guide.role + places.get(node.id).guide.name + '，查看本章推荐任务'"
                    :title="places.get(node.id).guide.role + ' · ' + places.get(node.id).guide.name"
                    @focus="revealFocus($event,node)" @click="campTasks(node)">
              <i class="guide-art" :class="{ examiner:places.get(node.id).chapterIndex === 7 }" :style="{ backgroundPositionX:(places.get(node.id).chapterIndex % 7) / 6 * 100 + '%' }"></i>
              <span class="guide-name">{{ places.get(node.id).guide.name }}</span>
            </button>
          </template>
          <div v-if="actor" class="actor" :class="{ walking }" :style="actorStyle" aria-hidden="true">
            <span class="shadow"></span><span class="sprite" :style="spriteStyle"></span>
          </div>
        </div>
      </div>
      <div class="map-controls" aria-label="地图视野">
        <button type="button" title="查看全图" aria-label="查看全图" @click="camera.overview"><el-icon><FullScreen /></el-icon></button>
        <button type="button" title="定位角色" aria-label="定位角色" @click="locateActor"><el-icon><Aim /></el-icon></button>
        <span></span>
        <button type="button" title="放大地图" aria-label="放大地图" :disabled="scale >= maxScale - .001" @click="camera.zoomBy(1.5)"><el-icon><Plus /></el-icon></button>
        <button type="button" title="缩小地图" aria-label="缩小地图" :disabled="overWorld" @click="camera.zoomBy(1/1.5)"><el-icon><Minus /></el-icon></button>
      </div>
      <div class="compass" aria-hidden="true">N<span></span></div>
      <button v-show="!overWorld" type="button" class="mini-map" title="查看全图" aria-label="从小地图返回全图" @click="camera.overview">
        <canvas ref="mini" aria-hidden="true"></canvas><i :style="miniWindow"></i>
      </button>
      <p v-if="hint" class="map-hint" role="status">{{ hint }}</p>
    </div>
    <footer>
      <div class="legend"><span><i class="available"></i>可挑战</span><span><i class="complete"></i>已通关</span><span><i class="locked"></i>未解锁</span></div>
      <p id="atlas-help"><span class="desktop-help">点岛名放大，方向键选路，E 进入。</span><span class="touch-help">点地点前往，双指缩放。</span></p>
    </footer>
  </div>
</template>

<script setup>
import { ref, shallowRef, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { FullScreen, Aim, Plus, Minus, Lock, Check, Flag } from '@element-plus/icons-vue'
import { paintAtlas } from './atlasPaint'
import { routeBetween, routeToRoad, routeBetweenRoads, nearestOnRoad, directionalNeighbor } from './worldAtlas'
import { useAtlasCamera } from './useAtlasCamera'
import traveler from './assets/traveler.webp'

const props = defineProps({
  atlas: { type:Object, required:true }, places: { type:Map, required:true },
  selectedId: String, restoreFocus: Boolean, visited: { type: Set, required: true }
})
const emit = defineEmits(['select','enter','mission','arrive'])
const terrain = ref(null), viewport = ref(null), mini = ref(null)
const camera = useAtlasCamera(viewport, () => props.atlas)
const { scale,fit,maxScale,overWorld,boardStyle,dragging } = camera
const detailed = computed(() => scale.value >= .74 || scale.value > fit.value * 1.4)
const padding = computed(() => ({
  x:Math.ceil(Math.max(0,camera.width.value / fit.value - props.atlas.width) / 2),
  y:Math.ceil(Math.max(0,camera.height.value / fit.value - props.atlas.height) / 2)
}))
const terrainStyle = computed(() => ({
  left:-padding.value.x / props.atlas.width * 100 + '%', top:-padding.value.y / props.atlas.height * 100 + '%',
  width:(1 + 2 * padding.value.x / props.atlas.width) * 100 + '%',
  height:(1 + 2 * padding.value.y / props.atlas.height) * 100 + '%'
}))
const actor = shallowRef(null), facing = ref(0), frame = ref(0), walking = ref(false), hint = ref('')
const selectedNode = computed(() => props.atlas.nodeById.get(props.selectedId))
const nextId = computed(() => props.atlas.nodes.find(node => node.kind === 'level' && props.places.get(node.id)?.status === 'LEARNING')?.id)
const trailNodes = computed(() => props.atlas.nodes.filter(node => {
  const place = props.places.get(node.id)
  return node.kind === 'hub' ? place?.chapter.levelsCompleted > 0 : place?.status === 'COMPLETED'
}).map(node => node.id))
const position = (x,y) => ({ left:x / props.atlas.width * 100 + '%', top:y / props.atlas.height * 100 + '%' })
const actorStyle = computed(() => ({ ...position(actor.value.x,actor.value.y), width:(detailed.value ? 39 : 26) + 'px' }))
const spriteStyle = computed(() => ({
  backgroundImage:'url(' + traveler + ')', backgroundPosition:frame.value * 50 + '% ' + facing.value * (100/3) + '%'
}))
const miniWindow = computed(() => {
  const worldW = props.atlas.width * scale.value, worldH = props.atlas.height * scale.value
  return { left:Math.max(0,-camera.offset.value.x/worldW)*100+'%', top:Math.max(0,-camera.offset.value.y/worldH)*100+'%',
    width:Math.min(1,camera.width.value/worldW)*100+'%', height:Math.min(1,camera.height.value/worldH)*100+'%' }
})
const hubId = region => 'hub:' + region.chapterId
const hub = region => props.places.get(hubId(region))
const regionOf = node => props.atlas.regions.find(region => region.chapterId === node.chapterId)
const accessibleName = node => '第 ' + node.number + ' 关 ' + props.places.get(node.id).name + '，' + props.places.get(node.id).statusText
let raf, hintTimer, actorId, motion

function renderTerrain() {
  if (!terrain.value || !mini.value) return
  const { width, height } = props.atlas
  const {x:padX, y:padY} = padding.value
  const paintedWidth = width + 2 * padX, paintedHeight = height + 2 * padY
  const density = Math.min(1.7, 2680 / paintedWidth, 1900 / paintedHeight)
  terrain.value.width = Math.ceil(paintedWidth * density)
  terrain.value.height = Math.ceil(paintedHeight * density)
  paintAtlas(terrain.value, props.atlas, {padX,padY, trailNodes: new Set(trailNodes.value)})
  mini.value.width = 172
  mini.value.height = Math.round(172 * height / width)
  mini.value.getContext('2d')?.drawImage(terrain.value, padX*density, padY*density, width*density, height*density, 0, 0, mini.value.width, mini.value.height)
}
function message(text) {
  clearTimeout(hintTimer)
  hint.value = text
  hintTimer = setTimeout(() => { hint.value = '' },2300)
}
function halt() {
  cancelAnimationFrame(raf)
  walking.value = false
  frame.value = 0
}
function animate(path, destination, arrival) {
  if (!path.length) return
  halt()
  const lengths = [0]
  for (let i = 1; i < path.length; i++) lengths.push(lengths[i-1] + Math.hypot(path[i].x - path[i-1].x,path[i].y - path[i-1].y))
  const total = lengths.at(-1)
  function finish() {
    walking.value = false
    actor.value = path.at(-1)
    actorId = destination
    frame.value = 0
    arrival?.()
  }
  if (!total || motion?.matches) { finish(); return }
  actorId = null
  walking.value = true
  const start = performance.now(), duration = Math.min(2800,Math.max(280,total*4))
  let segment = 1
  function tick(now) {
    const dist = total * Math.min(1,(now - start) / duration)
    while (segment < path.length - 1 && lengths[segment] <= dist) segment++
    const a = path[segment-1], b = path[segment], dx = b.x-a.x, dy = b.y-a.y
    const fraction = (dist-lengths[segment-1]) / (lengths[segment]-lengths[segment-1] || 1)
    actor.value = { x:a.x+fraction*dx, y:a.y+fraction*dy }
    facing.value = Math.abs(dx) > Math.abs(dy) ? (dx < 0 ? 1 : 2) : (dy < 0 ? 3 : 0)
    frame.value = 1+Math.floor((now-start)/130)%2
    if (dist < total) raf = requestAnimationFrame(tick)
    else finish()
  }
  raf = requestAnimationFrame(tick)
}
function actorHit() {
  return actor.value && nearestOnRoad(props.atlas,actor.value.x,actor.value.y)
}
function travel(node, arrival) {
  const path = actorId
    ? routeBetween(props.atlas,actorId,node.id)
    : routeToRoad(props.atlas,node.id,actorHit()).reverse()
  animate(path,node.id,() => {
    if (node.kind === 'hub') emit('arrive',node.chapterId)
    arrival?.()
  })
}
function choose(node) {
  emit('select',node.id)
  travel(node)
}
function visitRegion(region) {
  const node = props.atlas.nodeById.get(hubId(region))
  choose(node)
  camera.focusRegion(region)
}
function visitCampById(id) {
  const region = props.atlas.regions.find(r => r.chapterId === id)
  if (region) visitRegion(region)
}
function focusLevel(id) {
  const node = props.atlas.nodeById.get('level:' + id)
  if (!node) return
  choose(node)
  camera.focusRegion(regionOf(node), node)
}
defineExpose({visitCampById, focusLevel})
function campTasks(node) {
  emit('select',node.id)
  travel(node, () => emit('mission',node.chapterId))
}
function locateActor() {
  if (!actor.value) return
  const region = closestRegion(actor.value)
  if (region) camera.focusRegion(region,actor.value)
}
function revealFocus(event,p) {
  if (event.target.matches(':focus-visible')) camera.reveal(p)
}
function pointerLeave(event) {
  if (!viewport.value.hasPointerCapture(event.pointerId)) camera.end(event)
}
function closestRegion(point) {
  let best, gap = Infinity
  for (const region of props.atlas.regions) {
    const distance = Math.hypot((point.x-region.x)/region.width,(point.y-region.y)/region.height)
    if (distance < gap) { best=region; gap=distance }
  }
  return best
}
function terrainClick(event) {
  viewport.value.focus({ preventScroll:true })
  const point = camera.pointInWorld(camera.eventPosition(event))
  if (!detailed.value) {
    const region = closestRegion(point)
    if (region && Math.hypot((point.x-region.x)/region.width,(point.y-region.y)/region.height) < .58) {
      visitRegion(region)
      return
    }
  }
  const hit = nearestOnRoad(props.atlas,point.x,point.y,14/scale.value)
  if (!hit) { message('点道路或地图上的地点前往。'); return }
  const endId = hit.t < .045 ? hit.edge.from : hit.t > .955 ? hit.edge.to : null
  if (endId) { choose(props.atlas.nodeById.get(endId)); return }
  const path = actorId ? routeToRoad(props.atlas,actorId,hit) : routeBetweenRoads(props.atlas,actorHit(),hit)
  animate(path, null)
}
function onKey(event) {
  if (event.ctrlKey || event.altKey || event.metaKey) return
  const key = event.key.toLowerCase()
  const direction = { arrowleft:[-1,0], a:[-1,0], arrowright:[1,0], d:[1,0], arrowup:[0,-1], w:[0,-1], arrowdown:[0,1], s:[0,1] }[key]
  if (direction) {
    event.preventDefault()
    if (walking.value && event.repeat) return
    const hit = actorId ? null : actorHit()
    let next
    if (hit) {
      const ends = [props.atlas.nodeById.get(hit.edge.from),props.atlas.nodeById.get(hit.edge.to)]
      const score = p => (p.x-hit.x)*direction[0] + (p.y-hit.y)*direction[1]
      next = ends.sort((a,b) => score(b)-score(a))[0]
    } else next = directionalNeighbor(props.atlas, actorId, ...direction)
    if (!next) { message('这个方向没有相连的路，试试其他方向。'); return }
    choose(next)
    camera.reveal(next)
  } else if (key === 'e' || (key === 'enter' && event.target === viewport.value)) {
    event.preventDefault()
    if (!actorId) { message('到达据点后可以进入关卡。'); return }
    const place = props.places.get(actorId)
    if (place?.level && place.status === 'LOCKED') message('关卡未解锁，可在关卡介绍中查看先修条件。')
    else if (place?.level) emit('enter',actorId)
    else { emit('arrive',place.chapter.id); emit('mission',place.chapter.id) }
  } else if (key === '+' || key === '=' || key === '-') {
    event.preventDefault()
    camera.zoomBy(key === '-' ? 1/1.5 : 1.5)
  } else if (key === 'home') {
    event.preventDefault()
    camera.overview()
  }
}
function reset(initial) {
  halt()
  const node = props.atlas.nodeById.get(props.selectedId) || props.atlas.nodes[0]
  actor.value = node
  actorId = node?.id
  renderTerrain()
  if (initial && node && (props.restoreFocus || viewport.value.clientWidth < 460)) camera.focusRegion(regionOf(node),node)
  else if (initial) camera.overview()
}
onMounted(() => {
  motion = window.matchMedia('(prefers-reduced-motion: reduce)')
  reset(true)
})
watch(() => props.atlas, () => reset(false), { flush:'post' })
watch(padding, renderTerrain, { flush:'post' })
watch(() => trailNodes.value.join(','), renderTerrain, { flush:'post' })
onBeforeUnmount(() => { halt(); clearTimeout(hintTimer) })
</script>

<style scoped>
.atlas-frame { overflow:hidden; border:1px solid #bdcbbd; border-radius:15px; background:#fffdf4; box-shadow:0 4px 15px #27453515; }
.atlas-frame {display:flex;flex-direction:column;}
.map-stage { position:relative; flex:auto; width:100%; aspect-ratio:var(--map-ratio); min-height:350px; max-height:clamp(420px,54vh,630px); background:#85bcb4; }
.viewport,.world-plane,.terrain { position:absolute; inset:0; }
.viewport { overflow:clip; touch-action:none; user-select:none; cursor:grab; background:linear-gradient(115deg,#b4d6c1,#80b9b2 45%,#89c0b9 85%,#a4cabe); }
.viewport.dragging,.viewport.dragging * {cursor:grabbing;}
.world-plane { right:auto; bottom:auto; }
.terrain { display:block; width:100%; height:100%; }
.region-name,.outpost,.camp,.actor,.survey-dot { position:absolute; }
.region-name { z-index:4; display:inline-flex; align-items:center; justify-content:center; gap:6px; transform:translate(-50%,-50%); max-width:145px; padding:5px 10px 5px 5px; background:#fffae8f2; color:#274538; font-size:13px; font-weight:700; line-height:1.2; border:1px solid #8c9d81; border-radius:16px; box-shadow:0 2px 4px #34594738; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
.region-name span { display:grid;place-items:center;min-width:18px;height:18px;background:#dce3ca;color:#45593c;border-radius:50%;font:700 11px/1 system-ui; }
.region-name.active,.region-name:hover { border-color:#8a733e; background:#fff0c7; }
.region-name.active span { color:#fff5d9; background:#886d3b; }
.region-name.visited { border-color:#4b725b; }
.region-name.earned { background:#fae5a1;border:2px solid #88783d; }
.region-name.earned span { color:#f9edc6;background:#466849; }
.outpost,.camp { z-index:3; transform:translate(-50%,-50%); width:38px;height:38px; display:grid;place-items:center;border:0;padding:0;background:none; }
.camp { margin-left:19px; }
.marker,.camp .el-icon { display:grid; place-items:center; width:23px; height:23px; border:2px solid #fff3cd; box-shadow:0 1px 3px #233f3480; border-radius:50%; background:#9b6827; color:#fff; font-size:12px; }
.outpost.locked .marker {background:#62716a;} .outpost.completed .marker {background:#2c6650;}
.outpost.selected .marker {outline:2px solid #ffeabb;outline-offset:2px;}
.node-label { position:absolute; pointer-events:none; top:calc(50% + 13px); white-space:nowrap; max-width:100px; overflow:hidden; text-overflow:ellipsis; padding:2px 6px; border-radius:4px; color:#fff; background:#274d41ed; font-size:10px; line-height:1.4; font-weight:600; }
.outpost.selected .node-label,.outpost:hover .node-label,.outpost:focus-visible .node-label { background:#fff0c9; color:#304632; z-index:3; }
.camp .el-icon { background:#fff2cc; color:#5c5130; border-color:#987a41; border-radius:6px; font-size:15px; }
.guide-art { position:absolute; width:26px; height:39px; bottom:14px; background:url('./assets/camp-guides.png') no-repeat left top / 700% 100%; image-rendering:pixelated; filter:drop-shadow(0 1px 1px #32483970); }
.guide-art.examiner { filter:hue-rotate(310deg) saturate(.8) drop-shadow(0 1px 1px #32483970); }
.guide-name { position:absolute; white-space:nowrap; top:22px; background:#fcf2d2; border:1px solid #ae9363; padding:1px 6px; border-radius:4px; font-size:11px; color:#604c2c; font-weight:700; }
.survey-dot { transform:translate(-50%,-50%); pointer-events:none; border-radius:50%; width:7px;height:7px; background:#af762f; border:1px solid #fff3cc; }
.survey-dot.completed {background:#265f49;} .survey-dot.locked {background:#687969;}
.survey-dot.beacon {width:11px;height:11px;background:#f5cd62;border:2px solid #6e5125;box-shadow:0 0 0 4px #fff0b896;}
.survey-dot.completed {border-color:#faf0ba;box-shadow:0 0 0 1px #65875a;}
.actor { z-index:5; pointer-events:none; transform:translate(-70%,-94%); aspect-ratio:96 / 112; }
.actor .sprite { position:absolute; inset:0; background-repeat:no-repeat; background-size:300% 400%; }
.actor .shadow { position:absolute; bottom:0; left:28%; right:20%; height:10%; background:#22443766;border-radius:50%; }
.map-controls { position:absolute; z-index:6; top:11px; right:11px; display:flex; gap:3px; align-items:center; padding:3px; border:1px solid #8fac96; border-radius:9px; background:#fbfbeced; box-shadow:0 2px 5px #42664f20; }
.map-controls button { display:grid;place-items:center;width:34px;height:33px;border:0;border-radius:6px;color:#345342;background:transparent;font-size:17px; }
.map-controls button:hover { background:#dfe9d7; } .map-controls button:disabled { color:#7e9183; opacity:.65; }
.map-controls>span {width:1px;height:17px;background:#c2cdb6;}
.compass { position:absolute; top:12px; left:13px; display:grid;justify-items:center; gap:2px; pointer-events:none; color:#2d584e; font:bold 10px Georgia,serif; }
.compass span {width:12px; height:20px; background:linear-gradient(90deg,#43695a 50%,#e6e6ca 50%);clip-path:polygon(50% 0,100% 88%,50% 69%,0 88%);}
.mini-map { position:absolute; bottom:10px;right:10px;width:88px;border:2px solid #f5f3dd;border-radius:5px;box-shadow:0 1px 6px #31544470; padding:0; overflow:hidden; }
.mini-map canvas { width:100%; display:block; } .mini-map i { position:absolute;border:1px solid #fffaf0;outline:1px solid #3d6453; background:#fff6da25; }
.map-hint { position:absolute; bottom:13px;left:50%; transform:translateX(-50%); padding:6px 10px; border-radius:6px;background:#294b3eeb;color:#fff9ea; font-size:12px; text-align:center;pointer-events:none; }
footer { display:flex; justify-content:space-between; align-items:center; gap:8px; flex-wrap:wrap; border-top:1px solid #becdbd; padding:10px 15px; font-size:12px; color:#53665a; }
.legend,.legend span {display:inline-flex;align-items:center;gap:6px;} .legend {gap:13px;}
.legend i {width:7px;height:7px;border-radius:50%;} .available {background:#a57531;} .complete {background:#306750;} .locked {background:#667369;} .outpost.locked {background:none;}
.touch-help {display:none;}
button:focus-visible,.viewport:focus-visible {outline:3px solid #9c6422;outline-offset:-2px;}
@media(max-width:620px) {
  .region-name {max-width:94px; font-size:11px; gap:3px; padding:4px 6px 4px 3px; }
  .region-name span {min-width:14px;height:14px;font-size:9px;}
  .map-controls {top:auto;bottom:5px;right:6px;} .map-controls button {width:29px;height:28px;}
  .map-stage {min-height:298px;}
  .mini-map {width:58px;bottom:6px;left:6px;right:auto;}
  .survey-dot {width:5px;height:5px;}
  footer {padding:9px 11px;gap:5px;font-size:11px;}
}
@media(max-width:620px),(pointer:coarse) { .touch-help {display:inline;} .desktop-help {display:none;} }
</style>
