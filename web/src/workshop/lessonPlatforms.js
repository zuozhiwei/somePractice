// Platform directives occupy whole lines outside Markdown code fences.
const platformStart = /^:::(windows|mac|linux)((?: (?:windows|mac|linux))*)\s*$/

export function splitPlatformSections(markdown = '') {
  if (!markdown.includes(':::')) return [{ systems: [], text: markdown }]
  const sections = []
  let lines = [], systems = [], fence = ''
  function finishSection() {
    if (lines.length) sections.push({ systems, text: lines.join('\n') })
    lines = []
  }
  for (const line of markdown.split(/\r?\n/)) {
    const marker = line.match(/^ {0,3}(`{3,}|~{3,})(.*)$/)
    if (fence) {
      if (marker && marker[1][0] === fence[0] && marker[1].length >= fence.length && !marker[2].trim()) fence = ''
    } else if (marker) {
      fence = marker[1]
    } else if (systems.length && line.trimEnd() === ':::') {
      finishSection()
      systems = []
      continue
    } else if (!systems.length && platformStart.test(line)) {
      finishSection()
      systems = line.slice(3).trim().split(/ +/)
      continue
    }
    lines.push(line)
  }
  // Keep unfinished staff-authored sections readable while they're being edited.
  if (systems.length) systems = []
  finishSection()
  return sections
}

export function markdownForPlatform(sections, platform) {
  return sections.filter(section => !section.systems.length || section.systems.includes(platform))
    .map(section => section.text).join('\n')
}
