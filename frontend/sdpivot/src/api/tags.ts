import client from './client'

export interface TagDimension {
  id: string
  tenant_id?: number
  code: string
  name: string
  description: string
  enabled: boolean
  sort_order: number
}

export interface TagEntry {
  id: string
  dimension_id: string
  name: string
  color?: string
  sort_order: number
}

export interface TagFeedbackItem {
  id: string
  document_id: string
  document_title: string
  tag_id: string
  tag_name: string
  original_tag: string
  feedback: 'correct' | 'incorrect'
  status: 'pending' | 'reviewed' | 'rejected'
  created_by: string
  created_at: string
}

export function getTagDictionary() {
  return client.get<{ dimensions: TagDimension[]; tags: TagEntry[] }>('/admin/tags')
}

export function getTagDimensions() {
  return client.get<{ dimensions: TagDimension[] }>('/admin/tag-dimensions')
}

export function createTagDimension(data: Partial<TagDimension>) {
  return client.post<TagDimension>('/admin/tag-dimensions', data)
}

export function updateTagDimension(id: string, data: Partial<TagDimension>) {
  return client.put<TagDimension>(`/admin/tag-dimensions/${encodeURIComponent(id)}`, data)
}

export function deleteTagDimension(id: string) {
  return client.delete(`/admin/tag-dimensions/${encodeURIComponent(id)}`)
}

export function createTag(data: Partial<TagEntry>) {
  return client.post<TagEntry>('/admin/tags', data)
}

export function updateTag(id: string, data: Partial<TagEntry>) {
  return client.put<TagEntry>(`/admin/tags/${encodeURIComponent(id)}`, data)
}

export function deleteTag(id: string) {
  return client.delete(`/admin/tags/${encodeURIComponent(id)}`)
}

export function getTagFeedbackQueue() {
  return client.get<{ items: TagFeedbackItem[]; total: number }>('/admin/tag-feedback/queue')
}

export function reviewTagFeedback(ids: string[], decision: 'reviewed' | 'rejected') {
  return client.post('/admin/tag-feedback/review', { ids, decision })
}
