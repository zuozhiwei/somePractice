import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  buildAtlas, directionalNeighbor, nearestOnRoad,
  routeBetween, routeToRoad, routeBetweenRoads,
} from '../src/world/worldAtlas.js'

function chapters(sizes) {
  let id = 800
  return sizes.map((count, i) => ({
    id: 19 + i * 2,
    levels: Array.from({length: count}, () => ({id: ++id})),
  }))
}
const position = ({x,y}) => ({x,y})
const length = route => route.reduce((sum,p,i) => i ? sum + Math.hypot(p.x-route[i-1].x,p.y-route[i-1].y) : sum, 0)

test('every real checkpoint and empty settlement appears once and can be reached', () => {
  for (const sizes of [[],[0],[1,2],[6,6,6,6,6,6,6],[6,6,6,6,6,6,6,4],[8,0,16,2,3,34,1,4,2,1,0]]) {
    const atlas = buildAtlas(chapters(sizes))
    assert.equal(atlas.regions.length,sizes.length)
    assert.equal(atlas.nodes.length,sizes.reduce((sum,n) => sum+n, sizes.length))
    assert.equal(atlas.nodes.length,atlas.nodeById.size)
    for (const n of atlas.nodes) {
      assert.ok(n.x >= 0 && n.x <= atlas.width && n.y >= 0 && n.y <= atlas.height)
      const route = routeBetween(atlas,atlas.nodes[0].id,n.id)
      assert.deepEqual(route[0],position(atlas.nodes[0]))
      assert.deepEqual(route.at(-1),position(n))
    }
  }
})

test('the seven regions contain local loops, crossroads and alternate inter-island bridges', () => {
  const atlas = buildAtlas(chapters(Array(7).fill(6)))
  assert.equal(atlas.edges.filter(e => e.crossing).length,12)
  for (const r of atlas.regions) {
    const nodes = atlas.nodes.filter(n => n.chapterId === r.chapterId)
    const edges = atlas.edges.filter(e => !e.crossing && atlas.nodeById.get(e.from).chapterId === r.chapterId)
    assert.ok(edges.length >= nodes.length)
    const hub = nodes.find(n => n.kind === 'hub')
    const neighbors = new Set([[0,-1],[0,1],[-1,0],[1,0]].map(([dx,dy]) => directionalNeighbor(atlas,hub.id,dx,dy)?.id).filter(Boolean))
    assert.ok(neighbors.size >= 3)
  }
})

test('road routes use the shorter branch and stay on surveyed ground between vertices', () => {
  const atlas = buildAtlas(chapters([6,6,6]))
  // Compute independent all-pairs distances and compare geometry against the weighted road graph.
  const ids = atlas.nodes.map(n => n.id)
  const matrix = ids.map((_,i) => ids.map((__,j) => i === j ? 0 : Infinity))
  for (const e of atlas.edges) {
    const i = ids.indexOf(e.from), j = ids.indexOf(e.to)
    matrix[i][j] = matrix[j][i] = e.length
  }
  for (let k = 0; k < ids.length; k++) {
    for (let i = 0; i < ids.length; i++) {
      for (let j = 0; j < ids.length; j++) matrix[i][j] = Math.min(matrix[i][j],matrix[i][k]+matrix[k][j])
    }
  }
  for (const [i, from] of ids.entries()) {
    for (const [j, to] of ids.entries()) {
      const route = routeBetween(atlas,from,to)
      assert.ok(Math.abs(length(route) - matrix[i][j]) < .00001)
      for (let step = 1; step < route.length; step++) {
        const a = route[step-1], b = route[step]
        assert.ok(nearestOnRoad(atlas,(a.x+b.x)/2,(a.y+b.y)/2,.01))
      }
    }
  }
  assert.deepEqual(routeBetween(atlas,ids[0],'missing'),[])
  assert.equal(nearestOnRoad(atlas,-50,-50,12),null)
})

test('changing destination halfway along an edge starts at the traveler, even on a bridge', () => {
  const atlas = buildAtlas(chapters([6,6]))
  const edge = atlas.edges.find(e => e.crossing)
  function hitBetween(i,t) {
    const a=edge.points[i], b=edge.points[i+1]
    return nearestOnRoad(atlas,a.x+(b.x-a.x)*t,a.y+(b.y-a.y)*t,.01)
  }
  const a = hitBetween(0,.55), b = hitBetween(2,.42)
  const direct = routeBetweenRoads(atlas,a,b)
  assert.deepEqual(direct[0],position(a))
  assert.deepEqual(direct.at(-1),position(b))
  assert.ok(Math.abs(length(direct) - (b.t-a.t)*edge.length) < .00001)
  assert.deepEqual(routeBetweenRoads(atlas,b,a),direct.slice().reverse())
  const target = atlas.nodes[0]
  const backwards = routeToRoad(atlas,target.id,a).reverse()
  assert.deepEqual(backwards[0],position(a))
  assert.deepEqual(backwards.at(-1),position(target))
})
