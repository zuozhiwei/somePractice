// Navigation coordinates are authored against the 1536 × 1024 island artwork.
// A character's feet can only occupy these road corridors, including the bridge.
export const WORLD = { width: 1536, height: 1024 }
export const SPAWN = { x: 350, y: 590 }

const roads = [
  [[315,537],[330,565],[350,590],[410,603],[470,603],[525,594],[550,585],[600,556],[631,512],[650,482],[680,483]],
  [[680,483],[665,446],[655,418],[600,393],[549,371],[520,346],[527,318],[552,283],[576,262],[602,261],[620,240],[640,228],[660,221],[688,220],[716,220],[744,222],[772,225],[800,230],[828,236],[852,244],[872,252],[887,260],[894,272],[905,280],[920,287],[940,288],[960,284],[980,287],[996,291],[983,307],[986,331],[990,353],[999,365]],
  [[680,483],[700,480],[747,490],[790,483],[845,460],[883,442],[912,421],[937,392],[960,380],[999,365]],
  [[912,421],[940,432],[958,448],[960,486],[960,529],[965,565],[1002,597],[1053,620]],
  [[996,291],[1012,290],[1034,278],[1048,269],[1066,257],[1082,246],[1098,242],[1112,231],[1120,216],[1132,205],[1146,196],[1157,190],[1174,186],[1190,188],[1206,192],[1218,197],[1225,204],[1234,214],[1250,222],[1270,222],[1288,214],[1303,209],[1313,199],[1318,182],[1327,172]],
]

export const LOCATIONS = [
  { id: 'harbor', name: '旅人酒馆', x: 330, y: 565, greeting: '欢迎回来，旅人。先歇歇脚，看看今天留给你的委托。' },
  { id: 'windmill', name: '风车小屋', x: 520, y: 346, greeting: '风车又转起来了。屋主递来一张新的委托单。' },
  { id: 'bridge', name: '林间驿站', x: 602, y: 261, greeting: '驿站的信使等候多时，这封委托是专门留给你的。' },
  { id: 'lake', name: '湖畔营地', x: 747, y: 490, greeting: '篝火还温热着。营地的伙伴为你准备好了下一项挑战。' },
  { id: 'mountain', name: '山麓哨所', x: 1098, y: 242, greeting: '守望者从山路归来，把一份委托交到你手中。' },
  { id: 'castle', name: '古堡庭院', x: 1327, y: 172, greeting: '古堡的大门为每位旅人敞开。管家取出了属于你的委托。' },
]

const nodeMap = new Map()
const pointId = ([x, y]) => `${x},${y}`
for (const road of roads) {
  for (const [x, y] of road) nodeMap.set(pointId([x,y]), { id: pointId([x,y]), x, y })
}
export const ROAD_NODES = [...nodeMap.values()]
export const ROAD_SEGMENTS = roads.flatMap(road => road.slice(1).map((point, i) => ({
  from: nodeMap.get(pointId(road[i])), to: nodeMap.get(pointId(point)),
  radius: road[i][0] < 410 && road[i][1] > 530 ? 16 : 9,
})))

const distance = (a, b) => Math.hypot(a.x - b.x, a.y - b.y)

export function projectToRoad(point) {
  let best = null
  ROAD_SEGMENTS.forEach((segment, index) => {
    const { from, to } = segment
    const dx = to.x - from.x, dy = to.y - from.y
    const t = Math.max(0, Math.min(1, ((point.x - from.x) * dx + (point.y - from.y) * dy) / (dx * dx + dy * dy)))
    const projected = { x: from.x + t * dx, y: from.y + t * dy }
    const gap = distance(point, projected)
    if (!best || gap < best.distance) best = { ...projected, distance: gap, segment: index }
  })
  return best
}

export function isRoadPosition(point) {
  return ROAD_SEGMENTS.some(segment => {
    const dx = segment.to.x - segment.from.x, dy = segment.to.y - segment.from.y
    const t = Math.max(0, Math.min(1, ((point.x - segment.from.x) * dx + (point.y - segment.from.y) * dy) / (dx * dx + dy * dy)))
    return Math.hypot(point.x - segment.from.x - dx * t, point.y - segment.from.y - dy * t) <= segment.radius
  })
}

/** Dijkstra on the authored roads; start/end are virtual nodes on their segments. */
export function findRoadRoute(start, destination) {
  const from = projectToRoad(start), to = projectToRoad(destination)
  if (!from || !to || from.distance > 25 || to.distance > 60) return null
  const graph = new Map(ROAD_NODES.map(node => [node.id, []]))
  graph.set('start', []); graph.set('end', [])
  const connect = (a, b, cost) => { graph.get(a).push({ id: b, cost }); graph.get(b).push({ id: a, cost }) }
  for (const segment of ROAD_SEGMENTS) connect(segment.from.id, segment.to.id, distance(segment.from, segment.to))
  for (const [id, projection] of [['start', from], ['end', to]]) {
    const segment = ROAD_SEGMENTS[projection.segment]
    connect(id, segment.from.id, distance(projection, segment.from))
    connect(id, segment.to.id, distance(projection, segment.to))
  }
  if (from.segment === to.segment) connect('start', 'end', distance(from, to))
  const costs = new Map([['start', 0]]), previous = new Map(), pending = new Set(graph.keys())
  while (pending.size) {
    const current = [...pending].reduce((best, id) => (costs.get(id) ?? Infinity) < (costs.get(best) ?? Infinity) ? id : best)
    if (!costs.has(current)) return null
    pending.delete(current)
    if (current === 'end') break
    for (const edge of graph.get(current)) {
      const cost = costs.get(current) + edge.cost
      if (cost < (costs.get(edge.id) ?? Infinity)) { costs.set(edge.id, cost); previous.set(edge.id, current) }
    }
  }
  const ids = ['end']
  while (ids[0] !== 'start') { const id = previous.get(ids[0]); if (!id) return null; ids.unshift(id) }
  const positions = new Map(ROAD_NODES.map(node => [node.id, node]))
  positions.set('start', from); positions.set('end', to)
  return ids.map(id => ({ x: positions.get(id).x, y: positions.get(id).y }))
}
