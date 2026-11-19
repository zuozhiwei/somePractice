<template>
  <div class="tutorial-reader">
    <div ref="article" v-html="renderedMarkdown"></div>
    <div v-if="plantumlSrc" class="plantuml-block">
      <h4>PlantUML 图表</h4>
      <img :src="plantumlSrc" alt="PlantUML diagram" style="max-width:100%" />
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted, watch } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import 'highlight.js/styles/github.css'
import plantumlEncoder from 'plantuml-encoder'

const props = defineProps({ content: String })

const plantumlSrc = computed(() => {
  if (!props.content) return ''
  const match = props.content.match(/@startuml([\s\S]*?)@enduml/)
  if (!match) return ''
  const encoded = plantumlEncoder.encode('@startuml\n' + match[1].trim() + '\n@enduml')
  return `https://www.plantuml.com/plantuml/svg/${encoded}`
})

const renderedMarkdown = computed(() => {
  if (!props.content) return ''
  const cleaned = props.content.replace(/```plantuml[\s\S]*?```/g, '')
  return DOMPurify.sanitize(marked.parse(cleaned))
})
const article = ref(null)
async function highlightCode() {
  const blocks = Array.from(article.value?.querySelectorAll('pre code[class*="language-"]') || [])
  if (!blocks.length) return
  try {
    const { default: highlighter } = await import('highlight.js/lib/common')
    for (const block of blocks) {
      if (!block.isConnected || block.dataset.highlighted) continue
      const language = [...block.classList].find(name => name.startsWith('language-'))?.slice(9)
      if (!language || !highlighter.getLanguage(language)) continue
      const result = highlighter.highlight(block.textContent, { language, ignoreIllegals: true })
      block.innerHTML = DOMPurify.sanitize(result.value, { ALLOWED_TAGS: ['span'], ALLOWED_ATTR: ['class'] })
      block.dataset.highlighted = 'yes'
    }
  } catch { /* The sanitized code remains readable if the optional highlighting chunk fails. */ }
}
onMounted(highlightCode)
watch(renderedMarkdown, highlightCode, { flush: 'post' })
</script>

<style scoped>
.tutorial-reader { line-height: 1.85; overflow-wrap: anywhere; min-width: 0; }
.tutorial-reader :deep(img) { max-width: 100%; height: auto; }
.tutorial-reader :deep(table) { display: block; overflow-x: auto; border-collapse: collapse; }
.tutorial-reader :deep(td), .tutorial-reader :deep(th) { border: 1px solid var(--color-border); padding: 5px 9px; }
.tutorial-reader :deep(ul),.tutorial-reader :deep(ol) { padding-inline-start: 1.5em; }
.tutorial-reader :deep(pre) { background: #f6f8fa; padding: 12px; border-radius: 6px; overflow-x: auto; }
.tutorial-reader :deep(code) { font-size: 13px; }
.tutorial-reader :deep(h1) { font-size: 22px; margin: 16px 0 8px; }
.tutorial-reader :deep(h2) { font-size: 18px; margin: 14px 0 6px; }
.tutorial-reader :deep(p) { margin: 8px 0; }
.plantuml-block { margin-top: 16px; padding: 12px; background: #fafafa; border-radius: 8px; }
</style>
