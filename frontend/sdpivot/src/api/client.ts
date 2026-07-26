import axios from 'axios'
import { STORAGE_KEYS } from '../utils/storage'

const client = axios.create({
  baseURL: '/api/v1/sdp',
  timeout: 30000,
})

client.interceptors.request.use((config) => {
  const token = localStorage.getItem(STORAGE_KEYS.accessToken)
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

client.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401) {
      if (error.config?.url === '/auth/login') return Promise.reject(error)

      const refreshToken = localStorage.getItem(STORAGE_KEYS.refreshToken)
      if (refreshToken && !error.config._retry) {
        error.config._retry = true
        try {
          const res = await axios.post('/api/v1/sdp/auth/refresh', { refresh_token: refreshToken })
          localStorage.setItem(STORAGE_KEYS.accessToken, res.data.access_token)
          localStorage.setItem(STORAGE_KEYS.refreshToken, res.data.refresh_token)
          error.config.headers.Authorization = `Bearer ${res.data.access_token}`
          return client(error.config)
        } catch {
          localStorage.removeItem(STORAGE_KEYS.accessToken)
          localStorage.removeItem(STORAGE_KEYS.refreshToken)
          window.location.href = '/login'
        }
      }
      localStorage.removeItem(STORAGE_KEYS.accessToken)
      localStorage.removeItem(STORAGE_KEYS.refreshToken)
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default client
