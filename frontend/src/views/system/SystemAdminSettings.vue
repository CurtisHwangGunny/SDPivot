<template>
  <div class="admin-settings">
    <div class="section-header">
      <h2>{{ text.title }}</h2>
      <p>{{ text.description }}</p>
    </div>

    <t-tabs v-model="activeTab" class="settings-tabs">
      <t-tab-panel value="overview" :label="text.overviewTab" />
      <t-tab-panel value="platform" :label="text.systemTab" />
      <t-tab-panel value="security" :label="text.securityTab" />
      <t-tab-panel value="audit" :label="text.auditTab" />
    </t-tabs>

    <div v-if="activeTab === 'overview'" class="tab-content"><AdminOverview /></div>

    <div v-else-if="activeTab === 'platform'" class="tab-content">
      <t-loading :loading="systemLoading">
        <div class="config-grid">
          <section class="config-card config-card--wide">
            <div class="card-heading">
              <div><h3>{{ text.processing }}</h3><p>{{ text.processingDesc }}</p></div>
              <t-button theme="primary" :loading="saving === 'params'" @click="saveParams">{{ text.save }}</t-button>
            </div>
            <div class="form-grid form-grid--four">
              <t-form-item :label="text.chunkSize"><t-input-number v-model="paramsForm.chunk_size" :min="1" :max="65536" /></t-form-item>
              <t-form-item :label="text.threshold"><t-input-number v-model="paramsForm.threshold" :min="0" :max="1" :step="0.05" /></t-form-item>
              <t-form-item :label="text.tokenLimit"><t-input-number v-model="paramsForm.token_limit" :min="0" :max="1000000" /></t-form-item>
              <t-form-item :label="text.concurrency"><t-input-number v-model="paramsForm.concurrency" :min="1" :max="1024" /></t-form-item>
            </div>
          </section>

          <section class="config-card">
            <div class="card-heading">
              <div><h3>{{ text.storage }}</h3><p>{{ text.storageDesc }}</p></div>
              <t-button theme="primary" :loading="saving === 'storage'" @click="saveStorage">{{ text.save }}</t-button>
            </div>
            <t-form label-align="top">
              <div class="form-grid">
                <t-form-item :label="text.provider"><t-select v-model="storageForm.provider" :options="storageProviders" /></t-form-item>
                <t-form-item :label="text.bucket"><t-input v-model="storageForm.bucket" /></t-form-item>
              </div>
              <t-form-item :label="text.endpoint"><t-input v-model="storageForm.endpoint" placeholder="https://storage.example.com" /></t-form-item>
              <div class="form-grid">
                <t-form-item :label="text.region"><t-input v-model="storageForm.region" /></t-form-item>
                <t-form-item :label="text.pathPrefix"><t-input v-model="storageForm.path_prefix" /></t-form-item>
              </div>
              <div class="form-grid">
                <t-form-item :label="text.accessKey" :help="secretHelp(storageConfigured.access)"><t-input v-model="storageSecrets.accessKey" type="password" /></t-form-item>
                <t-form-item :label="text.secretKey" :help="secretHelp(storageConfigured.secret)"><t-input v-model="storageSecrets.secretKey" type="password" /></t-form-item>
              </div>
              <div class="switch-row">
                <label><t-switch v-model="storageForm.use_ssl" />{{ text.useSSL }}</label>
                <label><t-switch v-model="storageForm.force_path_style" />{{ text.pathStyle }}</label>
              </div>
            </t-form>
          </section>

          <section class="config-card">
            <div class="card-heading">
              <div><h3>{{ text.wechat }}</h3><p>{{ text.wechatDesc }}</p></div>
              <t-button theme="primary" :loading="saving === 'wechat'" @click="saveWeChat">{{ text.save }}</t-button>
            </div>
            <t-form label-align="top">
              <div class="switch-row switch-row--top"><label><t-switch v-model="wechatForm.enabled" />{{ text.enabled }}</label></div>
              <t-form-item :label="text.appId"><t-input v-model="wechatForm.app_id" /></t-form-item>
              <t-form-item :label="text.appSecret" :help="secretHelp(wechatConfigured)"><t-input v-model="wechatSecret" type="password" /></t-form-item>
              <t-form-item :label="text.redirectUrl"><t-input v-model="wechatForm.redirect_url" placeholder="https://app.example.com/callback" /></t-form-item>
            </t-form>
          </section>

          <section class="config-card config-card--wide">
            <div class="card-heading">
              <div><h3>{{ text.sms }}</h3><p>{{ text.smsDesc }}</p></div>
              <t-button theme="primary" :loading="saving === 'sms'" @click="saveSMS">{{ text.save }}</t-button>
            </div>
            <t-form label-align="top">
              <div class="form-grid form-grid--four">
                <t-form-item :label="text.provider"><t-select v-model="smsForm.provider" :options="smsProviders" /></t-form-item>
                <t-form-item :label="text.region"><t-input v-model="smsForm.region" /></t-form-item>
                <t-form-item :label="text.signName"><t-input v-model="smsForm.sign_name" /></t-form-item>
                <t-form-item :label="text.timeout"><t-input-number v-model="smsForm.timeout_seconds" :min="1" :max="120" /></t-form-item>
              </div>
              <t-form-item :label="text.endpoint"><t-input v-model="smsForm.endpoint" placeholder="https://sms.example.com" /></t-form-item>
              <div class="form-grid form-grid--four">
                <t-form-item :label="text.templateId"><t-input v-model="smsForm.template_id" /></t-form-item>
                <t-form-item :label="text.appId"><t-input v-model="smsForm.app_id" /></t-form-item>
                <t-form-item :label="text.sender"><t-input v-model="smsForm.sender" /></t-form-item>
                <t-form-item :label="text.customHeaders"><t-input v-model="smsHeaders" placeholder='{"X-API-Version":"1"}' /></t-form-item>
              </div>
              <div class="form-grid">
                <t-form-item :label="text.accessKey" :help="secretHelp(smsConfigured.access)"><t-input v-model="smsSecrets.accessKey" type="password" /></t-form-item>
                <t-form-item :label="text.secretKey" :help="secretHelp(smsConfigured.secret)"><t-input v-model="smsSecrets.secretKey" type="password" /></t-form-item>
              </div>
            </t-form>
          </section>

          <section class="config-card config-card--wide">
            <div class="card-heading">
              <div><h3>{{ text.tags }}</h3><p>{{ text.tagsDesc }}</p></div>
              <div class="card-actions"><t-button variant="outline" @click="addTag">{{ text.addTag }}</t-button><t-button theme="primary" :loading="saving === 'tags'" @click="saveTags">{{ text.save }}</t-button></div>
            </div>
            <div v-if="tags.length === 0" class="empty-inline">{{ text.noTags }}</div>
            <div v-for="(tag, index) in tags" :key="tag.id || index" class="tag-row">
              <t-select v-model="tag.dimension_id" :options="dimensionOptions" :placeholder="text.dimension" />
              <t-input v-model="tag.name" :placeholder="text.tagName" />
              <t-color-picker v-model="tag.color" format="HEX" :enable-alpha="false" />
              <t-input-number v-model="tag.sort_order" :step="1" />
              <t-button theme="danger" variant="text" @click="tags.splice(index, 1)">{{ text.remove }}</t-button>
            </div>
          </section>
        </div>
      </t-loading>
    </div>

    <div v-else-if="activeTab === 'security'" class="tab-content"><AdminSecuritySettings /></div>
    <div v-else class="tab-content"><AdminAuditLog /></div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  getPlatformGlobalParams, getPlatformSMSConfig, getPlatformStorageConfig, getPlatformTagDictionary,
  listTagDimensions,
  getPlatformWeChatLoginConfig,
  updatePlatformGlobalParams, updatePlatformSMSConfig, updatePlatformStorageConfig,
  updatePlatformTagDictionary, updatePlatformWeChatLoginConfig,
  type PlatformGlobalParams, type PlatformSMSConfigInput, type PlatformStorageConfigInput,
  type PlatformTagDictionaryEntry, type PlatformTagDimension, type PlatformWeChatLoginConfigInput,
} from '@/api/system'
import AdminAuditLog from './AdminAuditLog.vue'
import AdminOverview from './AdminOverview.vue'
import AdminSecuritySettings from './AdminSecuritySettings.vue'

const { locale } = useI18n()
const activeTab = ref<'overview' | 'platform' | 'security' | 'audit'>('overview')
const saving = ref('')
const systemLoading = ref(false)
const loadedTabs = reactive({ platform: false })

const en = {
  title: 'Platform administration', description: 'Monitor platform health, manage services and security, and investigate system activity.', overviewTab: 'Overview', systemTab: 'Platform', securityTab: 'Security', auditTab: 'Audit log',
  save: 'Save', reset: 'Reset', resetConfirm: 'Reset this policy to its environment or built-in default?', configured: 'Configured. Leave blank to keep the current secret.', notConfigured: 'Not configured.',
  processing: 'Processing defaults', processingDesc: 'Defaults used by platform ingestion and model workloads.', chunkSize: 'Chunk size', threshold: 'Threshold', tokenLimit: 'Token limit', concurrency: 'Concurrency',
  storage: 'Object storage', storageDesc: 'Platform S3 or MinIO connection and credentials.', provider: 'Provider', endpoint: 'Endpoint', region: 'Region', bucket: 'Bucket', pathPrefix: 'Path prefix', accessKey: 'Access key', secretKey: 'Secret key', useSSL: 'Use SSL', pathStyle: 'Force path-style URLs',
  wechat: 'WeChat login', wechatDesc: 'Configure the platform-wide WeChat OAuth callback.', enabled: 'Enabled', appId: 'App ID', appSecret: 'App secret', redirectUrl: 'Redirect URL',
  sms: 'SMS provider', smsDesc: 'Configure platform verification and notification delivery.', signName: 'Sign name', timeout: 'Timeout (seconds)', templateId: 'Template ID', sender: 'Sender', customHeaders: 'Custom headers (JSON)',
  tags: 'Global tag dictionary', tagsDesc: 'Manage classification values by platform dimension.', addTag: 'Add tag', noTags: 'No global tags configured.', tagName: 'Tag name', dimension: 'Dimension', remove: 'Remove',
  securityNotice: 'Security policy changes affect all users immediately. IP allowlist entries are validated by the backend, but enforcement depends on the deployed server version.',
  action: 'Action', outcome: 'Outcome', actorId: 'Actor user ID', search: 'Search', loadMore: 'Load older entries', end: 'No older entries', saved: 'Configuration saved', loadFailed: 'Failed to load configuration', saveFailed: 'Failed to save configuration', invalidHeaders: 'Custom headers must be a JSON object.', requiredFields: 'Complete all required fields before saving.',
}
const zh = {
  title: '平台管理', description: '监控平台状态，管理服务与安全策略，并审查系统活动。', overviewTab: '概览', systemTab: '平台配置', securityTab: '安全配置', auditTab: '审计日志',
  save: '保存', reset: '恢复默认', resetConfirm: '确定恢复为环境变量或内置默认值？', configured: '已配置，留空将保留当前密钥。', notConfigured: '尚未配置。',
  processing: '全局处理参数', processingDesc: '平台文档处理与模型任务使用的默认参数。', chunkSize: '分块大小', threshold: '阈值', tokenLimit: 'Token 上限', concurrency: '并发数',
  storage: '对象存储', storageDesc: '平台 S3 或 MinIO 连接与凭证。', provider: '服务商', endpoint: '服务地址', region: '区域', bucket: '存储桶', pathPrefix: '路径前缀', accessKey: '访问密钥', secretKey: '密钥', useSSL: '启用 SSL', pathStyle: '强制 Path Style',
  wechat: '微信登录', wechatDesc: '配置平台统一微信 OAuth 回调。', enabled: '启用', appId: 'App ID', appSecret: 'App Secret', redirectUrl: '回调地址',
  sms: '短信服务', smsDesc: '配置平台验证码与通知发送服务。', signName: '签名', timeout: '超时（秒）', templateId: '模板 ID', sender: '发送方', customHeaders: '自定义请求头（JSON）',
  tags: '全局标签字典', tagsDesc: '按平台分类维度管理所有空间使用的标签。', addTag: '添加标签', noTags: '尚未配置全局标签。', tagName: '标签名称', dimension: '分类维度', remove: '删除',
  securityNotice: '安全策略修改会立即影响所有用户。IP 白名单由后端校验，但是否执行访问限制取决于部署的服务端版本。',
  action: '操作类型', outcome: '结果', actorId: '操作者用户 ID', search: '查询', loadMore: '加载更早记录', end: '没有更早记录', saved: '配置已保存', loadFailed: '加载配置失败', saveFailed: '保存配置失败', invalidHeaders: '自定义请求头必须是 JSON 对象。', requiredFields: '请填写所有必填项。',
}
const text = computed(() => locale.value.startsWith('zh') ? zh : en)

const storageProviders = [{ label: 'MinIO', value: 'minio' }, { label: 'Amazon S3', value: 's3' }]
const smsProviders = ['custom', 'aliyun', 'tencent', 'huawei'].map(value => ({ label: value === 'custom' ? 'Custom' : value[0].toUpperCase() + value.slice(1), value }))
const storageForm = reactive<PlatformStorageConfigInput>({ provider: 'minio', endpoint: '', region: '', bucket: '', use_ssl: false, force_path_style: true, path_prefix: '' })
const storageSecrets = reactive({ accessKey: '', secretKey: '' })
const storageConfigured = reactive({ access: false, secret: false })
const smsForm = reactive<PlatformSMSConfigInput>({ provider: 'custom', endpoint: '', region: '', sign_name: '', template_id: '', app_id: '', sender: '', custom_headers: {}, timeout_seconds: 10 })
const smsSecrets = reactive({ accessKey: '', secretKey: '' })
const smsConfigured = reactive({ access: false, secret: false })
const smsHeaders = ref('{}')
const wechatForm = reactive<PlatformWeChatLoginConfigInput>({ enabled: false, app_id: '', redirect_url: '' })
const wechatSecret = ref('')
const wechatConfigured = ref(false)
const paramsForm = reactive<PlatformGlobalParams>({ chunk_size: 512, threshold: 0.5, token_limit: 4096, concurrency: 32 })
const tags = ref<PlatformTagDictionaryEntry[]>([])
const dimensions = ref<PlatformTagDimension[]>([])
const dimensionOptions = computed(() => dimensions.value.map(dimension => ({ label: dimension.name, value: dimension.id })))

function secretHelp(configured: boolean) { return configured ? text.value.configured : text.value.notConfigured }
function messageError(error: any, fallback: string) { MessagePlugin.error(error?.message || fallback) }
function assignFields<T extends object>(target: T, source: Partial<T>) { Object.assign(target, source) }

async function loadSystem() {
  systemLoading.value = true
  try {
    const [storage, sms, wechat, params, dictionary, tagDimensions] = await Promise.all([getPlatformStorageConfig(), getPlatformSMSConfig(), getPlatformWeChatLoginConfig(), getPlatformGlobalParams(), getPlatformTagDictionary(), listTagDimensions()])
    assignFields(storageForm, storage); storageConfigured.access = storage.access_key_id_configured; storageConfigured.secret = storage.secret_access_key_configured
    assignFields(smsForm, sms); smsConfigured.access = sms.access_key_id_configured; smsConfigured.secret = sms.access_key_secret_configured; smsHeaders.value = JSON.stringify(sms.custom_headers || {}, null, 2)
    assignFields(wechatForm, wechat); wechatConfigured.value = wechat.app_secret_configured
    assignFields(paramsForm, params); dimensions.value = tagDimensions || []; tags.value = (dictionary.tags || []).map(tag => ({ ...tag }))
    loadedTabs.platform = true
  } catch (error) { messageError(error, text.value.loadFailed) } finally { systemLoading.value = false }
}

async function withSave(key: string, action: () => Promise<void>) {
  saving.value = key
  try { await action(); MessagePlugin.success(text.value.saved) } catch (error) { messageError(error, text.value.saveFailed) } finally { saving.value = '' }
}
function validURL(value: string) { try { const url = new URL(value); return url.protocol === 'http:' || url.protocol === 'https:' } catch { return false } }
async function saveParams() { await withSave('params', async () => { assignFields(paramsForm, await updatePlatformGlobalParams({ ...paramsForm })) }) }
async function saveStorage() {
  if (!storageForm.endpoint.trim() || !storageForm.bucket.trim() || !validURL(storageForm.endpoint)) return void MessagePlugin.warning(text.value.requiredFields)
  await withSave('storage', async () => {
    const input: PlatformStorageConfigInput = { ...storageForm }
    if (storageSecrets.accessKey.trim()) input.access_key_id = storageSecrets.accessKey.trim()
    if (storageSecrets.secretKey.trim()) input.secret_access_key = storageSecrets.secretKey.trim()
    const result = await updatePlatformStorageConfig(input); assignFields(storageForm, result); storageConfigured.access = result.access_key_id_configured; storageConfigured.secret = result.secret_access_key_configured; storageSecrets.accessKey = ''; storageSecrets.secretKey = ''
  })
}
async function saveSMS() {
  if (!smsForm.endpoint.trim() || !validURL(smsForm.endpoint)) return void MessagePlugin.warning(text.value.requiredFields)
  let headers: Record<string, string>
  try { const parsed = JSON.parse(smsHeaders.value || '{}'); if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') throw new Error(); headers = Object.fromEntries(Object.entries(parsed).map(([key, value]) => [key, String(value)])) } catch { return void MessagePlugin.warning(text.value.invalidHeaders) }
  await withSave('sms', async () => {
    const input: PlatformSMSConfigInput = { ...smsForm, custom_headers: headers }
    if (smsSecrets.accessKey.trim()) input.access_key_id = smsSecrets.accessKey.trim()
    if (smsSecrets.secretKey.trim()) input.access_key_secret = smsSecrets.secretKey.trim()
    const result = await updatePlatformSMSConfig(input); assignFields(smsForm, result); smsConfigured.access = result.access_key_id_configured; smsConfigured.secret = result.access_key_secret_configured; smsSecrets.accessKey = ''; smsSecrets.secretKey = ''
  })
}
async function saveWeChat() {
  if (wechatForm.enabled && (!wechatForm.app_id.trim() || !validURL(wechatForm.redirect_url))) return void MessagePlugin.warning(text.value.requiredFields)
  await withSave('wechat', async () => { const input = { ...wechatForm }; if (wechatSecret.value.trim()) input.app_secret = wechatSecret.value.trim(); const result = await updatePlatformWeChatLoginConfig(input); assignFields(wechatForm, result); wechatConfigured.value = result.app_secret_configured; wechatSecret.value = '' })
}
function addTag() { tags.value.push({ id: crypto.randomUUID(), dimension_id: dimensions.value[0]?.id || '', name: '', color: '#0052D9', sort_order: tags.value.length * 10 }) }
async function saveTags() { if (tags.value.some(tag => !tag.dimension_id || !tag.name.trim())) return void MessagePlugin.warning(text.value.requiredFields); await withSave('tags', async () => { const result = await updatePlatformTagDictionary({ tags: tags.value }); tags.value = result.tags.map(tag => ({ ...tag })) }) }

watch(activeTab, tab => { if (tab === 'platform' && !loadedTabs.platform) void loadSystem() })
</script>

<style scoped>
.admin-settings { display: grid; gap: 18px; }
.section-header h2 { margin: 0; color: var(--td-text-color-primary); }
.section-header p, .card-heading p, .security-row p { margin: 7px 0 0; color: var(--td-text-color-secondary); line-height: 1.6; }
.settings-tabs { border-bottom: 1px solid var(--td-component-stroke); }
.tab-content { min-height: 420px; }
.config-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
.config-card { padding: 20px; border: 1px solid var(--td-component-border); border-radius: 10px; background: var(--td-bg-color-container); }
.config-card--wide { grid-column: 1 / -1; }
.card-heading, .card-actions, .switch-row, .audit-toolbar, .row-actions { display: flex; align-items: center; gap: 10px; }
.card-heading { justify-content: space-between; align-items: flex-start; margin-bottom: 18px; }
.card-heading h3, .security-row h4 { margin: 0; color: var(--td-text-color-primary); }
.form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 16px; }
.form-grid--four { grid-template-columns: repeat(4, minmax(0, 1fr)); }
.switch-row { gap: 24px; }
.switch-row label { display: flex; align-items: center; gap: 8px; color: var(--td-text-color-primary); }
.switch-row--top { margin-bottom: 18px; }
.tag-row { display: grid; grid-template-columns: minmax(160px, .8fr) minmax(180px, 1fr) 150px 120px auto; gap: 12px; align-items: center; padding: 10px 0; border-top: 1px solid var(--td-component-stroke); }
.empty-inline { padding: 24px; text-align: center; color: var(--td-text-color-placeholder); background: var(--td-bg-color-secondarycontainer); border-radius: 8px; }
.security-notice { margin-bottom: 14px; }
.security-row { display: grid; grid-template-columns: minmax(0, 1fr) minmax(280px, 45%); gap: 24px; align-items: center; padding: 18px 0; border-bottom: 1px solid var(--td-component-stroke); }
.security-row:last-child { border-bottom: 0; }
.security-control { display: grid; gap: 10px; justify-items: stretch; }
.row-actions { justify-content: flex-end; }
.audit-toolbar { flex-wrap: wrap; padding: 16px; margin-bottom: 14px; border-radius: 10px; background: var(--td-bg-color-secondarycontainer); }
.audit-toolbar :deep(.t-select), .audit-toolbar :deep(.t-input) { width: 210px; }
.audit-table { overflow: hidden; border: 1px solid var(--td-component-border); border-radius: 10px; }
.details-cell { display: inline-block; max-width: 280px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; }
.load-more { display: flex; justify-content: center; padding: 16px; color: var(--td-text-color-placeholder); }
@media (max-width: 900px) { .config-grid, .form-grid--four { grid-template-columns: 1fr 1fr; } .config-card { grid-column: 1 / -1; } }
@media (max-width: 640px) { .config-grid, .form-grid, .form-grid--four, .security-row, .tag-row { grid-template-columns: 1fr; } .card-heading { flex-direction: column; } .card-heading > .t-button, .card-actions { width: 100%; } .audit-toolbar :deep(.t-select), .audit-toolbar :deep(.t-input), .audit-toolbar > * { width: 100%; } }
</style>
