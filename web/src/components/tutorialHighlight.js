import highlighter from 'highlight.js/lib/core'

const languages = {
  xml: () => import('highlight.js/lib/languages/xml'),
  markdown: () => import('highlight.js/lib/languages/markdown'),
  css: () => import('highlight.js/lib/languages/css'),
  javascript: () => import('highlight.js/lib/languages/javascript'),
  bash: () => import('highlight.js/lib/languages/bash'),
  java: () => import('highlight.js/lib/languages/java'),
  json: () => import('highlight.js/lib/languages/json'),
  sql: () => import('highlight.js/lib/languages/sql'),
  yaml: () => import('highlight.js/lib/languages/yaml')
}
const aliases = { html: 'xml', js: 'javascript', md: 'markdown', sh: 'bash', yml: 'yaml' }

export async function highlightExample(text, label) {
  if (['text', 'plaintext', 'txt'].includes(label)) return null
  const language = aliases[label] || label
  let engine = highlighter
  if (!engine.getLanguage(language)) {
    if (Object.hasOwn(languages, language)) {
      const grammar = await languages[language]()
      if (!engine.getLanguage(language)) engine.registerLanguage(language, grammar.default)
    } else {
      // Teacher-authored lessons can also use the remaining common languages.
      engine = (await import('highlight.js/lib/common')).default
    }
  }
  return engine.getLanguage(language) ? engine.highlight(text, { language, ignoreIllegals: true }).value : null
}
