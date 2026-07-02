import { create } from 'zustand'
import type { User } from '../types'
import { authApi } from '../api/client'

interface AuthState {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (phone: string, password: string) => Promise<void>
  loginByEmail: (email: string, password: string) => Promise<void>
  register: (data: { phone?: string; email?: string; password: string; nickname?: string }) => Promise<void>
  logout: () => void
  loadFromStorage: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  isLoading: false,

  login: async (phone, password) => {
    set({ isLoading: true })
    try {
      const res = await authApi.login({ phone, password })
      const { access_token, refresh_token, user } = res.data
      localStorage.setItem('access_token', access_token)
      localStorage.setItem('refresh_token', refresh_token)
      localStorage.setItem('user', JSON.stringify(user))
      set({ user, isAuthenticated: true, isLoading: false })
    } catch (err) {
      set({ isLoading: false })
      throw err
    }
  },

  loginByEmail: async (email, password) => {
    set({ isLoading: true })
    try {
      const res = await authApi.login({ email, password })
      const { access_token, refresh_token, user } = res.data
      localStorage.setItem('access_token', access_token)
      localStorage.setItem('refresh_token', refresh_token)
      localStorage.setItem('user', JSON.stringify(user))
      set({ user, isAuthenticated: true, isLoading: false })
    } catch (err) {
      set({ isLoading: false })
      throw err
    }
  },

  register: async (data) => {
    set({ isLoading: true })
    try {
      const res = await authApi.register(data)
      const { access_token, refresh_token, user } = res.data
      localStorage.setItem('access_token', access_token)
      localStorage.setItem('refresh_token', refresh_token)
      localStorage.setItem('user', JSON.stringify(user))
      set({ user, isAuthenticated: true, isLoading: false })
    } catch (err) {
      set({ isLoading: false })
      throw err
    }
  },

  logout: () => {
    const refreshToken = localStorage.getItem('refresh_token')
    if (refreshToken) {
      authApi.logout(refreshToken).catch((err) => { console.error('API error:', err) })
    }
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    localStorage.removeItem('user')
    set({ user: null, isAuthenticated: false })
  },

  loadFromStorage: () => {
    const token = localStorage.getItem('access_token')
    const userStr = localStorage.getItem('user')
    if (token && userStr) {
      try {
        const user = JSON.parse(userStr)
        set({ user, isAuthenticated: true })
      } catch {
        localStorage.clear()
      }
    }
  },
}))
