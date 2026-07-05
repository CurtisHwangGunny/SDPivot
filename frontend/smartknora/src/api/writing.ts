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

export function generateContent(data: { category: string; prompt: string; space_id?: string; source_type?: string; web_search_enabled?: boolean }) {
  return client.post<{ content: string; category: string; sources_count: number }>('/writing/generate', data)
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

export function getOpsDashboard() {
  return client.get('/ops/dashboard')
}

export function listAnnouncements() {
  return client.get('/ops/announcements')
}

export function createAnnouncement(data: { title: string; content: string }) {
  return client.post('/ops/announcements', data)
}
