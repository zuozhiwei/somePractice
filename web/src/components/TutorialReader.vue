<template>
  <div class="tutorial-reader">
    <div v-html="renderedMarkdown"></div>
    <div v-if="plantumlSrc" class="plantuml-block">
      <h4>PlantUML 图表</h4>
      <img :src="plantumlSrc" alt="PlantUML diagram" style="max-width:100%" />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import hljs from 'highlight.js'
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
  marked.setOptions({
    highlight: (code, lang) => {
      if (lang && hljs.getLanguage(lang)) {
        return hljs.highlight(code, { language: lang }).value
      }
      return hljs.highlightAuto(code).value
    }
  })
  return DOMPurify.sanitize(marked.parse(cleaned))
})
</script>

<style scoped>
.tutorial-reader { line-height: 1.8; }
.tutorial-reader :deep(pre) { background: #f6f8fa; padding: 12px; border-radius: 6px; overflow-x: auto; }
.tutorial-reader :deep(code) { font-size: 13px; }
.tutorial-reader :deep(h1) { font-size: 22px; margin: 16px 0 8px; }
.tutorial-reader :deep(h2) { font-size: 18px; margin: 14px 0 6px; }
.tutorial-reader :deep(p) { margin: 8px 0; }
.plantuml-block { margin-top: 16px; padding: 12px; background: #fafafa; border-radius: 8px; }
</style>
