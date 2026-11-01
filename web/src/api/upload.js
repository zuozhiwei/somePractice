import request from './request'

export const uploadApi = {
  // 获取七牛直传凭证（登录用户可用）
  token: () => request.get('/upload/token')
}
