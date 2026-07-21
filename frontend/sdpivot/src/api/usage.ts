import client from './client'

export function getUsageSummary(params?: { start_date?: string; end_date?: string }) {
  return client.get('/usage/summary', { params })
}

export function getUsageHistory(params?: { page?: number; page_size?: number }) {
  return client.get('/usage/history', { params })
}
