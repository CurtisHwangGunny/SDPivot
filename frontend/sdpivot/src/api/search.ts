import client from './client'

export interface GlobalSearchResult {
  chunk_id: string
  chunk_index: number
  content: string
  document_id: string
  document_title: string
  space_id: string
  space_name: string
}

export interface SearchHistoryEntry {
  query: string
  searched_at: string
}

export function globalSearch(q: string, limit = 50) {
  return client.get<{ query: string; results: GlobalSearchResult[]; total: number }>('/search', { params: { q, limit } })
}

export function getSearchHistory() {
  return client.get<{ history: SearchHistoryEntry[] }>('/search/history')
}

export function saveSearchHistory(query: string) {
  return client.post('/search/history', { query })
}
