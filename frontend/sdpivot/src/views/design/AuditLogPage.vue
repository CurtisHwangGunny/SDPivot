<template>
  <SdpSidebarLayout mode="admin">
    <main class="audit-page">
      <div class="audit-page__shell">
        <header class="audit-page__header">
          <div><p>Audit Trail</p><h1>审计日志</h1><span>追踪 SDPivot·文枢 登录、管理操作与知识访问记录。</span></div>
          <div class="audit-page__actions"><SdpButton variant="secondary" :loading="loading" @click="load">刷新</SdpButton><SdpButton :disabled="loading" @click="exportLogs">导出 CSV</SdpButton></div>
        </header>

        <SdpNotice v-if="error" type="danger" :title="error" />

        <section class="audit-page__filters" aria-label="审计日志筛选">
          <div class="audit-page__tabs">
            <button v-for="tab in tabs" :key="tab.value" type="button" :class="{ active: category === tab.value }" @click="selectCategory(tab.value)">{{ tab.label }}</button>
          </div>
          <label>模块<select v-model="module" :disabled="category !== 'all'" @change="applyFilters"><option value="">全部模块</option><option value="user">用户管理</option><option value="knowledge">知识管理</option><option value="login">登录</option></select></label>
          <label>操作<input v-model.trim="action" type="search" placeholder="如 document.uploaded" @keyup.enter="applyFilters" /></label>
          <SdpButton variant="secondary" @click="applyFilters">应用筛选</SdpButton>
        </section>

        <section class="audit-page__panel">
          <div class="audit-page__panel-head"><div><p>Records</p><h2>{{ categoryLabel }}</h2></div><span>共 {{ total }} 条</span></div>
          <div class="audit-page__table-wrap">
            <table>
              <thead><tr><th>时间</th><th>用户</th><th>模块 / 操作</th><th>资源</th><th>结果</th><th>IP 地址</th></tr></thead>
              <tbody>
                <tr v-for="row in rows" :key="row.id">
                  <td><time>{{ formatTime(row.created_at) }}</time></td>
                  <td><strong>{{ row.username || row.user_id || '系统' }}</strong><small>{{ roleLabel(row.actor_role) }}</small></td>
                  <td><strong>{{ moduleLabel(row.module) }}</strong><small>{{ row.action }}</small></td>
                  <td><strong>{{ resourceLabel(row.resource_type) }}</strong><small>{{ row.resource_id || '—' }}</small></td>
                  <td><span class="audit-page__outcome" :class="`audit-page__outcome--${row.outcome}`">{{ row.outcome === 'success' ? '成功' : '拒绝' }}</span></td>
                  <td><code>{{ row.ip_address || '—' }}</code></td>
                </tr>
                <tr v-if="!loading && rows.length === 0"><td colspan="6" class="audit-page__empty">当前筛选条件下暂无审计记录</td></tr>
              </tbody>
            </table>
          </div>
          <footer class="audit-page__pagination"><span>第 {{ page }} / {{ totalPages }} 页</span><div><button type="button" :disabled="page <= 1 || loading" @click="changePage(page - 1)">上一页</button><button type="button" :disabled="page >= totalPages || loading" @click="changePage(page + 1)">下一页</button></div></footer>
        </section>
      </div>
    </main>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { exportAdminAuditLogs, getAuditLogs, getKnowledgeAuditLogs, getLoginAuditLogs, type AuditLogItem } from '@/api/admin'
import { SdpButton, SdpNotice } from '@/components/design'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'

type AuditCategory = 'all' | 'login' | 'knowledge'
const tabs: Array<{ label: string; value: AuditCategory }> = [{ label: '管理员操作', value: 'all' }, { label: '登录日志', value: 'login' }, { label: '知识访问', value: 'knowledge' }]
const category = ref<AuditCategory>('all'), module = ref(''), action = ref(''), page = ref(1), pageSize = 20
const rows = ref<AuditLogItem[]>([]), total = ref(0), loading = ref(true), error = ref('')
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const categoryLabel = computed(() => tabs.find((tab) => tab.value === category.value)?.label || '审计记录')

async function load() {
  loading.value = true; error.value = ''
  const params = { page: page.value, page_size: pageSize, action: action.value || undefined, module: category.value === 'all' ? module.value || undefined : undefined }
  try {
    const response = category.value === 'login' ? await getLoginAuditLogs(params) : category.value === 'knowledge' ? await getKnowledgeAuditLogs(params) : await getAuditLogs(params)
    rows.value = response.data.logs || []; total.value = response.data.total || 0
  } catch (e) { error.value = errorMessage(e, '审计日志加载失败，请检查管理员权限后重试。') } finally { loading.value = false }
}
function selectCategory(value: AuditCategory) { category.value = value; module.value = ''; page.value = 1; void load() }
function applyFilters() { page.value = 1; void load() }
function changePage(value: number) { page.value = value; void load() }
async function exportLogs() {
  error.value = ''
  try {
    const response = await exportAdminAuditLogs({ module: category.value === 'all' ? module.value || undefined : category.value, action: action.value || undefined })
    const url = URL.createObjectURL(response.data); const link = document.createElement('a'); link.href = url; link.download = `sdpivot-audit-${new Date().toISOString().slice(0, 10)}.csv`; link.click(); URL.revokeObjectURL(url)
  } catch (e) { error.value = errorMessage(e, '审计日志导出失败。') }
}
function errorMessage(error: unknown, fallback: string) { const value = error as { response?: { data?: { error?: string } } }; return value.response?.data?.error || fallback }
function formatTime(value: string) { return value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '—' }
function roleLabel(value: string) { return ({ super_admin: '超级管理员', department_admin: '部门管理员', knowledge_editor: '知识编辑员', knowledge_viewer: '知识查看员' } as Record<string, string>)[value] || value || '系统' }
function moduleLabel(value: string) { return ({ login: '登录', user: '用户管理', knowledge: '知识管理', document: '文档', space: '空间' } as Record<string, string>)[value] || value || '其他' }
function resourceLabel(value: string) { return ({ user: '用户', document: '文档', space: '知识空间' } as Record<string, string>)[value] || value || '—' }
onMounted(load)
</script>

<style scoped>
.audit-page{min-height:100dvh;color:var(--ink-900);background:var(--ink-100);font-family:var(--font-body)}.audit-page__shell{display:grid;gap:var(--space-5);width:min(100%,var(--content-max-width));margin:auto;padding:var(--space-8)}.audit-page__header,.audit-page__actions,.audit-page__filters,.audit-page__tabs,.audit-page__panel-head,.audit-page__pagination,.audit-page__pagination div{display:flex;align-items:center}.audit-page__header,.audit-page__panel-head,.audit-page__pagination{justify-content:space-between;gap:var(--space-4)}.audit-page__header p,.audit-page__panel-head p{color:var(--brand-700);font:var(--font-weight-semibold) var(--text-xs)/1 var(--font-mono);letter-spacing:.08em;text-transform:uppercase}.audit-page__header h1{margin-top:var(--space-1);font:var(--font-weight-bold) var(--text-4xl)/1.1 var(--font-display)}.audit-page__header span{display:block;margin-top:var(--space-2);color:var(--ink-600)}.audit-page__actions,.audit-page__filters{gap:var(--space-3)}.audit-page__filters,.audit-page__panel{padding:var(--space-5);border:1px solid var(--ink-200);border-radius:var(--radius-lg);background:var(--ink-50);box-shadow:var(--shadow-xs)}.audit-page__filters{flex-wrap:wrap}.audit-page__tabs{padding:var(--space-1);border:1px solid var(--ink-200);border-radius:var(--radius-sm);background:var(--ink-100)}.audit-page__tabs button,.audit-page__pagination button{padding:var(--space-2) var(--space-3);border:0;border-radius:var(--radius-xs);background:transparent;font:inherit;font-weight:var(--font-weight-semibold);cursor:pointer}.audit-page__tabs button.active{color:var(--brand-900);background:var(--brand-200)}.audit-page__filters label{display:grid;gap:var(--space-1);color:var(--ink-600);font-size:var(--text-xs);font-weight:var(--font-weight-semibold)}.audit-page__filters input,.audit-page__filters select{min-width:12rem;padding:var(--space-2) var(--space-3);border:1px solid var(--ink-300);border-radius:var(--radius-sm);background:var(--ink-50);font:inherit}.audit-page__panel-head{margin-bottom:var(--space-5)}.audit-page__panel-head h2{margin-top:var(--space-1);font:var(--font-weight-bold) var(--text-xl)/1.2 var(--font-display)}.audit-page__panel-head>span,.audit-page__pagination{color:var(--ink-500);font-size:var(--text-xs)}.audit-page__table-wrap{overflow:auto}table{width:100%;border-collapse:collapse;text-align:left}th,td{padding:var(--space-3);border-bottom:1px solid var(--ink-200);vertical-align:top}th{color:var(--ink-500);font-size:var(--text-xs);white-space:nowrap}td{font-size:var(--text-sm)}td strong,td small{display:block}td small{max-width:18rem;margin-top:var(--space-1);overflow:hidden;color:var(--ink-500);font:var(--text-xs)/1.4 var(--font-mono);text-overflow:ellipsis;white-space:nowrap}time,code{font:var(--text-xs)/1.4 var(--font-mono);white-space:nowrap}.audit-page__outcome{display:inline-flex;padding:var(--space-1) var(--space-2);border-radius:var(--radius-pill);font-size:var(--text-xs);font-weight:bold}.audit-page__outcome--success{color:var(--brand-900);background:var(--brand-100)}.audit-page__outcome--denied{color:var(--ink-50);background:var(--ink-800)}.audit-page__empty{padding:var(--space-10);color:var(--ink-500);text-align:center}.audit-page__pagination{padding-top:var(--space-4)}.audit-page__pagination div{gap:var(--space-2)}.audit-page__pagination button{border:1px solid var(--ink-200)}button:disabled{cursor:not-allowed;opacity:.45}@media(max-width:48rem){.audit-page__shell{padding:var(--space-4)}.audit-page__header{align-items:flex-start;flex-direction:column}.audit-page__actions{width:100%}.audit-page__filters{align-items:stretch;flex-direction:column}.audit-page__tabs{overflow:auto}.audit-page__filters input,.audit-page__filters select{width:100%}}
</style>
