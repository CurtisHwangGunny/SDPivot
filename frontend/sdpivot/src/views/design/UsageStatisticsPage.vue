<template>
  <SdpSidebarLayout>
    <main class="sdp-usage">
      <div class="sdp-usage__shell">
        <header class="sdp-usage__topbar">
          <div>
            <p>Analytics / Usage</p>
            <h1>用量统计</h1>
            <span>查看存储、活跃度、Token、文档与问答服务的使用情况。</span>
          </div>
          <div class="sdp-usage__actions">
            <SdpButton variant="secondary" :loading="loading" aria-label="刷新用量统计" @click="loadUsageStats">刷新数据</SdpButton>
            <SdpButton :disabled="tokenRows.length === 0" aria-label="导出 Token 消费报表" @click="exportTokenReport">导出报表</SdpButton>
          </div>
        </header>

        <SdpErrorState v-if="loadError" type="server" title="用量数据加载失败" :description="loadError" retryable @retry="loadUsageStats" />

        <template v-else>
          <section class="sdp-usage__kpis" aria-label="用户活跃与问答关键指标">
            <article v-for="item in headlineStats" :key="item.label" class="sdp-usage__kpi">
              <p>{{ item.label }}</p>
              <strong>{{ loading ? '—' : item.value }}</strong>
              <small>{{ item.note }}</small>
            </article>
          </section>

          <section class="sdp-usage__grid sdp-usage__grid--storage" aria-label="存储用量">
            <article class="sdp-usage__panel">
              <header class="sdp-usage__section-head">
                <div><p>Storage</p><h2>存储用量概览</h2></div>
                <span>{{ storagePercent }}%</span>
              </header>
              <div class="sdp-usage__storage-total">
                <strong>{{ formatBytes(stats.storage.used_bytes) }}</strong>
                <span>共 {{ formatBytes(stats.storage.total_bytes) }}</span>
              </div>
              <div class="sdp-usage__progress" role="progressbar" aria-label="存储使用率" aria-valuemin="0" aria-valuemax="100" :aria-valuenow="storagePercent">
                <span :style="{ width: `${storagePercent}%` }" />
              </div>
              <p class="sdp-usage__caption">剩余 {{ formatBytes(Math.max(stats.storage.total_bytes - stats.storage.used_bytes, 0)) }}</p>
            </article>

            <article class="sdp-usage__panel">
              <header class="sdp-usage__section-head">
                <div><p>Distribution</p><h2>空间存储分布</h2></div>
                <span>{{ stats.storage.by_space.length }} 个空间</span>
              </header>
              <ul v-if="stats.storage.by_space.length" class="sdp-usage__bars" aria-label="按空间划分的存储用量图表">
                <li v-for="item in stats.storage.by_space" :key="item.name">
                  <div><strong>{{ item.name }}</strong><span>{{ formatBytes(item.value) }}</span></div>
                  <div class="sdp-usage__bar" aria-hidden="true"><span :style="{ width: `${percentage(item.value, maxSpaceStorage)}%` }" /></div>
                </li>
              </ul>
              <p v-else class="sdp-usage__empty">暂无空间存储数据</p>
            </article>
          </section>

          <section class="sdp-usage__panel" aria-labelledby="token-title">
            <header class="sdp-usage__section-head sdp-usage__section-head--wrap">
              <div><p>Token Consumption</p><h2 id="token-title">Token 消费报告</h2></div>
              <div class="sdp-usage__segmented" aria-label="Token 报告分组方式">
                <button v-for="option in tokenGroupOptions" :key="option.value" type="button" :class="{ active: tokenGroup === option.value }" :aria-pressed="tokenGroup === option.value" @click="tokenGroup = option.value">{{ option.label }}</button>
              </div>
            </header>
            <div class="sdp-usage__token-summary"><span>总消耗</span><strong>{{ formatNumber(stats.tokens.total_tokens) }}</strong><small>tokens</small></div>
            <div class="sdp-usage__table-wrap">
              <table class="sdp-usage__table">
                <caption class="sr-only">{{ tokenGroup === 'person' ? '按人员' : '按部门' }}统计的 Token 消费</caption>
                <thead><tr><th scope="col">{{ tokenGroup === 'person' ? '人员' : '部门' }}</th><th scope="col">输入 Token</th><th scope="col">输出 Token</th><th scope="col">总计</th><th scope="col">占比</th></tr></thead>
                <tbody>
                  <tr v-for="item in tokenRows" :key="item.id || item.name">
                    <th scope="row">{{ item.name || '未命名' }}</th>
                    <td>{{ formatNumber(item.prompt_tokens) }}</td>
                    <td>{{ formatNumber(item.completion_tokens) }}</td>
                    <td><strong>{{ formatNumber(item.total_tokens) }}</strong></td>
                    <td>{{ percentage(item.total_tokens, tokenRowsTotal) }}%</td>
                  </tr>
                  <tr v-if="tokenRows.length === 0"><td colspan="5" class="sdp-usage__empty">暂无 Token 消费数据</td></tr>
                </tbody>
              </table>
            </div>
          </section>

          <section class="sdp-usage__grid" aria-label="文档统计">
            <article class="sdp-usage__panel">
              <header class="sdp-usage__section-head"><div><p>Documents</p><h2>按类型统计</h2></div><span>共 {{ formatNumber(stats.documents.total) }} 份</span></header>
              <ul v-if="stats.documents.by_type.length" class="sdp-usage__donut-layout">
                <li v-for="item in stats.documents.by_type" :key="item.name">
                  <span aria-hidden="true" /><strong>{{ item.name }}</strong><small>{{ formatNumber(item.value) }} · {{ percentage(item.value, stats.documents.total) }}%</small>
                </li>
              </ul>
              <p v-else class="sdp-usage__empty">暂无文档类型数据</p>
            </article>

            <article class="sdp-usage__panel">
              <header class="sdp-usage__section-head"><div><p>Tags</p><h2>按标签统计</h2></div><span>热门标签</span></header>
              <div v-if="stats.documents.by_tag.length" class="sdp-usage__tags">
                <span v-for="item in stats.documents.by_tag" :key="item.name"><strong>{{ item.name }}</strong>{{ formatNumber(item.value) }}</span>
              </div>
              <p v-else class="sdp-usage__empty">暂无文档标签数据</p>
            </article>
          </section>
        </template>
      </div>
    </main>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { getUsageStats, type UsageStats } from '@/api/usage'
import { SdpButton, SdpErrorState } from '@/components/design'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'

type TokenGroup = 'person' | 'department'

const emptyStats = (): UsageStats => ({
  storage: { used_bytes: 0, total_bytes: 0, by_space: [] },
  activity: { dau: 0, wau: 0, mau: 0 },
  tokens: { total_tokens: 0, by_person: [], by_department: [] },
  documents: { total: 0, by_type: [], by_tag: [] },
  qa: { count: 0, avg_response_time: 0, satisfaction: 0 },
})

const stats = ref<UsageStats>(emptyStats())
const loading = ref(true)
const loadError = ref('')
const tokenGroup = ref<TokenGroup>('person')
const tokenGroupOptions: Array<{ value: TokenGroup; label: string }> = [
  { value: 'person', label: '按人员' },
  { value: 'department', label: '按部门' },
]

const storagePercent = computed(() => percentage(stats.value.storage.used_bytes, stats.value.storage.total_bytes))
const maxSpaceStorage = computed(() => Math.max(...stats.value.storage.by_space.map((item) => item.value), 0))
const tokenRows = computed(() => tokenGroup.value === 'person' ? stats.value.tokens.by_person : stats.value.tokens.by_department)
const tokenRowsTotal = computed(() => tokenRows.value.reduce((total, item) => total + item.total_tokens, 0))
const headlineStats = computed(() => [
  { label: '日活跃用户 DAU', value: formatNumber(stats.value.activity.dau), note: '过去 24 小时' },
  { label: '周活跃用户 WAU', value: formatNumber(stats.value.activity.wau), note: '过去 7 天' },
  { label: '月活跃用户 MAU', value: formatNumber(stats.value.activity.mau), note: '过去 30 天' },
  { label: '问答次数', value: formatNumber(stats.value.qa.count), note: `平均响应 ${formatDuration(stats.value.qa.avg_response_time)}` },
  { label: '问答满意度', value: formatPercent(stats.value.qa.satisfaction), note: '用户反馈结果' },
])

async function loadUsageStats() {
  loading.value = true
  loadError.value = ''
  try {
    const response = await getUsageStats()
    stats.value = response.data
  } catch (error: unknown) {
    loadError.value = errorMessage(error, '无法获取用量统计，请稍后重试。')
  } finally {
    loading.value = false
  }
}

function formatNumber(value: number) {
  return new Intl.NumberFormat('zh-CN').format(Number(value) || 0)
}

function formatBytes(value: number) {
  const bytes = Number(value) || 0
  if (bytes < 1024) return `${formatNumber(bytes)} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let amount = bytes / 1024
  let unitIndex = 0
  while (amount >= 1024 && unitIndex < units.length - 1) {
    amount /= 1024
    unitIndex += 1
  }
  return `${amount.toLocaleString('zh-CN', { maximumFractionDigits: 1 })} ${units[unitIndex]}`
}

function formatDuration(value: number) {
  const seconds = Number(value) || 0
  return seconds < 1 ? `${Math.round(seconds * 1000)}ms` : `${seconds.toLocaleString('zh-CN', { maximumFractionDigits: 1 })}s`
}

function formatPercent(value: number) {
  const normalized = value > 0 && value <= 1 ? value * 100 : value
  return `${Math.min(Math.max(normalized, 0), 100).toLocaleString('zh-CN', { maximumFractionDigits: 1 })}%`
}

function percentage(value: number, total: number) {
  if (!total) return 0
  return Math.min(Math.max(Math.round((value / total) * 100), 0), 100)
}

function exportTokenReport() {
  const heading = tokenGroup.value === 'person' ? '人员' : '部门'
  const rows = tokenRows.value.map((item) => [item.name, item.prompt_tokens, item.completion_tokens, item.total_tokens])
  const csv = [[heading, '输入 Token', '输出 Token', '总 Token'], ...rows]
    .map((row) => row.map((cell) => `"${String(cell).replaceAll('"', '""')}"`).join(','))
    .join('\n')
  const url = URL.createObjectURL(new Blob([`\uFEFF${csv}`], { type: 'text/csv;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = url
  link.download = `usage-tokens-${tokenGroup.value}.csv`
  link.click()
  URL.revokeObjectURL(url)
}

function errorMessage(error: unknown, fallback: string) {
  const value = error as { message?: string; response?: { data?: { error?: string; message?: string } } }
  return value?.response?.data?.error || value?.response?.data?.message || value?.message || fallback
}

onMounted(loadUsageStats)
</script>

<style scoped>
.sdp-usage { min-height: 100dvh; color: var(--ink-900); background: var(--ink-100); font-family: var(--font-body); }
.sdp-usage__shell { display: grid; gap: var(--space-6); width: min(100%, var(--content-max-width)); margin-inline: auto; padding: var(--space-8); }
.sdp-usage__topbar, .sdp-usage__section-head, .sdp-usage__actions, .sdp-usage__storage-total, .sdp-usage__token-summary { display: flex; align-items: center; }
.sdp-usage__topbar { justify-content: space-between; gap: var(--space-6); }
.sdp-usage__topbar p, .sdp-usage__section-head p { color: var(--brand-700); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); letter-spacing: .08em; text-transform: uppercase; }
.sdp-usage__topbar h1 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-4xl); line-height: var(--leading-tight); }
.sdp-usage__topbar > div > span { display: block; margin-top: var(--space-2); color: var(--ink-600); font-size: var(--text-base); }
.sdp-usage__actions { gap: var(--space-3); }
.sdp-usage__kpis { display: grid; grid-template-columns: repeat(5, minmax(var(--space-0), 1fr)); gap: var(--space-3); }
.sdp-usage__kpi, .sdp-usage__panel { border: 1px solid var(--ink-200); background: var(--ink-50); box-shadow: var(--shadow-xs); }
.sdp-usage__kpi { padding: var(--space-5); border-radius: var(--radius-md); }
.sdp-usage__kpi p { color: var(--ink-600); font-size: var(--text-xs); }
.sdp-usage__kpi strong { display: block; margin-top: var(--space-2); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-3xl); line-height: var(--leading-tight); }
.sdp-usage__kpi small { display: block; margin-top: var(--space-2); color: var(--ink-500); font-size: var(--text-xs); }
.sdp-usage__grid { display: grid; grid-template-columns: repeat(2, minmax(var(--space-0), 1fr)); gap: var(--space-4); }
.sdp-usage__grid--storage { grid-template-columns: minmax(var(--space-0), .8fr) minmax(var(--space-0), 1.2fr); }
.sdp-usage__panel { min-width: var(--space-0); padding: var(--space-6); border-radius: var(--radius-lg); }
.sdp-usage__section-head { justify-content: space-between; gap: var(--space-4); margin-bottom: var(--space-5); }
.sdp-usage__section-head h2 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-xl); }
.sdp-usage__section-head > span { color: var(--ink-600); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-usage__storage-total { justify-content: space-between; gap: var(--space-4); margin-bottom: var(--space-4); }
.sdp-usage__storage-total strong { color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-4xl); }
.sdp-usage__storage-total span, .sdp-usage__caption { color: var(--ink-600); font-size: var(--text-sm); }
.sdp-usage__progress, .sdp-usage__bar { overflow: hidden; border-radius: var(--radius-pill); background: var(--ink-200); }
.sdp-usage__progress { height: var(--space-3); }
.sdp-usage__progress span, .sdp-usage__bar span { display: block; height: 100%; border-radius: inherit; background: linear-gradient(90deg, var(--brand-500), var(--brand-700)); }
.sdp-usage__caption { margin-top: var(--space-3); }
.sdp-usage__bars { display: grid; gap: var(--space-4); list-style: none; }
.sdp-usage__bars li > div:first-child { display: flex; justify-content: space-between; gap: var(--space-4); margin-bottom: var(--space-2); }
.sdp-usage__bars strong { overflow: hidden; color: var(--ink-900); font-size: var(--text-sm); text-overflow: ellipsis; white-space: nowrap; }
.sdp-usage__bars span { color: var(--ink-600); font-size: var(--text-xs); }
.sdp-usage__bar { height: var(--space-2); }
.sdp-usage__section-head--wrap { flex-wrap: wrap; }
.sdp-usage__segmented { display: inline-flex; padding: var(--space-1); border: 1px solid var(--ink-200); border-radius: var(--radius-sm); background: var(--ink-100); }
.sdp-usage__segmented button { padding: var(--space-2) var(--space-3); border: 0; border-radius: var(--radius-xs); color: var(--ink-700); background: transparent; font-family: var(--font-body); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); cursor: pointer; }
.sdp-usage__segmented button.active { color: var(--ink-950); background: var(--brand-400); }
.sdp-usage__segmented button:focus-visible { outline: 2px solid var(--brand-500); outline-offset: 2px; }
.sdp-usage__token-summary { gap: var(--space-2); margin-bottom: var(--space-5); color: var(--ink-600); }
.sdp-usage__token-summary strong { color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-3xl); }
.sdp-usage__token-summary small { font-family: var(--font-mono); font-size: var(--text-xs); }
.sdp-usage__table-wrap { overflow-x: auto; }
.sdp-usage__table { width: 100%; border-collapse: collapse; }
.sdp-usage__table th, .sdp-usage__table td { padding: var(--space-3) var(--space-4); border-top: 1px solid var(--ink-200); color: var(--ink-700); font-size: var(--text-sm); text-align: right; white-space: nowrap; }
.sdp-usage__table thead th { border-top: 0; color: var(--ink-600); background: var(--ink-100); font-size: var(--text-xs); }
.sdp-usage__table th:first-child, .sdp-usage__table td:first-child { text-align: left; }
.sdp-usage__table tbody th, .sdp-usage__table td strong { color: var(--ink-900); font-weight: var(--font-weight-semibold); }
.sdp-usage__donut-layout { display: grid; grid-template-columns: repeat(2, minmax(var(--space-0), 1fr)); gap: var(--space-3); list-style: none; }
.sdp-usage__donut-layout li { display: grid; grid-template-columns: var(--space-3) 1fr; column-gap: var(--space-3); padding: var(--space-3); border-radius: var(--radius-sm); background: var(--ink-100); }
.sdp-usage__donut-layout li > span { width: var(--space-3); height: var(--space-3); margin-top: var(--space-1); border-radius: var(--radius-pill); background: var(--brand-600); }
.sdp-usage__donut-layout strong { color: var(--ink-900); font-size: var(--text-sm); }
.sdp-usage__donut-layout small { grid-column: 2; color: var(--ink-600); font-size: var(--text-xs); }
.sdp-usage__tags { display: flex; flex-wrap: wrap; gap: var(--space-3); }
.sdp-usage__tags span { display: inline-flex; align-items: center; gap: var(--space-2); padding: var(--space-2) var(--space-3); border: 1px solid var(--brand-200); border-radius: var(--radius-pill); color: var(--ink-700); background: var(--brand-50); font-size: var(--text-xs); }
.sdp-usage__tags strong { color: var(--brand-800); }
.sdp-usage__empty { padding: var(--space-6); color: var(--ink-600); font-size: var(--text-sm); text-align: center; }
@media (max-width: 72rem) { .sdp-usage__kpis { grid-template-columns: repeat(3, 1fr); } }
@media (max-width: 56rem) { .sdp-usage__shell { padding: var(--space-6); } .sdp-usage__topbar { align-items: flex-start; flex-direction: column; } .sdp-usage__grid, .sdp-usage__grid--storage { grid-template-columns: 1fr; } }
@media (max-width: 40rem) { .sdp-usage__shell { padding: var(--space-4); } .sdp-usage__topbar h1 { font-size: var(--text-3xl); } .sdp-usage__actions, .sdp-usage__actions :deep(.sdp-button) { width: 100%; } .sdp-usage__actions { flex-direction: column; } .sdp-usage__kpis { grid-template-columns: 1fr; } .sdp-usage__panel { padding: var(--space-5); } .sdp-usage__storage-total { align-items: flex-start; flex-direction: column; } .sdp-usage__donut-layout { grid-template-columns: 1fr; } }
</style>
