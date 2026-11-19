import request from './request'

export const mapsApi = {
  list: () => request.get('/maps'),
  atlas: () => request.get('/maps/atlas'),
  visit: id => request.post(`/maps/${id}/visit`),
  detail: id => request.get(`/maps/${id}`),
  progress: () => request.get('/maps/progress'),
  levelDetail: id => request.get(`/levels/${id}`),
  levelTree: () => request.get('/levels/tree'),
  mentorDialog: mapId => request.get('/mentor/dialog', { params: { mapId } })
}
