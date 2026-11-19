import assert from 'node:assert/strict'
import { test } from 'node:test'
import { chapterStops, currentLevel, locate, roadHit, ROAD } from '../src/world/expedition.js'

const levels = Array.from({length: 14}, (_, i) => ({
  id: 902 + i * 3, name: `测试关卡 ${i + 1}`, status: i < 5 ? 'COMPLETED' : i < 7 ? 'LEARNING' : 'LOCKED',
}))

test('maps every API level exactly once across road segments', () => {
  const pages = [0, 1, 2].map(page => chapterStops(levels, page, 'IN_PROGRESS'))
  assert.deepEqual(pages.map(stops => stops.length), [6, 6, 2])
  assert.deepEqual(pages.flat().map(stop => stop.id), levels.map(level => level.id))
  assert.equal(pages[2][1].number, 14)
  for (const stops of pages) {
    assert.equal(new Set(stops.map(s => s.distance)).size, stops.length)
    for (const stop of stops) assert.ok(ROAD.some(([x,y]) => stop.x === x && stop.y === y))
  }
})
test('locked chapters remain preview-only even when an individual prerequisite is met', () => {
  const stops = chapterStops(levels, 1, 'LOCKED')
  assert.ok(stops.every(stop => stop.status === 'LOCKED'))
  assert.equal(levels[6].status, 'LEARNING')
  assert.deepEqual(chapterStops([], 0, 'IN_PROGRESS'), [])
})
test('focuses the first playable level instead of an earlier locked dependency', () => {
  const [first, second] = levels
  const unordered = [{...first, status:'LOCKED'}, second, levels[6]]
  assert.equal(currentLevel(unordered, 'IN_PROGRESS').id, levels[6].id)
  assert.equal(currentLevel(unordered, 'LOCKED').id, first.id)
  assert.equal(currentLevel([], 'IN_PROGRESS'), undefined)
})
test('movement follows the same road in either direction and clamps at the coast', () => {
  assert.deepEqual([locate(-5).x, locate(-5).y], ROAD[0])
  assert.deepEqual([locate(Infinity).x, locate(Infinity).y], ROAD.at(-1))
  const mid = Math.hypot(25,4)/2
  assert.equal(locate(mid).x, 120.5)
  assert.equal(locate(mid).y, 323)
})

test('tapping between checkpoints projects onto the trail while sea clicks miss it', () => {
  const mid = roadHit(120, 325, 5)
  assert.ok(mid.distance > 0 && mid.distance < 25)
  assert.ok(Math.hypot(locate(mid.distance).x - 120, locate(mid.distance).y - 325) < 3)
  assert.equal(roadHit(30, 90, 20), null)
})
