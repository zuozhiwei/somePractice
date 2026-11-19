import { computed, ref, shallowRef, onBeforeUnmount, onMounted } from 'vue'

const clamp = (value, min, max) => Math.max(min, Math.min(max, value))
const midpoint = (a, b) => ({ x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 })
const separation = (a, b) => Math.hypot(a.x - b.x, a.y - b.y)

export function useAtlasCamera(viewport, getAtlas) {
  const width = ref(1), height = ref(1), scale = ref(1)
  const offset = shallowRef({ x: 0, y: 0 })
  const fit = computed(() => Math.min(width.value / getAtlas().width, height.value / getAtlas().height))
  const maxScale = computed(() => Math.max(fit.value * 1.5, 2.4))
  const overWorld = computed(() => scale.value <= fit.value * 1.06)
  const boardStyle = computed(() => ({
    width: getAtlas().width * scale.value + 'px',
    height: getAtlas().height * scale.value + 'px',
    transform: 'translate(' + offset.value.x + 'px,' + offset.value.y + 'px)'
  }))
  const center = () => ({ x: width.value / 2, y: height.value / 2 })
  function pointInWorld(pos) {
    return { x: (pos.x - offset.value.x) / scale.value, y: (pos.y - offset.value.y) / scale.value }
  }
  function eventPosition(event) {
    const rect = viewport.value.getBoundingClientRect()
    return { x: event.clientX - rect.left, y: event.clientY - rect.top }
  }
  function origin(value, available, extent) {
    return extent < available ? (available - extent) / 2 : clamp(value, available - extent, 0)
  }
  function pan(x, y) {
    offset.value = {
      x: origin(x, width.value, getAtlas().width * scale.value),
      y: origin(y, height.value, getAtlas().height * scale.value)
    }
  }
  function position(world, anchor = center()) {
    pan(anchor.x - world.x * scale.value, anchor.y - world.y * scale.value)
  }
  function setView(world, zoom, anchor = center()) {
    scale.value = clamp(zoom, fit.value, maxScale.value)
    position(world, anchor)
  }
  function zoomBy(factor, anchor = center()) {
    setView(pointInWorld(anchor), scale.value * factor, anchor)
  }
  function overview() {
    const { width: w, height: h } = getAtlas()
    setView({ x:w / 2, y:h / 2 }, fit.value)
  }
  function focusRegion(region, target = region) {
    const zoom = Math.min((width.value - 16) / region.width, (height.value - 25) / region.height)
    setView(target, Math.max(.74, zoom))
  }
  function reveal(point) {
    const x = point.x * scale.value + offset.value.x, y = point.y * scale.value + offset.value.y
    if (x < 40 || x > width.value - 40 || y < 32 || y > height.value - 28) position(point)
  }
  function resize() {
    if (!viewport.value || !viewport.value.clientWidth) return
    const wasFit = overWorld.value || width.value === 1
    const oldCenter = pointInWorld(center())
    width.value = viewport.value.clientWidth
    height.value = viewport.value.clientHeight
    if (wasFit) overview()
    else setView(oldCenter, scale.value)
  }

  const pointers = new Map()
  const dragging = ref(false)
  let drag = null, pinch = null, swallowClick = false, observer
  function start(event) {
    if (event.button !== 0) return
    const p = eventPosition(event)
    pointers.set(event.pointerId, p)
    if (pointers.size === 1) {
      swallowClick = false
      drag = { start:p, origin:offset.value }
    } else if (pointers.size === 2) {
      const [a,b] = [...pointers.values()]
      pinch = { distance:Math.max(1, separation(a,b)), world:pointInWorld(midpoint(a,b)), scale:scale.value }
      drag = null
      dragging.value = true
      for (const id of pointers.keys()) viewport.value.setPointerCapture(id)
    }
  }
  function move(event) {
    if (!pointers.has(event.pointerId)) return
    const p = eventPosition(event)
    pointers.set(event.pointerId, p)
    if (pinch && pointers.size >= 2) {
      const [a,b] = [...pointers.values()]
      setView(pinch.world, pinch.scale * separation(a,b) / pinch.distance, midpoint(a,b))
    } else if (drag) {
      if (!dragging.value && separation(drag.start, p) < 5) return
      dragging.value = true
      if (!viewport.value.hasPointerCapture(event.pointerId)) viewport.value.setPointerCapture(event.pointerId)
      pan(drag.origin.x + p.x - drag.start.x, drag.origin.y + p.y - drag.start.y)
    }
  }
  function end(event) {
    if (!pointers.delete(event.pointerId)) return
    if (dragging.value) swallowClick = true
    pinch = null
    drag = pointers.size === 1 ? { start:[...pointers.values()][0], origin:offset.value } : null
    if (!pointers.size) dragging.value = false
  }
  function cancelClick(event) {
    if (swallowClick && event.detail > 0) {
      event.stopPropagation()
      event.preventDefault()
      swallowClick = false
    }
  }
  function wheel(event) {
    if (!event.ctrlKey && !event.metaKey) return
    event.preventDefault()
    zoomBy(Math.exp(clamp(-event.deltaY * .008, -.55, .55)), eventPosition(event))
  }
  onMounted(() => {
    resize()
    observer = new ResizeObserver(resize)
    observer.observe(viewport.value)
    viewport.value.addEventListener('wheel', wheel, { passive:false })
  })
  onBeforeUnmount(() => {
    observer?.disconnect()
    viewport.value?.removeEventListener('wheel', wheel)
    pointers.clear()
  })
  return { width,height,scale,fit,maxScale,offset,overWorld,boardStyle,dragging,overview,position,focusRegion,reveal,zoomBy,eventPosition,pointInWorld,start,move,end,cancelClick }
}
