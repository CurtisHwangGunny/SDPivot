import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('access_token') || '')
  const refreshToken = ref(localStorage.getItem('refresh_token') || '')
  const user = ref<any>(null)

  function setAuth(data: { access_token: string; refresh_token: string; user: any }) {
    token.value = data.access_token
    refreshToken.value = data.refresh_token
    user.value = data.user
    localStorage.setItem('access_token', data.access_token)
    localStorage.setItem('refresh_token', data.refresh_token)
  }

  function clearAuth() {
    token.value = ''
    refreshToken.value = ''
    user.value = null
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
  }

  function clearOpsAuth() {
    localStorage.removeItem('ops_access_token')
    localStorage.removeItem('ops_refresh_token')
    localStorage.removeItem('ops_user')
  }

  const isLoggedIn = () => !!token.value

  return { token, refreshToken, user, setAuth, clearAuth, clearOpsAuth, isLoggedIn }
})
