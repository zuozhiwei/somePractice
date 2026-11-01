import { TILE, WORLD, NODES, ZONES, PATH, pathLen, pathPointAt, pathProject } from './data.js'

const SPEED = 140

const BAND_COLORS = {
  sky: '#aee3ff', snow: '#e6f2f7', town: '#f8e8c8', castle: '#c6d3e8',
  mine: '#c9a57d', sand: '#f6e3b4', village: '#a8e6a3'
}
const STOP_DIST = 40   // 未解锁关卡前的停车距离（沿路径，约等于圆牌半径）

const clamp = (v, lo, hi) => (v < lo ? lo : v > hi ? hi : v)
const compactNodePos = index => {
  const row = Math.floor(index / 6)
  const col = index % 6
  return { x: row % 2 === 0 ? 100 + col * 180 : 1000 - col * 180, y: 834 - row * 128 }
}

function hexagon(ctx, x, y, radius) {
  ctx.beginPath()
  for (let i = 0; i < 6; i++) {
    const angle = Math.PI / 6 + i * Math.PI / 3
    const px = x + Math.cos(angle) * radius
    const py = y + Math.sin(angle) * radius
    if (i === 0) ctx.moveTo(px, py)
    else ctx.lineTo(px, py)
  }
  ctx.closePath()
}

function drawStar(ctx, x, y, radius, color) {
  ctx.beginPath()
  for (let i = 0; i < 10; i++) {
    const angle = -Math.PI / 2 + i * Math.PI / 5
    const r = i % 2 === 0 ? radius : radius * .45
    const px = x + Math.cos(angle) * r
    const py = y + Math.sin(angle) * r
    if (i === 0) ctx.moveTo(px, py)
    else ctx.lineTo(px, py)
  }
  ctx.closePath()
  ctx.fillStyle = color
  ctx.fill()
}

// 卡通道具绘制（区域装饰，保留城堡/矿车/雪人/云/彩虹等手绘，树木花草改走像素素材）
function drawMinecart(ctx, x, y) {
  ctx.fillStyle = '#8a5f32'
  ctx.fillRect(x - 20, y - 16, 40, 18)
  ctx.fillStyle = '#5d3f1f'
  ctx.fillRect(x - 20, y - 16, 40, 5)
  ctx.strokeStyle = '#7a4e28'; ctx.lineWidth = 2
  ctx.beginPath(); ctx.moveTo(x - 20, y - 16); ctx.lineTo(x + 20, y - 16); ctx.stroke()
  ctx.fillStyle = '#2b2b2b'
  ctx.beginPath(); ctx.arc(x - 12, y + 5, 5, 0, Math.PI * 2); ctx.fill()
  ctx.beginPath(); ctx.arc(x + 12, y + 5, 5, 0, Math.PI * 2); ctx.fill()
}

function drawCastle(ctx, x, y) {
  ctx.fillStyle = '#8fa3b8'
  ctx.fillRect(x - 14, y - 30, 28, 38)
  for (let i = -1; i <= 1; i++) ctx.fillRect(x + i * 10 - 4, y - 36, 8, 8)
  ctx.fillStyle = '#5b6b7d'
  ctx.fillRect(x - 4, y - 8, 8, 16)
  ctx.fillStyle = '#d94f4f'
  ctx.beginPath(); ctx.moveTo(x, y - 46); ctx.lineTo(x, y - 36); ctx.lineTo(x + 10, y - 41); ctx.closePath(); ctx.fill()
}

function drawMound(ctx, x, y, rx) {
  ctx.fillStyle = '#ffffff'
  ctx.beginPath(); ctx.ellipse(x, y, rx, rx * 0.4, 0, 0, Math.PI); ctx.fill()
}

function drawSnowman(ctx, x, y) {
  ctx.fillStyle = '#ffffff'
  ctx.beginPath(); ctx.arc(x, y, 15, 0, Math.PI * 2); ctx.fill()
  ctx.beginPath(); ctx.arc(x, y - 24, 10, 0, Math.PI * 2); ctx.fill()
  ctx.fillStyle = '#2b2b2b'
  ctx.fillRect(x - 10, y - 38, 20, 5)
  ctx.fillRect(x - 6, y - 47, 12, 10)
  ctx.fillStyle = '#e8e8e8'
  ctx.beginPath(); ctx.arc(x - 3, y - 25, 1.6, 0, Math.PI * 2); ctx.fill()
  ctx.beginPath(); ctx.arc(x + 3, y - 25, 1.6, 0, Math.PI * 2); ctx.fill()
  ctx.fillStyle = '#ff8f4d'
  ctx.beginPath(); ctx.moveTo(x - 4, y - 21); ctx.lineTo(x + 5, y - 21); ctx.lineTo(x, y - 17); ctx.closePath(); ctx.fill()
}

function drawCloud(ctx, x, y) {
  ctx.fillStyle = '#ffffff'
  ctx.beginPath(); ctx.arc(x - 12, y, 9, 0, Math.PI * 2); ctx.fill()
  ctx.beginPath(); ctx.arc(x, y - 6, 13, 0, Math.PI * 2); ctx.fill()
  ctx.beginPath(); ctx.arc(x + 12, y, 9, 0, Math.PI * 2); ctx.fill()
}

function drawRainbow(ctx, x, y) {
  const colors = ['#ff6b6b', '#ffd93d', '#4ecdc4', '#8a5cf6']
  ctx.lineWidth = 5; ctx.lineCap = 'round'
  for (let i = 0; i < colors.length; i++) {
    ctx.strokeStyle = colors[i]
    ctx.beginPath(); ctx.arc(x, y, 22 - i * 5, Math.PI, 0); ctx.stroke()
  }
}

export class WorldEngine {
  constructor(canvas, { sprites, zones, deviceScale = 1 } = {}) {
    // zones 由外部注入备用；区域数据以 data.js 的 ZONES 为准
    this.canvas = canvas
    this.ctx = canvas.getContext('2d')
    this.ctx.imageSmoothingEnabled = false
    this.deviceScale = deviceScale || 1
    this.sprites = sprites
    this.pathTotal = pathLen()
    this.nodeD = NODES.map((_, i) => {
      const p = compactNodePos(i)
      return pathProject(p.x, p.y)?.d || 0
    })
    this.d = this.nodeD[0]            // 起点 = 第 1 节点
    this.walkTarget = null            // 自动行走目标 d；null 不自动走
    this.pendingEmit = null           // 走到后要 emit 的事件（'click-node'|'enter-node'）
    this.facing = 'up'                // 初始朝上（路径向上）
    this.walkTimer = 0
    this.levelStatuses = new Map()
    this.zoneMeta = {}
    this.nodeNames = new Map()
    this.camera = { x: 0, y: 0, scale: 1 }
    this.fitScale = 1
    this.cameraTouched = false
    this.runners = []
    this.runnerMode = 'fellows'
    this.events = {}
    this.keys = {}
    this.time = 0
    this.now = 0
    this.running = false
    this.raf = 0
    this.lastT = 0
    this._inputBound = false
    const spawn = pathPointAt(this.d)
    this.lastZone = this.zoneOf(spawn.x, spawn.y)
    this.boundLoop = this.loop.bind(this)
    this.bindInput()
    this.bgCanvas = document.createElement('canvas')
    this.bgCanvas.width = WORLD.width
    this.bgCanvas.height = WORLD.height
    this.prerenderBackground()
  }

  on(ev, cb) { (this.events[ev] ||= []).push(cb) }

  emit(ev, ...args) { for (const cb of this.events[ev] || []) cb(...args) }

  setLevelStatuses(m) { this.levelStatuses = m }

  setZoneMeta(mapId, meta) { this.zoneMeta[mapId] = meta }

  setNodeNames(m) { this.nodeNames = m }

  setRunners(list, mode = 'fellows') {
    this.runners = Array.isArray(list) ? list : []
    this.runnerMode = mode === 'apprentices' ? 'apprentices' : 'fellows'
  }

  // 视野自适应：按画布尺寸动态决定缩放，让世界内容在 2K/1080p/笔记本 200% 下都有合适放大感
  recomputeScale() {
    const c = this.canvas
    const cw = c.clientWidth || 800
    const ch = c.clientHeight || 600
    this.fitScale = clamp(Math.min(cw / WORLD.width, ch / WORLD.height), 0.45, 1.6)
    if (!this.cameraTouched) this.fitWorld()
    else this.clampCamera()
  }

  viewportSize() {
    return {
      w: this.canvas.width / this.camera.scale / this.deviceScale,
      h: this.canvas.height / this.camera.scale / this.deviceScale
    }
  }

  clampCamera() {
    const view = this.viewportSize()
    this.camera.x = clamp(this.camera.x, 0, Math.max(0, WORLD.width - view.w))
    this.camera.y = clamp(this.camera.y, 0, Math.max(0, WORLD.height - view.h))
  }

  fitWorld() {
    this.cameraTouched = false
    this.camera.scale = this.fitScale || 1
    const view = this.viewportSize()
    this.camera.x = Math.max(0, (WORLD.width - view.w) / 2)
    this.camera.y = Math.max(0, (WORLD.height - view.h) / 2)
  }

  focusPlayer() {
    this.cameraTouched = true
    this.camera.scale = clamp(this.fitScale * 1.45, this.fitScale, 2.6)
    const p = pathPointAt(this.d)
    const view = this.viewportSize()
    this.camera.x = p.x - view.w / 2
    this.camera.y = p.y - view.h / 2
    this.clampCamera()
  }

  panBy(dx, dy) {
    this.cameraTouched = true
    this.camera.x -= dx / this.camera.scale
    this.camera.y -= dy / this.camera.scale
    this.clampCamera()
  }

  zoomAt(factor, clientX, clientY) {
    const rect = this.canvas.getBoundingClientRect()
    const old = this.camera.scale
    const next = clamp(old * factor, this.fitScale * 0.8, 2.6)
    const wx = (clientX - rect.left) / old + this.camera.x
    const wy = (clientY - rect.top) / old + this.camera.y
    this.cameraTouched = true
    this.camera.scale = next
    this.camera.x = wx - (clientX - rect.left) / next
    this.camera.y = wy - (clientY - rect.top) / next
    this.clampCamera()
  }

  start() {
    if (this.running) return
    this.running = true
    this.bindInput()
    this.raf = requestAnimationFrame(this.boundLoop)
  }

  stop() {
    if (!this.running) return
    this.running = false
    cancelAnimationFrame(this.raf)
    this.unbindInput()
  }

  loop(t) {
    if (!this.running) return
    const dt = Math.min(0.05, (t - (this.lastT || t)) / 1000)
    this.lastT = t
    this.now = t
    this.update(dt)
    this.draw()
    this.raf = requestAnimationFrame(this.boundLoop)
  }

  getSnapshot() {
    const p = pathPointAt(this.d)
    return { d: this.d, x: p.x, y: p.y, zone: this.zoneOf(p.x, p.y), facing: this.facing }
  }

  statusOf(levelId) {
    const s = this.levelStatuses.get(levelId)
    return s === undefined ? 'LOCKED' : s
  }

  zoneOf(x, y) {
    return (ZONES.find(z => y >= z.tile.y && y < z.tile.y + z.tile.h) || { key: 'village' }).key
  }

  // 沿路径从 fromD 出发、向 dir(+1 前进 / -1 后退) 方向，最近的一个未解锁(LOCKED)关卡节点距离
  blockedD(fromD, dir) {
    let limit = dir > 0 ? this.pathTotal : 0
    for (let i = 0; i < NODES.length; i++) {
      if (this.statusOf(NODES[i].levelId) !== 'LOCKED') continue
      const nd = this.nodeD[i]
      if (dir > 0 && nd > fromD + 1 && nd < limit) limit = nd
      else if (dir < 0 && nd < fromD - 1 && nd > limit) limit = nd
    }
    return limit
  }

  walkToD(d) {
    const target0 = Math.min(this.pathTotal, Math.max(0, d))
    const moveDir = Math.sign(target0 - this.d)
    let target = target0
    if (moveDir !== 0) {
      const limit = this.blockedD(this.d, moveDir)
      target = moveDir > 0 ? Math.min(target, limit - STOP_DIST) : Math.max(target, limit + STOP_DIST)
    }
    this.walkTarget = target
  }

  getD() { return this.d }

  nodePos(i) {
    return compactNodePos(i)
  }

  nodeAt(wx, wy) {
    let best = null, bestD = 45
    NODES.forEach((n, i) => {
      const p = this.nodePos(i)
      const dist = Math.hypot(wx - p.x, wy - p.y)
      if (dist < bestD) { bestD = dist; best = n }
    })
    return best
  }

  interact() {
    const n = this.nearestNode()
    if (!n) return
    this.emit(this.statusOf(n.levelId) === 'LOCKED' ? 'click-node' : 'enter-node', n.levelId)
  }

  nearestNode() {
    let best = null, bestD = 70
    NODES.forEach((n, i) => {
      const dist = Math.abs(this.nodeD[i] - this.d)   // 沿路径距离
      if (dist < bestD) { bestD = dist; best = n }
    })
    return best
  }

  update(dt) {
    this.time += dt
    const k = this.keys
    const p = pathPointAt(this.d)
    // 按键与路径朝向映射：竖直段用 ↑/↓，水平段用 ←/→（与屏幕方向一致）
    const vertical = Math.abs(p.dy) >= Math.abs(p.dx)
    let dir = 0
    if (vertical) {
      if (k['arrowup'] || k['w']) dir += p.dy > 0 ? -1 : 1
      if (k['arrowdown'] || k['s']) dir += p.dy > 0 ? 1 : -1
    } else {
      if (k['arrowright'] || k['d']) dir += p.dx > 0 ? 1 : -1
      if (k['arrowleft'] || k['a']) dir += p.dx > 0 ? -1 : 1
    }
    // 前方未解锁关卡拦截：只能停到其圆牌边缘，无法穿过
    const edge = d => {
      const limit = this.blockedD(this.d, d)
      return d > 0 ? limit - STOP_DIST : limit + STOP_DIST
    }
    if (this.walkTarget !== null) {
      const moveDir = Math.sign(this.walkTarget - this.d)
      if (moveDir !== 0) {
        const e = edge(moveDir)
        this.walkTarget = moveDir > 0 ? Math.min(this.walkTarget, e) : Math.max(this.walkTarget, e)
      }
      const diff = this.walkTarget - this.d
      if (Math.abs(diff) < 2) {
        this.d = this.walkTarget
        this.walkTarget = null
        if (this.pendingEmit) {
          const ev = this.pendingEmit
          this.pendingEmit = null
          this.emit(ev.ev, ev.levelId)
        }
      } else {
        this.d += Math.sign(diff) * Math.min(SPEED * dt, Math.abs(diff))
      }
    } else if (dir) {
      const nd = this.d + dir * SPEED * dt
      const e = edge(dir)
      this.d = dir > 0 ? Math.min(nd, e) : Math.max(nd, e)
    }
    this.d = Math.min(this.pathTotal, Math.max(0, this.d))
    if (this.walkTarget === null && (dir || this.walkTimer > 0)) {
      this.walkTimer += dt
    } else { this.walkTimer = 0 }
    this.facing = Math.abs(p.dx) > Math.abs(p.dy) ? (p.dx > 0 ? 'right' : 'left') : (p.dy > 0 ? 'down' : 'up')
    const zone = this.zoneOf(p.x, p.y)
    if (zone !== this.lastZone) { this.lastZone = zone; this.emit('zone-change', zone) }
  }

  prerenderBackground() {
    const c = this.bgCanvas
    const ctx = c.getContext('2d')
    // 1) 地面：区域基础色 + 像素纹理平铺（grass/sand/snow）
    for (const z of ZONES) {
      ctx.fillStyle = BAND_COLORS[z.key]
      ctx.fillRect(0, z.tile.y, WORLD.width, z.tile.h)
      const list = this.groundTiles(z.key)
      if (list && list.length) {
        const cols = Math.ceil(WORLD.width / TILE)
        for (let ty = z.tile.y; ty < z.tile.y + z.tile.h; ty += TILE) {
          for (let tx = 0; tx < cols * TILE; tx += TILE) {
            const t = list[(((tx / TILE) * 7 + (ty / TILE) * 13) | 0) % list.length]
            if (t) ctx.drawImage(t, tx, ty)
          }
        }
        // 蒙层：把纹理统一到区域色调
        ctx.globalAlpha = 0.38
        ctx.fillStyle = BAND_COLORS[z.key]
        ctx.fillRect(0, z.tile.y, WORLD.width, z.tile.h)
        ctx.globalAlpha = 1
      }
    }
    // 2) 色带间波浪分界线（每相邻带边界画浅色弧）
    for (const z of ZONES.slice(0, -1)) {
      const y = z.tile.y + z.tile.h
      ctx.strokeStyle = 'rgba(255,255,255,0.65)'
      ctx.lineWidth = 6
      ctx.beginPath()
      ctx.moveTo(0, y + 3)
      ctx.bezierCurveTo(WORLD.width * 0.25, y - 14, WORLD.width * 0.5, y + 14, WORLD.width * 0.75, y - 14)
      ctx.lineTo(WORLD.width, y + 3)
      ctx.stroke()
    }
    // 3) 区域道具
    this.drawBandProps(ctx)
    // 4) 道路：深色外缘 + 暖色石板主路，避免看起来像普通流程线
    ctx.lineCap = 'round'; ctx.lineJoin = 'round'
    ctx.strokeStyle = '#3f3542'; ctx.lineWidth = 60
    ctx.beginPath(); ctx.moveTo(PATH[0].x, PATH[0].y)
    for (let i = 1; i < PATH.length; i++) ctx.lineTo(PATH[i].x, PATH[i].y)
    ctx.stroke()
    ctx.strokeStyle = '#c9945b'; ctx.lineWidth = 46
    ctx.beginPath(); ctx.moveTo(PATH[0].x, PATH[0].y)
    for (let i = 1; i < PATH.length; i++) ctx.lineTo(PATH[i].x, PATH[i].y)
    ctx.stroke()
    this.drawPaving(ctx)
    ctx.lineCap = 'butt'; ctx.lineJoin = 'miter'
  }

  // 各区域地面纹理：grass/sand/snow 用像素 tile，矿洞/天空用手绘底色
  groundTiles(zone) {
    const tiles = this.sprites?.tiles
    if (!tiles) return null
    if (zone === 'village' || zone === 'town' || zone === 'castle') return tiles.grass
    if (zone === 'sand') return tiles.sand
    if (zone === 'snow') return tiles.snow
    return null
  }

  // 沿路径铺石板（两色交替），让道路有游戏地图质感
  drawPaving(ctx) {
    const L = pathLen()
    const step = 30
    for (let d = 0; d < L; d += step) {
      const p = pathPointAt(d)
      ctx.save()
      ctx.translate(p.x, p.y)
      ctx.rotate(Math.atan2(p.dy, p.dx))
      ctx.fillStyle = Math.floor(d / step) % 2 === 0 ? '#e8c687' : '#d7aa6d'
      if (ctx.roundRect) { ctx.beginPath(); ctx.roundRect(-11, -16, 22, 32, 5); ctx.fill() }
      else ctx.fillRect(-11, -16, 22, 32)
      ctx.strokeStyle = 'rgba(87,56,38,.18)'; ctx.lineWidth = 1
      ctx.strokeRect(-11, -16, 22, 32)
      ctx.restore()
    }
  }

  drawBandProps(ctx) {
    // 紧凑赛道的章节标题与轻量装饰：保持全景清爽、让 7 个区域一眼可辨
    ZONES.forEach((zone, index) => {
      const { y, h } = zone.tile
      ctx.fillStyle = 'rgba(15,23,42,0.14)'
      ctx.fillRect(0, y, WORLD.width, h)
      ctx.fillStyle = 'rgba(255,255,255,0.72)'
      ctx.font = 'bold 18px sans-serif'
      ctx.textAlign = 'left'
      ctx.fillText(`${index + 1}. ${zone.name}`, 24, y + 28)
      ctx.globalAlpha = 0.22
      ctx.fillStyle = '#ffffff'
      for (let i = 0; i < 10; i++) {
        const x = 48 + ((i * 131 + index * 79) % 980)
        const yy = y + 42 + ((i * 37 + index * 11) % Math.max(32, h - 50))
        ctx.beginPath(); ctx.arc(x, yy, 2 + (i % 3), 0, Math.PI * 2); ctx.fill()
      }
      ctx.globalAlpha = 1
    })
    return

    // 道具统一列 P=78（避开 5 条路径列），区域道具用 Kenney 像素素材 + 少量手绘地标
    const T = this.sprites?.tiles
    const D = this.sprites?.decor
    const P = 78
    // village：房子 + 树 + 花丛 + 草
    if (T.house) ctx.drawImage(T.house[0], P - 8, 1760)
    if (T.house2) ctx.drawImage(T.house2[0], P - 4, 1872)
    if (T.tree) { ctx.drawImage(T.tree[0], P - 20, 1804); ctx.drawImage(T.tree[0], P - 12, 1924) }
    const flowers = [D.flower, D.flowerRed, D.flowerPurple, D.tuft]
    for (let i = 0; i < 9; i++) {
      const art = flowers[i % 4]
      if (!art) continue
      const fx = P + (i % 2) * 16 - 8
      const fy = 1818 + i * 25
      if (fy > 2036) break
      ctx.drawImage(art, fx, fy)
    }
    // sand：仙人掌 + 岩石 + 沙丘
    if (D.cactus) { ctx.drawImage(D.cactus, P, 1490); ctx.drawImage(D.cactus, P - 8, 1605) }
    if (D.rockFlat) { ctx.drawImage(D.rockFlat, P, 1560); ctx.drawImage(D.rockFlat, P + 10, 1690) }
    ctx.strokeStyle = '#e8c879'; ctx.lineWidth = 5; ctx.lineCap = 'round'
    ctx.beginPath(); ctx.ellipse(P, 1735, 22, 7, 0, 0, Math.PI); ctx.stroke()
    // mine：岩石群 + 矿车
    if (D.rockC) { ctx.drawImage(D.rockC, P, 1200); ctx.drawImage(D.rockC, P - 10, 1310); ctx.drawImage(D.rockC, P + 6, 1390) }
    drawMinecart(ctx, P, 1440)
    // castle：城堡 + 树
    drawCastle(ctx, P, 950); drawCastle(ctx, P, 1080)
    if (T.tree2) ctx.drawImage(T.tree2[0], P - 12, 1005)
    // town：小屋群 + 树
    if (T.house) ctx.drawImage(T.house[0], P - 16, 620)
    if (T.house2) ctx.drawImage(T.house2[0], P + 8, 725)
    if (T.house) ctx.drawImage(T.house[0], P - 4, 838)
    if (T.tree) ctx.drawImage(T.tree[0], P + 12, 780)
    // snow：雪丘 + 雪人 + 覆雪岩
    drawMound(ctx, P, 340, 22); drawMound(ctx, P, 420, 18); drawMound(ctx, P, 495, 14)
    drawSnowman(ctx, P, 556)
    if (D.rockFlat) ctx.drawImage(D.rockFlat, P - 8, 380)
    // sky：云 + 彩虹 + 浮岛
    drawCloud(ctx, P, 60); drawCloud(ctx, P, 140); drawCloud(ctx, P, 220)
    drawRainbow(ctx, P, 265)
    ctx.fillStyle = '#7f9c8a'
    ctx.beginPath(); ctx.ellipse(P, 96, 36, 13, 0, 0, Math.PI * 2); ctx.fill()
    ctx.fillStyle = '#6b8a76'
    ctx.beginPath(); ctx.ellipse(P, 96, 22, 8, 0, 0, Math.PI * 2); ctx.fill()
  }

  draw() {
    const ctx = this.ctx
    const s = this.camera.scale
    const dpr = this.deviceScale
    const cw = this.canvas.width / s / dpr
    const ch = this.canvas.height / s / dpr
    const pp = pathPointAt(this.d)
    this.clampCamera()
    const camX = this.camera.x
    const camY = this.camera.y
    ctx.fillStyle = '#0f172a'
    ctx.fillRect(0, 0, this.canvas.width, this.canvas.height)
    ctx.save()
    ctx.scale(s * dpr, s * dpr)
    ctx.translate(-camX, -camY)
    // 背景：预渲染图（色带+波浪分界+道具+路径双描边）
    ctx.drawImage(this.bgCanvas, 0, 0)
    // 节点（42 个，全部视口裁剪）
    for (let i = 0; i < NODES.length; i++) {
      const n = NODES[i]
      const p = this.nodePos(i)
      if (p.x < camX - 60 || p.x > camX + cw + 60 || p.y < camY - 110 || p.y > camY + ch + 90) continue
      const st = this.statusOf(n.levelId)
      const bob = st === 'LEARNING' ? Math.sin((this.now || 0) / 250) * 4 : 0
      const cy = p.y + bob
      const lm = this.sprites?.landmarks?.[n.landmark]
      // 当前关卡是一面呼吸旗帜；已完成是奖章；未解锁则是被封印的路标。
      if (st === 'LEARNING') {
        const pulse = Math.sin((this.now || 0) / 250) * 3
        const grad = ctx.createLinearGradient(p.x, cy - 6, p.x, cy - 90)
        grad.addColorStop(0, 'rgba(250,204,21,0.3)')
        grad.addColorStop(1, 'rgba(245,158,11,0)')
        ctx.fillStyle = grad
        ctx.beginPath()
        ctx.moveTo(p.x - 16, cy - 5); ctx.lineTo(p.x - 26 - pulse, cy - 90); ctx.lineTo(p.x + 26 + pulse, cy - 90); ctx.lineTo(p.x + 16, cy - 5)
        ctx.closePath(); ctx.fill()
      }
      const marker = st === 'COMPLETED'
        ? { fill: '#174d4a', stroke: '#f6c85f', text: '#fff7d6' }
        : st === 'LEARNING'
          ? { fill: '#5b3ab5', stroke: '#fde68a', text: '#fff7d6' }
          : { fill: '#465266', stroke: '#94a3b8', text: '#dbe4ef' }
      ctx.save()
      ctx.shadowColor = st === 'LEARNING' ? 'rgba(250,204,21,.7)' : 'rgba(15,23,42,.35)'
      ctx.shadowBlur = st === 'LEARNING' ? 14 : 5
      hexagon(ctx, p.x, cy, 27)
      ctx.fillStyle = marker.fill; ctx.fill()
      ctx.lineWidth = 3; ctx.strokeStyle = marker.stroke; ctx.stroke()
      ctx.restore()
      if (lm) {
        if (st === 'LOCKED') ctx.globalAlpha = 0.45
        ctx.drawImage(lm, p.x - 16, cy - 17)
        ctx.globalAlpha = 1
      }
      if (st === 'COMPLETED') drawStar(ctx, p.x + 23, cy - 23, 8, '#facc15')
      if (st === 'LEARNING') {
        ctx.strokeStyle = '#fef3c7'; ctx.lineWidth = 3
        ctx.beginPath(); ctx.moveTo(p.x - 18, cy + 24); ctx.lineTo(p.x - 18, cy - 40); ctx.stroke()
        ctx.fillStyle = '#f97316'
        ctx.beginPath(); ctx.moveTo(p.x - 17, cy - 39); ctx.lineTo(p.x + 12, cy - 31); ctx.lineTo(p.x - 17, cy - 18); ctx.closePath(); ctx.fill()
      }
      if (st === 'LOCKED' && this.sprites?.fx?.lock) ctx.drawImage(this.sprites.fx.lock, p.x - 8, cy - 9)
      ctx.font = 'bold 12px monospace'; ctx.textAlign = 'center'; ctx.textBaseline = 'middle'
      ctx.fillStyle = marker.text
      ctx.fillText(String(i + 1), p.x, cy + 1)
      // 名字铭牌：置于徽章下方
      const name = this.nodeNames.get(n.levelId) || ''
      if (name) {
        ctx.font = 'bold 12px monospace'
        const w = ctx.measureText(name).width + 18
        const tagX = p.x - w / 2, tagY = cy + 35
        ctx.fillStyle = 'rgba(15,23,42,.78)'
        if (ctx.roundRect) { ctx.beginPath(); ctx.roundRect(tagX, tagY, w, 20, 10); ctx.fill() }
        else ctx.fillRect(tagX, tagY, w, 20)
        ctx.fillStyle = '#f8fafc'
        ctx.fillText(name, p.x, tagY + 11)
      }
      // 同门 / 徒弟位置：同关卡时从左侧依次排开，避免居中堆叠遮挡关卡。
      const runners = this.runners.filter(runner => Number(runner.currentLevelId) === Number(n.levelId))
      if (runners.length) {
        const shown = runners.slice(0, 3)
        const baseX = p.x - 31
        const runnerColor = this.runnerMode === 'apprentices' ? '#f97316' : '#8b5cf6'
        shown.forEach((runner, j) => {
          const fx = baseX + j * 24
          const fy = cy - 52
          hexagon(ctx, fx, fy, 12)
          ctx.fillStyle = runnerColor; ctx.fill()
          ctx.lineWidth = 2; ctx.strokeStyle = '#ffffff'; ctx.stroke()
          ctx.font = 'bold 10px sans-serif'; ctx.textAlign = 'center'; ctx.textBaseline = 'middle'
          ctx.fillStyle = '#ffffff'
          ctx.fillText(String(runner.name || runner.username || '?').slice(0, 1).toUpperCase(), fx, fy + .5)
        })
        if (runners.length > shown.length) {
          const countX = baseX + shown.length * 24
          ctx.fillStyle = '#334155'
          if (ctx.roundRect) { ctx.beginPath(); ctx.roundRect(countX - 10, cy - 62, 24, 20, 8); ctx.fill() }
          else ctx.fillRect(countX - 10, cy - 62, 24, 20)
          ctx.font = 'bold 9px sans-serif'; ctx.fillStyle = '#ffffff'
          ctx.fillText('+' + (runners.length - shown.length), countX + 2, cy - 52)
        }
      }
    }
    // 角色：像素小人（四向行走动画 + 脚下阴影 + 扬尘）
    const spPlayer = this.sprites?.player
    if (spPlayer) {
      const frames = spPlayer[this.facing] || spPlayer.down
      const walk = this.walkTimer > 0
      const frame = walk ? frames[Math.floor(this.time * 10) % 3] : frames[0]
      ctx.save()
      ctx.fillStyle = 'rgba(0,0,0,0.22)'
      ctx.beginPath(); ctx.ellipse(pp.x, pp.y + 15, 13, 5, 0, 0, Math.PI * 2); ctx.fill()
      if (frame) ctx.drawImage(frame, Math.round(pp.x - 16), Math.round(pp.y - 16))
      if (walk) {
        const ph = Math.sin(this.time * 18) * 4
        ctx.fillStyle = 'rgba(0,0,0,0.12)'
        ctx.beginPath(); ctx.arc(pp.x - 14 + ph, pp.y + 17, 3, 0, Math.PI * 2); ctx.fill()
        ctx.beginPath(); ctx.arc(pp.x + 14 - ph, pp.y + 17, 3, 0, Math.PI * 2); ctx.fill()
      }
      ctx.restore()
    }
    // 提示条（nearestNode() 目标）
    const tgt = this.nearestNode()
    if (tgt) {
      const ti = NODES.findIndex(n => n.levelId === tgt.levelId)
      const tp = this.nodePos(ti)
      const barY = tp.y - 116   // 光柱上方
      const text = this.statusOf(tgt.levelId) === 'LOCKED' ? '按 E · 前置关卡未通关' : '按 E 进入关卡'
      ctx.font = 'bold 14px monospace'
      const w = ctx.measureText(text).width + 26
      const bx = tp.x - w / 2
      ctx.fillStyle = 'rgba(15,23,42,0.78)'
      if (ctx.roundRect) { ctx.beginPath(); ctx.roundRect(bx, barY, w, 24, 12); ctx.fill() }
      else ctx.fillRect(bx, barY, w, 24)
      ctx.fillStyle = '#ffffff'
      ctx.fillText(text, tp.x, barY + 12)
    }
    ctx.restore()
  }

  bindInput() {
    if (this._inputBound) return
    this._inputBound = true
    this._keyDown = e => {
      const k = e.key.toLowerCase()
      if (['a', 'd', 'w', 's', 'arrowleft', 'arrowright', 'arrowup', 'arrowdown'].includes(k)) this.keys[k] = true
      if (k === 'e') this.interact()
    }
    this._keyUp = e => { this.keys[e.key.toLowerCase()] = false }
    this._click = e => {
      if (Date.now() < (this._ignoreClickUntil || 0)) return
      const rect = this.canvas.getBoundingClientRect()
      const ratio = this.canvas.width / rect.width
      // T3 修复（T2 Low-1）：draw() 变换为 scale(s*dpr) 后 translate(-cam)，逆变换需再除 deviceScale
      // ratio = 设备像素/CSS 像素 = deviceScale，此处显式除 this.deviceScale（dpr 场景下点击命中才准确）
      const wx = (e.clientX - rect.left) * ratio / this.camera.scale / this.deviceScale + this.camera.x
      const wy = (e.clientY - rect.top) * ratio / this.camera.scale / this.deviceScale + this.camera.y
      const node = this.nodeAt(wx, wy)
      const proj = pathProject(wx, wy)
      if (proj) this.walkToD(proj.d)
      if (node) {
        this.pendingEmit = {
          ev: this.statusOf(node.levelId) === 'LOCKED' ? 'click-node' : 'enter-node',
          levelId: node.levelId
        }
      }
    }
    this._pointerDown = e => {
      this._dragStart = { x: e.clientX, y: e.clientY }
      this._dragged = false
      this.canvas.setPointerCapture?.(e.pointerId)
    }
    this._pointerMove = e => {
      if (!this._dragStart) return
      const dx = e.clientX - this._dragStart.x
      const dy = e.clientY - this._dragStart.y
      if (Math.abs(dx) + Math.abs(dy) > 3) this._dragged = true
      if (this._dragged) {
        this.panBy(dx, dy)
        this._dragStart = { x: e.clientX, y: e.clientY }
      }
    }
    this._pointerUp = e => {
      if (this._dragged) this._ignoreClickUntil = Date.now() + 80
      this._dragStart = null
      this._dragged = false
      this.canvas.releasePointerCapture?.(e.pointerId)
    }
    this._wheel = e => {
      e.preventDefault()
      this.zoomAt(e.deltaY > 0 ? 0.9 : 1.1, e.clientX, e.clientY)
    }
    window.addEventListener('keydown', this._keyDown)
    window.addEventListener('keyup', this._keyUp)
    this.canvas.addEventListener('click', this._click)
    this.canvas.addEventListener('pointerdown', this._pointerDown)
    this.canvas.addEventListener('pointermove', this._pointerMove)
    this.canvas.addEventListener('pointerup', this._pointerUp)
    this.canvas.addEventListener('pointercancel', this._pointerUp)
    this.canvas.addEventListener('wheel', this._wheel, { passive: false })
  }

  unbindInput() {
    if (!this._inputBound) return
    this._inputBound = false
    window.removeEventListener('keydown', this._keyDown)
    window.removeEventListener('keyup', this._keyUp)
    this.canvas.removeEventListener('click', this._click)
    this.canvas.removeEventListener('pointerdown', this._pointerDown)
    this.canvas.removeEventListener('pointermove', this._pointerMove)
    this.canvas.removeEventListener('pointerup', this._pointerUp)
    this.canvas.removeEventListener('pointercancel', this._pointerUp)
    this.canvas.removeEventListener('wheel', this._wheel)
    this._keyDown = null
    this._keyUp = null
    this._click = null
    this._pointerDown = null
    this._pointerMove = null
    this._pointerUp = null
    this._wheel = null
  }
}
