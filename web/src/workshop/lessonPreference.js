import { ref } from 'vue'

const key = 'workshop.lesson-platform'
const supported = ['windows', 'mac', 'linux']
function normalize(value) {
  return supported.includes(value) ? value : 'windows'
}

export const lessonPlatform = ref('windows')
try { lessonPlatform.value = normalize(window.localStorage.getItem(key)) } catch { /* Storage may be unavailable. */ }

export function setLessonPlatform(value) {
  lessonPlatform.value = normalize(value)
  try { window.localStorage.setItem(key, lessonPlatform.value) } catch { /* Keep the selection for this visit. */ }
}

window.addEventListener('storage', event => {
  if (event.key === key || event.key === null) lessonPlatform.value = normalize(event.newValue)
})
