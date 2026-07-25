import axios from 'axios'
import { STORAGE_KEYS } from '../utils/storage'

const opsClient = axios.create({
  baseURL: '/api/v1/sdp',
  timeout: 30000,
})

let refreshPromise: Promise<string> | null = null

opsClient.interceptors.request.use((config) => {
  const token = localStorage.getItem(STORAGE_KEYS.opsAccessToken)
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

function clearOpsSession() {
  localStorage.removeItem(STORAGE_KEYS.opsAccessToken)
  localStorage.removeItem(STORAGE_KEYS.opsRefreshToken)
  localStorage.removeItem(STORAGE_KEYS.opsUser)
}

async function refreshOpsToken() {
  const refreshToken = localStorage.getItem(STORAGE_KEYS.opsRefreshToken)
  if (!refreshToken) throw new Error('missing ops refresh token')
  const response = await axios.post('/api/v1/sdp/ops/refresh', { refresh_token: refreshToken })
  const accessToken = response.data.access_token
  localStorage.setItem(STORAGE_KEYS.opsAccessToken, accessToken)
  localStorage.setItem(STORAGE_KEYS.opsRefreshToken, response.data.refresh_token)
  return accessToken
}

function redirectToOpsLogin() {
  if (window.location.pathname !== '/ops-login') window.location.href = '/ops-login'
}

opsClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config as any
    if (error.response?.status === 401 && originalRequest && !originalRequest._retry) {
      originalRequest._retry = true
      try {
        if (!refreshPromise) refreshPromise = refreshOpsToken().finally(() => { refreshPromise = null })
        const accessToken = await refreshPromise
        originalRequest.headers = originalRequest.headers || {}
        originalRequest.headers.Authorization = `Bearer ${accessToken}`
        return opsClient(originalRequest)
      } catch {
        clearOpsSession()
        redirectToOpsLogin()
      }
    } else if (error.response?.status === 401) {
      clearOpsSession()
      redirectToOpsLogin()
    }
    return Promise.reject(error)
  },
)

export function getOpsDashboard() { return opsClient.get('/ops/dashboard') }
export function listEnterprises(params?: any) { return opsClient.get('/ops/enterprises', { params }) }
export function getEnterprise(id: string) { return opsClient.get(`/ops/enterprises/${id}`) }
export function updateEnterpriseStatus(id: string, status: string) { return opsClient.put(`/ops/enterprises/${id}/status`, { status }) }
export function listUsers(params?: any) { return opsClient.get('/ops/users', { params }) }
export function createUser(data: { phone?: string; email?: string; password: string; nickname?: string }) { return opsClient.post('/ops/users', data) }
export function importUsers(file: File) {
  const form = new FormData()
  form.append('file', file)
  return opsClient.post<{
    total: number
    imported: number
    failed: number
    errors: Array<{ row: number; field?: string; value?: string; message: string }>
  }>('/ops/users/import', form)
}
export function updateUserStatus(id: string, isActive: boolean) { return opsClient.put(`/ops/users/${id}/status`, { is_active: isActive }) }
export function getAuditLogs(params?: any) { return opsClient.get('/ops/audit-logs', { params }) }
export function exportAuditLogs(params?: any) { return opsClient.get('/ops/audit-logs/export', { params, responseType: 'blob' }) }
export function createAnnouncement(data: { title: string; content: string }) { return opsClient.post('/ops/announcements', data) }
export function listAnnouncements() { return opsClient.get('/ops/announcements') }
export function deleteAnnouncement(id: string) { return opsClient.delete(`/ops/announcements/${id}`) }
export function getActiveAnnouncements() { return opsClient.get('/ops/announcements/active') }
export function listSensitiveWords(params?: any) { return opsClient.get('/ops/filters/words', { params }) }
export function createSensitiveWord(data: { word: string; category?: string }) { return opsClient.post('/ops/filters/words', data) }
export function deleteSensitiveWord(id: string) { return opsClient.delete(`/ops/filters/words/${id}`) }
export function listFilterHits(params?: any) { return opsClient.get('/ops/filters/hits', { params }) }
export function updateFilterHit(orgId: string, action: string) { return opsClient.put(`/ops/filters/hits/${orgId}`, { action }) }
export function listConfigs() { return opsClient.get('/ops/config') }
export function updateConfig(key: string, value: string, description?: string) { return opsClient.put(`/ops/config/${key}`, { value, description }) }
export function getTrialConfig() { return opsClient.get('/ops/config/trial') }
export function updateTrialConfig(data: { trial_days: number; extended_trial_days: number }) { return opsClient.put('/ops/config/trial', data) }
export function listModels() { return opsClient.get('/ops/models') }
export function createModel(data: any) { return opsClient.post('/ops/models', data) }
export function updateModel(id: string, data: any) { return opsClient.put(`/ops/models/${id}`, data) }
export function deleteModel(id: string) { return opsClient.delete(`/ops/models/${id}`) }
export function setDefaultModel(id: string) { return opsClient.put(`/ops/models/${id}/default`) }

export default opsClient
