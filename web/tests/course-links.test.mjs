import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import { marked } from 'marked'
import { splitPlatformSections, markdownForPlatform } from '../src/workshop/lessonPlatforms.js'

const source = new URL('../../internal/curriculum/', import.meta.url)
const catalog = JSON.parse(await readFile(new URL('catalog.json', source), 'utf8'))

for (const platform of ['windows', 'mac', 'linux']) {
  test('lesson references resolve for ' + platform, async () => {
    const lessons = new Map()
    const references = []
    for (const { id } of catalog.levels) {
      const path = 'lessons/' + String(id).padStart(2, '0') + '.md'
      const raw = await readFile(new URL(path, source), 'utf8')
      const body = markdownForPlatform(splitPlatformSections(raw), platform)
      const guide = body.split(/^## 任务 \d+ \| /m, 1)[0]
      const titles = new Set()
      marked.walkTokens(marked.lexer(guide), token => {
        if (token.type === 'heading' && [2, 3].includes(token.depth)) {
          titles.add(token.tokens.map(part => part.text).join(''))
        }
      })
      const tasks = catalog.tasks.filter(task => task.level === id).map(task => task.id)
      lessons.set(id, { titles, tasks })
      marked.walkTokens(marked.lexer(body), token => {
        if (token.type === 'link' && token.href.startsWith('/level/')) {
          references.push({ from: id, href: token.href })
        }
      })
    }
    for (const { from, href } of references) {
      const url = new URL(href, 'http://localhost')
      const lesson = lessons.get(Number(url.pathname.slice('/level/'.length)))
      const message = platform + ' level ' + from + ' links to ' + href
      assert.ok(lesson, message)
      if (url.searchParams.has('task')) {
        assert.ok(lesson.tasks.includes(Number(url.searchParams.get('task'))), message)
      }
      if (url.hash) {
        assert.equal(url.searchParams.get('guide'), '1', message)
        assert.ok(lesson.titles.has(decodeURIComponent(url.hash.slice(1))), message)
      }
    }
  })
}
