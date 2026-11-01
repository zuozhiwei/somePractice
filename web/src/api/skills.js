import request from './request'

export const skillsApi = {
  getTree: () => request.get('/levels/tree'),
  getDetail: id => request.get(`/levels/${id}`)
}
