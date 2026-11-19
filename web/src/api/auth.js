import request from './request'

export const authApi = {
  login: data => request.post('/auth/login', data),
  register: data => request.post('/auth/register', data),
  profile: () => request.get('/auth/profile'),
  updateProfile: data => request.put('/auth/profile', data),
  forgotPassword: identifier => request.post('/auth/forgot-password', { identifier }, { timeout: 30000 }),
  resetPassword: data => request.post('/auth/reset-password', data)
}
