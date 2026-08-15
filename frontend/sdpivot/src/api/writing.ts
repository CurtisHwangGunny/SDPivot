import client from './client'

export interface WritingDraft {
  id: string
  title: string
  category: string
  content: string
  status: string
  space_id: string
  source_type?: string
  web_search_enabled?: boolean
  created_at: string
  updated_at: string
}

export interface WritingCategory {
  id: string
  name: string
  description: string
  sort: number
  created_at: string
  updated_at: string
}

export interface WritingTemplate {
  id: string
  category_id: string
  name: string
  content: string
  is_builtin: boolean
  is_default?: boolean
  source?: WritingTemplateSource
  sort: number
  created_at: string
  updated_at: string
}

export type WritingTemplateSource = 'personal' | 'admin' | 'builtin' | 'system'

export interface PersonalWritingTemplate {
  id: string
  category_id: string
  name: string
  content: string
  sort: number
  created_at?: string
  updated_at?: string
}

export interface WritingTemplatePreferences {
  default_template_id?: string | null
  category_order?: string[]
  templates: PersonalWritingTemplate[]
}

export interface ResolvedWritingTemplate {
  id: string
  category_id: string
  name: string
  content: string
  source: Exclude<WritingTemplateSource, 'system'>
}

export const CATEGORIES = [
  { value: 'notice', label: '通知' },
  { value: 'announcement', label: '公告' },
  { value: 'tech_doc', label: '技术文档' },
  { value: 'meeting_minutes', label: '会议纪要' },
  { value: 'policy_interpretation', label: '制度解读' },
  { value: 'report', label: '报告' },
  { value: 'work_summary', label: '工作总结' },
  { value: 'research_report', label: '研究报告' },
]

export type WritingSourceType = 'knowledge_base' | 'knowledge_plus_web'

export interface GenerateContentResponse {
  content: string
  category: string
  source_type: WritingSourceType
  web_search_enabled: boolean
  sources_count: number
  knowledge_sources_count: number
  web_sources_count: number
  model_id: string
  model: string
}

export function generateContent(data: {
  category: string
  prompt: string
  space_id?: string
  source_type?: WritingSourceType
  web_search_enabled?: boolean
  custom_template?: string
}) {
  return client.post<GenerateContentResponse>('/writing/generate', data)
}

export function createDraft(data: { title: string; category: string; space_id?: string; source_type?: string; web_search_enabled?: boolean }) {
  return client.post<{ draft: WritingDraft }>('/writing/drafts', data)
}

export function listDrafts() {
  return client.get<{ drafts: WritingDraft[] }>('/writing/drafts')
}

export function getDraft(id: string) {
  return client.get<{ draft: WritingDraft }>(`/writing/drafts/${id}`)
}

export function updateDraft(id: string, data: { title?: string; content?: string; status?: string }) {
  return client.put(`/writing/drafts/${id}`, data)
}

export function deleteDraft(id: string) {
  return client.delete(`/writing/drafts/${id}`)
}

export function exportDraft(id: string, format: string) {
  return client.post(`/writing/drafts/${id}/export`, { format }, { responseType: 'blob' })
}

export function listWritingCategories() {
  return client.get<{ categories: WritingCategory[] }>('/writing/categories')
}

export function createWritingCategory(data: Pick<WritingCategory, 'name' | 'description' | 'sort'>) {
  return client.post<{ category: WritingCategory }>('/writing/categories', data)
}

export function updateWritingCategory(id: string, data: Partial<Pick<WritingCategory, 'name' | 'description' | 'sort'>>) {
  return client.put(`/writing/categories/${id}`, data)
}

export function deleteWritingCategory(id: string) {
  return client.delete(`/writing/categories/${id}`)
}

export function listWritingTemplates(categoryId?: string) {
  return client.get<{ templates: WritingTemplate[] }>('/writing/templates', { params: categoryId ? { category_id: categoryId } : undefined })
}

export function createWritingTemplate(data: Pick<WritingTemplate, 'category_id' | 'name' | 'content' | 'is_builtin' | 'sort'>) {
  return client.post<{ template: WritingTemplate }>('/writing/templates', data)
}

export function updateWritingTemplate(id: string, data: Partial<Pick<WritingTemplate, 'category_id' | 'name' | 'content' | 'is_builtin' | 'sort'>>) {
  return client.put(`/writing/templates/${id}`, data)
}

export function deleteWritingTemplate(id: string) {
  return client.delete(`/writing/templates/${id}`)
}

export function getMyWritingTemplates(resolved = false) {
  return client.get<{ preferences: WritingTemplatePreferences; resolved_templates?: ResolvedWritingTemplate[] }>('/my/writing/templates', { params: resolved ? { resolved: 1 } : undefined })
}

export function createMyWritingTemplate(data: { category_id: string; name: string; content: string; sort?: number; set_default?: boolean }) {
  return client.post<{ template: PersonalWritingTemplate; preferences: WritingTemplatePreferences }>('/my/writing/templates', data)
}

export function updateMyWritingTemplate(id: string, data: Partial<{ category_id: string; name: string; content: string; sort: number; set_default: boolean }>) {
  return client.put<{ template: PersonalWritingTemplate; preferences: WritingTemplatePreferences }>(`/my/writing/templates/${id}`, data)
}

export function deleteMyWritingTemplate(id: string) {
  return client.delete(`/my/writing/templates/${id}`)
}

export function listAdminWritingTemplates(categoryId?: string) {
  return client.get<{ templates: WritingTemplate[] }>('/admin/writing/templates', { params: categoryId ? { category_id: categoryId } : undefined })
}

export function createAdminWritingTemplate(data: { category_id: string; name: string; content: string; sort?: number; is_default?: boolean; is_builtin?: boolean }) {
  return client.post<{ template: WritingTemplate }>('/admin/writing/templates', data)
}

export function updateAdminWritingTemplate(id: string, data: Partial<{ category_id: string; name: string; content: string; sort: number; is_default: boolean; is_builtin: boolean }>) {
  return client.put(`/admin/writing/templates/${id}`, data)
}

export function deleteAdminWritingTemplate(id: string) {
  return client.delete(`/admin/writing/templates/${id}`)
}

export function getOpsDashboard() {
  return client.get('/ops/dashboard')
}

export function listAnnouncements() {
  return client.get('/ops/announcements')
}

export function createAnnouncement(data: { title: string; content: string }) {
  return client.post('/ops/announcements', data)
}
