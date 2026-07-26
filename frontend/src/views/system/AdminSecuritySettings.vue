<template>
  <t-loading :loading="loading">
    <div class="security-page">
      <t-alert theme="warning" :message="copy.notice" />
      <section class="security-card">
        <div class="card-copy"><span class="eyebrow">{{ copy.network }}</span><h3>{{ copy.allowlist }}</h3><p>{{ copy.allowlistDesc }}</p></div>
        <div class="card-control"><t-tag-input v-model="whitelist.entries" clearable :placeholder="copy.allowlistPlaceholder" /><span class="hint">{{ copy.allowlistHint }}</span><t-button theme="primary" :loading="saving === 'whitelist'" @click="saveWhitelist">{{ copy.save }}</t-button></div>
      </section>
      <section class="security-card">
        <div class="card-copy"><span class="eyebrow">{{ copy.identity }}</span><h3>{{ copy.password }}</h3><p>{{ copy.passwordDesc }}</p></div>
        <div class="card-control form-control">
          <t-form-item :label="copy.minLength"><t-input-number v-model="password.min_length" :min="6" :max="128" /></t-form-item>
          <t-form-item :label="copy.rotation"><t-input-number v-model="password.rotation_days" :min="0" :max="3650" /></t-form-item>
          <label class="switch-control"><t-switch v-model="password.complexity" />{{ copy.complexity }}</label>
          <t-button theme="primary" :loading="saving === 'password'" @click="savePassword">{{ copy.save }}</t-button>
        </div>
      </section>
      <section class="security-card">
        <div class="card-copy"><span class="eyebrow">{{ copy.protection }}</span><h3>{{ copy.lockout }}</h3><p>{{ copy.lockoutDesc }}</p></div>
        <div class="card-control form-control">
          <t-form-item :label="copy.attempts"><t-input-number v-model="lockout.max_failed_attempts" :min="0" :max="100" /></t-form-item>
          <t-form-item :label="copy.duration"><t-input-number v-model="lockout.lockout_minutes" :min="1" :max="10080" /></t-form-item>
          <span class="hint">{{ copy.disabledHint }}</span>
          <t-button theme="primary" :loading="saving === 'lockout'" @click="saveLockout">{{ copy.save }}</t-button>
        </div>
      </section>
    </div>
  </t-loading>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import {
  getIPWhitelist, getLoginLockoutPolicy, getPasswordPolicy, updateIPWhitelist,
  updateLoginLockoutPolicy, updatePasswordPolicy,
} from '@/api/system'

const { locale } = useI18n()
const loading = ref(false)
const saving = ref('')
const whitelist = reactive({ entries: [] as string[] })
const password = reactive({ min_length: 8, complexity: true, rotation_days: 90 })
const lockout = reactive({ max_failed_attempts: 5, lockout_minutes: 30 })
const copy = computed(() => locale.value.startsWith('zh') ? {
  notice: '安全策略会立即作用于整个平台。修改 IP 白名单前，请确保当前访问地址包含在新规则中。', network: '网络边界', allowlist: 'IP 白名单', allowlistDesc: '仅允许指定 IP 地址或 CIDR 网段访问已认证接口。留空表示不限制。', allowlistPlaceholder: '输入 IP 或 CIDR 后回车', allowlistHint: '支持 IPv4、IPv6 与 CIDR，例如 10.0.0.0/24。', identity: '身份安全', password: '密码策略', passwordDesc: '设置平台账号密码强度与定期轮换要求。', minLength: '最小长度', rotation: '轮换周期（天）', complexity: '要求大小写字母、数字和特殊字符', protection: '登录防护', lockout: '失败登录锁定', lockoutDesc: '达到连续失败次数后临时锁定账号，降低暴力破解风险。', attempts: '最大失败次数', duration: '锁定时长（分钟）', disabledHint: '最大失败次数设为 0 将关闭账号锁定。', save: '保存策略', saved: '安全策略已更新', failed: '安全策略更新失败', loadFailed: '安全配置加载失败',
} : {
  notice: 'Security policies apply platform-wide immediately. Before changing the IP allowlist, ensure your current address remains covered.', network: 'Network boundary', allowlist: 'IP allowlist', allowlistDesc: 'Restrict authenticated API access to exact IP addresses or CIDR ranges. An empty list allows all addresses.', allowlistPlaceholder: 'Enter an IP or CIDR and press Enter', allowlistHint: 'IPv4, IPv6, and CIDR are supported, for example 10.0.0.0/24.', identity: 'Identity security', password: 'Password policy', passwordDesc: 'Set password strength and rotation requirements for platform accounts.', minLength: 'Minimum length', rotation: 'Rotation period (days)', complexity: 'Require upper/lower case, number, and special character', protection: 'Login protection', lockout: 'Failed-login lockout', lockoutDesc: 'Temporarily lock accounts after repeated failures to reduce brute-force risk.', attempts: 'Maximum failed attempts', duration: 'Lockout duration (minutes)', disabledHint: 'Set maximum failed attempts to 0 to disable account lockout.', save: 'Save policy', saved: 'Security policy updated', failed: 'Failed to update security policy', loadFailed: 'Failed to load security configuration',
})

async function load() {
  loading.value = true
  try {
    const [ip, passwordPolicy, lockoutPolicy] = await Promise.all([getIPWhitelist(), getPasswordPolicy(), getLoginLockoutPolicy()])
    whitelist.entries = [...ip.entries]
    Object.assign(password, passwordPolicy)
    Object.assign(lockout, lockoutPolicy)
  } catch (err: any) { MessagePlugin.error(err?.message || copy.value.loadFailed) } finally { loading.value = false }
}
async function save(key: string, action: () => Promise<unknown>) {
  saving.value = key
  try { await action(); MessagePlugin.success(copy.value.saved) } catch (err: any) { MessagePlugin.error(err?.message || copy.value.failed) } finally { saving.value = '' }
}
async function saveWhitelist() { await save('whitelist', async () => { Object.assign(whitelist, await updateIPWhitelist(whitelist.entries)) }) }
async function savePassword() { await save('password', async () => { Object.assign(password, await updatePasswordPolicy({ ...password })) }) }
async function saveLockout() { await save('lockout', async () => { Object.assign(lockout, await updateLoginLockoutPolicy({ ...lockout })) }) }
onMounted(load)
</script>

<style scoped>
.security-page { display: grid; gap: 14px; }
.security-card { display: grid; grid-template-columns: minmax(240px, .8fr) minmax(360px, 1.2fr); gap: 40px; padding: 24px; border: 1px solid var(--td-component-border); border-radius: 12px; background: var(--td-bg-color-container); }
.card-copy h3 { margin: 6px 0 8px; color: var(--td-text-color-primary); }.card-copy p, .hint { color: var(--td-text-color-secondary); line-height: 1.6; }.card-copy p { margin: 0; }
.eyebrow { color: var(--td-brand-color); font-size: 11px; font-weight: 700; letter-spacing: .08em; text-transform: uppercase; }
.card-control { display: grid; align-content: center; gap: 12px; }.card-control > .t-button { justify-self: end; min-width: 112px; }.hint { font-size: 12px; }
.form-control { grid-template-columns: repeat(2, minmax(0, 1fr)); }.form-control .switch-control, .form-control .hint { grid-column: 1 / -1; }.form-control > .t-button { grid-column: 2; }
.switch-control { display: flex; align-items: center; gap: 10px; color: var(--td-text-color-primary); }
@media (max-width: 760px) { .security-card { grid-template-columns: 1fr; gap: 20px; }.form-control { grid-template-columns: 1fr; }.form-control > .t-button { grid-column: 1; }.card-control > .t-button { width: 100%; } }
</style>
