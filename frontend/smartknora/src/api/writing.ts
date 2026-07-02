import client from './client'

export interface WritingDraft {
  id: string
  title: string
  category: string
  content: string
  status: string
  space_id: string
  created_at: string
  updated_at: string
}

export const CATEGORIES = [
  { value: 'work_summary', label: '工作总结' },
  { value: 'research_report', label: '研究报告' },
  { value: 'project_proposal', label: '项目方案' },
  { value: 'meeting_minutes', label: '会议纪要' },
  { value: 'tech_doc', label: '技术文档' },
  { value: 'business_plan', label: '商业计划书' },
  { value: 'weekly_report', label: '周报日报' },
  { value: 'notice', label: '通知公告' },
]

export function generateContent(data: { category: string; prompt: string; space_id?: string }) {
  return client.post<{ content: string; category: string; sources_count: number }>('/writing/generate', data)
}

export function createDraft(data: { title: string; category: string; space_id?: string }) {
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
  return client.post(`/writing/drafts/${id}/export`, { format })
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
