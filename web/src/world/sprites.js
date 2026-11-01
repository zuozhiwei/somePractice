import { TILE } from './data.js'
import { KENNEY } from './assets/kenney-data.js'

export const PALETTES = {}

export function pixelCanvas(pixelRows, palette, scale = 1) {
  if (!Array.isArray(pixelRows) || pixelRows.length === 0) throw new Error('pixelCanvas: empty rows')
  const h = pixelRows.length
  const w = pixelRows[0].length
  for (const row of pixelRows) {
    if (row.length !== w) throw new Error(`pixelCanvas: ragged rows (expected ${w}, got ${row.length})`)
  }
  const cv = document.createElement('canvas')
  cv.width = w * scale
  cv.height = h * scale
  const ctx = cv.getContext('2d')
  const img = ctx.createImageData(w * scale, h * scale)
  for (let y = 0; y < h; y++) {
    for (let x = 0; x < w; x++) {
      const c = palette[pixelRows[y][x]] || '#0000'
      const r = parseInt(c.slice(1, 3), 16)
      const g = parseInt(c.slice(3, 5), 16)
      const b = parseInt(c.slice(5, 7), 16)
      const a = c.length === 9 ? parseInt(c.slice(7, 9), 16) : (c === '#0000' ? 0 : 255)
      for (let dy = 0; dy < scale; dy++) {
        for (let dx = 0; dx < scale; dx++) {
          const i = ((y * scale + dy) * (w * scale) + (x * scale + dx)) * 4
          img.data[i] = r
          img.data[i + 1] = g
          img.data[i + 2] = b
          img.data[i + 3] = a
        }
      }
    }
  }
  ctx.putImageData(img, 0, 0)
  return cv
}

export function pixelDataCanvas(pixelData, scale = 1) {
  const h = pixelData.length
  const w = pixelData[0].length
  const cv = document.createElement('canvas')
  cv.width = w * scale
  cv.height = h * scale
  const ctx = cv.getContext('2d')
  const img = ctx.createImageData(w * scale, h * scale)
  for (let y = 0; y < h; y++) {
    for (let x = 0; x < w; x++) {
      const p = pixelData[y][x]
      for (let dy = 0; dy < scale; dy++) {
        for (let dx = 0; dx < scale; dx++) {
          const i = ((y * scale + dy) * (w * scale) + (x * scale + dx)) * 4
          img.data[i] = p[0]
          img.data[i + 1] = p[1]
          img.data[i + 2] = p[2]
          img.data[i + 3] = p[3]
        }
      }
    }
  }
  ctx.putImageData(img, 0, 0)
  return cv
}

function overlay(base, x, y, art) {
  const out = base.map(r => r.split(''))
  art.forEach((row, ay) => {
    for (let ax = 0; ax < row.length; ax++) {
      const ch = row[ax]
      if (ch === '.') continue
      const gx = x + ax
      const gy = y + ay
      if (gy >= 0 && gy < out.length && gx >= 0 && gx < out[gy].length) out[gy][gx] = ch
    }
  })
  return out.map(r => r.join(''))
}

// Kenney Nature Kit/RPG Urban Pack 采样色，供预览与旧引用
// 采样自 web/src/world/assets/kenney-data.js：grass #38cbab / sand #c0b29a /
// water #59b6d8 / tree foliage #42dfab / house wall #ffc999 / flower #fbaf46 /
// sign stone #5c6278。n/N（snow）保留原手工瓦片颜色，保证 snow 视觉不变。
const TILE_PAL = {
  g: '#38cbab', G: '#33bdae', f: '#f7f2dc',
  s: '#c0b29a', S: '#c6bc9f',
  r: '#8d8d8d', R: '#6f6f6f', l: '#b5b5b5',
  n: '#f4f8ff', N: '#d9e4f5',
  w: '#59b6d8', W: '#64c0e1', d: '#6bccea',
  t: '#42dfab', T: '#2f7d33', k: '#296360', L: '#66c06a',
  F: '#fbaf46', f: '#f7f2dc',
  h: '#ffc999', H: '#dc8652', r: '#c0392b', R: '#a93226', D: '#654140', o: '#ffe9a8',
  b: '#a8743f', B: '#8a5f32',
  p: '#b8b0a4', P: '#9a9284',
  A: '#c69c5a', K: '#7a4e28',
  y: '#9aa0a8', Y: '#7f858d',
  u: '#f1c40f'
}
PALETTES.tiles = TILE_PAL

// 地形/建筑/植被瓦片来自 Kenney（web/src/world/assets/kenney-data.js，16x16 像素数据 [r,g,b,a]）；
// snow 用原手工瓦片兜底（KENNEY 无 snow），视觉不变。
// 说明：grass/sand/water 为多变体数组（classifier 产出）；tree/house/flower/sign 为单 tile。
// buildSprites 统一输出「画布数组」：每类一个数组（grass 每变体一画布，单 tile 类为单元素数组），
// engine 侧统一按数组取（hash2 种子 / 时间帧）。
const KENNEY_TILE_KEYS = ['grass', 'sand', 'water', 'tree', 'tree2', 'house', 'house2', 'flower', 'flower2', 'sign']
const VARIANT_CLASSES = ['grass', 'sand', 'water']
const TILES = {}
for (const k of KENNEY_TILE_KEYS) {
  TILES[k] = KENNEY[k] ? KENNEY[k] : null
}
TILES.snow = [
  'nnnnnnnnnnnnnnnn',
  'nNnnnnnnnnnnNnnn',
  'nnnnnNnnnnnnnnnn',
  'nnnnnnnnnnNnnnnn',
  'nNnnnnnnnnnnnNnn',
  'nnnnnNnnnnnnnnnn',
  'nnnnnnnnnnnnnnnn',
  'nnnnnnnnNnnnnnnn',
  'nNnnnnnnnnnnnNnn',
  'nnnnnNnnnnnnnnnn',
  'nnnnnnnnnnnnnnnn',
  'nNnnnnnnnNnnnnnn',
  'nnnnnnnnnnnNnnnn',
  'nnnnnNnnnnnnnnnn',
  'nnnnnnnnnnnnnnnn',
  'nnnnnnnnNnnnnnnn'
]

const DECOR = ['flower', 'flowerRed', 'flowerPurple', 'tuft', 'rockFlat', 'rockC', 'cactus']

const PLAYER_PAL = {
  p: '#f0c8a0', P: '#e0b088', h: '#5a3a2a', h2: '#4a2d1f',
  e: '#2b2b2b', c: '#c0392b', c2: '#a93226', b: '#2c3e50', b2: '#1f2d3d', s: '#7f8c8d'
}
PALETTES.player = PLAYER_PAL

const FACE_DOWN = [
  '......hhhh......',
  '.....hhhhhh.....',
  '.....pppppp.....',
  '.....peppep.....',
  '.....pppppp.....',
  '......pppp......',
  '.....cccccc.....',
  '....cccccccc....',
  '....ccccc2cc....',
  '....cccccccc....',
  '.....cccccc.....'
]
const FACE_UP = [
  '......hhhh......',
  '.....hhhhhh.....',
  '.....hhhhhh.....',
  '.....hhh2hh.....',
  '.....hhhhhh.....',
  '......hhhh......',
  '.....cccccc.....',
  '....cccccccc....',
  '....ccccc2cc....',
  '....cccccccc....',
  '.....cccccc.....'
]
const FACE_LEFT = [
  '..hhhhhhhh......',
  '.hhhhhhhhhh.....',
  '.ppppppppp......',
  '.peppppppp......',
  '.ppppppppp......',
  '.pppppppp.......',
  '..pppppp........',
  '..ccccccc.......',
  '.cccccccccc.....',
  '.ccccccccccc....',
  '.ccccccccccc....',
  '..bbbbbbbbb.....',
  '.bbbbbbbbbbb....',
  '.bbbbbbbbbbb....',
  '.ssbbbbbbss.....',
  '.ss...ss........'
]
const LEGS_STAND = [
  '.....bbbbbb.....',
  '....bbbbbbbb....',
  '....bbbbbbbb....',
  '.....bssssb.....',
  '....ss....ss....'
]
const LEGS_WALK_A = [
  '.....bbbbbb.....',
  '....bbbbbbbb....',
  '...bbbb..bbbb...',
  '...bss....bss...',
  '...ss......ss...'
]
const LEGS_WALK_B = [
  '.....bbbbbb.....',
  '....bbbbbbbb....',
  '....bbbb..bbbb..',
  '..bss.....bss...',
  '..ss.......ss...'
]

function mirror(rows) {
  return rows.map(r => [...r].reverse().join(''))
}

function makePlayerFrames() {
  const down = [
    [...FACE_DOWN, ...LEGS_STAND],
    [...FACE_DOWN, ...LEGS_WALK_A],
    [...FACE_DOWN, ...LEGS_WALK_B]
  ]
  const up = [
    [...FACE_UP, ...LEGS_STAND],
    [...FACE_UP, ...LEGS_WALK_A],
    [...FACE_UP, ...LEGS_WALK_B]
  ]
  const leftStand = FACE_LEFT
  const leftA = [
    ...FACE_LEFT.slice(0, 8),
    '.ccccccccccc2...',
    '.ccccccccccc....',
    '..bbbbbbbbb.....',
    '.bbbbbbbbbbb....',
    '.bbbbbbbbbb.....',
    '.bbssbbbbss.....',
    '.ss...ss........'
  ]
  const leftB = [
    ...FACE_LEFT.slice(0, 8),
    '.ccccccccccc....',
    '.ccccccccccc2...',
    '..bbbbbbbbb.....',
    '.bbbbbbbbbb.....',
    '.bbbbbbbbbbb....',
    '.ssbbbbbbss.....',
    '...ss...ss......'
  ]
  return {
    down,
    up,
    left: [leftStand, leftA, leftB],
    right: [leftStand, leftA, leftB].map(f => mirror(f))
  }
}

const MENTOR_ZONES = ['village', 'sand', 'mine', 'castle', 'town', 'snow', 'sky']

const M_ROBE = {
  village: '#2980b9', sand: '#8e6b3a', mine: '#6b7f8a',
  castle: '#16a085', town: '#8d6e63', snow: '#ecf0f1', sky: '#f1c40f'
}
const M_BEARD = { village: '#ecf0f1', sand: '#5a3a2a', mine: '#9aa3ab', castle: '#5a3a2a', town: '#5a3a2a', snow: '#5a3a2a', sky: '#ecf0f1' }
const M_HAIR = { mine: '#8f9aa5' }

const M_HEAD = {
  village: [
    '....aaaaaaaa....',
    '...aaaaaaaaaa...',
    '...aahhhhhhaa...',
    '...hhhhhhhhhh...',
    '...pppppppppp...',
    '..ppepppppeppp..',
    '..pppppppppppp..',
    '..vvvvvvvvvvvv..'
  ],
  sand: [
    '.....hhhhhh.....',
    '....zzzzzzzz....',
    '...zppppppppz...',
    '...pppppppppp...',
    '..pppppppppppp..',
    '..ppmmmmpppppp..',
    '..pppppppppppp..',
    '...pppppppppp...'
  ],
  mine: [
    '....qqqqqqqq....',
    '...qqqqqqqqqq...',
    '....uqqqqqqu....',
    '...HHHHHHHHHH...',
    '...pepppppppp...',
    '..pppppppppppp..',
    '..vvvvvvvvvvvv..',
    '..vvvvvvvvvvvv..'
  ],
  castle: [
    '....hhhhhhhh....',
    '...hhhhhhhhhh...',
    '...hhhhhhhhhh...',
    '...pppppppppp...',
    '..pppppppppppp..',
    '..pppepppppppp..',
    '...pppppppppp...',
    '....pppppppp....'
  ],
  town: [
    '.....aaaaa......',
    '....aaaaaaaaa...',
    '....aaaaaaaaa...',
    '....aaaaaaaaa...',
    '.....hhhhhh.....',
    '....pppppppp....',
    '...pppppppppp...',
    '...ppeppppppp...'
  ],
  snow: [
    '....hhhhhh......',
    'hh.hhhhhhhh.....',
    'hh.pppppppp.....',
    'hh.peppppepp....',
    'hh.pppppppppp...',
    'hh.pppppppppp...',
    'hh.pppppppp.....',
    '....pppppp......'
  ],
  sky: [
    '....hhhhhhhh....',
    '...hhhhhhhhhh...',
    '...pppppppppp...',
    '..pppppppppppp..',
    '..ppepppppeppp..',
    '..vvvvvvvvvvvv..',
    '..vvvvvvvvvvvv..',
    '...vvvvvvvvvv...'
  ]
}

const M_BODY = [
  '....RRRRRRRR....',
  '...RRRRRRRRRR...',
  '...RRRRRRRRRR...',
  '..RRRRRRRRRRRR..',
  '..rrRRRRRRRRrr..',
  '..WWWWWWWWWWWW..',
  '..RRRRRRRRRRRR..',
  '...RRRRRRRRRR...',
  '...RRRRRRRRRR...',
  '..RRRRRRRRRRRR..',
  '..RRRRRRRRRRRR..',
  '..rrRRRRRRRRrr..',
  '.rrRRRRRRRRRRrr.',
  '.rrrrrrrrrrrrrr.',
  '....ssssssss....',
  '....ssssssss....'
]

const M_ACCESSORY = {
  sand: {
    art: [
      '.......#.......',
      '.......#.......',
      '......##.......',
      '.....###.......',
      '.....kk........',
      '.....kk........',
      '.....kk........'
    ],
    x: 9,
    y: 9
  },
  castle: {
    art: [
      '.fff............',
      '.ff.............',
      '.fffff..........',
      '.ffff...........',
      '.fff............',
      '.kk.............',
      '.kk.............'
    ],
    x: 0,
    y: 9
  },
  town: {
    art: [
      '....gggg......',
      '...g....g.....',
      '...g.gg.g.....',
      '...gggggg.....',
      '....gggg......'
    ],
    x: 6,
    y: 11
  },
  snow: [
    {
      art: [
        '..........GG...',
        '...........zz..',
        '............z..',
        '............z..',
        '............z..',
        '...........zz..',
        '..........GG...',
        '.........zz....'
      ],
      x: 1,
      y: 8
    }
  ],
  sky: {
    art: [
      '...vvvvvvv..uu',
      '...vvvvvvv..kk',
      '...vvvvvvv..kk',
      '...vvvvvvv..kk',
      '...vvvvvvv..kk',
      '...vvvvvvv..kk',
      '...vvvvvvv..kk',
      '....vvvvv...kk',
      '....vvvvv...kk',
      '.....vvv....kk',
      '.....vvv....kk',
      '.....vvv....kk',
      '......v.....kk',
      '......v.....kk'
    ],
    x: 1,
    y: 8
  }
}

function makeMentor(zoneKey) {
  const head = M_HEAD[zoneKey]
  let full = [...head, ...M_BODY]
  const accs = Array.isArray(M_ACCESSORY[zoneKey]) ? M_ACCESSORY[zoneKey] : [M_ACCESSORY[zoneKey]]
  for (const acc of accs) {
    if (acc) full = overlay(full, acc.x, acc.y, acc.art)
  }
  const palette = {
    p: '#f0c8a0', P: '#e0b088', e: '#2b2b2b', m: '#5a3a2a',
    h: '#5a3a2a', h2: '#4a2d1f', H: '#8f9aa5', v: M_BEARD[zoneKey],
    R: M_ROBE[zoneKey], r: M_ROBE[zoneKey], W: '#d9a13b', s: '#4a4f55',
    a: '#d9b44a', z: '#b03a2e', q: '#566573', u: '#f1c40f',
    f: '#f2e6c9', k: '#7a4e28', g: '#b8860b', G: '#b8860b',
    '#': '#4a4f55', '.': '#0000'
  }
  const frames = [0, 1, 2].map(() => full)
  frames[1] = full.map((r, i) => i === 22 ? '...sssssssss....' : i === 23 ? '....ssssssss....' : r)
  frames[2] = full.map((r, i) => i === 22 ? '....ssssssss....' : i === 23 ? '...sssssssss....' : r)
  const portrait = pixelCanvas(full.slice(0, 16).map(r => r.padEnd(16, '.')), palette, 2)
  const cv = document.createElement('canvas')
  cv.width = 64
  cv.height = 64
  const ctx = cv.getContext('2d')
  ctx.strokeStyle = '#ffffff'
  ctx.lineWidth = 1
  ctx.strokeRect(16.5, 16.5, 31, 31)
  ctx.drawImage(portrait, 16, 16)
  return { body: { down: frames.map(f => pixelCanvas(f, palette, 2)) }, portrait: cv }
}

const LANDMARK_PAL = {
  A: '#c69c5a', K: '#7a4e28', u: '#f1c40f', r: '#8d8d8d', R: '#6f6f6f', l: '#b5b5b5',
  g: '#d4a017', b: '#8a5f32', Y: '#f1c40f', w: '#f5f0e1',
  f: '#f5f0e1', j: '#b8b0a4', c: '#5b616b', q: '#9aa0a8',
  '#': '#5b616b',
  z: '#e8edf2', G: '#b8860b', y: '#8d949c', V: '#5f6a75',
  o: '#7ec8f2', O: '#bfe6fa', W: '#6b6f76'
}
PALETTES.landmarks = LANDMARK_PAL

const LANDMARKS = {
  sign: [
    '................',
    '....AAAAAAA.....',
    '....AAAAAAA.....',
    '....AAuAAuA.....',
    '....AAAAAAA.....',
    '......K.........',
    '......K.........',
    '......K.........',
    '......K.........',
    '......K.........',
    '......K.........',
    '......K.........',
    '......K.........',
    '......K.........',
    '.....KKKKK......',
    '................'
  ],
  stone: [
    '................',
    '................',
    '......rr........',
    '.....rrrr.......',
    '....rrlrrr......',
    '....rrrrrr......',
    '...rrrrrrrr.....',
    '...rrrrrrrr.....',
    '..rrrRRRRrrr....',
    '..rrRRRRRRrr....',
    '..rrRRRRRRrr....',
    '..rrRRRRRRrr....',
    '...rRRRRRrr.....',
    '...RRRRRRRR.....',
    '..rRRRRRRRrr....',
    '................'
  ],
  chest: [
    '................',
    '...gggggggggg...',
    '...gggggggggg...',
    '...gwwwwwwwwg...',
    '...gggggggggg...',
    '...gggggggggg...',
    '...bbbbbbbbbb...',
    '...bbbbbbbbbb...',
    '...bbbbbbbbbb...',
    '...bbbbbbbbbb...',
    '...bbbbbbbbbb...',
    '...bbbYYYYbbb...',
    '...bbbbbbbbbb...',
    '...gggggggggg...',
    '...bbbbbbbbbb...',
    '................'
  ],
  book: [
    '................',
    '................',
    '....ffffffff....',
    '...ffffffffff...',
    '...fjffffffjf...',
    '...fjffffjjff...',
    '...ffffffffjj...',
    '...fjffffffff...',
    '...ffffffffff...',
    '....ffffffff....',
    '.....cccccc.....',
    '.....cccccc.....',
    '....cccccccc....',
    '....cccccccc....',
    '................',
    '................'
  ],
  anvil: [
    '................',
    '....#####.......',
    '...#######......',
    '...##ll###......',
    '...#######......',
    '....#####.......',
    '....#####.......',
    '....##.##.......',
    '.....###........',
    '....#####.......',
    '...#######......',
    '..########......',
    '..#########.....',
    '..#########.....',
    '................',
    '................'
  ],
  sword: [
    '................',
    '...z........z...',
    '....z......z....',
    '.....z....z.....',
    '......zzzz......',
    '.....GG..GG.....',
    '....GGGGGGGG....',
    '.....GG..GG.....',
    '......zzzz......',
    '.....z....z.....',
    '....z......z....',
    '...z........z...',
    '..z..........z..',
    '................',
    '................',
    '................'
  ],
  beacon: [
    '................',
    '....oooooo......',
    '...oooooooo.....',
    '...oOOOOoooo....',
    '....oooooo......',
    '....OOOOOO......',
    '....yyyyyy......',
    '....yyyyyy......',
    '....yyyyyy......',
    '...yyyyyyyy.....',
    '...yyyyyyyy.....',
    '...yyyyyyyy.....',
    '...yYYYYYYy.....',
    '..yYYYYYYYYYy...',
    '..yyyyyyyyyyyy..',
    '................'
  ]
}

const FX_PAL = {
  Y: '#f1c40f', l: '#f9e17a', y: '#aab2bd', d: '#5f6a75', o: '#f7f2dc', z: '#e8edf2'
}
PALETTES.fx = FX_PAL

const FX = {
  star: [
    '.......Y........',
    '......YYY.......',
    '.....YYYYY......',
    '....YYYYYYY.....',
    '...YYYYYYYYY....',
    '..YYYYYYYYYYY...',
    '..lYYYYYYYYYY...',
    '..lYYYYYYYYYY...',
    '...YYYYYYYYY....',
    '....YYYYYYY.....',
    '.....YYYYY......',
    '......YYY.......',
    '.......Y........',
    '................',
    '................',
    '................'
  ],
  lock: [
    '...yy......yy...',
    '..yyy......yyy..',
    '..yyy......yyy..',
    '...yy......yy...',
    '...yyyyyyyyyy...',
    '..yyyyyyyyyyyy..',
    '..yyyyyyyyyyyy..',
    '..yyyydddddyyy..',
    '..yyyydyddyyyy..',
    '..yyyyyddyyyyyy.',
    '..yyyyyyyyyyyy..',
    '..yyyyyyyyyyyy..',
    '...yyyyyyyyyy...',
    '................',
    '................',
    '................'
  ],
  sparkle: [
    [
      '.......Y........',
      '.......Y........',
      '.......Y........',
      '.......Y........',
      '.......Y........',
      '...YYYYYYYYY....',
      '...YYYYYYYYY....',
      '.......Y........',
      '.......Y........',
      '.......Y........',
      '.......Y........',
      '.......Y........',
      '................',
      '................',
      '................',
      '................'
    ],
    [
      '................',
      '.......Y........',
      '.......Y........',
      '.......Y........',
      '.......Y........',
      '....YYYYYYY.....',
      '....YYYYYYY.....',
      '.......Y........',
      '.......Y........',
      '.......Y........',
      '.......Y........',
      '................',
      '................',
      '................',
      '................',
      '................'
    ],
    [
      '................',
      '................',
      '.......Y........',
      '.......Y........',
      '.......Y........',
      '......YYYY......',
      '......YYYY......',
      '.......Y........',
      '.......Y........',
      '.......Y........',
      '................',
      '................',
      '................',
      '................',
      '................',
      '................'
    ]
  ]
}

export const DEBUG = {
  TILES, DECOR, PLAYER_PAL, FACE_DOWN, FACE_UP, FACE_LEFT,
  LEGS_STAND, LEGS_WALK_A, LEGS_WALK_B,
  M_HEAD, M_BODY, M_ACCESSORY, M_ROBE, M_BEARD, MENTOR_ZONES,
  LANDMARKS, LANDMARK_PAL, FX, FX_PAL, TILE_PAL
}

export function buildSprites() {
  const scale = TILE / 16
  // 统一画布数组形状：grass/sand/water 多变体 → 每变体一 32x32 画布；
  // 单 tile 键（tree/house/flower/sign）→ 单元素数组；snow（字符行手工兜底）→ 单元素数组。
  function toCanvasList(src) {
    if (!src) return null
    if (typeof src[0] === 'string') return [pixelCanvas(src, TILE_PAL, scale)]
    if (Array.isArray(src[0][0][0])) return src.map(t => pixelDataCanvas(t, scale))
    return [pixelDataCanvas(src, scale)]
  }
  const tiles = {}
  for (const [k, src] of Object.entries(TILES)) {
    tiles[k] = toCanvasList(src)
  }
  const decor = {}
  for (const k of DECOR) {
    const d = KENNEY.decor && KENNEY.decor[k]
    if (d) decor[k] = pixelDataCanvas(d, scale)
  }
  const player = makePlayerFrames()
  const mentors = {}
  for (const z of MENTOR_ZONES) mentors[z] = makeMentor(z)
  const landmarks = {}
  for (const [k, rows] of Object.entries(LANDMARKS)) {
    landmarks[k] = pixelCanvas(rows, LANDMARK_PAL, scale)
  }
  const fx = {
    star: pixelCanvas(FX.star, FX_PAL, scale),
    lock: pixelCanvas(FX.lock, FX_PAL, scale),
    sparkle: FX.sparkle.map(rows => pixelCanvas(rows, FX_PAL, 1))
  }
  return {
    tiles,
    decor,
    player: {
      down: player.down.map(f => pixelCanvas(f, PLAYER_PAL, scale)),
      up: player.up.map(f => pixelCanvas(f, PLAYER_PAL, scale)),
      left: player.left.map(f => pixelCanvas(f, PLAYER_PAL, scale)),
      right: player.right.map(f => pixelCanvas(f, PLAYER_PAL, scale))
    },
    mentors,
    landmarks,
    fx
  }
}
