import { defineStore } from 'pinia'
import { ref } from 'vue'
import { STORAGE_KEYS } from '../utils/storage'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem(STORAGE_KEYS.accessToken) || '')
  const refreshToken = ref(localStorage.getItem(STORAGE_KEYS.refreshToken) || '')
  const user = ref<any>(null)

  function setAuth(data: { access_token: string; refresh_token: string; user: any }) {
    token.value = data.access_token
    refreshToken.value = data.refresh_token
    user.value = data.user
    localStorage.setItem(STORAGE_KEYS.accessToken, data.access_token)
    localStorage.setItem(STORAGE_KEYS.refreshToken, data.refresh_token)
  }

  function clearAuth() {
    token.value = ''
    refreshToken.value = ''
    user.value = null
    localStorage.removeItem(STORAGE_KEYS.accessToken)
    localStorage.removeItem(STORAGE_KEYS.refreshToken)
  }

  function clearOpsAuth() {
    localStorage.removeItem(STORAGE_KEYS.opsAccessToken)
    localStorage.removeItem(STORAGE_KEYS.opsRefreshToken)
    localStorage.removeItem(STORAGE_KEYS.opsUser)
  }

  const isLoggedIn = () => !!token.value

  return { token, refreshToken, user, setAuth, clearAuth, clearOpsAuth, isLoggedIn }
})
