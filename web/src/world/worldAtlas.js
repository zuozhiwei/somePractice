// The graph describes where a traveler can walk. Course prerequisites are a separate server decision.
const TAU = Math.PI * 2
const hexSteps = [[0,-1],[-1,0],[-1,1],[0,1],[1,0],[1,-1]]
// Surveyed routes of the seven settlements, in clockwise order around their meeting place.
const settlementRoads = [
  [[-128,48],[-131,-31],[-34,-79],[89,-55],[146,15],[49,81]],
  [[-118,50],[-145,-15],[-78,-75],[37,-87],[137,-22],[79,69]],
  [[-99,69],[-150,-5],[-64,-64],[57,-83],[142,-10],[61,67]],
  [[-145,16],[-112,-58],[4,-81],[116,-55],[125,36],[-21,81]],
  [[-133,39],[-117,-43],[-16,-75],[104,-39],[143,24],[17,88]],
  [[-93,66],[-147,-2],[-55,-69],[56,-83],[149,-24],[92,55]],
  [[-115,52],[-142,-30],[-47,-80],[87,-66],[144,11],[28,81]],
]
const distance = (a, b) => Math.hypot(a.x - b.x, a.y - b.y)
const point = ({ x, y }) => ({ x, y })

function hexSites(count) {
  const sites = [[0, 0]]
  for (let ring = 1; sites.length < count; ring++) {
    let q = ring, r = 0
    for (const [dq, dr] of hexSteps) {
      for (let step = 0; step < ring; step++) {
        sites.push([q, r])
        q += dq
        r += dr
      }
    }
  }
  return sites.slice(0, count)
}

function ringsOf(levels) {
  const rings = []
  let offset = 0
  for (let ring = 1; offset < levels.length; ring++) {
    const group = levels.slice(offset, offset + 6 * ring)
    rings.push(group)
    offset += group.length
  }
  return rings
}

function buildRegions(chapters) {
  const sites = hexSites(chapters.length)
  const regions = chapters.map((chapter, index) => {
    const rings = ringsOf(chapter.levels || [])
    const width = 410 + 216 * Math.max(0, rings.length - 1)
    const [q, r] = sites[index]
    return { chapterId: chapter.id, index, q, r, width, height: width * .71, rings }
  })
  const cell = regions.reduce((max, region) => Math.max(max, region.width), 410) * 1.10
  for (const region of regions) {
    region.x = (region.q + region.r / 2) * cell + 9 * Math.sin(region.index * 3)
    region.y = region.r * cell * .57 + 6 * Math.sin(region.index * 5)
  }
  if (!regions.length) return { width: 510, height: 360, regions }
  const left = Math.min(...regions.map(r => r.x - r.width / 2)) - 30
  const top = Math.min(...regions.map(r => r.y - r.height / 2)) - 54
  let width = 0, height = 0
  for (const region of regions) {
    region.x -= left
    region.y -= top
    region.labelX = region.x
    region.labelY = region.y - region.height * .48
    width = Math.max(width, region.x + region.width / 2 + 30)
    height = Math.max(height, region.y + region.height / 2 + 36)
  }
  return { width: Math.ceil(width), height: Math.ceil(height), regions }
}

function newEdge(a, b, crossing, index) {
  const dx = b.x - a.x, dy = b.y - a.y
  const bend = (index % 2 ? 1 : -1) * (crossing ? .025 : .052)
  const points = [
    point(a),
    { x: a.x + dx * .32 - dy * bend, y: a.y + dy * .32 + dx * bend },
    { x: a.x + dx * .68 + dy * bend * .35, y: a.y + dy * .68 - dx * bend * .35 },
    point(b),
  ]
  const lengths = [0]
  for (let i = 1; i < points.length; i++) lengths.push(lengths[i - 1] + distance(points[i - 1], points[i]))
  return { from: a.id, to: b.id, points, lengths, length: lengths.at(-1), crossing }
}

function closest(nodes, target) {
  return nodes.reduce((best, node) => distance(node, target) < distance(best, target) ? node : best)
}

export function buildAtlas(chapters) {
  const atlas = buildRegions(chapters)
  atlas.nodes = []
  atlas.edges = []
  atlas.nodeById = new Map()
  atlas.adjacent = new Map()
  function addNode(node) {
    atlas.nodes.push(node)
    atlas.nodeById.set(node.id, node)
    atlas.adjacent.set(node.id, [])
    return node
  }
  function connect(a, b, crossing = false) {
    if (a.id === b.id) return
    const edge = newEdge(a, b, crossing, atlas.edges.length)
    atlas.edges.push(edge)
    atlas.adjacent.get(a.id).push({ id: b.id, edge })
    atlas.adjacent.get(b.id).push({ id: a.id, edge })
  }
  const ports = []
  for (const region of atlas.regions) {
    const { index, chapterId, x, y, rings } = region
    const hub = addNode({ id: 'hub:' + chapterId, chapterId, kind: 'hub', x, y: y + 6 })
    let previous = [hub], number = 0
    for (const [ring, group] of rings.entries()) {
      const nodes = group.map((level, i) => {
        const angle = 2.57 + TAU * i / group.length + .13 * Math.sin(index * 4 + i * 3) + (ring % 2) * .3
        const [offsetX, offsetY] = ring === 0 && group.length === 6
          ? settlementRoads[index % settlementRoads.length][i]
          : [(135 + 90 * ring) * Math.cos(angle), (83 + 58 * ring) * Math.sin(angle)]
        return addNode({
          id: 'level:' + level.id, levelId: level.id, chapterId, kind: 'level', number: ++number,
          x: x + offsetX, y: y + offsetY,
        })
      })
      for (let i = 0; i < nodes.length; i++) {
        if (i > 0) connect(nodes[i - 1], nodes[i])
        if (ring || i % 2 === 0) connect(nodes[i], closest(previous, nodes[i]))
      }
      if (nodes.length > 2) connect(nodes.at(-1), nodes[0])
      previous = nodes
    }
    ports.push(previous)
    delete region.rings
  }
  // Three forward neighbors cover every shared side of the hexagonal region grid once.
  const bySite = new Map(atlas.regions.map(region => [[region.q, region.r].join(','), region]))
  for (const region of atlas.regions) {
    for (const [dq, dr] of [[1,0],[0,1],[1,-1]]) {
      const neighbor = bySite.get([region.q + dq, region.r + dr].join(','))
      if (!neighbor) continue
      const a = closest(ports[region.index], neighbor)
      const b = closest(ports[neighbor.index], a)
      connect(a, b, true)
    }
  }
  return atlas
}

export function directionalNeighbor(atlas, fromId, dx, dy) {
  const origin = atlas.nodeById.get(fromId)
  if (!origin || (!dx && !dy)) return null
  let best = null, bestScore = .30
  for (const { id } of atlas.adjacent.get(fromId)) {
    const next = atlas.nodeById.get(id)
    const score = ((next.x - origin.x) * dx + (next.y - origin.y) * dy) / (distance(next, origin) * Math.hypot(dx,dy))
    if (score > bestScore) {
      best = next
      bestScore = score
    }
  }
  return best
}

export function routeBetween(atlas, fromId, toId) {
  if (!atlas.nodeById.has(fromId) || !atlas.nodeById.has(toId)) return []
  const frontier = new Set([fromId]), cost = new Map([[fromId, 0]]), via = new Map()
  while (frontier.size) {
    let current = null
    for (const id of frontier) {
      if (current === null || cost.get(id) < cost.get(current)) current = id
    }
    if (current === toId) break
    frontier.delete(current)
    for (const { id, edge } of atlas.adjacent.get(current)) {
      const nextCost = cost.get(current) + edge.length
      if (nextCost >= (cost.get(id) ?? Infinity)) continue
      cost.set(id, nextCost)
      via.set(id, { previous: current, edge })
      frontier.add(id)
    }
  }
  if (!cost.has(toId)) return []
  const pieces = []
  for (let end = toId; end !== fromId;) {
    const { previous, edge } = via.get(end)
    pieces.push(edge.from === previous ? edge.points.slice(1) : edge.points.slice(0, -1).reverse())
    end = previous
  }
  return [point(atlas.nodeById.get(fromId)), ...pieces.reverse().flat()]
}

export function nearestOnRoad(atlas, x, y, radius = Infinity) {
  let gap = radius, nearest = null
  for (const edge of atlas.edges) {
    for (let i = 1; i < edge.points.length; i++) {
      const a = edge.points[i - 1], b = edge.points[i]
      const dx = b.x - a.x, dy = b.y - a.y
      const ratio = Math.max(0, Math.min(1, ((x - a.x) * dx + (y - a.y) * dy) / (dx * dx + dy * dy)))
      const px = a.x + ratio * dx, py = a.y + ratio * dy
      const nextGap = Math.hypot(x - px, y - py)
      if (nextGap > gap) continue
      const along = edge.lengths[i - 1] + ratio * (edge.lengths[i] - edge.lengths[i - 1])
      nearest = { x: px, y: py, edge, t: along / edge.length }
      gap = nextGap
    }
  }
  return nearest
}

function section(a, b) {
  const { edge } = a
  const low = Math.min(a.t, b.t) * edge.length
  const high = Math.max(a.t, b.t) * edge.length
  const inner = edge.points.filter((_, i) => edge.lengths[i] > low && edge.lengths[i] < high)
  return [point(a), ...(a.t <= b.t ? inner : inner.reverse()), point(b)]
}

function shortest(routes) {
  let result = [], distanceSoFar = Infinity
  for (const route of routes) {
    if (!route.length) continue
    const length = route.reduce((sum, p, i) => sum + (i ? distance(p, route[i - 1]) : 0), 0)
    if (length < distanceSoFar) {
      distanceSoFar = length
      result = route
    }
  }
  return result.filter((p, i) => !i || distance(result[i - 1], p) > .000001)
}

function exits(hit) {
  const { edge } = hit
  return [
    { id: edge.from, path: section(hit, { ...edge.points[0], t: 0 }) },
    { id: edge.to, path: section(hit, { ...edge.points.at(-1), t: 1 }) },
  ]
}

export function routeToRoad(atlas, fromId, hit) {
  if (!hit || !atlas.nodeById.has(fromId)) return []
  return shortest(exits(hit).map(({id, path}) => [...routeBetween(atlas, fromId, id), ...path.reverse()]))
}

export function routeBetweenRoads(atlas, start, end) {
  if (!start || !end) return []
  const routes = []
  if (start.edge === end.edge) routes.push(section(start, end))
  for (const source of exits(start)) {
    for (const target of exits(end)) {
      routes.push([...source.path, ...routeBetween(atlas, source.id, target.id), ...target.path.slice().reverse()])
    }
  }
  return shortest(routes)
}
