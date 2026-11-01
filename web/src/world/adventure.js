import { WORLD, SPAWN, LOCATIONS, findRoadRoute, isRoadPosition } from './island.js'

const SPEED = 145
const FRAME_WIDTH = 96, FRAME_HEIGHT = 112
const DIRECTIONS = { down: 0, left: 1, right: 2, up: 3 }

function loadImage(url) {
  return new Promise((resolve, reject) => {
    const image = new Image()
    image.onload = () => resolve(image)
    image.onerror = () => reject(new Error('游戏素材加载失败'))
    image.src = url
  })
}

// The scene owns movement and presentation. Assignments come from the server.
export class AdventureEngine {
  constructor(canvas, { background, character, onEnter, onNearby = () => {}, onHint = () => {} }) {
    this.canvas = canvas
    this.ctx = canvas.getContext('2d')
    this.assetUrls = { background, character }
    this.onEnter = onEnter
    this.onNearby = onNearby
    this.onHint = onHint
    this.player = { ...SPAWN, facing: 'down', walking: false }
    this.keys = new Set()
    this.control = { x: 0, y: 0 }
    this.route = []
    this.destination = null
    this.nearby = null
    this.paused = false
    this.running = false
    this.clock = 0
    this.walkClock = 0
    this.lastTime = 0
    this.frame = 0
    this.zoom = 1
    this.camera = { x: 0, y: 0, w: WORLD.width, h: WORLD.height }
    this.onKeyDown = event => {
      if (this.paused || !(event.target === canvas || event.target === document.body)) return
      const key = event.key.toLowerCase()
      if (['w', 'a', 's', 'd', 'arrowup', 'arrowdown', 'arrowleft', 'arrowright'].includes(key)) {
        event.preventDefault(); this.keys.add(key); this.cancelRoute()
      }
      if ((key === 'e' || key === ' ') && !event.repeat) { event.preventDefault(); this.interact() }
    }
    this.onKeyUp = event => this.keys.delete(event.key.toLowerCase())
    this.onBlur = () => { this.keys.clear(); this.control = { x: 0, y: 0 }; this.lastTime = 0 }
    this.onClick = event => {
      if (this.paused) return
      canvas.focus({ preventScroll: true })
      const rect = canvas.getBoundingClientRect()
      const point = {
        x: (event.clientX - rect.left) / rect.width * this.camera.w + this.camera.x,
        y: (event.clientY - rect.top) / rect.height * this.camera.h + this.camera.y,
      }
      const location = LOCATIONS.find(item => Math.hypot(point.x - item.x, point.y - item.y) < 34)
      this.travelTo(location || point, location || null)
    }
    this.loop = this.loop.bind(this)
  }
  async prepare() {
    const [background, character] = await Promise.all([loadImage(this.assetUrls.background), loadImage(this.assetUrls.character)])
    this.background = background; this.character = character
  }
  start() {
    if (this.running) return
    this.running = true
    this.zoom = this.canvas.clientWidth < 700 ? 2.2 : 1
    window.addEventListener('keydown', this.onKeyDown)
    window.addEventListener('keyup', this.onKeyUp)
    window.addEventListener('blur', this.onBlur)
    this.canvas.addEventListener('click', this.onClick)
    this.canvas.focus({ preventScroll: true })
    this.frame = requestAnimationFrame(this.loop)
  }
  stop() {
    this.running = false; cancelAnimationFrame(this.frame)
    window.removeEventListener('keydown', this.onKeyDown)
    window.removeEventListener('keyup', this.onKeyUp)
    window.removeEventListener('blur', this.onBlur)
    this.canvas.removeEventListener('click', this.onClick)
    this.onBlur()
  }
  cancelRoute() { this.route = []; this.destination = null }
  setControl(x, y) { this.control = { x, y }; if (x || y) this.cancelRoute() }
  setPaused(paused) { this.paused = paused; this.onBlur(); if (paused) this.cancelRoute() }
  toggleZoom() { this.zoom = this.zoom > 1 ? 1 : 2.2 }
  travelTo(point, location = null) {
    const route = findRoadRoute(this.player, point)
    if (!route) { this.onHint('沿着小路探索吧，那边暂时走不过去。'); return false }
    this.route = route; this.destination = location; this.keys.clear(); this.control = { x: 0, y: 0 }
    return true
  }
  interact() { if (!this.paused && this.nearby) this.onEnter(this.nearby) }
  update(dt) {
    this.player.walking = false
    if (this.paused) return
    const k = this.keys
    let x = this.control.x + Number(k.has('d') || k.has('arrowright')) - Number(k.has('a') || k.has('arrowleft'))
    let y = this.control.y + Number(k.has('s') || k.has('arrowdown')) - Number(k.has('w') || k.has('arrowup'))
    const keyboard = Math.hypot(x, y)
    if (keyboard) { x /= keyboard; y /= keyboard }
    let remaining = SPEED * dt
    const old = { x: this.player.x, y: this.player.y }
    while (remaining > 0) {
      let step = Math.min(2, remaining)
      if (!keyboard) {
        if (!this.route.length) break
        const waypoint = this.route[0]
        const dx = waypoint.x - this.player.x, dy = waypoint.y - this.player.y
        const gap = Math.hypot(dx, dy)
        if (gap < .1) { this.route.shift(); continue }
        x = dx / gap; y = dy / gap; step = Math.min(step, gap)
      }
      const point = { x: this.player.x + x * step, y: this.player.y + y * step }
      if (isRoadPosition(point)) { this.player.x = point.x; this.player.y = point.y }
      else if (keyboard) {
        if (isRoadPosition({ x: point.x, y: this.player.y })) this.player.x = point.x
        else if (isRoadPosition({ x: this.player.x, y: point.y })) this.player.y = point.y
      } else { this.cancelRoute(); break }
      remaining -= step
    }
    const dx = this.player.x - old.x, dy = this.player.y - old.y
    this.player.walking = Math.hypot(dx, dy) > .05
    if (this.player.walking) {
      this.walkClock += dt
      this.player.facing = Math.abs(dx) > Math.abs(dy) ? dx > 0 ? 'right' : 'left' : dy > 0 ? 'down' : 'up'
    } else this.walkClock = 0
    const nearby = LOCATIONS.find(item => Math.hypot(this.player.x - item.x, this.player.y - item.y) < 38) || null
    if (nearby?.id !== this.nearby?.id) { this.nearby = nearby; this.onNearby(nearby) }
    if (!this.route.length && this.destination) {
      const destination = this.destination; this.destination = null
      if (Math.hypot(this.player.x - destination.x, this.player.y - destination.y) < 38) this.onEnter(destination)
    }
  }
  loop(time) {
    if (!this.running) return
    const dt = Math.min((time - (this.lastTime || time)) / 1000, .05)
    this.lastTime = time; this.clock += dt
    this.update(dt); this.draw()
    this.frame = requestAnimationFrame(this.loop)
  }
  draw() {
    const ctx = this.ctx, canvas = this.canvas
    const dpr = Math.min(window.devicePixelRatio || 1, 2)
    const width = Math.max(1, Math.round(canvas.clientWidth * dpr)), height = Math.max(1, Math.round(canvas.clientHeight * dpr))
    if (canvas.width !== width || canvas.height !== height) { canvas.width = width; canvas.height = height }
    const aspect = canvas.clientWidth / Math.max(1, canvas.clientHeight)
    this.camera.h = WORLD.height / this.zoom
    this.camera.w = Math.min(WORLD.width, this.camera.h * aspect)
    this.camera.h = this.camera.w / aspect
    this.camera.x = Math.max(0, Math.min(WORLD.width - this.camera.w, this.player.x - this.camera.w / 2))
    this.camera.y = Math.max(0, Math.min(WORLD.height - this.camera.h, this.player.y - this.camera.h / 2))
    ctx.setTransform(width / this.camera.w, 0, 0, height / this.camera.h, 0, 0)
    ctx.translate(-this.camera.x, -this.camera.y)
    ctx.drawImage(this.background, 0, 0, WORLD.width, WORLD.height)
    if (this.route.length) {
      ctx.save(); ctx.beginPath(); ctx.moveTo(this.player.x, this.player.y)
      for (const point of this.route) ctx.lineTo(point.x, point.y)
      ctx.strokeStyle = '#fff1a5b0'; ctx.lineWidth = 3; ctx.setLineDash([3, 8]); ctx.lineDashOffset = -this.clock * 12
      ctx.stroke(); ctx.restore()
    }
    for (const location of LOCATIONS) {
      const active = this.nearby?.id === location.id
      ctx.save(); ctx.translate(location.x, location.y)
      ctx.fillStyle = active ? '#fff4ad' : '#f4d389'
      const bob = Math.sin(this.clock * 2.6 + location.x) * 3
      ctx.shadowColor = '#ffe58a'; ctx.shadowBlur = active ? 18 : 8
      ctx.beginPath(); ctx.moveTo(0,-39+bob); ctx.lineTo(5,-33+bob); ctx.lineTo(0,-27+bob); ctx.lineTo(-5,-33+bob); ctx.closePath(); ctx.fill()
      ctx.shadowBlur = 0
      if (active) {
        ctx.font = 'bold 13px sans-serif'; ctx.textAlign = 'center'; ctx.lineWidth = 4; ctx.strokeStyle = '#1e3934'; ctx.fillStyle = '#fff6d8'
        ctx.strokeText(location.name, 0, -53); ctx.fillText(location.name, 0, -53)
      }
      ctx.restore()
    }
    const { x, y, facing, walking } = this.player
    ctx.fillStyle = 'rgba(19,39,30,.25)'; ctx.beginPath(); ctx.ellipse(x,y,10,4,0,0,Math.PI*2); ctx.fill()
    const frame = walking ? 1 + Math.floor(this.walkClock * 7) % 2 : 0
    ctx.drawImage(this.character, frame * FRAME_WIDTH, DIRECTIONS[facing] * FRAME_HEIGHT, FRAME_WIDTH, FRAME_HEIGHT, x - 24, y - 54, 48, 56)
  }
}
