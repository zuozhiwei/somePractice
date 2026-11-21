<template>
  <div class="tutorial-reader">
    <fieldset v-if="platformOptions.length" class="platform-picker">
      <legend>我的电脑</legend>
      <div class="platform-options">
        <label v-for="option in platformOptions" :key="option.value">
          <input type="radio" :name="platformGroup" :value="option.value" :checked="platform === option.value"
                 @change="setLessonPlatform(option.value)" />
          <span>{{ option.label }}</span>
        </label>
      </div>
    </fieldset>
    <details v-if="outline.length >= 3" class="guide-outline">
      <summary>参考目录 <span>共 {{ outline.length }} 节</span></summary>
      <ol>
        <li v-for="(title, index) in outline" :key="index">
          <button type="button" @click="goToStep(index)">{{ title }}</button>
        </li>
      </ol>
    </details>
    <div ref="article" class="markdown-body" @click="followLessonLink" v-html="renderedMarkdown"></div>
    <div v-if="plantumlSrc" class="plantuml-block">
      <h4>PlantUML 图表</h4>
      <img :src="plantumlSrc" alt="本节教程的 PlantUML 图表" loading="lazy" />
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted, watch, useId } from 'vue'
import { useRouter } from 'vue-router'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import 'highlight.js/styles/github.css'
import { splitPlatformSections, markdownForPlatform } from '../workshop/lessonPlatforms'
import { lessonPlatform, setLessonPlatform } from '../workshop/lessonPreference'

const props = defineProps({ content: String, showOutline: Boolean, sectionAnchor: String })

const platformGroup = useId()
const sections = computed(() => splitPlatformSections(props.content || ''))
const platformOptions = computed(() => {
  const branches = sections.value.filter(section => section.systems.length)
  if (!branches.length) return []
  const options = [{ value: 'windows', label: 'Windows' }, { value: 'mac', label: 'Mac' }]
  if (branches.some(section => section.systems.length === 1 && section.systems[0] === 'linux')
    || (lessonPlatform.value === 'linux' && branches.some(section => section.systems.includes('linux')))) {
    options.push({ value: 'linux', label: 'Linux' })
  }
  return options
})
const platform = computed(() => platformOptions.value.some(option => option.value === lessonPlatform.value)
  ? lessonPlatform.value : 'windows')
const content = computed(() => markdownForPlatform(sections.value, platform.value))
const router = useRouter()
const plantumlSrc = ref('')
watch(() => content.value.match(/@startuml([\s\S]*?)@enduml/)?.[0], async (diagram, _, onCleanup) => {
  let current = true
  onCleanup(() => { current = false })
  plantumlSrc.value = ''
  if (!diagram) return
  try {
    const { default: encoder } = await import('plantuml-encoder')
    if (current) plantumlSrc.value = 'https://www.plantuml.com/plantuml/svg/' + encoder.encode(diagram)
  } catch { /* Leave the diagram source available when its encoder cannot load. */ }
}, { immediate: true })

const renderedMarkdown = computed(() => {
  if (!content.value) return ''
  const cleaned = plantumlSrc.value ? content.value.replace(/```plantuml[\s\S]*?```/g, '') : content.value
  return DOMPurify.sanitize(marked.parse(cleaned))
})
const article = ref(null)
const outline = ref([])
function headings() {
  return Array.from(article.value?.querySelectorAll('h2, h3') || [])
}
function goToStep(index) {
  const heading = headings()[index]
  if (!heading) return
  heading.tabIndex = -1
  heading.focus({ preventScroll: true })
  heading.scrollIntoView({ block: 'start' })
}

function scrollToSection() {
  if (!props.sectionAnchor) return
  const fragment = props.sectionAnchor.slice(1)
  let title = fragment
  try { title = decodeURIComponent(fragment) } catch { /* A heading may contain a literal percent sign. */ }
  const index = headings().findIndex(heading => [title, fragment].includes(heading.textContent.trim()))
  if (index >= 0) goToStep(index)
}

function followLessonLink(event) {
  const link = event.target.closest('a[href]')
  if (!link || link.target || link.hasAttribute('download') || event.defaultPrevented || event.button !== 0
    || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) return
  if (link.origin !== window.location.origin || !/^\/level\/\d+$/.test(link.pathname)) return
  event.preventDefault()
  const destination = link.pathname + link.search + link.hash
  if (router.resolve(destination).fullPath === router.currentRoute.value.fullPath) scrollToSection()
  else router.push(destination)
}

function addCopyButton(code) {
  const pre = code.parentNode
  if (pre.parentNode.classList.contains('code-example')) return
  const wrapper = document.createElement('div')
  wrapper.className = 'code-example'
  const button = document.createElement('button')
  button.type = 'button'
  button.className = 'copy-example'
  button.textContent = '复制'
  button.setAttribute('aria-label', '复制这段示例')
  button.setAttribute('aria-live', 'polite')
  pre.before(wrapper)
  wrapper.append(button, pre)
  button.addEventListener('click', async () => {
    try {
      await navigator.clipboard.writeText(code.textContent)
      button.textContent = '已复制'
      button.setAttribute('aria-label', '已复制这段示例')
    } catch {
      const range = document.createRange()
      range.selectNodeContents(code)
      const selection = window.getSelection()
      selection.removeAllRanges()
      selection.addRange(range)
      button.textContent = '已选中，请使用系统复制'
      button.setAttribute('aria-label', button.textContent)
    }
  })
}
async function highlightCode() {
  outline.value = props.showOutline ? headings().map(heading => heading.textContent) : []
  const blocks = Array.from(article.value?.querySelectorAll('pre > code') || [])
  blocks.forEach(addCopyButton)
  if (!blocks.length) return
  try {
    const { highlightExample } = await import('./tutorialHighlight')
    for (const block of blocks) {
      if (!block.isConnected || block.dataset.highlighted) continue
      const language = [...block.classList].find(name => name.startsWith('language-'))?.slice(9)
      if (!language) continue
      const result = await highlightExample(block.textContent, language)
      if (result == null || !block.isConnected) continue
      block.innerHTML = DOMPurify.sanitize(result, { ALLOWED_TAGS: ['span'], ALLOWED_ATTR: ['class'] })
      block.dataset.highlighted = 'yes'
    }
  } catch { /* The sanitized code remains readable if the optional highlighting chunk fails. */ }
}
function prepareArticle() {
  for (const link of article.value?.querySelectorAll('a[href]') || []) {
    if (['http:', 'https:'].includes(link.protocol) && link.origin !== window.location.origin) {
      link.target = '_blank'
      link.rel = 'noopener noreferrer'
    }
  }
  highlightCode()
  scrollToSection()
}
onMounted(prepareArticle)
watch(renderedMarkdown, prepareArticle, { flush: 'post' })
watch(() => props.sectionAnchor, scrollToSection, { flush: 'post' })
</script>

<style scoped>
.tutorial-reader { font-size: 15px; line-height: 1.8; overflow-wrap: anywhere; min-width: 0; }
.platform-picker { display: flex; align-items: center; flex-wrap: wrap; gap: 6px 12px; border: 0; margin: 0 0 14px; padding: 0; min-width: 0; }
.platform-picker legend { float: left; margin: 0; padding: 7px 0; font-size: 13px; color: var(--color-text-secondary); }
.platform-options { display: inline-flex; border: 1px solid var(--color-border); background: #f2f5ef; border-radius: 9px; padding: 2px; }
.platform-options label { position: relative; cursor: pointer; }
.platform-options input { position: absolute; opacity: 0; width: 1px; height: 1px; }
.platform-options span { display: grid; place-items: center; min-height: 34px; min-width: 52px; padding: 0 11px; border-radius: 6px; color: #42594b; font-size: 13px; }
.platform-options input:checked + span { color: white; background: #235747; font-weight: 650; }
.platform-options input:focus-visible + span { outline: 2px solid var(--color-primary); outline-offset: 2px; }
.guide-outline { border-block: 1px solid var(--color-border); margin-block: 2px 18px; }
.guide-outline summary { cursor: pointer; font-weight: 650; padding-block: 9px; color: var(--color-primary); }
.guide-outline summary span { margin-inline-start: 5px; font-size: 12px; color: var(--color-text-secondary); font-weight: 400; }
.guide-outline ol { list-style: none; padding: 0 0 7px; }
.guide-outline button, .tutorial-reader :deep(.copy-example) {
  border: 0; background: transparent; color: var(--color-primary); text-align: start;
}
.guide-outline button { padding: 4px 0; text-decoration: underline; text-underline-offset: 3px; }
.markdown-body :deep(h2), .markdown-body :deep(h3) { scroll-margin-top: calc(var(--mobile-header-height, 55px) + 10px); }
.tutorial-reader :deep(.code-example) {
  display: flow-root; margin-block: 14px; border: 1px solid var(--color-border);
  border-radius: 9px; background: #f4f5ee; text-align: end;
}
.tutorial-reader :deep(.copy-example) { padding: 5px 12px; min-height: 32px; font: inherit; font-size: 12px; font-weight: 650; }
.tutorial-reader :deep(.code-example pre) { margin: 0; padding-top: 0; border: 0; text-align: start; }
.tutorial-reader :deep(img) { max-width: 100%; height: auto; border-radius: 8px; }
.tutorial-reader :deep(a) { color: var(--color-primary); text-underline-offset: 3px; }
.tutorial-reader :deep(table) { display: block; overflow-x: auto; border-collapse: collapse; margin-block: 14px; }
.tutorial-reader :deep(td), .tutorial-reader :deep(th) { border: 1px solid var(--color-border); padding: 6px 10px; }
.tutorial-reader :deep(th) { text-align: start; background: #edf2e8; }
.tutorial-reader :deep(ul), .tutorial-reader :deep(ol) { padding-inline-start: 1.5em; margin-block: 9px; }
.tutorial-reader :deep(li + li) { margin-top: 5px; }
.tutorial-reader :deep(pre) {
  background: #f4f5ee; margin-block: 14px; padding: 13px 16px; border: 1px solid var(--color-border);
  border-radius: 9px; overflow-x: auto; overscroll-behavior-inline: contain;
}
.tutorial-reader :deep(code) { font-size: .88em; }
.tutorial-reader :deep(:not(pre) > code) { background: #edf1e7; color: #215745; padding: .12em .32em; border-radius: 4px; }
.tutorial-reader :deep(pre code) { color: #263b31; overflow-wrap: normal; }
.tutorial-reader :deep(h1) { font-size: 22px; margin: 19px 0 8px; }
.tutorial-reader :deep(h2) { font-size: 18px; margin: 18px 0 6px; }
.tutorial-reader :deep(h3), .tutorial-reader :deep(h4) { font-size: 16px; margin: 15px 0 5px; }
.tutorial-reader :deep(p) { margin: 8px 0; }
.tutorial-reader :deep(blockquote) {
  border-inline-start: 3px solid #86a08c; background: #f2f5ed; color: #3e5147; border-radius: 4px;
  padding: 5px 13px; margin-block: 14px;
}
.tutorial-reader :deep(hr) { border: 0; border-top: 1px solid var(--color-border); margin-block: 18px; }
.markdown-body > :deep(:first-child) { margin-block-start: 0; }
.plantuml-block { margin-top: 16px; padding: 12px; background: #f4f5ee; border-radius: 8px; }
</style>
