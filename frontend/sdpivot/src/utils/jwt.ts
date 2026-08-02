import { STORAGE_KEYS } from './storage'

export function getRoleFromToken(): string {
  const token = localStorage.getItem(STORAGE_KEYS.accessToken)
  if (!token) return ''
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return payload.role || 'knowledge_viewer'
  } catch {
    return 'knowledge_viewer'
  }
}
