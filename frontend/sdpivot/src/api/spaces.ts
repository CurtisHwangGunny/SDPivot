import client from './client'

export interface Space {
  id: string
  name: string
  description: string
  visibility: string
  owner_id: string
  created_at: string
  updated_at: string
}

export function listSpaces() {
  return client.get<{ spaces: Space[] }>('/spaces')
}

export function createSpace(data: { name: string; description?: string; visibility?: string }) {
  return client.post<Space>('/spaces', data)
}

export function getSpace(id: string) {
  return client.get<Space>(`/spaces/${id}`)
}

export function deleteSpace(id: string) {
  return client.delete(`/spaces/${id}`)
}

export function listSpaceMembers(id: string) {
  return client.get(`/spaces/${id}/members`)
}
