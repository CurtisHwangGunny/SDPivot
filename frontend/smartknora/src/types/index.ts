// smartKnora TypeScript 类型定义

export interface User {
  id: string
  username: string
  email: string
  avatar?: string
  tenant_id: number
  is_active: boolean
  created_at: string
}

export interface UserProfile {
  id: string
  user_id: string
  phone?: string
  nickname: string
  status: string
}

export interface AuthResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  user: User
}

export interface Organization {
  id: string
  name: string
  description: string
  avatar: string
  owner_id: string
  invite_code: string
  created_at: string
}

export interface OrgExt {
  auth_status: 'trial' | 'pending' | 'certified' | 'expired'
  auth_type: string
  auth_expires_at?: string
  days_remaining: number
}

export interface OrgMember {
  id: string
  org_id: string
  user_id: string
  role: 'owner' | 'admin' | 'editor' | 'viewer'
  status: string
  joined_at: string
}

export interface KnowledgeSpace {
  id: string
  tenant_id: number
  org_id?: string
  name: string
  description: string
  visibility: 'private' | 'team' | 'org'
  icon: string
  creator_id?: string
  created_at: string
  updated_at: string
}

export interface SpaceMember {
  id: string
  space_id: string
  user_id: string
  role: 'owner' | 'editor' | 'viewer'
  created_at: string
}

export interface SpaceCategory {
  id: string
  tenant_id: number
  name: string
  color: string
  created_at: string
}

export interface TokenUsageSummary {
  total_prompt_tokens: number
  total_completion_tokens: number
  total_tokens: number
  request_count: number
}

export interface ApiResponse<T = unknown> {
  data?: T
  error?: string
  message?: string
}
