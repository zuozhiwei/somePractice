import { reactive } from 'vue'

export const submissionPageSize = 10

export function submissionPage(fetch, queue) {
  return reactive({ fetch, queue, rows: [], page: 1, total: 0, loading: false, failed: false, request: 0 })
}

export function reviewPages(fetch) {
  return reactive({
    mode: 'pending',
    pending: submissionPage(fetch, 'pending'),
    reviewed: submissionPage(fetch, 'reviewed'),
    get current() { return this[this.mode] }
  })
}

export async function loadSubmissionPage(list, page = list.page) {
  const request = ++list.request
  list.page = page
  list.loading = true
  list.failed = false
  try {
    const { data: response } = await list.fetch(page - 1, submissionPageSize, list.queue)
    if (request !== list.request) return
    const { content, total } = response.data
    const last = Math.max(1, Math.ceil(total / submissionPageSize))
    if (page > last) return loadSubmissionPage(list, last)
    list.rows = content
    list.total = total
  } catch {
    if (request === list.request) list.failed = true
  } finally {
    if (request === list.request) list.loading = false
  }
}
