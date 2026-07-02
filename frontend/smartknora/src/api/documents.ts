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

export function listDocuments(params: { space_id?: string; parse_status?: string; search?: string; page?: number; page_size?: number }) {
  return client.get<{ documents: Document[]; total: number; page: number; page_size: number }>('/documents', { params })
}

export function getDocument(id: string) {
  return client.get<{ document: Document }>(`/documents/${id}`)
}

export function deleteDocument(id: string) {
  return client.delete(`/documents/${id}`)
}

export function reparseDocument(id: string) {
  return client.post(`/documents/${id}/reparse`)
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
