<template>
  <div class="audit-page">
    <div class="audit-toolbar">
      <t-select v-model="filters.action" clearable :placeholder="copy.action" :options="actions" />
      <t-select v-model="filters.outcome" clearable :placeholder="copy.outcome" :options="outcomes" />
      <t-input v-model="filters.user" clearable :placeholder="copy.user" />
      <label class="date-filter"><span>{{ copy.from }}</span><input v-model="filters.start" type="datetime-local" /></label>
      <label class="date-filter"><span>{{ copy.to }}</span><input v-model="filters.end" type="datetime-local" /></label>
      <t-button theme="primary" :loading="loading" @click="load(true)">{{ copy.search }}</t-button>
      <t-button variant="text" @click="clearFilters">{{ copy.clear }}</t-button>
    </div>
    <t-loading :loading="loading && entries.length === 0">
      <div class="audit-table">
        <t-table :data="entries" :columns="columns" row-key="id" hover>
          <template #created_at="{ row }"><span class="time-cell">{{ formatDate(row.created_at) }}</span></template>
          <template #action="{ row }"><div class="action-cell"><strong>{{ row.action }}</strong><span>{{ row.request_method }} {{ row.request_path }}</span></div></template>
          <template #actor_user_id="{ row }"><div class="actor-cell"><strong>{{ row.actor_user_id || copy.system }}</strong><span>{{ row.actor_role || '—' }}</span></div></template>
          <template #target_id="{ row }"><div class="target-cell"><span>{{ row.target_type || '—' }}</span><strong>{{ row.target_id || row.target_user_id || '—' }}</strong></div></template>
          <template #outcome="{ row }"><t-tag :theme="row.outcome === 'success' ? 'success' : 'danger'" variant="light">{{ row.outcome }}</t-tag></template>
          <template #details="{ row }"><t-button variant="text" size="small" @click="selected = row">{{ copy.view }}</t-button></template>
        </t-table>
        <div v-if="!entries.length && !loading" class="empty"><t-empty :description="copy.empty" /></div>
        <div class="load-more"><t-button v-if="hasMore" variant="outline" :loading="loading" @click="load(false)">{{ copy.loadMore }}</t-button><span v-else-if="entries.length">{{ copy.end }}</span></div>
      </div>
    </t-loading>

    <t-dialog v-model:visible="detailsVisible" :header="copy.details" :footer="false" width="680px">
      <dl v-if="selected" class="detail-list">
        <div><dt>{{ copy.action }}</dt><dd>{{ selected.action }}</dd></div><div><dt>{{ copy.actor }}</dt><dd>{{ selected.actor_user_id || copy.system }}</dd></div>
        <div><dt>{{ copy.request }}</dt><dd>{{ selected.request_method }} {{ selected.request_path }}</dd></div><div><dt>{{ copy.target }}</dt><dd>{{ selected.target_type }} {{ selected.target_id || selected.target_user_id }}</dd></div>
        <div><dt>{{ copy.time }}</dt><dd>{{ formatDate(selected.created_at) }}</dd></div><div><dt>{{ copy.outcome }}</dt><dd>{{ selected.outcome }}</dd></div>
      </dl>
      <pre class="detail-json">{{ formatDetails(selected?.details) }}</pre>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { listSystemAuditLog, type AuditLog } from '@/api/system'

const { locale } = useI18n()
const loading = ref(false)
const entries = ref<AuditLog[]>([])
const cursor = ref(0)
const hasMore = ref(true)
const selected = ref<AuditLog | null>(null)
const detailsVisible = computed({ get: () => selected.value !== null, set: value => { if (!value) selected.value = null } })
const filters = reactive({ action: '', outcome: '' as '' | 'success' | 'denied', user: '', start: '', end: '' })
const copy = computed(() => locale.value.startsWith('zh') ? {
  action: '操作类型', outcome: '结果', user: '操作者用户 ID', from: '开始', to: '结束', search: '查询', clear: '清除', system: '系统', view: '查看', empty: '没有匹配的审计记录', loadMore: '加载更早记录', end: '已加载全部记录', details: '审计详情', actor: '操作者', request: '请求', target: '目标', time: '时间', loadFailed: '审计日志加载失败',
} : {
  action: 'Action', outcome: 'Outcome', user: 'Actor user ID', from: 'From', to: 'To', search: 'Search', clear: 'Clear', system: 'System', view: 'View', empty: 'No matching audit records', loadMore: 'Load older entries', end: 'All entries loaded', details: 'Audit details', actor: 'Actor', request: 'Request', target: 'Target', time: 'Time', loadFailed: 'Failed to load audit log',
})
const actions = ['system.setting_changed', 'system.admin_promoted', 'system.admin_revoked', 'system.admin_operation', 'rbac.access_denied'].map(value => ({ label: value, value }))
const outcomes = ['success', 'denied'].map(value => ({ label: value, value }))
const columns = computed(() => [
  { colKey: 'created_at', title: copy.value.time, width: 170 }, { colKey: 'action', title: copy.value.action, minWidth: 240 },
  { colKey: 'actor_user_id', title: copy.value.actor, minWidth: 180 }, { colKey: 'target_id', title: copy.value.target, minWidth: 160 },
  { colKey: 'outcome', title: copy.value.outcome, width: 100 }, { colKey: 'details', title: copy.value.details, width: 80 },
])

async function load(reset: boolean) {
  if (loading.value) return
  if (reset) { entries.value = []; cursor.value = 0; hasMore.value = true }
  loading.value = true
  try {
    const result = await listSystemAuditLog({
      after_id: cursor.value || undefined, limit: 50, action: filters.action || undefined,
      outcome: filters.outcome || undefined, user: filters.user.trim() || undefined,
      start_time: filters.start ? new Date(filters.start).toISOString() : undefined,
      end_time: filters.end ? new Date(filters.end).toISOString() : undefined,
    })
    const rows = result.data || []
    entries.value.push(...rows)
    cursor.value = result.next_cursor || 0
    hasMore.value = rows.length === 50 && cursor.value > 0
  } catch (err: any) { MessagePlugin.error(err?.message || copy.value.loadFailed) } finally { loading.value = false }
}
function clearFilters() { Object.assign(filters, { action: '', outcome: '', user: '', start: '', end: '' }); void load(true) }
function formatDate(value: string) { return new Date(value).toLocaleString(locale.value) }
function formatDetails(value?: AuditLog['details']) { if (!value) return '—'; if (typeof value === 'string') return value; return JSON.stringify(value, null, 2) }
onMounted(() => load(true))
</script>

<style scoped>
.audit-page { display: grid; gap: 14px; }.audit-toolbar { display: flex; align-items: flex-end; flex-wrap: wrap; gap: 10px; padding: 16px; border-radius: 12px; background: var(--td-bg-color-secondarycontainer); }
.audit-toolbar :deep(.t-select), .audit-toolbar :deep(.t-input) { width: 190px; }.date-filter { display: grid; gap: 4px; color: var(--td-text-color-secondary); font-size: 12px; }
.date-filter input { height: 32px; padding: 0 10px; border: 1px solid var(--td-component-border); border-radius: 4px; outline: none; color: var(--td-text-color-primary); background: var(--td-bg-color-container); }
.date-filter input:focus { border-color: var(--td-brand-color); }.audit-table { overflow: hidden; border: 1px solid var(--td-component-border); border-radius: 12px; background: var(--td-bg-color-container); }
.time-cell { white-space: nowrap; }.action-cell, .actor-cell, .target-cell { display: grid; gap: 3px; min-width: 0; }.action-cell strong, .actor-cell strong, .target-cell strong { overflow: hidden; color: var(--td-text-color-primary); text-overflow: ellipsis; white-space: nowrap; }.action-cell span, .actor-cell span, .target-cell span { overflow: hidden; color: var(--td-text-color-secondary); font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
.empty { padding: 28px; }.load-more { display: flex; justify-content: center; padding: 15px; color: var(--td-text-color-placeholder); }.detail-list { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin: 0 0 16px; }.detail-list div { min-width: 0; }.detail-list dt { color: var(--td-text-color-secondary); font-size: 12px; }.detail-list dd { margin: 4px 0 0; overflow-wrap: anywhere; color: var(--td-text-color-primary); }.detail-json { max-height: 320px; margin: 0; padding: 14px; overflow: auto; border-radius: 8px; color: var(--td-text-color-primary); background: var(--td-bg-color-secondarycontainer); font: 12px/1.6 ui-monospace, SFMono-Regular, Menlo, monospace; white-space: pre-wrap; overflow-wrap: anywhere; }
@media (max-width: 640px) { .audit-toolbar > *, .audit-toolbar :deep(.t-select), .audit-toolbar :deep(.t-input) { width: 100%; }.detail-list { grid-template-columns: 1fr; } }
</style>
