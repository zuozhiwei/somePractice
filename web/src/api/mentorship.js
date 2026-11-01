import request from './request'

export const mentorshipApi = {
  request: mentorId => request.post('/mentorship/request', { mentorId }),
  approve: id => request.put(`/mentorship/${id}/approve`),
  reject: id => request.put(`/mentorship/${id}/reject`),
  cancel: id => request.delete(`/mentorship/${id}/cancel`),
  quit: id => request.put(`/mentorship/${id}/quit`),
  inviteLink: () => request.get('/mentorship/invite-link'),
  my: () => request.get('/mentorship/my'),
  fellows: () => request.get('/mentorship/fellows'),
  apprenticePositions: () => request.get('/mentorship/apprentices'),
  searchUsers: q => request.get('/mentorship/search-users', { params: { q } })
}
