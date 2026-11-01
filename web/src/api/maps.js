import request from './request'

export const mapsApi = {
  list: () => request.get('/maps'),
  detail: id => request.get(`/maps/${id}`),
  progress: () => request.get('/maps/progress'),
  levelDetail: id => request.get(`/levels/${id}`),
  levelTree: () => request.get('/levels/tree'),
  mentorDialog: mapId => request.get('/mentor/dialog', { params: { mapId } })
}
