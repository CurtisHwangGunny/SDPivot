import client from './client'

export interface QASession {
  id: string
  title: string
  space_id: string
  created_at: string
  updated_at: string
}

export interface QAMessage {
  id: string
  session_id: string
  role: string
  content: string
  sources: string
  created_at: string
}

export interface QAAvailableModel {
  id: string
  name: string
  display_name: string
  is_default: boolean
}

export function listQAModels() {
  return client.get<{ models: QAAvailableModel[] }>('/qa/models')
}

export const listModels = listQAModels

export function createSession(data: { title?: string; space_id?: string }) {
  return client.post<{ session: QASession }>('/qa/sessions', data)
}

export function listSessions() {
  return client.get<{ sessions: QASession[] }>('/qa/sessions')
}

export function getMessages(sessionId: string) {
  return client.get<{ messages: QAMessage[] }>(`/qa/sessions/${sessionId}/messages`)
}

export function sendMessage(sessionId: string, content: string, modelId?: string) {
  return client.post<{
    user_message: QAMessage
    assistant_message: QAMessage
    model_id: string
    model: string
  }>(`/qa/sessions/${sessionId}/messages`, { content, model_id: modelId })
}

export function deleteSession(sessionId: string) {
  return client.delete(`/qa/sessions/${sessionId}`)
}

export function getAdminStats() {
  return client.get('/admin/stats')
}

export function getAdminMembers() {
  return client.get('/admin/members')
}

export function getAdminSpaces() {
  return client.get('/admin/spaces')
}
