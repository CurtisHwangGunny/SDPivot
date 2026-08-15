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

export function updateSpace(id: string, data: Partial<Pick<Space, 'name' | 'description' | 'visibility'>>) {
  return client.put(`/spaces/${id}`, data)
}

export function deleteSpace(id: string) {
  return client.delete(`/spaces/${id}`)
}

export function listSpaceMembers(id: string) {
  return client.get<{ members: AdminMember[] }>(`/spaces/${id}/members`)
}

export function addSpaceMember(id: string, data: { user_ids: string[]; department_ids: string[] }) {
  return client.post(`/spaces/${id}/members`, data)
}

export interface SpaceMemberCandidate {
  id: string
  name: string
  username: string
  account: string
  email: string
  department_id?: string | null
  department_name: string
  is_member: boolean
}

export interface SpaceMemberCandidateDepartment {
  id: string
  name: string
  parent_id: string
  member_count: number
  existing_member_count: number
  is_fully_added: boolean
}

export function listSpaceMemberCandidates(id: string) {
  return client.get<{
    users: SpaceMemberCandidate[]
    departments: SpaceMemberCandidateDepartment[]
    total: number
  }>(`/spaces/${id}/members/candidates`, { params: { page_size: 500 } })
}

export function removeSpaceMember(id: string, userId: string) {
  return client.delete(`/spaces/${id}/members/${userId}`)
}

export interface AdminStats {
  space_count: number
  document_count: number
  member_count: number
  model_count?: number
}

export interface AdminMember {
  id?: string
  user_id: string
  name?: string
  nickname?: string
  username?: string
  email?: string
  role: string
  access_role?: string
  department_id?: string
  department?: string
  status?: string
  created_at?: string
  last_active_at?: string
}

export function getAdminStats() {
  return client.get<AdminStats>('/admin/stats')
}

export function listMembers() {
  return client.get<{ members: AdminMember[] }>('/admin/members')
}
