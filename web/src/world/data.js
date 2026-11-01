export const TILE = 32

export const WORLD = { width: 1100, height: 900 }

export const ZONES = [
  { key: 'village', mapId: 1, name: '新手村草原', tile: { x: 0, y: 768, w: 1100, h: 132 } },
  { key: 'sand', mapId: 2, name: '沙丘集市', tile: { x: 0, y: 640, w: 1100, h: 128 } },
  { key: 'mine', mapId: 3, name: '山地矿洞', tile: { x: 0, y: 512, w: 1100, h: 128 } },
  { key: 'castle', mapId: 4, name: '城堡王国', tile: { x: 0, y: 384, w: 1100, h: 128 } },
  { key: 'town', mapId: 5, name: '工匠城镇', tile: { x: 0, y: 256, w: 1100, h: 128 } },
  { key: 'snow', mapId: 6, name: '雪山之巅', tile: { x: 0, y: 128, w: 1100, h: 128 } },
  { key: 'sky', mapId: 7, name: '天空之城', tile: { x: 0, y: 0, w: 1100, h: 128 } }
]

export const PATH = [
  { x: 100, y: 834 }, { x: 1000, y: 834 }, { x: 1000, y: 706 },
  { x: 100, y: 706 }, { x: 100, y: 578 }, { x: 1000, y: 578 },
  { x: 1000, y: 450 }, { x: 100, y: 450 }, { x: 100, y: 322 },
  { x: 1000, y: 322 }, { x: 1000, y: 194 }, { x: 100, y: 194 },
  { x: 100, y: 66 }, { x: 1000, y: 66 }
]

function segLen(a, b) { return Math.hypot(b.x - a.x, b.y - a.y) }

export function pathLen() {
  let total = 0
  for (let i = 1; i < PATH.length; i++) total += segLen(PATH[i - 1], PATH[i])
  return total
}

export function pathPointAt(d) {
  const L = pathLen()
  let acc = 0
  for (let i = 1; i < PATH.length; i++) {
    const a = PATH[i - 1], b = PATH[i]
    const l = segLen(a, b)
    if (acc + l >= d || i === PATH.length - 1) {
      const t = l === 0 ? 0 : Math.min(1, Math.max(0, (d - acc) / l))
      const x = a.x + (b.x - a.x) * t
      const y = a.y + (b.y - a.y) * t
      const il = l === 0 ? 1 : 1 / l
      return { x, y, dx: (b.x - a.x) * il, dy: (b.y - a.y) * il }
    }
    acc += l
  }
  return { x: PATH[PATH.length - 1].x, y: PATH[PATH.length - 1].y, dx: 0, dy: -1 }
}

export function pathProject(x, y) {
  let best = null
  let acc = 0
  for (let i = 1; i < PATH.length; i++) {
    const a = PATH[i - 1], b = PATH[i]
    const abx = b.x - a.x, aby = b.y - a.y
    const l2 = abx * abx + aby * aby
    let t = l2 === 0 ? 0 : ((x - a.x) * abx + (y - a.y) * aby) / l2
    t = Math.min(1, Math.max(0, t))
    const px = a.x + abx * t, py = a.y + aby * t
    const dist = Math.hypot(x - px, y - py)
    const d = acc + Math.sqrt(l2) * t
    if (!best || dist < best.dist) best = { d, dist }
    acc += Math.sqrt(l2)
  }
  return best
}

export const NODES = [
  { levelId: 1, zone: 'village', x: 6,  y: 12, landmark: 'sign' },
  { levelId: 2, zone: 'village', x: 13, y: 10, landmark: 'book' },
  { levelId: 3, zone: 'village', x: 20, y: 12, landmark: 'anvil' },
  { levelId: 4, zone: 'village', x: 27, y: 9,  landmark: 'sign' },
  { levelId: 5, zone: 'village', x: 33, y: 11, landmark: 'chest' },
  { levelId: 6, zone: 'village', x: 37, y: 7,  landmark: 'beacon' },
  { levelId: 7,  zone: 'sand', x: 42, y: 12, landmark: 'sign' },
  { levelId: 8,  zone: 'sand', x: 49, y: 10, landmark: 'chest' },
  { levelId: 9,  zone: 'sand', x: 56, y: 12, landmark: 'book' },
  { levelId: 10, zone: 'sand', x: 63, y: 9,  landmark: 'anvil' },
  { levelId: 11, zone: 'sand', x: 69, y: 11, landmark: 'sign' },
  { levelId: 12, zone: 'sand', x: 75, y: 7,  landmark: 'beacon' },
  { levelId: 13, zone: 'mine', x: 6,  y: 24, landmark: 'anvil' },
  { levelId: 14, zone: 'mine', x: 13, y: 26, landmark: 'stone' },
  { levelId: 15, zone: 'mine', x: 20, y: 24, landmark: 'sign' },
  { levelId: 16, zone: 'mine', x: 27, y: 26, landmark: 'chest' },
  { levelId: 17, zone: 'mine', x: 33, y: 24, landmark: 'book' },
  { levelId: 18, zone: 'mine', x: 37, y: 21, landmark: 'beacon' },
  { levelId: 19, zone: 'castle', x: 42, y: 24, landmark: 'sword' },
  { levelId: 20, zone: 'castle', x: 49, y: 26, landmark: 'sign' },
  { levelId: 21, zone: 'castle', x: 56, y: 24, landmark: 'stone' },
  { levelId: 22, zone: 'castle', x: 63, y: 26, landmark: 'anvil' },
  { levelId: 23, zone: 'castle', x: 69, y: 24, landmark: 'book' },
  { levelId: 24, zone: 'castle', x: 75, y: 21, landmark: 'beacon' },
  { levelId: 25, zone: 'town', x: 6,  y: 39, landmark: 'anvil' },
  { levelId: 26, zone: 'town', x: 13, y: 41, landmark: 'sign' },
  { levelId: 27, zone: 'town', x: 20, y: 39, landmark: 'book' },
  { levelId: 28, zone: 'town', x: 27, y: 41, landmark: 'chest' },
  { levelId: 29, zone: 'town', x: 33, y: 39, landmark: 'stone' },
  { levelId: 30, zone: 'town', x: 37, y: 36, landmark: 'beacon' },
  { levelId: 31, zone: 'snow', x: 42, y: 39, landmark: 'sword' },
  { levelId: 32, zone: 'snow', x: 49, y: 41, landmark: 'sign' },
  { levelId: 33, zone: 'snow', x: 56, y: 39, landmark: 'chest' },
  { levelId: 34, zone: 'snow', x: 63, y: 41, landmark: 'anvil' },
  { levelId: 35, zone: 'snow', x: 69, y: 39, landmark: 'book' },
  { levelId: 36, zone: 'snow', x: 75, y: 36, landmark: 'beacon' },
  { levelId: 37, zone: 'sky', x: 24, y: 44, landmark: 'beacon' },
  { levelId: 38, zone: 'sky', x: 31, y: 46, landmark: 'stone' },
  { levelId: 39, zone: 'sky', x: 38, y: 44, landmark: 'chest' },
  { levelId: 40, zone: 'sky', x: 45, y: 46, landmark: 'sword' },
  { levelId: 41, zone: 'sky', x: 52, y: 44, landmark: 'sign' },
  { levelId: 42, zone: 'sky', x: 57, y: 42, landmark: 'beacon' }
]

export const MENTOR_ARCHETYPES = {
  village: { name: '村长', title: '新手村村长 · 教你把第一行代码敲响' },
  sand: { name: '铁匠', title: '沙丘铁匠铺老板 · 一锤一锤打出好代码' },
  mine: { name: '老矿匠', title: '矿洞老匠 · 挖到底层才见真金' },
  castle: { name: '军师', title: '城堡军师 · 运筹帷幄，决胜千里' },
  town: { name: '工匠大师', title: '工匠城镇大师 · 好系统是雕出来的' },
  snow: { name: '剑客', title: '雪山剑客 · 快、稳、准' },
  sky: { name: '长老', title: '天空之城长老 · 最后的试炼' }
}

export function mentorLines(map, level, tasks, mentorName) {
  const lines = [
    `${mentorName}：来了，学徒。这一带想闯出名堂，先从「${level.name}」开始。`,
    `${mentorName}：这一关的关键是——${level.description}。`
  ]
  const todo = tasks.filter(t => t.submissionStatus !== 'APPROVED')
  if (todo.length) {
    lines.push(`${mentorName}：眼下有 ${todo.length} 件作业要做：「${todo[0].title}」，做完把它交给我。`)
  } else {
    lines.push(`${mentorName}：好小子，这关的作业全交齐了。去下一个师父那儿吧。`)
  }
  lines.push(`${mentorName}：要交作业就点下面的「提交作业」。截图、笔记都行，我看过才算数。`)
  return lines
}
