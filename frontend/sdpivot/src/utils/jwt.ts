import { STORAGE_KEYS } from './storage'

export function getRoleFromToken(): string {
  const token = localStorage.getItem(STORAGE_KEYS.accessToken)
  if (!token) return ''
  try {
    const encodedPayload = token.split('.')[1]
    if (!encodedPayload) return 'knowledge_viewer'
    const base64 = encodedPayload.replace(/-/g, '+').replace(/_/g, '/').padEnd(Math.ceil(encodedPayload.length / 4) * 4, '=')
    const payload = JSON.parse(atob(base64))
    return payload.access_role || payload.role || 'knowledge_viewer'
  } catch {
    return 'knowledge_viewer'
  }
}
