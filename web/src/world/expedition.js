// Coordinates are in scene space. Each segment follows the ground or crosses a bridge.
export const WIDTH = 1000
export const HEIGHT = 456
export const STOPS_PER_PAGE = 6
export const ROAD = [
  [108,325], [133,321], [148,306], [174,302], [194,313], [215,328],
  [236,329], [254,317], [279,311], [310,311], [336,300], [395,279],
  [423,269], [438,254], [464,245], [483,238], [493,221], [519,213],
  [545,220], [565,216], [578,202], [598,194], [619,191], [647,182],
  [715,171], [740,174], [756,187], [776,192], [795,182], [806,163],
  [828,155], [845,162], [862,158], [877,147],
]
const stopIndices = [1, 8, 14, 21, 27, 33]
const lengths = ROAD.map(() => 0)
for (let i = 1; i < ROAD.length; i++) {
  lengths[i] = lengths[i - 1] + Math.hypot(ROAD[i][0] - ROAD[i - 1][0], ROAD[i][1] - ROAD[i - 1][1])
}

export function locate(distance) {
  const d = Math.max(0, Math.min(lengths.at(-1), distance))
  const index = lengths.findIndex(length => length >= d)
  if (!index) return { x: ROAD[0][0], y: ROAD[0][1], facing: 'right' }
  const [ax, ay] = ROAD[index - 1], [bx, by] = ROAD[index]
  const fraction = (d - lengths[index - 1]) / (lengths[index] - lengths[index - 1])
  return { x: ax + fraction * (bx - ax), y: ay + fraction * (by - ay), facing: 'right' }
}

export const ROAD_LENGTH = lengths.at(-1)

// Find the nearest point on the polyline, including points between authored vertices.
export function roadHit(x, y, radius) {
  let closest = null
  for (let i = 1; i < ROAD.length; i++) {
    const [ax, ay] = ROAD[i - 1], [bx, by] = ROAD[i]
    const dx = bx - ax, dy = by - ay
    const t = Math.max(0, Math.min(1, ((x - ax) * dx + (y - ay) * dy) / (dx * dx + dy * dy)))
    const gap = Math.hypot(x - ax - dx * t, y - ay - dy * t)
    if (gap <= radius && (!closest || gap < closest.gap)) {
      closest = { gap, distance: lengths[i - 1] + t * (lengths[i] - lengths[i - 1]) }
    }
  }
  return closest
}

export function chapterStops(levels, page, chapterStatus) {
  return levels.slice(page * STOPS_PER_PAGE, (page + 1) * STOPS_PER_PAGE).map((level, i, group) => {
    const slot = group.length === 1 ? 0 : Math.round(i * (STOPS_PER_PAGE - 1) / (group.length - 1))
    const index = stopIndices[slot]
    const distance = lengths[index]
    return { ...level, status: chapterStatus === 'LOCKED' ? 'LOCKED' : level.status, number: page * STOPS_PER_PAGE + i + 1, distance, ...locate(distance) }
  })
}

export function currentLevel(levels, chapterStatus) {
  if (chapterStatus === 'LOCKED') return levels[0]
  return levels.find(level => level.status === 'LEARNING') || levels.find(level => level.status === 'COMPLETED') || levels[0]
}

export const STATUS = { LOCKED: '未解锁', LEARNING: '可挑战', COMPLETED: '已通关' }
