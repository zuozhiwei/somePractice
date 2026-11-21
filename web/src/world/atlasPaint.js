import { biomes, house, landmarks, peak, pine, random, ship, stone } from './coast.js'

const TAU = Math.PI * 2

function path(ctx, points, close = false) {
  ctx.beginPath()
  points.forEach((p, i) => i ? ctx.lineTo(p.x, p.y) : ctx.moveTo(p.x, p.y))
  if (close) ctx.closePath()
}

function fill(ctx, points, color) {
  path(ctx, points, true)
  ctx.fillStyle = color
  ctx.fill()
}

function stroke(ctx, points, color, width) {
  path(ctx, points)
  ctx.strokeStyle = color
  ctx.lineWidth = width
  ctx.stroke()
}

function segment(ctx, ax, ay, bx, by, color, width = 1) {
  stroke(ctx, [{x: ax, y: ay}, {x: bx, y: by}], color, width)
}

function shift(contour, dy) {
  return contour.map(p => ({x: p.x, y: p.y + dy}))
}

function expanded(contour, region, k) {
  return contour.map(p => ({x: region.x + (p.x - region.x) * k, y: region.y + (p.y - region.y) * k}))
}

function coastline(region) {
  const {x, y, width, height, index} = region
  const rng = random(index * 997 + 77)
  return Array.from({length: 60}, (_, i) => {
    const angle = i * TAU / 60
    const inlet = Math.max(0, Math.cos(angle-index*1.18-1))
    const coast = .95 + .072 * Math.cos(angle * 3 + index * 1.7) + .055 * Math.sin(angle * 5 + index)
      + .038 * Math.sin(angle * 9 + index * .3) + .065 * (rng() - .5) - .16 * inlet**18
    return {x: x + Math.cos(angle) * width / 2 * coast, y: y + Math.sin(angle) * height / 2 * coast}
  })
}

function contains(contour, x, y) {
  let inside = false
  for (let i = 0, j = contour.length - 1; i < contour.length; j = i++) {
    const a = contour[i], b = contour[j]
    if ((a.y > y) !== (b.y > y) && x < (b.x - a.x) * (y - a.y) / (b.y - a.y) + a.x) inside = !inside
  }
  return inside
}

function paintSea(ctx, atlas) {
  const {width, height} = atlas
  const sea = ctx.createLinearGradient(0, 0, width, height)
  sea.addColorStop(0, '#b5d7c1')
  sea.addColorStop(.40, '#80b9b2')
  sea.addColorStop(.78, '#83bcb9')
  sea.addColorStop(1, '#a4cabe')
  ctx.fillStyle = sea
  ctx.fillRect(0,0,width,height)
  // Soft shoal bands and clustered ripples keep open water distinct from traversable bridges.
  const rng = random(6843)
  const count = Math.min(4200, Math.ceil(width * height / 1100))
  ctx.lineWidth = 1.05
  for (let i = 0; i < count; i++) {
    const x = rng() * width, y = rng() * height, size = 3 + rng() * 8
    ctx.strokeStyle = i % 3 ? '#f3f1d469' : '#498e933d'
    ctx.beginPath()
    ctx.moveTo(x - size, y)
    ctx.quadraticCurveTo(x, y + 2.7, x + size, y - .8)
    ctx.stroke()
    if (i % 10 === 0) segment(ctx, x + 2, y + 3, x + size + 3, y + 2.7, '#f7f3d840')
  }
}

function paintLand(ctx, region, contour, biome) {
  const rng = random(8231 + region.index * 371)
  const face = [12,9,20,15,12,21,24,14][region.index % biomes.length]
  const shelf = shift(contour, 7)
  for (const [k, color] of [[1.145,'#86bdb0'],[1.10,'#a3cbb4'],[1.064,'#c4dcc0'],[1.032,'#d6dcad']]) {
    fill(ctx, expanded(shelf, region, k), color)
  }
  fill(ctx, shift(contour, face + 3), '#a8bc96')
  for (let i = 0; i < contour.length; i++) {
    const a = contour[i], b = contour[(i + 1) % contour.length]
    const colors = ['#a79770','#a8a27a','#b7a880','#c2b58b','#a99f79']
    fill(ctx, [a,b,{x:b.x,y:b.y + face},{x:a.x,y:a.y + face}], colors[(i + region.index) % colors.length])
    if (a.x > b.x && i % 3 === 0) segment(ctx,a.x - 1,a.y + 4,a.x - 2,a.y + face - 2,'#7d795d52',1.2)
  }
  path(ctx, contour, true)
  ctx.save()
  ctx.clip()
  const meadow = ctx.createLinearGradient(region.x,region.y-region.height*.5,region.x,region.y+region.height*.5)
  meadow.addColorStop(0,biome.sun)
  meadow.addColorStop(.47,biome.grass)
  meadow.addColorStop(1,biome.shade)
  ctx.fillStyle = meadow
  ctx.fillRect(region.x-region.width*.6,region.y-region.height*.6,region.width*1.2,region.height*1.2)
  // Shallow terraces follow the coast and have irregular breaks rather than radial spokes.
  for (let i = 0; i < contour.length; i += 3) {
    const panel = [0,1,2,3].map(d => contour[(i+d) % contour.length])
    const inner = panel.slice().reverse().map(p => {
      const k = .75 + .13 * rng()
      return {x: region.x + (p.x - region.x)*k, y: region.y + (p.y-region.y)*k}
    })
    fill(ctx,[...panel,...inner], i % 2 ? '#ffffd02b' : '#607b5225')
  }
  for (let i = 0; i < 135; i++) {
    const x = region.x + (rng() - .5) * region.width
    const y = region.y + (rng() - .5) * region.height
    const w = 2 + rng() * 3
    stroke(ctx, [{x:x-w,y},{x,y:y-1.3},{x:x+w,y:y-.4}], i % 3 ? '#6a83513d' : '#ffefba85', 1)
  }
  ctx.restore()
  path(ctx, contour, true)
  ctx.lineWidth = 1.3
  ctx.strokeStyle = '#ede6b789'
  ctx.stroke()
}

function sample(edge, d) {
  let index = 1
  while (index < edge.points.length - 1 && edge.lengths[index] < d) index++
  const a = edge.points[index - 1], b = edge.points[index]
  const length = edge.lengths[index] - edge.lengths[index - 1]
  const t = (Math.max(0, Math.min(edge.length, d)) - edge.lengths[index - 1]) / length
  return { x: a.x + (b.x - a.x)*t, y: a.y + (b.y - a.y)*t, nx: -(b.y-a.y)/length, ny: (b.x-a.x)/length }
}

function paintCrossing(ctx, edge, a, b, stoneBridge) {
  const deck = []
  for (let d = 0; d <= edge.length; d += 5) {
    const p = sample(edge, d)
    if (!contains(a,p.x,p.y) && !contains(b,p.x,p.y)) deck.push(p)
  }
  if (deck.length < 2) return
  stroke(ctx,deck,'#587e7169',14)
  stroke(ctx,deck,stoneBridge ? '#817d69' : '#886843',11)
  const left = [], right = []
  for (const [i, p] of deck.entries()) {
    const dx = p.nx * 4.9, dy = p.ny * 4.9
    const color = stoneBridge ? (i % 2 ? '#d2cbaa' : '#b9b395') : (i % 2 ? '#e0ba82' : '#c8a16e')
    segment(ctx,p.x-dx,p.y-dy,p.x+dx,p.y+dy,color,3.7)
    if (i % 4 === 0 || i === deck.length-1) {
      segment(ctx,p.x+dx,p.y+dy,p.x+dx,p.y+dy-5.5,'#746b52',2)
      segment(ctx,p.x-dx,p.y-dy,p.x-dx,p.y-dy-5.5,'#746b52',2)
      left.push({x:p.x+dx,y:p.y+dy-5})
      right.push({x:p.x-dx,y:p.y-dy-5})
    }
  }
  stroke(ctx,left,stoneBridge ? '#a59f83' : '#9b7c50',1.5)
  stroke(ctx,right,stoneBridge ? '#a59f83' : '#9b7c50',1.5)
}

function farmland(ctx, x, y, s) {
  const corners = [[-22,-1],[4,-11],[25,-3],[-2,8]]
  fill(ctx, corners.map(([dx,dy]) => ({x:x+s*dx,y:y+s*dy})), '#aa915b')
  for (let i=0; i<5; i++) {
    const px=x+s*(-17+i*4), py=y+s*(-.8+i*1.45)
    segment(ctx,px,py,px+22*s,py-8.5*s,i%2 ? '#edce81' : '#dabb74',1.9*s)
  }
}

function crystals(ctx, x, y, s) {
  for (const [dx,dy,h] of [[-6,0,9],[0,2,17],[6,-1,11]]) {
    const px=x+s*dx, py=y+s*dy
    fill(ctx,[{x:px-3*s,y:py},{x:px-4*s,y:py-5*s},{x:px+1*s,y:py-h*s},
      {x:px+4*s,y:py-6*s},{x:px+2*s,y:py+1*s}],'#609d92')
    segment(ctx,px-2*s,py-5*s,px+1*s,py-h*s,'#cfdfac',1.4*s)
  }
}

function waterfall(ctx, region, contour) {
  const x = region.x - region.width * .22
  const south = contour.filter(p => p.y > region.y)
  const lip = south.reduce((a,b) => Math.abs(a.x-x) < Math.abs(b.x-x) ? a : b)
  const {x:px, y:py} = lip
  ctx.save()
  ctx.translate(px,py)
  ctx.beginPath()
  ctx.ellipse(3,21,15,3,0,0,TAU)
  ctx.fillStyle='#cfe5d080'
  ctx.fill()
  fill(ctx,[{x:-8,y:-8},{x:-5,y:-10},{x:4,y:-6},{x:3,y:-1},{x:4,y:10},{x:9,y:21},
    {x:0,y:22},{x:-5,y:17},{x:-6,y:1}],'#8ebdb0')
  stroke(ctx,[{x:-3,y:-3},{x:-3,y:7},{x:0,y:17},{x:2,y:20}],'#e6f1d8',2.1)
  stroke(ctx,[{x:1,y:-2},{x:1,y:6},{x:4,y:16}],'#e6f1d8',1)
  ctx.restore()
}

function nearRoad(edges, x, y, gap) {
  return edges.some(edge => edge.points.slice(1).some((b,i) => {
    const a = edge.points[i]
    const dx = b.x - a.x, dy = b.y - a.y
    const t = Math.max(0, Math.min(1, ((x-a.x)*dx + (y-a.y)*dy)/(dx*dx + dy*dy)))
    return Math.hypot(x-a.x-t*dx, y-a.y-t*dy) < gap
  }))
}

function paintVillage(ctx, region, contour, roads, nodes) {
  const {x, y, width, height, index} = region
  const theme = index % biomes.length
  const biome = biomes[theme]
  const rng = random(143 + index * 257)
  const sprites = []
  const add = (draw, sx, sy, scale) => sprites.push({draw,x:sx,y:sy,scale})
  function available(sx, sy, margin) {
    return contains(contour,sx,sy) && !nearRoad(roads,sx,sy,margin) &&
      nodes.every(n => Math.hypot(n.x-sx, n.y-sy) > margin + 6) &&
      (Math.abs(sx - x) > 36 || sy < y - 62 || sy > y + 16)
  }
  // A few tall silhouettes, several hamlets, and small clustered woodlands are legible at both zooms.
  for (let i = 0; i < 95; i++) {
    const sx = x + (rng() - .5)*width*.89, sy = y + (rng() - .5)*height*.83
    const scale = .45 + rng()*.34
    if (!available(sx,sy,16) || sprites.some(s => Math.hypot(s.x - sx, s.y - sy) < 16)) continue
    let draw = pine
    if (i % 8 === 0 && sy < y && [2,5,6].includes(theme)) draw = peak
    else if (i % 9 === 0) draw = house
    else if ([0,4].includes(theme) && i % 6 === 0) draw = farmland
    else if ([2,6].includes(theme) && i % 4 === 0) draw = crystals
    else if (i % 5 === 0 || (theme === 2 && i % 2 === 0)) draw = stone
    add(draw, sx, sy, draw === peak ? scale*.92 : scale)
    if (draw === pine && i % 3 === 0 && available(sx-8,sy-4,14)) add(pine,sx-8,sy-4,scale*.73)
  }
  add((c, lx, ly, s) => {
    c.save()
    c.translate(lx,ly)
    c.scale(s,s)
    landmarks[theme](c,biome)
    c.restore()
  }, x+8, y-7, 1)
  sprites.sort((a,b) => a.y - b.y)
  for (const sprite of sprites) sprite.draw(ctx,sprite.x,sprite.y,sprite.scale,biome)
}

function paintShips(ctx, atlas, contours) {
  const rng = random(927), occupied = []
  const count = Math.min(atlas.regions.length + 1, 16)
  for (let i = 0; i < 400 && occupied.length < count; i++) {
    const x = 32 + rng() * (atlas.width-64), y = 42 + rng() * (atlas.height-68)
    if (atlas.regions.some((region,j) => contains(expanded(contours[j],region,1.19),x,y))) continue
    if (occupied.some(p => Math.hypot(p.x-x,p.y-y)<175) || nearRoad(atlas.edges,x,y,38)) continue
    const s = .50 + rng()*.33
    ship(ctx,x,y,s)
    occupied.push({x,y})
    for (let j=0; j<2; j++) {
      const px=x+20+10*j, py=y-21+j*4
      stroke(ctx,[{x:px-4,y:py-1.5},{x:px,y:py},{x:px+3,y:py-2.2}],'#537c715c',1.1)
    }
  }
}

/** Paint only the static world. The UI positions interactive markers and the moving traveler above it. */
export function paintAtlas(canvas, atlas, {padX = 0, padY = 0, trailNodes = new Set()} = {}) {
  const ctx = canvas.getContext('2d')
  if (!ctx || !atlas.width || !atlas.height) return
  ctx.save()
  const width = atlas.width + 2 * padX, height = atlas.height + 2 * padY
  const sx = canvas.width/width, sy = canvas.height/height
  ctx.setTransform(sx,0,0,sy,0,0)
  ctx.lineJoin = 'round'
  ctx.lineCap = 'round'
  paintSea(ctx, {width, height})
  ctx.translate(padX, padY)
  const contours = atlas.regions.map(coastline)
  const coasts = new Map(atlas.regions.map((r, i) => [r.chapterId, contours[i]]))
  for (const [i, region] of atlas.regions.entries()) {
    paintLand(ctx,region,contours[i],biomes[i%biomes.length])
    if ([0,5,6].includes(i % biomes.length)) waterfall(ctx,region,contours[i])
  }
  for (const edge of atlas.edges) stroke(ctx,edge.points,'#897a506e',8.4)
  for (const edge of atlas.edges) stroke(ctx,edge.points,'#efd394',6.2)
  ctx.setLineDash([3,7])
  for (const edge of atlas.edges) stroke(ctx,edge.points,'#fff0c2',1.3)
  ctx.setLineDash([])
  for (const edge of atlas.edges.filter(e => e.crossing)) {
    const a = atlas.nodeById.get(edge.from), b = atlas.nodeById.get(edge.to)
    paintCrossing(ctx,edge,coasts.get(a.chapterId),coasts.get(b.chapterId),(a.chapterId+b.chapterId)%3===0)
  }
  // Colour surveyed paths only after the surrounding coursework has been passed.
  for (const edge of atlas.edges) {
    if (!trailNodes.has(edge.from) || !trailNodes.has(edge.to)) continue
    stroke(ctx, edge.points, '#527556', 4.0)
    ctx.setLineDash([3, 7])
    stroke(ctx, edge.points, '#f6e49a', 1.8)
    ctx.setLineDash([])
  }
  for (const [i, region] of atlas.regions.entries()) {
    const roads = atlas.edges.filter(e => atlas.nodeById.get(e.from).chapterId===region.chapterId || atlas.nodeById.get(e.to).chapterId===region.chapterId)
    paintVillage(ctx,region,contours[i],roads,atlas.nodes.filter(n => n.chapterId===region.chapterId))
  }
  paintShips(ctx,atlas,contours)
  ctx.restore()
}
