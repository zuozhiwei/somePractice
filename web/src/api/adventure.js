import request from './request'

export const adventureApi = {
  mission: chapterId => request.get('/adventure/mission', { params: { chapterId: chapterId || undefined } }),
}
