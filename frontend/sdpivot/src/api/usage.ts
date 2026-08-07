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

export interface UsageReportItem {
  id: string
  name: string
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  request_count: number
}

export function getUsageSummary(params?: { start_date?: string; end_date?: string }) {
  return client.get<{ summary: UsageSummary }>('/usage/summary', { params })
}

export function getUsageHistory(params?: { page?: number; page_size?: number }) {
  return client.get('/usage/history', { params })
}

export function getUsageReportByUser(params?: { start_date?: string; end_date?: string }) {
  return client.get<{ items: UsageReportItem[] }>('/usage/report/by-user', { params })
}

export function getUsageReportByDepartment(params?: { start_date?: string; end_date?: string }) {
  return client.get<{ items: UsageReportItem[] }>('/usage/report/by-department', { params })
}

export async function downloadUsageReport(groupBy: 'user' | 'department', params?: { start_date?: string; end_date?: string }) {
  const response = await client.get('/usage/report/export', {
    params: { ...params, format: 'csv', group_by: groupBy },
    responseType: 'blob',
  })
  const url = URL.createObjectURL(response.data)
  const link = document.createElement('a')
  link.href = url
  link.download = `usage-report-${groupBy}.csv`
  link.click()
  URL.revokeObjectURL(url)
}
