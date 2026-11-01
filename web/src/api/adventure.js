import request from './request'

export const adventureApi = {
  mission: () => request.get('/adventure/mission'),
}
