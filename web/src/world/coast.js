const biomes = [
  { grass:'#aac477', sun:'#d2d98c', shade:'#8da862', pine:'#477b59', rock:'#839084', roof:'#ad6846' },
  { grass:'#d2aa70', sun:'#e8cb91', shade:'#bea064', pine:'#698660', rock:'#b38b64', roof:'#8b644e' },
  { grass:'#9da685', sun:'#bec296', shade:'#899679', pine:'#4d7264', rock:'#818788', roof:'#8c6956' },
  { grass:'#9eb780', sun:'#c5cc96', shade:'#8da375', pine:'#457b67', rock:'#858f8e', roof:'#706682' },
  { grass:'#b7b775', sun:'#d7cc91', shade:'#9fa265', pine:'#60865d', rock:'#91957c', roof:'#b16a45' },
  { grass:'#d3e2d9', sun:'#edf3e9', shade:'#b8cfc9', pine:'#668b7e', rock:'#91aaa7', roof:'#65859a' },
  { grass:'#b8c4a2', sun:'#dce2bf', shade:'#a5b294', pine:'#698e81', rock:'#9aabad', roof:'#727b9a' },
  { grass:'#b8b68c', sun:'#e3d5a7', shade:'#9aa27f', pine:'#56785d', rock:'#9a9982', roof:'#4d6665' },
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

function examinationHall(ctx) {
  ellipse(ctx, 1, 5, 52, 11, '#4e624539')
  polygon(ctx, [[-44,0],[-7,-13],[43,-1],[3,13]], '#bcb5a0')
  polygon(ctx, [[-38,-4],[-6,-14],[37,-4],[2,7]], '#e4d8ae')
  polygon(ctx, [[-27,-36],[2,-28],[2,3],[-27,-6]], '#e5c78f')
  polygon(ctx, [[2,-28],[29,-37],[29,-5],[2,3]], '#c19e72')
  for (const [x,y] of [[-24,-6],[-12,-2],[1,1],[14,-3],[26,-6]]) {
    line(ctx, [[x,y],[x,y-29]], '#9b4f3c', 3)
  }
  polygon(ctx, [[-42,-32],[-27,-35],[-11,-52],[13,-54],[26,-42],[41,-39],[32,-33],[1,-22],[-31,-29]], '#495f60')
  line(ctx, [[-42,-32],[-32,-29],[1,-22],[32,-33],[41,-39]], '#c59b58', 2.5)
  line(ctx, [[-12,-54],[-10,-51],[13,-54],[16,-58]], '#c59b58', 2)
  polygon(ctx, [[-10,-25],[6,-23],[6,-17],[-10,-19]], '#725332')
  line(ctx, [[-7,-22],[2,-21]], '#e9c476', 1.5)
  for (const [x,y] of [[-26,-28],[22,-30]]) {
    line(ctx, [[x,y],[x,y+5]], '#624d39')
    ellipse(ctx, x,y+8,3.8,5,'#b75c3e')
    line(ctx, [[x,y+12],[x,y+15]], '#d3a451')
  }
}
const landmarks = [windmill, lighthouse, mine, keep, foundry, alpineStation, skySanctuary, examinationHall]

// Shared art for the atlas and the original individual chapter scenes.
export { biomes, house, landmarks, peak, pine, random, ship, stone }
