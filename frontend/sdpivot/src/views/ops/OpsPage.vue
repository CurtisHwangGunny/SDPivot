<template>
  <div class="ops-page">
    <h1>运营管理端</h1>
    <t-loading v-if="loading" />
    <div v-else>
      <t-row :gutter="16" class="stats-row">
        <t-col :span="3"><t-card class="stat-card"><div class="stat-value">{{ stats.tenant_count || 0 }}</div><div class="stat-label">企业数</div></t-card></t-col>
        <t-col :span="3"><t-card class="stat-card"><div class="stat-value">{{ stats.user_count || 0 }}</div><div class="stat-label">用户数</div></t-card></t-col>
        <t-col :span="3"><t-card class="stat-card"><div class="stat-value">{{ stats.document_count || 0 }}</div><div class="stat-label">文档数</div></t-card></t-col>
        <t-col :span="3"><t-card class="stat-card"><div class="stat-value">{{ stats.token_usage_total || 0 }}</div><div class="stat-label">Token总量</div></t-card></t-col>
        <t-col :span="3"><t-card class="stat-card"><div class="stat-value">{{ stats.token_usage_today || 0 }}</div><div class="stat-label">今日Token</div></t-card></t-col>
        <t-col :span="3"><t-card class="stat-card"><div class="stat-value">{{ formatStorage(stats.storage_bytes) }}</div><div class="stat-label">存储用量</div></t-card></t-col>
      </t-row>

      <OpsPageBody />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { getOpsDashboard } from '@/api/ops'
import OpsPageBody from '@/components/ops/OpsPageBody.vue'

const loading = ref(true)
const stats = ref<any>({})

function formatStorage(bytes: number) {
  if (!bytes) return '0'
  if (bytes < 1024) return bytes + 'B'
  if (bytes < 1048576) return (bytes / 1024).toFixed(1) + 'KB'
  if (bytes < 1073741824) return (bytes / 1048576).toFixed(1) + 'MB'
  return (bytes / 1073741824).toFixed(1) + 'GB'
}

async function loadDashboard() {
  loading.value = true
  try {
    const res = await getOpsDashboard().catch(() => ({ data: {} }))
    stats.value = (res.data as any) || {}
  } catch {
    MessagePlugin.warning('加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadDashboard)
</script>

<style scoped>
.ops-page { max-width: 1200px; margin: 0 auto; }
.ops-page h1 { font-size: 24px; font-weight: 600; margin-bottom: 24px; }
.stats-row { margin-bottom: 16px; }
.stat-card { text-align: center; }
.stat-value { font-size: 28px; font-weight: 700; color: var(--brand-primary); }
.stat-label { font-size: 13px; color: var(--text-secondary); margin-top: 4px; }
</style>
