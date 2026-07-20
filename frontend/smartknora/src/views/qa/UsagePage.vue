<template>
  <div class="usage-shell">
    <section class="usage-hero">
      <div>
        <p class="usage-eyebrow">Consumption overview</p>
        <h1>用量统计</h1>
        <p class="usage-subtitle">聚合查看 Token 消耗、请求规模与模型使用趋势，让问答与写作消耗更透明。</p>
      </div>
      <div class="usage-hero-note">
        <span>数据来源</span>
        <strong>当前租户实时聚合</strong>
        <small>保留现有 summary 接口，不改变统计口径</small>
      </div>
    </section>

    <t-loading v-if="loading" text="正在加载用量统计..." class="usage-loading" />

    <template v-else>
      <section class="usage-metrics">
        <article class="usage-card accent-card">
          <span>输入 Token</span>
          <strong>{{ formatNumber(summary.total_input_tokens) }}</strong>
          <small>模型输入总量</small>
        </article>
        <article class="usage-card">
          <span>输出 Token</span>
          <strong>{{ formatNumber(summary.total_output_tokens) }}</strong>
          <small>模型输出总量</small>
        </article>
        <article class="usage-card">
          <span>请求次数</span>
          <strong>{{ formatNumber(summary.total_requests) }}</strong>
          <small>累计 API 调用次数</small>
        </article>
      </section>

      <section class="usage-insights">
        <div class="insight-panel main-panel">
          <div class="panel-head">
            <div>
              <p class="panel-kicker">Usage mix</p>
              <h2>用量结构</h2>
            </div>
          </div>
          <div class="ratio-grid">
            <div class="ratio-item">
              <span>输入占比</span>
              <strong>{{ inputShare }}%</strong>
            </div>
            <div class="ratio-item">
              <span>输出占比</span>
              <strong>{{ outputShare }}%</strong>
            </div>
            <div class="ratio-item">
              <span>单次均值</span>
              <strong>{{ avgPerRequest }}</strong>
            </div>
          </div>
        </div>

        <div class="insight-panel side-panel">
          <div class="panel-head compact">
            <div>
              <p class="panel-kicker">Interpretation</p>
              <h2>阅读提示</h2>
            </div>
          </div>
          <ul class="usage-tips">
            <li>输入 Token 高，通常意味着提示词更长或知识引用更多。</li>
            <li>输出 Token 高，通常说明生成内容更长或回答更完整。</li>
            <li>后续可继续叠加模型维度、时间维度与趋势图。</li>
          </ul>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { getUsageSummary } from '@/api/usage'

const loading = ref(true)
const summary = ref<any>({})

function formatNumber(value: number | string | undefined) {
  return Number(value || 0).toLocaleString('zh-CN')
}

const totalTokens = computed(() => Number(summary.value.total_input_tokens || 0) + Number(summary.value.total_output_tokens || 0))
const inputShare = computed(() => totalTokens.value ? ((Number(summary.value.total_input_tokens || 0) / totalTokens.value) * 100).toFixed(1) : '0.0')
const outputShare = computed(() => totalTokens.value ? ((Number(summary.value.total_output_tokens || 0) / totalTokens.value) * 100).toFixed(1) : '0.0')
const avgPerRequest = computed(() => {
  const total = totalTokens.value
  const requests = Number(summary.value.total_requests || 0)
  if (!requests) return '0'
  return Math.round(total / requests).toLocaleString('zh-CN')
})

async function load() {
  loading.value = true
  try {
    const res = await getUsageSummary()
    summary.value = res.data
  } catch {}
  finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.usage-shell {
  display: grid;
  gap: 24px;
}
.usage-hero,
.usage-metrics,
.usage-insights {
  display: grid;
  gap: 18px;
}
.usage-hero {
  grid-template-columns: minmax(0, 1.8fr) minmax(280px, 0.9fr);
  align-items: stretch;
}
.usage-eyebrow,
.panel-kicker {
  margin: 0 0 8px;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--brand-primary);
}
.usage-hero h1,
.panel-head h2 {
  margin: 0;
  font-size: 30px;
  line-height: 1.1;
  color: var(--text-primary);
}
.usage-subtitle {
  margin: 14px 0 0;
  max-width: 680px;
  color: var(--text-secondary);
  line-height: 1.7;
}
.usage-hero-note,
.usage-card,
.insight-panel {
  border-radius: 12px;
  border: 1px solid var(--border-soft);
  background: var(--surface-elevated);
  box-shadow: var(--shadow-soft);
}
.usage-hero-note {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 8px;
  padding: 24px;
  background: linear-gradient(135deg, rgba(0, 185, 107, 0.12), rgba(31, 41, 55, 0.04));
}
.usage-hero-note span,
.usage-card span,
.ratio-item span {
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-muted);
}
.usage-hero-note strong,
.usage-card strong,
.ratio-item strong {
  font-size: 28px;
  color: var(--text-primary);
}
.usage-hero-note small,
.usage-card small {
  color: var(--text-secondary);
}
.usage-metrics {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
.usage-card {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.accent-card {
  background: linear-gradient(145deg, color-mix(in srgb, var(--brand-primary) 14%, transparent), color-mix(in srgb, var(--surface-elevated) 96%, transparent));
}
.usage-insights {
  grid-template-columns: minmax(0, 1.6fr) minmax(280px, 0.9fr);
}
.insight-panel {
  padding: 24px;
}
.panel-head {
  margin-bottom: 18px;
}
.compact {
  margin-bottom: 14px;
}
.ratio-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}
.ratio-item {
  padding: 18px;
  border-radius: 12px;
  background: color-mix(in srgb, var(--sk-surface-soft) 88%, transparent);
  border: 1px solid var(--border-soft);
}
.usage-tips {
  margin: 0;
  padding-left: 18px;
  color: var(--text-secondary);
  line-height: 1.8;
}
.usage-loading {
  min-height: 240px;
  display: flex;
  align-items: center;
  justify-content: center;
}
@media (max-width: 1080px) {
  .usage-hero,
  .usage-insights,
  .usage-metrics,
  .ratio-grid {
    grid-template-columns: 1fr;
  }
}
</style>
