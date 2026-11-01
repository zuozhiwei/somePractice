import request from './request'

export const tasksApi = {
  list: levelId => request.get('/tasks', { params: { levelId } }),
  submit: (id, data) => request.post(`/tasks/${id}/submit`, data),
  taskSubmissions: taskId => request.get(`/tasks/${taskId}/submissions`),
  mySubmissions: (page = 0, size = 10) => request.get('/submissions/my', { params: { page, size } }),
  review: (id, data) => request.put(`/submissions/${id}/review`, data),
  aiReview: id => request.post(`/submissions/${id}/ai-review`, null, { timeout: 75000 }),
  apprenticeSubmissions: (page, size) => request.get('/submissions/apprentice', { params: { page, size } }),
  externalSubmissions: (page, size) => request.get('/submissions/external', { params: { page, size } }),
  submissionDetail: id => request.get(`/submissions/${id}`)
}
