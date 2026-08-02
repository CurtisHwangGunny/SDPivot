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

export interface OrganizationListItem extends Organization {
  member_count?: number
  days_remaining?: number
}

export interface OrganizationPayload {
  name: string
  description?: string
}

export function listOrgs() {
  return client.get<{ organizations: OrganizationListItem[] }>('/organizations')
}

export function createOrg(data: OrganizationPayload) {
  return client.post<Organization>('/organizations', data)
}

export function updateOrg(id: string, data: OrganizationPayload) {
  return client.put<Organization>(`/organizations/${id}`, data)
}

export function deleteOrg(id: string) {
  return client.delete(`/organizations/${id}`)
}

export function listOrganizations() {
  return listOrgs()
}

export function createOrganization(data: { name: string; description?: string }) {
  return createOrg(data)
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
