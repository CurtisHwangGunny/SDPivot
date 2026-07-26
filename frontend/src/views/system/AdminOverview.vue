<template>
  <t-loading :loading="loading">
    <div class="overview">
      <t-alert v-if="error" theme="error" :message="error">
        <template #operation><t-button size="small" @click="loadOverview">{{ copy.retry }}</t-button></template>
      </t-alert>
      <t-alert v-if="systemInfo?.db_migration_error" theme="error" :title="copy.migrationError" :message="systemInfo.db_migration_error" />

      <div class="metric-grid">
        <section class="metric-card metric-card--accent">
          <span class="metric-label">{{ copy.service }}</span>
          <strong>{{ systemInfo?.db_migration_error ? copy.degraded : copy.operational }}</strong>
          <span class="metric-meta">{{ systemInfo?.version || copy.unknown }} · {{ systemInfo?.edition || copy.standard }}</span>
        </section>
        <section class="metric-card">
          <span class="metric-label">{{ copy.admins }}</span>
          <strong>{{ formatNumber(adminCount) }}</strong>
          <span class="metric-meta">{{ copy.adminsMeta }}</span>
        </section>
        <section class="metric-card">
          <span class="metric-label">{{ copy.security }}</span>
          <strong>{{ securityScore }}/3</strong>
          <span class="metric-meta">{{ copy.securityMeta }}</span>
        </section>
        <section class="metric-card">
          <span class="metric-label">{{ copy.recentEvents }}</span>
          <strong>{{ formatNumber(recentEvents.length) }}</strong>
          <span class="metric-meta">{{ copy.recentEventsMeta }}</span>
        </section>
      </div>

      <div class="overview-grid">
        <section class="panel">
          <div class="panel-heading"><div><h3>{{ copy.runtime }}</h3><p>{{ copy.runtimeDesc }}</p></div><t-button variant="text" @click="loadOverview">{{ copy.refresh }}</t-button></div>
          <dl class="runtime-list">
            <div><dt>{{ copy.uptime }}</dt><dd>{{ formatUptime(systemInfo?.uptime_seconds) }}</dd></div>
            <div><dt>{{ copy.database }}</dt><dd>{{ systemInfo?.db_version || copy.unknown }}</dd></div>
            <div><dt>{{ copy.vector }}</dt><dd>{{ systemInfo?.vector_store_engine || copy.unknown }}</dd></div>
            <div><dt>{{ copy.keyword }}</dt><dd>{{ systemInfo?.keyword_index_engine || copy.unknown }}</dd></div>
          </dl>
        </section>
        <section class="panel">
          <div class="panel-heading"><div><h3>{{ copy.activity }}</h3><p>{{ copy.activityDesc }}</p></div></div>
          <div v-if="recentEvents.length" class="activity-list">
            <div v-for="entry in recentEvents" :key="entry.id" class="activity-row">
              <span class="activity-dot" :class="`activity-dot--${entry.outcome}`" />
              <div><strong>{{ entry.action }}</strong><span>{{ entry.actor_user_id || copy.system }} · {{ formatDate(entry.created_at) }}</span></div>
            </div>
          </div>
          <t-empty v-else :description="copy.noActivity" />
        </section>
      </div>
    </div>
  </t-loading>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  getIPWhitelist, getLoginLockoutPolicy, getPasswordPolicy, getSystemInfo,
  listSystemAdmins, listSystemAuditLog, type AuditLog, type SystemInfo,
} from '@/api/system'

const { locale } = useI18n()
const loading = ref(false)
const error = ref('')
const systemInfo = ref<SystemInfo | null>(null)
const adminCount = ref(0)
const securityScore = ref(0)
const recentEvents = ref<AuditLog[]>([])

const copy = computed(() => locale.value.startsWith('zh') ? {
  retry: '重试', refresh: '刷新', service: '服务状态', operational: '运行正常', degraded: '需要处理', unknown: '未知', standard: '标准版', admins: '系统管理员', adminsMeta: '拥有平台管理权限的账号', security: '安全基线', securityMeta: '已启用的关键安全控制', recentEvents: '近期审计事件', recentEventsMeta: '最新 8 条系统级操作', migrationError: '数据库迁移异常', runtime: '运行环境', runtimeDesc: '当前服务与基础设施状态。', uptime: '运行时长', database: '数据库版本', vector: '向量存储', keyword: '关键词索引', activity: '最近活动', activityDesc: '最新的平台管理操作。', system: '系统', noActivity: '暂无审计活动',
} : {
  retry: 'Retry', refresh: 'Refresh', service: 'Service status', operational: 'Operational', degraded: 'Attention needed', unknown: 'Unknown', standard: 'standard', admins: 'System admins', adminsMeta: 'Accounts with platform privileges', security: 'Security baseline', securityMeta: 'Key controls currently enabled', recentEvents: 'Recent audit events', recentEventsMeta: 'Latest 8 system-level actions', migrationError: 'Database migration issue', runtime: 'Runtime', runtimeDesc: 'Current service and infrastructure state.', uptime: 'Uptime', database: 'Database version', vector: 'Vector store', keyword: 'Keyword index', activity: 'Recent activity', activityDesc: 'Latest platform administration actions.', system: 'System', noActivity: 'No audit activity yet',
})

async function loadOverview() {
  loading.value = true
  error.value = ''
  try {
    const [info, admins, whitelist, password, lockout, audit] = await Promise.all([
      getSystemInfo(), listSystemAdmins({ limit: 1 }), getIPWhitelist(), getPasswordPolicy(),
      getLoginLockoutPolicy(), listSystemAuditLog({ limit: 8 }),
    ])
    systemInfo.value = info.data
    adminCount.value = admins.total
    securityScore.value = Number(whitelist.entries.length > 0) + Number(password.complexity && password.min_length >= 8) + Number(lockout.max_failed_attempts > 0)
    recentEvents.value = audit.data || []
  } catch (err: any) {
    error.value = err?.message || 'Failed to load administration overview'
  } finally {
    loading.value = false
  }
}

function formatNumber(value: number) { return new Intl.NumberFormat(locale.value).format(value) }
function formatDate(value: string) { return new Date(value).toLocaleString(locale.value) }
function formatUptime(value?: number) {
  if (value == null) return copy.value.unknown
  const days = Math.floor(value / 86400)
  const hours = Math.floor((value % 86400) / 3600)
  const minutes = Math.floor((value % 3600) / 60)
  return days ? `${days}d ${hours}h` : hours ? `${hours}h ${minutes}m` : `${minutes}m`
}

onMounted(loadOverview)
</script>

<style scoped>
.overview { display: grid; gap: 16px; }
.metric-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 14px; }
.metric-card, .panel { border: 1px solid var(--td-component-border); border-radius: 12px; background: var(--td-bg-color-container); }
.metric-card { position: relative; display: grid; gap: 8px; min-height: 112px; padding: 20px; overflow: hidden; }
.metric-card--accent::after { content: ''; position: absolute; right: -30px; top: -45px; width: 120px; height: 120px; border-radius: 50%; background: color-mix(in srgb, var(--td-brand-color) 14%, transparent); }
.metric-label, .metric-meta, .panel-heading p, .activity-row span { color: var(--td-text-color-secondary); }
.metric-card strong { font-size: 28px; line-height: 1; color: var(--td-text-color-primary); }
.metric-meta { font-size: 12px; }
.overview-grid { display: grid; grid-template-columns: minmax(0, .85fr) minmax(0, 1.15fr); gap: 16px; }
.panel { padding: 20px; }
.panel-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; margin-bottom: 16px; }
.panel-heading h3, .panel-heading p { margin: 0; }
.panel-heading p { margin-top: 5px; line-height: 1.5; }
.runtime-list { margin: 0; }
.runtime-list div { display: flex; justify-content: space-between; gap: 20px; padding: 13px 0; border-top: 1px solid var(--td-component-stroke); }
.runtime-list dt { color: var(--td-text-color-secondary); }.runtime-list dd { margin: 0; color: var(--td-text-color-primary); font-weight: 500; text-align: right; }
.activity-list { display: grid; }.activity-row { display: grid; grid-template-columns: 10px 1fr; gap: 12px; padding: 12px 0; border-top: 1px solid var(--td-component-stroke); }
.activity-row div { display: grid; gap: 4px; min-width: 0; }.activity-row strong { overflow: hidden; color: var(--td-text-color-primary); text-overflow: ellipsis; white-space: nowrap; }.activity-row span { font-size: 12px; }
.activity-dot { width: 8px; height: 8px; margin-top: 5px; border-radius: 50%; background: var(--td-success-color); }.activity-dot--denied { background: var(--td-error-color); }
@media (max-width: 1000px) { .metric-grid { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 700px) { .metric-grid, .overview-grid { grid-template-columns: 1fr; } }
</style>
