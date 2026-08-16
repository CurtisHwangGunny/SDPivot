import client from './client'

export interface Document {
  id: string
  space_id: string
  title: string
  file_name: string
  file_type: string
  file_size: number
  parse_status: string
  chunk_count: number
  embedding_status: string
  version: number
  tags: string
  created_at: string
  updated_at: string
}

export interface DocumentChunk {
  id: string
  document_id: string
  chunk_index: number
  content: string
  token_count: number
}

export interface DocumentParseStatus {
  progress: number
  status: 'pending' | 'parsing' | 'completed' | 'failed' | string
  error?: string
}

export function listDocuments(params: { space_id?: string; parse_status?: string; search?: string; page?: number; page_size?: number }) {
  return client.get<{ documents: Document[]; total: number; page: number; page_size: number }>('/documents', { params })
}

export function uploadDocument(data: { space_id: string; file: File; tags?: string }, onProgress?: (percent: number) => void, signal?: AbortSignal) {
  const form = new FormData()
  form.append('space_id', data.space_id)
  form.append('file', data.file)
  if (data.tags) form.append('tags', data.tags)
  return client.post<{ document: Document; message: string }>('/documents/upload', form, {
    signal,
    timeout: 5 * 60 * 1000,
    onUploadProgress: event => {
      if (event.total) onProgress?.(Math.round((event.loaded * 100) / event.total))
    },
  })
}

export function getDocument(id: string) {
  return client.get<{ document: Document }>(`/documents/${id}`)
}

export function getDocumentParseStatus(id: string) {
  return client.get<DocumentParseStatus>(`/documents/${id}/parse-status`)
}

export function deleteDocument(id: string) {
  return client.delete(`/documents/${id}`)
}

export function reparseDocument(id: string) {
  return client.post(`/documents/${id}/reparse`)
}

export function autoTagDocument(id: string) {
  return client.post<{ document_id: string; created: number }>(`/documents/${id}/auto-tag`)
}

export function getDocumentChunks(id: string) {
  return client.get<{ chunks: DocumentChunk[]; total: number }>(`/documents/${id}/chunks`)
}

export function getDocumentVersions(id: string) {
  return client.get<{ versions: any[] }>(`/documents/${id}/versions`)
}

export function uploadManual(data: { space_id: string; title: string; content: string; tags?: string }) {
  return client.post('/documents/manual', data)
}

export function uploadFromURL(data: { space_id: string; url: string; tags?: string }) {
  return client.post('/documents/url', data)
}

export function searchDocuments(data: { query: string; space_id?: string; top_k?: number }) {
  return client.post<{ results: any[]; total: number }>('/search', data)
}
