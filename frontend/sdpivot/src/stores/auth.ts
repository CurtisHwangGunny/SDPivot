import { defineStore } from 'pinia'
import { ref } from 'vue'
import { STORAGE_KEYS } from '../utils/storage'

function loadStoredUser() {
  const storedUser = localStorage.getItem(STORAGE_KEYS.user)
  if (!storedUser) return null
  try {
    return JSON.parse(storedUser)
  } catch {
    localStorage.removeItem(STORAGE_KEYS.user)
    return null
  }
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem(STORAGE_KEYS.accessToken) || '')
  const refreshToken = ref(localStorage.getItem(STORAGE_KEYS.refreshToken) || '')
  const user = ref<any>(loadStoredUser())

  function setAuth(data: { access_token: string; refresh_token: string; user: any }) {
    token.value = data.access_token
    refreshToken.value = data.refresh_token
    user.value = data.user
    localStorage.setItem(STORAGE_KEYS.accessToken, data.access_token)
    localStorage.setItem(STORAGE_KEYS.refreshToken, data.refresh_token)
    localStorage.setItem(STORAGE_KEYS.user, JSON.stringify(data.user))
  }

  function clearAuth() {
    token.value = ''
    refreshToken.value = ''
    user.value = null
    localStorage.removeItem(STORAGE_KEYS.accessToken)
    localStorage.removeItem(STORAGE_KEYS.refreshToken)
    localStorage.removeItem(STORAGE_KEYS.user)
  }

  function clearOpsAuth() {
    localStorage.removeItem(STORAGE_KEYS.opsAccessToken)
    localStorage.removeItem(STORAGE_KEYS.opsRefreshToken)
    localStorage.removeItem(STORAGE_KEYS.opsUser)
  }

  const isLoggedIn = () => !!token.value

  return { token, refreshToken, user, setAuth, clearAuth, clearOpsAuth, isLoggedIn }
})
