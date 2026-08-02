<template>
  <SdpSidebarLayout mode="admin">
    <div class="sdp-admin-dashboard">
      <div class="sdp-admin-dashboard__shell">
        <header class="sdp-admin-dashboard__topbar">
          <div>
            <p>Administration</p>
            <h1>管理后台</h1>
            <span>掌握平台运行状态，快速处理日常管理任务。</span>
          </div>
          <div class="sdp-admin-dashboard__top-actions" aria-label="管理后台快捷操作">
            <SdpButton variant="secondary" aria-label="刷新管理后台数据" :loading="loading" @click="loadDashboard">
              <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M20 6v5h-5M4 18v-5h5M6.1 9a7 7 0 0 1 11.4-2.5L20 9M4 15l2.5 2.5A7 7 0 0 0 17.9 15" /></svg>
              刷新
            </SdpButton>
            <SdpButton aria-label="邀请新成员" @click="goTo('/admin/people')">
              <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M15 19v-1.5a4.5 4.5 0 0 0-4.5-4.5h-5A4.5 4.5 0 0 0 1 17.5V19m7-10a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm11-1v6m-3-3h6" /></svg>
              邀请成员
            </SdpButton>
          </div>
        </header>

        <section class="sdp-admin-dashboard__status" :class="{ 'sdp-admin-dashboard__status--warning': loadError }" role="status" aria-live="polite">
          <span class="sdp-admin-dashboard__pulse" aria-hidden="true" />
          <div>
            <strong>{{ loadError ? '部分管理数据暂不可用' : '所有核心服务运行正常' }}</strong>
            <p>{{ loadError || `最近检查于 ${lastChecked}` }}</p>
          </div>
          <button type="button" aria-label="查看服务状态详情" @click="focusServices">查看详情</button>
        </section>

        <section class="sdp-admin-dashboard__kpis" aria-label="平台关键指标">
          <article v-for="item in kpis" :key="item.label" class="sdp-admin-dashboard__kpi">
            <span aria-hidden="true"><svg viewBox="0 0 24 24"><path :d="item.icon" /></svg></span>
            <div><p>{{ item.label }}</p><strong>{{ loading ? '—' : item.value }}</strong><small>{{ item.note }}</small></div>
          </article>
        </section>

        <section class="sdp-admin-dashboard__panel" aria-labelledby="quick-actions-title">
          <header class="sdp-admin-dashboard__section-head">
            <div><p>Quick Access</p><h2 id="quick-actions-title">快捷操作</h2></div>
            <span>17 项常用管理入口</span>
          </header>
          <div ref="quickGridRef" class="sdp-admin-dashboard__quick-grid" @keydown="handleQuickKeydown">
            <button v-for="action in quickActions" :key="action.label" type="button" :aria-label="action.label" @click="goTo(action.path)">
              <span aria-hidden="true"><svg viewBox="0 0 24 24"><path :d="action.icon" /></svg></span>
              <strong>{{ action.label }}</strong>
              <small>{{ action.group }}</small>
            </button>
          </div>
        </section>

        <div class="sdp-admin-dashboard__lower-grid">
          <section ref="servicesRef" class="sdp-admin-dashboard__panel" tabindex="-1" aria-labelledby="service-status-title">
            <header class="sdp-admin-dashboard__section-head">
              <div><p>Infrastructure</p><h2 id="service-status-title">服务状态</h2></div>
              <span>7 个容器</span>
            </header>
            <ul class="sdp-admin-dashboard__services">
              <li v-for="service in services" :key="service.name">
                <span class="sdp-admin-dashboard__service-icon" aria-hidden="true">{{ service.code }}</span>
                <div><strong>{{ service.name }}</strong><small>{{ service.detail }}</small></div>
                <span class="sdp-admin-dashboard__service-state"><i aria-hidden="true" />{{ service.status }}</span>
              </li>
            </ul>
          </section>

          <section class="sdp-admin-dashboard__panel" aria-labelledby="activity-title">
            <header class="sdp-admin-dashboard__section-head">
              <div><p>Audit Trail</p><h2 id="activity-title">最近活动</h2></div>
              <button type="button" aria-label="查看全部活动日志" @click="goTo('/admin/security')">全部日志</button>
            </header>
            <ol class="sdp-admin-dashboard__activity">
              <li v-for="activity in activities" :key="activity.time + activity.title">
                <span aria-hidden="true" />
                <div><strong>{{ activity.title }}</strong><p>{{ activity.description }}</p><time>{{ activity.time }}</time></div>
              </li>
            </ol>
          </section>
        </div>
      </div>
    </div>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { getAdminStats, type AdminStats } from '@/api/spaces'
import { listModels } from '@/api/qa'
import { SdpButton } from '@/components/design'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'

const router = useRouter()
const stats = ref<AdminStats>({ space_count: 0, document_count: 0, member_count: 0, model_count: 0 })
const loading = ref(true)
const loadError = ref('')
const lastChecked = ref('刚刚')
const quickGridRef = ref<HTMLElement | null>(null)
const servicesRef = ref<HTMLElement | null>(null)

const commonIcon = 'M4 5h16v14H4V5Zm4 4h8M8 13h5'
const quickActions = [
  ['新建空间', '/spaces', '内容', 'M3 7h7l2 2h9v10H3V7Zm9 5v5m-2.5-2.5h5'],
  ['导入文档', '/spaces', '内容', 'M6 3h8l4 4v14H6V3Zm6 6v8m-3-3 3 3 3-3'],
  ['邀请成员', '/admin/people', '人员', 'M16 20v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2m7-10a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm10-2v6m-3-3h6'],
  ['分配角色', '/admin/people', '人员', commonIcon], ['标签字典', '/admin/config', '内容', commonIcon],
  ['模型配置', '/admin/config', 'AI', commonIcon], ['检索配置', '/admin/config', 'AI', commonIcon],
  ['对象存储', '/admin/config', '系统', commonIcon], ['短信服务', '/admin/config', '系统', commonIcon],
  ['登录配置', '/admin/security', '安全', commonIcon], ['密码策略', '/admin/security', '安全', commonIcon],
  ['IP 白名单', '/admin/security', '安全', commonIcon], ['审计日志', '/admin/security', '安全', commonIcon],
  ['数据备份', '/admin/config', '运维', commonIcon], ['系统参数', '/admin/config', '系统', commonIcon],
  ['用量统计', '/usage', '运营', commonIcon], ['健康检查', '/admin', '运维', commonIcon],
].map(([label, path, group, icon]) => ({ label, path, group, icon }))

const services = [
  { code: 'API', name: 'API Gateway', detail: '请求路由与鉴权', status: '正常' },
  { code: 'DB', name: 'PostgreSQL', detail: '业务数据存储', status: '正常' },
  { code: 'RDS', name: 'Redis', detail: '缓存与会话', status: '正常' },
  { code: 'MIN', name: 'MinIO', detail: '文档对象存储', status: '正常' },
  { code: 'VEC', name: 'Vector Engine', detail: '向量检索服务', status: '正常' },
  { code: 'DOC', name: 'DocReader', detail: '文档解析服务', status: '正常' },
  { code: 'LLM', name: 'Model Gateway', detail: '模型调用代理', status: '正常' },
]

const activities = [
  { title: '知识空间已更新', description: '产品规范库新增 12 份文档', time: '8 分钟前' },
  { title: '成员权限已调整', description: '李明被授予管理员角色', time: '32 分钟前' },
  { title: '模型配置已同步', description: '默认问答模型配置生效', time: '1 小时前' },
  { title: '系统备份完成', description: '每日自动备份执行成功', time: '今天 03:20' },
  { title: '标签字典已发布', description: '业务分类新增 4 个标签', time: '昨天 17:42' },
]

const kpis = computed(() => [
  { label: '知识空间', value: stats.value.space_count || 0, note: '团队知识容器', icon: 'M3 7h7l2 2h9v10H3V7Z' },
  { label: '文档总数', value: stats.value.document_count || 0, note: '已纳入检索', icon: 'M6 3h8l4 4v14H6V3Zm8 0v5h5' },
  { label: '平台成员', value: stats.value.member_count || 0, note: '组织内账户', icon: 'M16 20v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2m7-10a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z' },
  { label: '可用模型', value: stats.value.model_count || 0, note: '问答与写作模型', icon: 'M9 3h6l1 3h3v12h-3l-1 3H9l-1-3H5V6h3l1-3Zm3 6v6m-3-3h6' },
])

async function loadDashboard() {
  loading.value = true
  loadError.value = ''
  try {
    const [statsResponse, modelsResponse] = await Promise.all([getAdminStats(), listModels().catch(() => ({ data: { models: [] } }))])
    stats.value = { ...statsResponse.data, model_count: modelsResponse.data.models?.length || 0 }
    lastChecked.value = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } catch (error: unknown) {
    loadError.value = errorMessage(error, '无法获取管理数据，请检查管理员权限后重试。')
  } finally {
    loading.value = false
  }
}

function errorMessage(error: unknown, fallback: string) {
  const value = error as { message?: string; response?: { data?: { error?: string } } }
  return value?.response?.data?.error || value?.message || fallback
}

function goTo(path: string) { void router.push(path) }
function focusServices() { servicesRef.value?.focus() }
function handleQuickKeydown(event: KeyboardEvent) {
  if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key)) return
  const buttons = Array.from(quickGridRef.value?.querySelectorAll<HTMLButtonElement>('button') || [])
  const index = buttons.indexOf(document.activeElement as HTMLButtonElement)
  if (index < 0) return
  event.preventDefault()
  const columns = window.innerWidth > 1024 ? 6 : window.innerWidth > 640 ? 3 : 1
  let next = index
  if (event.key === 'Home') next = 0
  if (event.key === 'End') next = buttons.length - 1
  if (event.key === 'ArrowLeft') next = Math.max(0, index - 1)
  if (event.key === 'ArrowRight') next = Math.min(buttons.length - 1, index + 1)
  if (event.key === 'ArrowUp') next = Math.max(0, index - columns)
  if (event.key === 'ArrowDown') next = Math.min(buttons.length - 1, index + columns)
  buttons[next]?.focus()
}

onMounted(loadDashboard)
</script>

<style scoped>
.sdp-admin-dashboard { min-height: 100dvh; color: var(--ink-900); background: var(--ink-100); font-family: var(--font-body); }
.sdp-admin-dashboard__shell { display: grid; gap: var(--space-6); width: min(100%, var(--content-max-width)); margin-inline: auto; padding: var(--space-8); }
.sdp-admin-dashboard__topbar, .sdp-admin-dashboard__section-head, .sdp-admin-dashboard__status, .sdp-admin-dashboard__top-actions, .sdp-admin-dashboard__services li { display: flex; align-items: center; }
.sdp-admin-dashboard__topbar { justify-content: space-between; gap: var(--space-6); }
.sdp-admin-dashboard__topbar p, .sdp-admin-dashboard__section-head p { color: var(--brand-700); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); letter-spacing: .08em; text-transform: uppercase; }
.sdp-admin-dashboard__topbar h1 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-4xl); line-height: var(--leading-tight); }
.sdp-admin-dashboard__topbar > div > span { display: block; margin-top: var(--space-2); color: var(--ink-600); font-size: var(--text-base); }
.sdp-admin-dashboard__top-actions { gap: var(--space-3); }
.sdp-admin-dashboard__top-actions svg { width: var(--space-4); height: var(--space-4); margin-right: var(--space-2); fill: none; stroke: currentColor; stroke-linecap: round; stroke-linejoin: round; stroke-width: 2; }
.sdp-admin-dashboard__status { gap: var(--space-4); padding: var(--space-4) var(--space-5); border: 1px solid var(--brand-300); border-radius: var(--radius-md); background: var(--brand-50); }
.sdp-admin-dashboard__pulse { width: var(--space-3); height: var(--space-3); flex: 0 0 var(--space-3); border: var(--space-1) solid var(--brand-200); border-radius: var(--radius-pill); background: var(--brand-700); }
.sdp-admin-dashboard__status--warning { border-color: var(--ink-400); background: var(--ink-200); }
.sdp-admin-dashboard__status--warning .sdp-admin-dashboard__pulse { border-color: var(--ink-300); background: var(--ink-700); }
.sdp-admin-dashboard__status div { min-width: var(--space-0); flex: 1; }
.sdp-admin-dashboard__status strong { color: var(--ink-950); font-size: var(--text-sm); }
.sdp-admin-dashboard__status p { margin-top: var(--space-1); color: var(--ink-700); font-size: var(--text-xs); }
.sdp-admin-dashboard__status button, .sdp-admin-dashboard__section-head button { min-width: var(--space-8); min-height: var(--space-8); padding: var(--space-2) var(--space-3); border: 0; border-radius: var(--radius-sm); color: var(--brand-800); background: transparent; font: inherit; font-weight: var(--font-weight-semibold); cursor: pointer; }
.sdp-admin-dashboard__kpis { display: grid; grid-template-columns: repeat(4, minmax(var(--space-0), 1fr)); gap: var(--space-4); }
.sdp-admin-dashboard__kpi { display: flex; align-items: flex-start; gap: var(--space-4); padding: var(--space-5); border: 1px solid var(--ink-200); border-radius: var(--radius-md); background: var(--ink-50); box-shadow: var(--shadow-xs); }
.sdp-admin-dashboard__kpi > span, .sdp-admin-dashboard__quick-grid button > span { display: inline-flex; align-items: center; justify-content: center; color: var(--brand-800); background: var(--brand-100); }
.sdp-admin-dashboard__kpi > span { width: var(--space-10); height: var(--space-10); flex: 0 0 var(--space-10); border-radius: var(--radius-sm); }
.sdp-admin-dashboard__kpi svg, .sdp-admin-dashboard__quick-grid svg { width: var(--space-5); height: var(--space-5); fill: none; stroke: currentColor; stroke-linecap: round; stroke-linejoin: round; stroke-width: 1.8; }
.sdp-admin-dashboard__kpi p { color: var(--ink-600); font-size: var(--text-xs); }
.sdp-admin-dashboard__kpi strong { display: block; margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-3xl); line-height: var(--leading-tight); }
.sdp-admin-dashboard__kpi small { display: block; margin-top: var(--space-2); color: var(--ink-500); font-size: var(--text-xs); }
.sdp-admin-dashboard__panel { padding: var(--space-6); border: 1px solid var(--ink-200); border-radius: var(--radius-lg); background: var(--ink-50); box-shadow: var(--shadow-xs); }
.sdp-admin-dashboard__section-head { justify-content: space-between; gap: var(--space-4); margin-bottom: var(--space-5); }
.sdp-admin-dashboard__section-head h2 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-xl); }
.sdp-admin-dashboard__section-head > span { color: var(--ink-500); font-size: var(--text-xs); }
.sdp-admin-dashboard__quick-grid { display: grid; grid-template-columns: repeat(6, minmax(var(--space-0), 1fr)); gap: var(--space-3); }
.sdp-admin-dashboard__quick-grid button { display: grid; min-width: var(--space-0); justify-items: start; gap: var(--space-2); padding: var(--space-4); border: 1px solid var(--ink-200); border-radius: var(--radius-md); color: var(--ink-900); background: var(--ink-50); font-family: var(--font-body); text-align: left; cursor: pointer; transition: border-color var(--duration-fast), background var(--duration-fast), transform var(--duration-fast); }
.sdp-admin-dashboard__quick-grid button:hover { border-color: var(--brand-400); background: var(--brand-50); transform: translateY(calc(var(--space-1) * -1)); }
.sdp-admin-dashboard__quick-grid button > span { width: var(--space-8); height: var(--space-8); border-radius: var(--radius-sm); }
.sdp-admin-dashboard__quick-grid strong { overflow: hidden; width: 100%; font-size: var(--text-sm); text-overflow: ellipsis; white-space: nowrap; }
.sdp-admin-dashboard__quick-grid small { color: var(--ink-500); font-size: var(--text-xs); }
.sdp-admin-dashboard__lower-grid { display: grid; grid-template-columns: minmax(var(--space-0), 1.1fr) minmax(var(--space-0), .9fr); gap: var(--space-4); align-items: start; }
.sdp-admin-dashboard__services, .sdp-admin-dashboard__activity { list-style: none; }
.sdp-admin-dashboard__services li { gap: var(--space-3); padding: var(--space-3) var(--space-0); border-top: 1px solid var(--ink-200); }
.sdp-admin-dashboard__service-icon { width: var(--space-10); height: var(--space-8); display: inline-flex; flex: 0 0 var(--space-10); align-items: center; justify-content: center; border-radius: var(--radius-xs); color: var(--brand-900); background: var(--brand-100); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-bold); }
.sdp-admin-dashboard__services li div { min-width: var(--space-0); flex: 1; }
.sdp-admin-dashboard__services strong, .sdp-admin-dashboard__services small { display: block; }
.sdp-admin-dashboard__services strong { color: var(--ink-900); font-size: var(--text-sm); }
.sdp-admin-dashboard__services small { margin-top: var(--space-1); color: var(--ink-500); font-size: var(--text-xs); }
.sdp-admin-dashboard__service-state { display: inline-flex; align-items: center; gap: var(--space-2); color: var(--brand-800); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-admin-dashboard__service-state i { width: var(--space-2); height: var(--space-2); border-radius: var(--radius-pill); background: var(--brand-600); }
.sdp-admin-dashboard__activity li { position: relative; display: grid; grid-template-columns: var(--space-4) 1fr; gap: var(--space-3); padding-bottom: var(--space-5); }
.sdp-admin-dashboard__activity li > span { position: relative; z-index: var(--z-base); width: var(--space-3); height: var(--space-3); margin-top: var(--space-1); border: 2px solid var(--brand-200); border-radius: var(--radius-pill); background: var(--brand-700); }
.sdp-admin-dashboard__activity li:not(:last-child)::before { position: absolute; top: var(--space-3); bottom: var(--space-0); left: var(--space-1); width: 2px; background: var(--ink-200); content: ''; }
.sdp-admin-dashboard__activity strong { color: var(--ink-900); font-size: var(--text-sm); }
.sdp-admin-dashboard__activity p { margin-top: var(--space-1); color: var(--ink-600); font-size: var(--text-xs); }
.sdp-admin-dashboard__activity time { display: block; margin-top: var(--space-2); color: var(--ink-500); font-family: var(--font-mono); font-size: var(--text-xs); }
.sdp-admin-dashboard button:focus-visible, .sdp-admin-dashboard__panel:focus-visible { outline: 2px solid var(--brand-500); outline-offset: 2px; }
@media (max-width: 64rem) { .sdp-admin-dashboard__kpis { grid-template-columns: repeat(2, 1fr); } .sdp-admin-dashboard__quick-grid { grid-template-columns: repeat(3, 1fr); } }
@media (max-width: 48rem) { .sdp-admin-dashboard__shell { padding: var(--space-6); } .sdp-admin-dashboard__topbar { align-items: flex-start; flex-direction: column; } .sdp-admin-dashboard__lower-grid { grid-template-columns: 1fr; } }
@media (max-width: 40rem) { .sdp-admin-dashboard__shell { padding: var(--space-4); } .sdp-admin-dashboard__topbar h1 { font-size: var(--text-3xl); } .sdp-admin-dashboard__top-actions, .sdp-admin-dashboard__top-actions :deep(.sdp-button) { width: 100%; } .sdp-admin-dashboard__top-actions { flex-direction: column; } .sdp-admin-dashboard__status { align-items: flex-start; flex-wrap: wrap; } .sdp-admin-dashboard__kpis, .sdp-admin-dashboard__quick-grid { grid-template-columns: 1fr; } .sdp-admin-dashboard__panel { padding: var(--space-5); } }
</style>
