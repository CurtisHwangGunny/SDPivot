import client from './client'

export interface Organization {
  id: string
  name: string
  description: string
  invite_code: string
  owner_id: string
  auth_status: string
  created_at: string
}

export function listOrganizations() {
  return client.get<{ organizations: Organization[] }>('/organizations')
}

export function createOrganization(data: { name: string; description?: string }) {
  return client.post<Organization>('/organizations', data)
}

export function joinOrganization(data: { org_id: string }) {
  return client.post('/organizations/join', data)
}

export function getOrganization(id: string) {
  return client.get<Organization>(`/organizations/${id}`)
}

export function listMembers(id: string) {
  return client.get(`/organizations/${id}/members`)
}
