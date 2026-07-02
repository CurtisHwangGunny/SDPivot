import client from './client'

export interface RegisterRequest {
  phone?: string
  email?: string
  password: string
  nickname?: string
}

export interface LoginRequest {
  phone?: string
  email?: string
  password: string
}

export interface AuthResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  user: {
    id: string
    username: string
    email: string
    phone?: string
    nickname?: string
    avatar: string
    tenant_id: number
    is_active: boolean
  }
}

export function register(data: RegisterRequest) {
  return client.post<AuthResponse>('/auth/register', data)
}

export function login(data: LoginRequest) {
  return client.post<AuthResponse>('/auth/login', data)
}

export function refreshToken(refreshToken: string) {
  return client.post<AuthResponse>('/auth/refresh', { refresh_token: refreshToken })
}

export function logout() {
  return client.post('/auth/logout', { refresh_token: localStorage.getItem('refresh_token') })
}

export function getCurrentUser() {
  return client.get('/auth/me')
}
