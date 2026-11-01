import request from './request'

export const leaderboardApi = {
  global: (page, size) => request.get('/leaderboard/global', { params: { page, size } })
}
