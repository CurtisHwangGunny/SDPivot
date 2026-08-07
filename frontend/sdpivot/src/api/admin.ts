import client from './client'

export type SettingsMap = Record<string, Record<string, string>>

export interface AdminApiToken {
  id: string
  name: string
  prefix: string
  scopes: string[]
  expires_at?: string | null
  last_used_at?: string | null
  revoked_at?: string | null
  created_at: string
}

export interface AuditLogItem {
  id: number
  user_id: string
  username: string
  actor_role: string
  module: string
  action: string
  resource_type: string
  resource_id: string
  request_path: string
  request_method: string
  outcome: string
  details: Record<string, unknown>
  ip_address: string
  created_at: string
}

export interface AuditLogResponse {
  logs: AuditLogItem[]
  total: number
  page: number
  page_size: number
}

export interface DashboardTrendPoint {
  date: string
  value: number
}

export interface AdminDashboardData {
  qa_trend: DashboardTrendPoint[]
  document_trend: DashboardTrendPoint[]
  token_trend: DashboardTrendPoint[]
}

export interface AdminRole {
  code: 'super_admin' | 'department_admin' | 'knowledge_editor' | 'knowledge_viewer'
  name: string
}

export interface AdminUserImportResult {
  total: number
  imported: number
  failed: number
  errors: Array<{ row: number; field?: string; message: string }>
}

export interface DepartmentNode {
  id: string
  name: string
  parent_id?: string
  children?: DepartmentNode[]
}

export interface CreatedAdminUser {
  user: { id: string; username: string; email: string }
  initial_password: string
}

export interface AdminModel {
  id: string
  name: string
  display_name: string
  type: string
  source: string
  description: string
  parameters: { base_url?: string; provider?: string; interface_type?: string }
  credentials?: { api_key?: { configured: boolean } }
  is_default: boolean
  is_builtin: boolean
  status: string
}

export interface ThirdPartyIntegration {
  id: string
  name: string
  type: 'users' | 'departments' | 'documents'
  base_url: string
  auth_type: 'none' | 'bearer' | 'basic'
  auth_configured: boolean
  enabled: boolean
  sync_rule: Record<string, unknown>
  last_sync_at?: string | null
  created_at: string
}

export function getSystemSettings() {
  return client.get<{ settings: SettingsMap }>('/admin/config')
}

export function saveSystemSettings(section: string, values: Record<string, string>) {
  return client.put(`/admin/config/${section}`, values)
}

export function getSecuritySettings() {
  return client.get<{ settings: SettingsMap }>('/admin/security')
}

export function saveSecuritySettings(section: string, values: Record<string, string>) {
  return client.put(`/admin/security/${section}`, values)
}

export function listAdminApiTokens() {
  return client.get<{ tokens: AdminApiToken[] }>('/admin/api-tokens')
}

export function createAdminApiToken(payload: { name: string; scopes: string[]; expires_at?: string | null }) {
  return client.post<{ token: string; data: AdminApiToken }>('/admin/api-tokens', payload)
}

export function updateAdminApiToken(id: string, payload: { name: string; scopes: string[] }) {
  return client.put(`/admin/api-tokens/${id}`, payload)
}

export function revokeAdminApiToken(id: string) {
  return client.delete(`/admin/api-tokens/${id}`)
}

export function regenerateAdminApiToken(id: string) {
  return client.post<{ token: string; data: AdminApiToken }>(`/admin/api-tokens/${id}/regenerate`)
}

export function getAuditLogs(params?: { page?: number; page_size?: number; module?: string; action?: string }) {
  return client.get<AuditLogResponse>('/admin/audit', { params })
}

export function getLoginAuditLogs(params?: { page?: number; page_size?: number; action?: string }) {
  return client.get<AuditLogResponse>('/admin/audit/login', { params })
}

export function getKnowledgeAuditLogs(params?: { page?: number; page_size?: number; action?: string }) {
  return client.get<AuditLogResponse>('/admin/audit/knowledge', { params })
}

export function exportAdminAuditLogs(params?: { module?: string; action?: string }) {
  return client.get('/admin/audit/export', { params, responseType: 'blob' })
}

export function getAdminDashboard() {
  return client.get<AdminDashboardData>('/admin/dashboard')
}

export function listAdminRoles() {
  return client.get<{ roles: AdminRole[] }>('/admin/roles')
}

export function batchImportAdminUsers(payload: unknown[] | { csv_base64: string }) {
  return client.post<AdminUserImportResult>('/admin/users/batch', payload)
}

export function createAdminUser(payload: { name: string; email?: string; phone?: string; department_id: string; access_role: 'knowledge_editor' | 'knowledge_viewer' }) {
  return client.post<CreatedAdminUser>('/admin/users', payload)
}

export function listAdminDepartments() {
  return client.get<{ success: boolean; data: DepartmentNode[] }>('/admin/departments')
}

export function updateAdminUserRole(id: string, role: AdminRole['code'], departmentId?: string) {
  return client.put(`/admin/users/${id}/role`, { role, department_id: departmentId || undefined })
}

export function listAdminModels() {
  return client.get<{ models: AdminModel[] }>('/admin/models')
}

export function createAdminModel(payload: Record<string, unknown>) {
  return client.post<{ model: AdminModel }>('/admin/models', payload)
}

export function updateAdminModel(id: string, payload: Record<string, unknown>) {
  return client.put(`/admin/models/${id}`, payload)
}

export function testAdminModel(id: string) {
  return client.post<{ message: string; response: string }>(`/admin/models/${id}/test`)
}

export function setDefaultAdminModel(id: string) {
  return client.post(`/admin/models/${id}/set-default`)
}

export function listThirdPartyIntegrations() {
  return client.get<{ integrations: ThirdPartyIntegration[] }>('/sdpivot/integrations')
}

export function createThirdPartyIntegration(payload: Record<string, unknown>) {
  return client.post('/sdpivot/integrations', payload)
}

export function updateThirdPartyIntegration(id: string, payload: Record<string, unknown>) {
  return client.put(`/sdpivot/integrations/${id}`, payload)
}

export function deleteThirdPartyIntegration(id: string) {
  return client.delete(`/sdpivot/integrations/${id}`)
}

export function syncThirdPartyIntegration(id: string, resource: ThirdPartyIntegration['type']) {
  return client.post(`/sdpivot/integrations/${id}/sync/${resource}`)
}
