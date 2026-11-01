import request from './request'

export const adminApi = {
  // Levels
  listLevels: () => request.get('/admin/levels'),
  getLevel: id => request.get(`/admin/levels/${id}`),
  createLevel: data => request.post('/admin/levels', data),
  updateLevel: (id, data) => request.put(`/admin/levels/${id}`, data),
  deleteLevel: id => request.delete(`/admin/levels/${id}`),
  // Tasks
  createTask: data => request.post('/admin/tasks', data),
  updateTask: (id, data) => request.put(`/admin/tasks/${id}`, data),
  deleteTask: id => request.delete(`/admin/tasks/${id}`),
  moveTask: (id, direction) => request.put(`/admin/tasks/${id}/move`, null, { params: { direction } }),
  // Users
  listUsers: (page, size) => request.get('/admin/users', { params: { page, size } }),
  updateRole: (id, role) => request.put(`/admin/users/${id}/role`, null, { params: { role } }),
  setLevel: (id, level) => request.put(`/admin/users/${id}/level`, null, { params: { level } }),
  deleteUser: id => request.delete(`/admin/users/${id}`),
  // Submissions
  listSubmissions: () => request.get('/admin/submissions'),
  // Invite Codes
  listInviteCodes: () => request.get('/admin/invite-codes'),
  createInviteCode: data => request.post('/admin/invite-codes', data),
  deleteInviteCode: id => request.delete(`/admin/invite-codes/${id}`),
  // Mentorship Graph
  getMentorshipGraph: () => request.get('/admin/mentorship-graph'),

  // Tutorials
  listTutorials: levelId => request.get('/tutorials', { params: { levelId } }),
  createTutorial: data => request.post('/tutorials', data),
  updateTutorial: (id, data) => request.put(`/tutorials/${id}`, data),
  deleteTutorial: id => request.delete(`/tutorials/${id}`)
}
