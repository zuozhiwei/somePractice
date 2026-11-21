import request from './request'

export const tasksApi = {
  list: levelId => request.get('/tasks', { params: { levelId } }),
  submit: (id, data) => request.post(`/tasks/${id}/submit`, data),
  taskSubmissions: (taskId, limit) => request.get(`/tasks/${taskId}/submissions`, { params: { limit } }),
  mySubmissions: (page = 0, size = 10) => request.get('/submissions/my', { params: { page, size } }),
  review: (id, data) => request.put(`/submissions/${id}/review`, data),
  aiReview: id => request.post(`/submissions/${id}/ai-review`, null, { timeout: 75000 }),
  apprenticeSubmissions: (page, size, queue) => request.get('/submissions/apprentice', { params: { page, size, queue } }),
  externalSubmissions: (page, size, queue) => request.get('/submissions/external', { params: { page, size, queue } }),
  submissionDetail: id => request.get(`/submissions/${id}`)
}
