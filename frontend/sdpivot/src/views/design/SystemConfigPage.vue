<template>
  <SdpSidebarLayout mode="admin">
    <main class="admin-form-page">
      <header><div><p>System Configuration</p><h1>系统配置</h1><span>集中维护 SDPivot·文枢 的基础服务与全局参数。</span></div><SdpButton :loading="saving" @click="saveActive">保存当前配置</SdpButton></header>
      <SdpNotice v-if="notice" :type="notice.type" :title="notice.text" />
      <nav class="tabs" aria-label="系统配置分类">
        <button v-for="tab in tabs" :key="tab.id" type="button" :class="{ active: active === tab.id }" @click="active = tab.id">{{ tab.label }}</button>
      </nav>
      <section class="panel" :aria-labelledby="`${active}-title`">
        <div class="panel-head"><div><p>{{ currentTab.eyebrow }}</p><h2 :id="`${active}-title`">{{ currentTab.label }}</h2></div><span>{{ currentTab.description }}</span></div>
        <div v-if="loading" class="empty">正在加载配置...</div>
        <div v-else class="fields">
          <label v-for="field in currentTab.fields" :key="field.key"><span>{{ field.label }}</span><small>{{ field.help }}</small><textarea v-if="field.multiline" v-model="settings[active][field.key]" rows="4" /><input v-else v-model="settings[active][field.key]" :type="field.secret ? 'password' : 'text'" :placeholder="field.placeholder" /></label>
        </div>
      </section>
    </main>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { getSystemSettings, saveSystemSettings, type SettingsMap } from '@/api/admin'
import { SdpButton, SdpNotice } from '@/components/design'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'

interface ConfigField { key: string; label: string; help: string; placeholder: string; secret?: boolean; multiline?: boolean }
interface ConfigTab { id: 'storage' | 'sms' | 'wechat' | 'search' | 'cli_mcp' | 'global'; label: string; eyebrow: string; description: string; fields: ConfigField[] }

const tabs: ConfigTab[] = [
  { id: 'storage', label: '存储', eyebrow: 'Storage', description: '对象存储与文件访问', fields: [{ key: 'provider', label: '存储类型', help: '例如 minio、s3 或 local', placeholder: 'minio' }, { key: 'endpoint', label: '服务地址', help: '对象存储服务 Endpoint', placeholder: 'https://storage.example.com' }, { key: 'bucket', label: 'Bucket', help: '文档存储桶名称', placeholder: 'sdpivot-documents' }, { key: 'access_key', label: 'Access Key', help: '访问凭据标识', placeholder: '', secret: true }] },
  { id: 'sms', label: '短信', eyebrow: 'SMS', description: '验证码与通知短信', fields: [{ key: 'provider', label: '服务商', help: '短信服务商标识', placeholder: 'tencent' }, { key: 'sign_name', label: '短信签名', help: '审核通过的短信签名', placeholder: 'SDPivot文枢' }, { key: 'template_id', label: '模板 ID', help: '登录验证码模板', placeholder: '' }] },
  { id: 'wechat', label: '微信', eyebrow: 'WeChat', description: '企业微信或公众号连接', fields: [{ key: 'corp_id', label: 'Corp ID', help: '企业微信组织标识', placeholder: '' }, { key: 'agent_id', label: 'Agent ID', help: '自建应用标识', placeholder: '' }, { key: 'secret', label: '应用 Secret', help: '仅输入需要更新的密钥', placeholder: '', secret: true }] },
  { id: 'search', label: '搜索', eyebrow: 'Search', description: '检索与召回默认参数', fields: [{ key: 'top_k', label: '默认 Top K', help: '检索返回候选数量', placeholder: '10' }, { key: 'score_threshold', label: '相关度阈值', help: '0 到 1 之间', placeholder: '0.5' }, { key: 'rerank_enabled', label: '启用重排', help: '填写 true 或 false', placeholder: 'true' }] },
  { id: 'cli_mcp', label: 'CLI·MCP', eyebrow: 'CLI / MCP', description: '自动化客户端接入策略', fields: [{ key: 'enabled', label: '允许接入', help: '填写 true 或 false', placeholder: 'true' }, { key: 'default_scopes', label: '默认 Scopes', help: '逗号分隔的权限范围', placeholder: 'knowledge.read,knowledge.write' }, { key: 'token_ttl_days', label: '默认有效天数', help: '新 Token 的建议有效期', placeholder: '90' }] },
  { id: 'global', label: '全局参数', eyebrow: 'Global', description: '平台级通用行为', fields: [{ key: 'site_name', label: '站点名称', help: '统一品牌建议保持 SDPivot·文枢', placeholder: 'SDPivot·文枢' }, { key: 'support_email', label: '支持邮箱', help: '管理通知与帮助入口', placeholder: 'support@example.com' }, { key: 'announcement', label: '全局提示', help: '展示给用户的系统提示文本', placeholder: '', multiline: true }] },
] 

type Section = typeof tabs[number]['id']
const active = ref<Section>('storage')
const settings = reactive<SettingsMap>(Object.fromEntries(tabs.map((tab) => [tab.id, {}])))
const loading = ref(true)
const saving = ref(false)
const notice = ref<{ type: 'success' | 'danger'; text: string } | null>(null)
const currentTab = computed(() => tabs.find((tab) => tab.id === active.value)!)

async function load() {
  loading.value = true
  try { Object.assign(settings, (await getSystemSettings()).data.settings) }
  catch (error) { notice.value = { type: 'danger', text: message(error, '系统配置加载失败') } }
  finally { loading.value = false }
}
async function saveActive() {
  saving.value = true
  notice.value = null
  try { await saveSystemSettings(active.value, settings[active.value]); notice.value = { type: 'success', text: `${currentTab.value.label}配置已保存` } }
  catch (error) { notice.value = { type: 'danger', text: message(error, '配置保存失败') } }
  finally { saving.value = false }
}
function message(error: unknown, fallback: string) { const value = error as { response?: { data?: { error?: string } } }; return value.response?.data?.error || fallback }
onMounted(load)
</script>

<style scoped>
.admin-form-page{min-height:100dvh;padding:var(--space-8);background:var(--ink-100);color:var(--ink-900);font-family:var(--font-body)}header,.panel-head{display:flex;align-items:center;justify-content:space-between;gap:var(--space-5)}header{max-width:var(--content-max-width);margin:0 auto var(--space-6)}header p,.panel-head p{color:var(--brand-700);font:var(--font-weight-semibold) var(--text-xs)/1 var(--font-mono);letter-spacing:.08em;text-transform:uppercase}h1{margin-top:var(--space-1);font:var(--font-weight-bold) var(--text-4xl)/1.1 var(--font-display);color:var(--ink-950)}header span,.panel-head>span{display:block;margin-top:var(--space-2);color:var(--ink-600)}.tabs,.panel{max-width:var(--content-max-width);margin-inline:auto}.tabs{display:flex;gap:var(--space-2);overflow:auto;margin-top:var(--space-5);padding:var(--space-2);border:1px solid var(--ink-200);border-radius:var(--radius-md);background:var(--ink-50)}.tabs button{padding:var(--space-3) var(--space-4);border:0;border-radius:var(--radius-sm);background:transparent;color:var(--ink-700);font:inherit;font-weight:var(--font-weight-semibold);white-space:nowrap;cursor:pointer}.tabs button.active{background:var(--brand-500);color:var(--ink-950)}.panel{margin-top:var(--space-4);padding:var(--space-6);border:1px solid var(--ink-200);border-radius:var(--radius-lg);background:var(--ink-50)}h2{margin-top:var(--space-1);font:var(--font-weight-bold) var(--text-xl)/1.2 var(--font-display)}.fields{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:var(--space-5);margin-top:var(--space-6)}label{display:grid;gap:var(--space-2)}label span{font-weight:var(--font-weight-semibold)}label small{color:var(--ink-500)}input,textarea{width:100%;padding:var(--space-3);border:1px solid var(--ink-300);border-radius:var(--radius-sm);background:var(--ink-50);color:var(--ink-900);font:inherit}textarea{resize:vertical}.empty{padding:var(--space-8);text-align:center;color:var(--ink-500)}@media(max-width:48rem){.admin-form-page{padding:var(--space-4)}header,.panel-head{align-items:flex-start;flex-direction:column}.fields{grid-template-columns:1fr}.panel{padding:var(--space-5)}}
</style>
