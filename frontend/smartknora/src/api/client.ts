import axios from 'axios'
import type { AuthResponse, User, Organization, KnowledgeSpace, SpaceMember, SpaceCategory, TokenUsageSummary, OrgMember } from '../types'

const API_BASE = '/api/v1/smartknora'

const client = axios.create({
  baseURL: API_BASE,
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
})

// Request interceptor: attach JWT
client.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Mutex for token refresh to prevent race conditions
let isRefreshing = false
let failedQueue: Array<{ resolve: Function; reject: Function }> = []

const processQueue = (error: any, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error)
    } else {
      prom.resolve(token)
    }
  })
  failedQueue = []
}

// Response interceptor: handle 401
client.interceptors.response.use(
  (res) => res,
  async (error) => {
    if (error.response?.status === 401) {
      const refreshToken = localStorage.getItem('refresh_token')
      if (!refreshToken) {
        localStorage.clear()
        setTimeout(() => { window.location.href = '/login' }, 100)
        return Promise.reject(error)
      }

      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject })
        }).then((token) => {
          error.config.headers.Authorization = `Bearer ${token}`
          return client.request(error.config)
        })
      }

      isRefreshing = true

      try {
        const res = await axios.post(`${API_BASE}/auth/refresh`, {
          refresh_token: refreshToken,
        })
        const { access_token, refresh_token } = res.data
        localStorage.setItem('access_token', access_token)
        localStorage.setItem('refresh_token', refresh_token)
        error.config.headers.Authorization = `Bearer ${access_token}`
        processQueue(null, access_token)
        return client.request(error.config)
      } catch (refreshError) {
        processQueue(refreshError, null)
        localStorage.clear()
        setTimeout(() => { window.location.href = '/login' }, 100)
        return Promise.reject(refreshError)
      } finally {
        isRefreshing = false
      }
    }
    return Promise.reject(error)
  }
)

// ─── Auth ────────────────────────────────────────────────────
export const authApi = {
  register: (data: { phone?: string; email?: string; password: string; nickname?: string }) =>
    client.post<AuthResponse>('/auth/register', data),

  login: (data: { phone?: string; email?: string; password: string }) =>
    client.post<AuthResponse>('/auth/login', data),

  refresh: (refreshToken: string) =>
    client.post<{ access_token: string; refresh_token: string; expires_in: number }>('/auth/refresh', { refresh_token: refreshToken }),

  logout: (refreshToken: string) =>
    client.post('/auth/logout', { refresh_token: refreshToken }),
}

// ─── User Profile ────────────────────────────────────────────
export const userApi = {
  getProfile: () => client.get<{ user_id: string }>('/profile'),
  updateProfile: (data: { nickname?: string; avatar_url?: string; phone?: string }) =>
    client.put('/profile', data),
  changePassword: (data: { old_password: string; new_password: string }) =>
    client.put('/password', data),
}

// ─── Organizations ───────────────────────────────────────────
export const orgApi = {
  create: (data: { name: string; description?: string; logo_url?: string }) =>
    client.post<{ organization: Organization }>('/organizations', data),

  list: () => client.get<{ organizations: Array<{ organization: Organization; auth_status: string; days_remaining: number }> }>('/organizations'),

  get: (id: string) =>
    client.get<{ organization: Organization; auth_status: string; days_remaining: number }>(`/organizations/${id}`),

  update: (id: string, data: { name?: string; description?: string; logo_url?: string }) =>
    client.put(`/organizations/${id}`, data),

  join: (inviteCode: string) =>
    client.post('/organizations/join', { invite_code: inviteCode }),

  listMembers: (id: string) => client.get<{ members: OrgMember[] }>(`/organizations/${id}/members`),

  addMember: (id: string, userId: string, role?: string) =>
    client.post(`/organizations/${id}/members`, { user_id: userId, role }),

  removeMember: (id: string, userId: string) =>
    client.delete(`/organizations/${id}/members/${userId}`),

  updateMemberRole: (id: string, userId: string, role: string) =>
    client.put(`/organizations/${id}/members/${userId}/role`, { role }),
}

// ─── Knowledge Spaces ────────────────────────────────────────
export const spaceApi = {
  create: (data: { name: string; description?: string; visibility?: string; icon?: string; org_id?: string }) =>
    client.post<{ space: KnowledgeSpace }>('/spaces', data),

  list: () => client.get<{ spaces: KnowledgeSpace[] }>('/spaces'),

  get: (id: string) => client.get<{ space: KnowledgeSpace }>(`/spaces/${id}`),

  update: (id: string, data: { name?: string; description?: string; visibility?: string; icon?: string }) =>
    client.put(`/spaces/${id}`, data),

  delete: (id: string) => client.delete(`/spaces/${id}`),

  listMembers: (id: string) => client.get<{ members: SpaceMember[] }>(`/spaces/${id}/members`),

  addMember: (id: string, userId: string, role?: string) =>
    client.post(`/spaces/${id}/members`, { user_id: userId, role }),

  removeMember: (id: string, userId: string) =>
    client.delete(`/spaces/${id}/members/${userId}`),

  updateMemberRole: (id: string, userId: string, role: string) =>
    client.put(`/spaces/${id}/members/${userId}`, { role }),
}

// ─── Categories ──────────────────────────────────────────────
export const categoryApi = {
  create: (data: { name: string; color?: string }) =>
    client.post<{ category: SpaceCategory }>('/categories', data),

  list: () => client.get<{ categories: SpaceCategory[] }>('/categories'),

  delete: (id: string) => client.delete(`/categories/${id}`),
}

// ─── Token Usage ─────────────────────────────────────────────
export const usageApi = {
  summary: (params?: { org_id?: string; user_id?: string; start_at?: string; end_at?: string }) =>
    client.get<{ summary: TokenUsageSummary }>('/usage/summary', { params }),

  history: (params?: { group_by?: string; start_at?: string; end_at?: string }) =>
    client.get('/usage/history', { params }),

  byModel: () => client.get('/usage/by-model'),
}

export default client
