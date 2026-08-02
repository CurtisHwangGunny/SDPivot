import client from './client'

export interface UsageBreakdownItem {
  name: string
  value: number
}

export interface TokenUsageItem {
  id?: string
  name: string
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
}

export interface UsageStats {
  storage: {
    used_bytes: number
    total_bytes: number
    by_space: UsageBreakdownItem[]
  }
  activity: {
    dau: number
    wau: number
    mau: number
  }
  tokens: {
    total_tokens: number
    by_person: TokenUsageItem[]
    by_department: TokenUsageItem[]
  }
  documents: {
    total: number
    by_type: UsageBreakdownItem[]
    by_tag: UsageBreakdownItem[]
  }
  qa: {
    count: number
    avg_response_time: number
    satisfaction: number
  }
}

export interface UsageSummary {
  total_prompt_tokens: number
  total_completion_tokens: number
  total_tokens: number
  request_count: number
}

export function getUsageSummary(params?: { start_date?: string; end_date?: string }) {
  return client.get<{ summary: UsageSummary }>('/usage/summary', { params })
}

export function getUsageHistory(params?: { page?: number; page_size?: number }) {
  return client.get('/usage/history', { params })
}
