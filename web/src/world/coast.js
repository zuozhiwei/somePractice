import { WIDTH, HEIGHT, ROAD } from './expedition.js'

// Painted in layers once per chapter and resize. Character movement uses a separate sprite.
const biomes = [
  { grass:'#aac477', sun:'#d2d98c', shade:'#8da862', pine:'#477b59', rock:'#839084', roof:'#ad6846' },
  { grass:'#d2aa70', sun:'#e8cb91', shade:'#bea064', pine:'#698660', rock:'#b38b64', roof:'#8b644e' },
  { grass:'#9da685', sun:'#bec296', shade:'#899679', pine:'#4d7264', rock:'#818788', roof:'#8c6956' },
  { grass:'#9eb780', sun:'#c5cc96', shade:'#8da375', pine:'#457b67', rock:'#858f8e', roof:'#706682' },
  { grass:'#b7b775', sun:'#d7cc91', shade:'#9fa265', pine:'#60865d', rock:'#91957c', roof:'#b16a45' },
  { grass:'#d3e2d9', sun:'#edf3e9', shade:'#b8cfc9', pine:'#668b7e', rock:'#91aaa7', roof:'#65859a' },
  { grass:'#b8c4a2', sun:'#dce2bf', shade:'#a5b294', pine:'#698e81', rock:'#9aabad', roof:'#727b9a' },
]
const lands = [
  [[30,284],[49,255],[80,244],[86,218],[124,204],[140,182],[178,177],[196,162],[237,168],[246,184],[267,181],[287,203],[317,210],[328,232],[351,243],[358,266],[382,282],[369,307],[339,315],[328,341],[299,348],[287,373],[257,376],[241,395],[199,394],[179,409],[136,399],[118,382],[79,380],[75,362],[46,350],[50,328],[27,310]],
  [[376,221],[394,189],[419,177],[432,152],[464,147],[485,121],[523,124],[544,105],[576,113],[597,99],[625,108],[638,128],[661,130],[672,153],[688,164],[680,183],[657,197],[652,219],[628,234],[620,265],[588,270],[575,289],[542,286],[520,311],[490,310],[462,331],[428,328],[406,309],[378,305],[368,287],[383,266],[371,248]],
  [[698,143],[708,120],[740,113],[750,88],[781,82],[799,67],[823,71],[847,49],[873,57],[892,48],[915,65],[920,85],[946,98],[953,125],[971,139],[962,161],[945,168],[937,192],[906,198],[892,221],[860,222],[839,243],[809,241],[788,254],[757,247],[748,227],[721,226],[712,205],[693,195],[697,174],[684,163]],
]
function random(seed) {
  let value = seed >>> 0
  return () => { value = (1664525 * value + 1013904223) >>> 0; return value / 4294967296 }
}
function polygon(ctx, points, color) {
  ctx.beginPath()
  points.forEach(([x,y], i) => i ? ctx.lineTo(x,y) : ctx.moveTo(x,y))
  ctx.closePath(); ctx.fillStyle = color; ctx.fill()
}
function line(ctx, points, color, width = 1) {
  ctx.beginPath(); points.forEach(([x,y], i) => i ? ctx.lineTo(x,y) : ctx.moveTo(x,y))
  ctx.strokeStyle = color; ctx.lineWidth = width; ctx.stroke()
}
function ellipse(ctx, x, y, rx, ry, color) {
  ctx.fillStyle = color; ctx.beginPath(); ctx.ellipse(x,y,rx,ry,0,0,Math.PI*2); ctx.fill()
}
function within([x,y], shape) {
  let inside = false
  for (let i=0, j=shape.length-1; i<shape.length; j=i++) {
    const [ax,ay]=shape[i], [bx,by]=shape[j]
    if ((ay>y)!==(by>y) && x<(bx-ax)*(y-ay)/(by-ay)+ax) inside=!inside
  }
  return inside
}
function drawLand(ctx, shape, b, rng, floating) {
  for (const [dy, color] of [[31,'#72a69b'],[26,'#d1d8ab'],[22,'#88947b']]) {
    polygon(ctx, shape.map(([x,y]) => [x,y+dy]),color)
  }
  shape.forEach(([x,y], i) => {
    const [u,v] = shape[(i+1)%shape.length]
    polygon(ctx, [[x,y],[u,v],[u,v+20],[x,y+20]], ['#a39370','#9b9875','#b6ab83','#af9e76'][i%4])
    if(i%2===0) line(ctx, [[x+2,y+6],[x+3,y+12]],'#857e663d',1.8)
  })
  if (floating) {
    for (let edge=0; edge<shape.length; edge++) {
      const [x,y] = shape[edge], [u,v] = shape[(edge+1)%shape.length]
      if (x <= u) continue
      const depth = 19 + (edge % 5) * 5
      polygon(ctx, [[x,y+17],[u,v+17],[(x+u)/2-3,(y+v)/2+depth],[x,y+21]], edge%2 ? '#8b9193' : '#9b9b96')
    }
  }
  polygon(ctx,shape,b.grass)
  // Low ridges and meadows follow the shoreline, with their edges clipped to the land.
  ctx.save(); ctx.clip()
  const cx=shape.reduce((sum,p)=>sum+p[0],0)/shape.length, cy=shape.reduce((sum,p)=>sum+p[1],0)/shape.length
  shape.forEach(([x,y],i) => {
    const next=shape[(i+1)%shape.length]
    polygon(ctx, [[x,y],next,[cx+(x-cx)*.65+8,cy+(y-cy)*.65]], i%3===0?b.shade:b.sun)
  })
  for(let i=0; i<170; i++) {
    const x=cx+(rng()-.5)*430, y=cy+(rng()-.5)*260
    if(!within([x,y],shape)) continue
    line(ctx,[[x-3,y],[x,y-1],[x+5,y]], i%4===0?'#fff4c785':'#667e4747',1.2)
  }
  ctx.restore()
}
function pine(ctx,x,y,s,b) {
  ctx.save(); ctx.translate(x,y); ctx.scale(s,s)
  ellipse(ctx,4,1,12,3,'#29473726')
  polygon(ctx,[[-2,0],[2,0],[3,-15],[-2,-15]],'#766446')
  for(const [top,base,r] of [[-31,-7,13],[-39,-16,11],[-45,-25,8]]) {
    polygon(ctx,[[0,top],[-r,base-3],[0,base+1],[r,base-3]],b.pine)
    polygon(ctx,[[0,top],[0,base+1],[r,base-3]],'#174e502d')
    polygon(ctx,[[0,top],[-r,base-3],[-3,base-5]],'#f6e6ac42')
  }
  ctx.restore()
}
function stone(ctx,x,y,s,b) {
  ctx.save(); ctx.translate(x,y); ctx.scale(s,s)
  ellipse(ctx,3,0,11,3,'#34473a23')
  polygon(ctx,[[-11,-1],[-8,-10],[-1,-16],[6,-12],[12,-2],[3,1]],b.rock)
  polygon(ctx,[[-11,-1],[-8,-10],[-1,-16],[0,-4]],'#ffffff40')
  polygon(ctx,[[0,-4],[6,-12],[12,-2],[3,1]],'#30483944')
  ctx.restore()
}
function peak(ctx,x,y,s,b) {
  ctx.save(); ctx.translate(x,y); ctx.scale(s,s)
  polygon(ctx,[[-51,-3],[-40,-23],[-28,-27],[-13,-49],[0,-67],[16,-40],[25,-34],[45,-9],[35,-3],[8,2],[-21,2]],b.rock)
  polygon(ctx,[[-51,-3],[-28,-27],[-13,-49],[0,-67],[-8,-28],[-26,-13]],'#eef1cf6b')
  polygon(ctx,[[0,-67],[5,-40],[16,-30],[25,-10],[36,-3],[45,-9],[16,-40]],'#465a586e')
  polygon(ctx,[[-13,-49],[0,-67],[13,-46],[7,-48],[5,-43],[0,-51],[-6,-46],[-7,-49]],'#f4f3dc')
  ctx.restore()
}
function house(ctx,x,y,s,b) {
  ctx.save();ctx.translate(x,y);ctx.scale(s,s)
  ellipse(ctx,6,1,25,6,'#29473725')
  polygon(ctx,[[-19,-24],[2,-17],[2,3],[-19,-5]],'#eee0b3')
  polygon(ctx,[[2,-17],[22,-25],[22,-5],[2,3]],'#c8b993')
  polygon(ctx,[[-24,-24],[-14,-41],[6,-33],[4,-15]],b.roof)
  polygon(ctx,[[-14,-41],[9,-48],[29,-28],[4,-15],[6,-33]],'#8e5d45')
  polygon(ctx,[[-14,-41],[8,-46],[26,-28],[6,-33]],b.roof)
  polygon(ctx,[[-14,-18],[-9,-16],[-9,-10],[-14,-12]],'#5c8278')
  polygon(ctx,[[10,-14],[16,-16],[16,-2],[10,0]],'#80684c')
  line(ctx,[[-23,-23],[4,-14],[27,-27]],'#775e43',1.6)
  ctx.restore()
}
function tower(ctx,x,y,s,b) {
  ctx.save();ctx.translate(x,y);ctx.scale(s,s)
  ellipse(ctx,7,0,25,6,'#29473729')
  polygon(ctx,[[-13,-39],[0,-34],[0,3],[-13,-2]],'#ebe6ca')
  polygon(ctx,[[0,-34],[15,-40],[15,-3],[0,3]],'#b9b9a2')
  polygon(ctx,[[-19,-41],[0,-66],[19,-42],[0,-34]],b.roof)
  polygon(ctx,[[0,-66],[19,-42],[0,-34]],'#48514758')
  line(ctx,[[0,-65],[0,-79]],'#716147',1.7)
  polygon(ctx,[[0,-79],[15,-78],[11,-75],[15,-72],[0,-73]],'#e6a54d')
  polygon(ctx,[[4,-21],[8,-23],[8,-15],[4,-13]],'#627c72')
  ctx.restore()
}
function bridge(ctx,from,to) {
  const [x,y]=from,[u,v]=to, dx=u-x,dy=v-y
  line(ctx,[from,to],'#7e674d',13)
  for(let i=0;i<=12;i++){
    const t=i/12, xx=x+dx*t, yy=y+dy*t
    line(ctx,[[xx-1,yy-5],[xx+2,yy+5]],i%2?'#cfad78':'#dfbd87',4)
    if(i%4===0) line(ctx,[[xx,yy-5],[xx,yy-12]],'#795f45',2.5)
  }
  line(ctx,[[x,y-11],[x+dx/2,y+dy/2-9],[u,v-11]],'#9c7c51',1.7)
}
function ship(ctx,x,y,s) {
  ctx.save();ctx.translate(x,y);ctx.scale(s,s)
  ellipse(ctx,4,6,18,3,'#538d8166')
  polygon(ctx,[[-18,0],[14,-3],[22,-7],[15,3],[-7,6]],'#815f42')
  line(ctx,[[-15,0],[15,-3]],'#e5c38c',2)
  line(ctx,[[1,1],[1,-31]],'#876e4a',2)
  polygon(ctx,[[-2,-29],[-4,-5],[-15,-4]],'#f5e9c1')
  polygon(ctx,[[3,-25],[4,-6],[15,-9]],'#d3a162')
  ctx.restore()
}
/** Each landmark is drawn in local coordinates, with its origin at ground level. */
function windmill(ctx) {
  ellipse(ctx, 9, 1, 31, 6, '#465c452c')
  polygon(ctx, [[-20,-2],[-12,-53],[4,-58],[15,-49],[24,-4],[2,4]], '#f4dfaa')
  polygon(ctx, [[4,-58],[15,-49],[24,-4],[2,4]], '#c6af80')
  polygon(ctx, [[-17,-50],[-7,-64],[6,-68],[20,-51],[4,-45]], '#895342')
  polygon(ctx, [[-10,-15],[-3,-18],[1,-16],[1,3],[-11,0]], '#635748')
  ctx.save()
  ctx.translate(-4,-47)
  ctx.rotate(-.42)
  for (let blade = 0; blade < 4; blade++) {
    line(ctx, [[0,0],[0,-44]], '#795c41', 2.8)
    polygon(ctx, [[-1,-15],[-1,-44],[8,-41],[6,-16]], '#fff4d4')
    for (const y of [-21,-28,-35]) line(ctx, [[0,y],[6,y+1]], '#c3a878', 1)
    ctx.rotate(Math.PI/2)
  }
  ellipse(ctx,0,0,4,4,'#594831')
  ctx.restore()
}

function stall(ctx, x, y, color) {
  line(ctx, [[x-14,y-1],[x-14,y-19],[x+15,y-23],[x+15,y-5]], '#795b42', 2.5)
  polygon(ctx, [[x-19,y-19],[x-8,y-30],[x+12,y-32],[x+22,y-21],[x+1,y-16]], color)
  polygon(ctx, [[x-19,y-19],[x+1,y-16],[x+22,y-21],[x+20,y-16],[x+1,y-12],[x-19,y-15]], '#f2d39d')
  polygon(ctx, [[x-15,y-6],[x,y-10],[x+17,y-7],[x+2,y-3]], '#865f40')
  for (const offset of [-7,-2,3,8]) ellipse(ctx,x+offset,y-8,2.4,1.7,'#ddaa56')
}

function lighthouse(ctx) {
  ellipse(ctx,5,1,24,6,'#5e60432c')
  polygon(ctx,[[-15,0],[-9,-55],[5,-58],[11,-54],[18,-2],[2,4]],'#f7ecd1')
  polygon(ctx,[[2,4],[18,-2],[11,-54],[5,-58]],'#cfbda0')
  polygon(ctx,[[-12,-23],[3,-20],[15,-26],[14,-37],[3,-32],[-11,-35]],'#b86747')
  polygon(ctx,[[-12,-53],[4,-50],[15,-56],[15,-60],[2,-57],[-12,-60]],'#6b7260')
  polygon(ctx,[[-9,-61],[-7,-72],[5,-75],[11,-70],[11,-61],[2,-57]],'#e8bd61')
  line(ctx,[[-8,-71],[-8,-60],[2,-57],[2,-72],[11,-70],[11,-62]],'#776c4c',2)
  polygon(ctx,[[-13,-71],[1,-85],[15,-73],[3,-69]],'#884f41')
  polygon(ctx,[[-8,-13],[-3,-14],[-2,-1],[-8,-2]],'#645645')
  stall(ctx,-31,4,'#994f38')
  stall(ctx,-10,16,'#448b83')
}

function mine(ctx) {
  ellipse(ctx,5,2,63,8,'#39483c22')
  polygon(ctx,[[-49,-4],[-43,-28],[-32,-36],[-26,-58],[-8,-74],[2,-65],[11,-69],[35,-45],[48,-14],[36,0],[7,5]],'#717e78')
  polygon(ctx,[[-49,-4],[-32,-36],[-26,-58],[-8,-74],[-13,-44],[-21,-32]],'#a3aaa0')
  polygon(ctx,[[-8,-74],[2,-65],[11,-69],[35,-45],[9,-50]],'#b0b1a1')
  polygon(ctx,[[-21,-1],[-21,-26],[-12,-41],[10,-38],[20,-28],[21,0]],'#38453f')
  polygon(ctx,[[-13,-1],[-12,-23],[-6,-32],[8,-30],[12,-2]],'#283731')
  line(ctx,[[-24,-1],[-24,-32],[-15,-43],[17,-37],[24,-26],[25,0]],'#775c40',6)
  line(ctx,[[-24,-32],[-15,-43],[17,-37]],'#ba9568',2.2)
  for (const x of [-12,6]) line(ctx,[[x,-4],[x-9,12]],'#504f45',2)
  for (const y of [0,4,9]) line(ctx,[[-15-y/2,y],[6-y/2,y+2]],'#a28e6b',2)
  polygon(ctx,[[23,-12],[43,-14],[42,-6],[29,-4]],'#a77649')
  for (const x of [30,40]) ellipse(ctx,x,-4,3,3,'#454e46')
  for (const [x,y] of [[28,-14],[34,-16],[39,-15],[-35,-27]]) {
    polygon(ctx,[[x-3,y],[x-2,y-6],[x+2,y-9],[x+5,y-3],[x+2,y+1]],'#78b8a0')
    polygon(ctx,[[x-2,y-6],[x+2,y-9],[x,y-1]],'#bce1ba')
  }
}

function keep(ctx, b) {
  ellipse(ctx,5,2,54,8,'#4b614632')
  polygon(ctx,[[-33,-38],[-2,-30],[-2,4],[-33,-4]],'#e8d9b4')
  polygon(ctx,[[-2,-30],[32,-42],[32,-7],[-2,4]],'#c0b399')
  for (let i=0; i<4; i++) {
    const x=-34+8*i, y=-41+2*i
    polygon(ctx,[[x,y],[x+5,y+1],[x+5,y+8],[x,y+7]],'#e8d9b4')
  }
  polygon(ctx,[[-20,-1],[-19,-15],[-14,-20],[-9,-14],[-9,2]],'#645d51')
  tower(ctx,-30,-10,.78,b)
  tower(ctx,27,-13,.85,b)
  tower(ctx,-2,-29,.95,b)
  polygon(ctx,[[2,-47],[11,-50],[11,-32],[7,-27],[2,-31]],'#9b5845')
}

function foundry(ctx, b) {
  polygon(ctx,[[4,-20],[5,-66],[16,-69],[22,-65],[20,-16]],'#a56548')
  polygon(ctx,[[15,-68],[22,-65],[20,-16],[13,-18]],'#824d3b')
  polygon(ctx,[[2,-66],[5,-72],[19,-74],[25,-69],[17,-65],[3,-63]],'#6f6452')
  house(ctx,-10,1,.98,b)
  polygon(ctx,[[1,-9],[8,-12],[12,-9],[12,-3],[2,0]],'#df914b')
  const cx=-29, cy=-8
  ellipse(ctx,cx,cy,10,14,'#67563f')
  ellipse(ctx,cx,cy,7.4,10.8,'#b69462')
  for (let i=0;i<6;i++) {
    const rad=i*Math.PI/3
    line(ctx,[[cx,cy],[cx+8*Math.cos(rad),cy+12*Math.sin(rad)]],'#725a3f',2)
  }
  ellipse(ctx,cx,cy,2.5,3,'#ded3a5')
}

function alpineStation(ctx, b) {
  house(ctx,-8,0,.94,b)
  polygon(ctx,[[-29,-24],[-21,-38],[-1,-45],[15,-30],[0,-23],[-6,-30],[-11,-23],[-19,-26],[-23,-21]],'#f2f4e8')
  polygon(ctx,[[16,-5],[11,-9],[17,-11],[29,-10],[28,-7]],'#adbcb1')
  line(ctx,[[20,-8],[23,-54],[26,-7]],'#5c6b61',2.4)
  line(ctx,[[13,-49],[30,-52]],'#5c6b61',2.5)
  line(ctx,[[24,-52],[48,-64],[62,-75]],'#66766e',1.2)
  line(ctx,[[49,-64],[49,-58]],'#616c66',1.3)
  polygon(ctx,[[43,-59],[50,-62],[57,-58],[57,-51],[49,-47],[43,-51]],'#ad5e46')
  polygon(ctx,[[44,-58],[50,-61],[55,-58],[49,-55],[45,-55]],'#e2eee4')
}

function skySanctuary(ctx) {
  ellipse(ctx,4,1,42,6,'#6772743a')
  polygon(ctx,[[-38,-3],[-3,-16],[35,-9],[39,-4],[1,9],[-38,1]],'#a8a6a0')
  polygon(ctx,[[-38,-3],[-3,-16],[35,-9],[0,5]],'#e1d8b8')
  for (const [x,y] of [[-23,-1],[22,-7],[-3,1]]) {
    polygon(ctx,[[x-4,y],[x-3,y-27],[x-6,y-29],[x+6,y-31],[x+4,y-26],[x+4,y-1]],'#f6eace')
    line(ctx,[[x+3,y-26],[x+2,y-1]],'#b6b5a9',1.5)
  }
  polygon(ctx,[[-30,-29],[-4,-42],[32,-34],[31,-30],[0,-22],[-28,-26]],'#a7aaa1')
  const cy=-55
  ctx.save()
  ctx.translate(0,cy)
  ctx.rotate(.3)
  ctx.strokeStyle='#b68e4c';ctx.lineWidth=3
  ctx.beginPath();ctx.ellipse(0,0,16,25,0,0,Math.PI*2);ctx.stroke()
  ctx.rotate(-1.05)
  ctx.beginPath();ctx.ellipse(0,0,9,26,0,0,Math.PI*2);ctx.stroke()
  ctx.restore()
  polygon(ctx,[[-7,cy],[0,cy-14],[8,cy],[0,cy+10]],'#80aaa2')
  polygon(ctx,[[0,cy-14],[8,cy],[0,cy+10]],'#557f7a')
}

const landmarks = [windmill, lighthouse, mine, keep, foundry, alpineStation, skySanctuary]

function landmark(ctx,x,y,scale,biome,draw) {
  ctx.save()
  ctx.translate(x,y)
  ctx.scale(scale,scale)
  draw(ctx,biome)
  ctx.restore()
}


export function paintScene(canvas, themeIndex, seed) {
  const ctx=canvas.getContext('2d')
  if(!ctx) return
  const chapter = themeIndex % biomes.length
  const b = biomes[chapter], rng = random(seed+713)
  ctx.setTransform(canvas.width/WIDTH,0,0,canvas.height/HEIGHT,0,0)
  const sea=ctx.createLinearGradient(0,0,WIDTH,HEIGHT)
  const water = [
    ['#9fcbc1','#b7d7c8','#81b6b1'], ['#8ab8aa','#afcdb6','#77aca7'],
    ['#82aaa5','#afcbc2','#83a8a6'], ['#8fbeb1','#bad3c0','#8ab4aa'],
    ['#a2c1b0','#c1d6bb','#8eb6ab'], ['#a1c5c9','#c4dcdb','#9cbec4'],
    ['#a4bdcf','#d5e1df','#a4c1d0'],
  ][chapter]
  sea.addColorStop(0,water[0]); sea.addColorStop(.55,water[1]); sea.addColorStop(1,water[2])
  ctx.fillStyle=sea;ctx.fillRect(0,0,WIDTH,HEIGHT)
  ctx.lineJoin='round';ctx.lineCap='round'
  for(let i=0;i<550;i++) {
    const x=rng()*WIDTH,y=rng()*HEIGHT,w=rng()*9+2
    line(ctx,[[x,y],[x+w*.6,y+.6],[x+w,y-.4]],i%3?'#f0f7df63':'#639e943d',1.3)
  }
  if (chapter === 6) {
    for (const [x,y,scale] of [[84,83,1.2],[369,184,.8],[672,335,1.4],[896,303,.85],[360,424,.85]]) {
      for (const [dx,dy,size] of [[-19,1,14],[-3,-4,20],[16,1,14],[29,3,8]]) {
        ellipse(ctx, x+dx*scale,y+dy*scale,size*scale,size*.42*scale,'#f0f3e5')
      }
    }
  }
  if (chapter === 5) {
    for (const [x,y] of [[110,97],[373,369],[738,303],[918,266],[852,368]]) {
      polygon(ctx,[[x-14,y],[x-4,y-4],[x+7,y-5],[x+16,y],[x+5,y+4],[x-7,y+4]],'#e5efea')
      line(ctx,[[x-14,y],[x-8,y+5],[x+6,y+5],[x+16,y]],'#a9c8c8',2)
    }
  }
  for(const shape of lands) {
    const contour=shape.map(([x,y])=>[x,y+18])
    for(const [width,color] of [[28,chapter===6?'#aec3cb':'#84c1af'],[18,chapter===6?'#beced1':'#a6d4b9'],[6,chapter===6?'#d3d9d3':'#e2e4c1']]) line(ctx,[...contour,contour[0]],color,width)
    drawLand(ctx,shape,b,rng,chapter===6)
  }
  line(ctx,ROAD,'#ac9e70',12); line(ctx,ROAD,'#e2c68b',9)
  ctx.setLineDash([3,7]);line(ctx,ROAD,'#f7deb0',1.7);ctx.setLineDash([])
  bridge(ctx,ROAD[10],ROAD[12]);bridge(ctx,ROAD[23],ROAD[25])
  const objects=[]
  function add(fn,x,y,scale){objects.push({fn,x,y,scale})}
  if ([2,5].includes(chapter)) {
    add(peak,191,211,.88); add(peak,218,215,.72)
    add(peak,524,148,.83); add(peak,551,146,1.28); add(peak,582,151,.88)
  } else {
    add(peak,527,146,.47); add(peak,557,143,.71)
  }
  const addLandmark = (x,y,scale) => add((c,px,py,s,theme) => landmark(c,px,py,s,theme,landmarks[chapter]),x,y,scale)
  addLandmark(199,258,1.03)
  if (chapter === 3 || chapter === 6) addLandmark(825,121,.94)
  else if (chapter === 1) addLandmark(892,109,.78)
  else add(tower,818,117,.75)
  if (chapter === 0) {
    for (const [x,y] of [[151,238],[164,227],[263,277]]) {
      polygon(ctx, [[x-13,y],[x+4,y-6],[x+16,y],[x+1,y+7]], '#af9a64')
      for (let row=0;row<4;row++) line(ctx,[[x-10+row*4,y+row],[x+5+row*3,y-5+row]],'#eed598',1.8)
    }
  }
  for(const [cx,cy,n] of [[97,267,7],[285,255,10],[214,375,8],[438,208,7],[534,269,7],[624,152,7],[755,139,6],[873,202,6]]) {
    for(let i=0;i<n;i++){
      const x=cx+(rng()-.5)*52,y=cy+(rng()-.5)*23
      if (themeIndex%7===1 || (themeIndex%7===2 && i%2===0)) add(stone,x,y,.55+rng()*.7)
      else add(pine,x,y,.45+rng()*.45)
    }
  }
  for(const [x,y,s] of [[145,250,.65],[158,271,.69],[459,190,.55],[595,245,.55],[788,132,.62]]) add(house,x,y,s)
  if (chapter === 1) {
    add((c,x,y) => stall(c,x,y,'#b56240'),244,271,1)
    add((c,x,y) => stall(c,x,y,'#648887'),264,258,1)
  } else if (chapter === 4) {
    addLandmark(561,157,.83)
  } else if (chapter === 5) {
    add(peak,864,111,.9); add(peak,893,103,.82)
  }
  for(const [x,y] of [[69,322],[107,351],[310,348],[469,292],[575,184],[651,159],[739,203],[906,174]]) add(stone,x,y,.55+rng()*.5)
  objects.sort((a,b)=>a.y-b.y).forEach(({fn,x,y,scale})=>fn(ctx,x,y,scale,b))
  if (chapter !== 6) {
    ship(ctx,363,363,.9); ship(ctx,691,269,.55)
  }
  for(const [x,y,s] of [[347,169,1],[355,162,.75],[671,95,.8],[679,101,.65]]) {
    line(ctx,[[x-5*s,y-2*s],[x,y],[x+4*s,y-3*s]],'#54796e',1.5*s)
  }
}
