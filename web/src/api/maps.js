import request from './request'

function versionCheck(etag) {
  if (!etag) return undefined
  return {
    headers: { 'If-None-Match': etag },
    validateStatus: status => status === 304 || (status >= 200 && status < 300)
  }
}

export const mapsApi = {
  list: () => request.get('/maps'),
  atlas: etag => request.get('/maps/atlas', versionCheck(etag)),
  visit: id => request.post(`/maps/${id}/visit`),
  detail: id => request.get(`/maps/${id}`),
  progress: () => request.get('/maps/progress'),
  levelDetail: (id, etag) => request.get(`/levels/${id}`, versionCheck(etag)),
  levelTree: () => request.get('/levels/tree'),
  mentorDialog: mapId => request.get('/mentor/dialog', { params: { mapId } })
}
