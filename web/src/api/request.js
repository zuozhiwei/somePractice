import axios from 'axios'
import { useAuthStore } from '../store/auth'

const request = axios.create({
  baseURL: '/api',
  timeout: 15000
})

request.interceptors.request.use(config => {
  const authStore = useAuthStore()
  if (authStore.token) {
    config.headers.Authorization = `Bearer ${authStore.token}`
  }
  return config
})

request.interceptors.response.use(
  res => res,
  err => {
    if (err.response?.status === 401 && err.config?.headers?.Authorization) {
      const authStore = useAuthStore()
      if (authStore.token && err.config.headers.Authorization === 'Bearer ' + authStore.token) {
        const next = window.location.pathname + window.location.search + window.location.hash
        authStore.logout()
        window.location.replace('/login?' + new URLSearchParams({ next }))
      }
    }
    return Promise.reject(err)
  }
)

export default request
