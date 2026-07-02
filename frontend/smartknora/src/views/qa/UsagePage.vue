<template>
  <div class="usage-page">
    <h1>用量统计</h1>
    <t-loading v-if="loading" />
    <div v-else class="stats-grid">
      <t-card class="stat-card">
        <div class="stat-value">{{ summary.total_input_tokens || 0 }}</div>
        <div class="stat-label">输入 Token</div>
      </t-card>
      <t-card class="stat-card">
        <div class="stat-value">{{ summary.total_output_tokens || 0 }}</div>
        <div class="stat-label">输出 Token</div>
      </t-card>
      <t-card class="stat-card">
        <div class="stat-value">{{ summary.total_requests || 0 }}</div>
        <div class="stat-label">API 调用次数</div>
      </t-card>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getUsageSummary } from '@/api/usage'

const loading = ref(true)
const summary = ref<any>({})

async function load() {
  loading.value = true
  try { const res = await getUsageSummary(); summary.value = res.data } catch {}
  finally { loading.value = false }
}
onMounted(load)
</script>
<style scoped>
.usage-page { max-width: 900px; margin: 0 auto; }
.usage-page h1 { font-size: 24px; font-weight: 600; margin-bottom: 24px; }
.stats-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; }
.stat-card { text-align: center; }
.stat-value { font-size: 32px; font-weight: 700; color: #014DB2; }
.stat-label { font-size: 14px; color: #999; margin-top: 4px; }
</style>
