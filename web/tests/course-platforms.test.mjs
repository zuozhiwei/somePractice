import assert from 'node:assert/strict'
import { test } from 'node:test'
import { splitPlatformSections, markdownForPlatform } from '../src/workshop/lessonPlatforms.js'

test('system-specific instructions keep shared text and literal code samples', () => {
  const example = [
    '准备文件夹', ':::windows', 'Win + E', ':::', ':::mac linux', './mvnw', ':::',
    '````text', '```', ':::windows', ':::', '````', '核对结果',
  ]
  const parts = splitPlatformSections(example.join('\r\n'))
  const literal = example.slice(7).join('\n')
  assert.equal(markdownForPlatform(parts, 'windows'), '准备文件夹\nWin + E\n' + literal)
  for (const system of ['mac', 'linux']) {
    assert.equal(markdownForPlatform(parts, system), '准备文件夹\n./mvnw\n' + literal)
  }
})

test('ordinary and unfinished staff-written material stays readable', () => {
  const plain = '### 步骤\n\n1. 打开项目\n2. 修改文字'
  assert.equal(markdownForPlatform(splitPlatformSections(plain), 'mac'), plain)
  assert.match(markdownForPlatform(splitPlatformSections(':::windows\n未写完'), 'mac'), /未写完/)
})
