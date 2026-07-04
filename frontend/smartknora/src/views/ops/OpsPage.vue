<template>
  <div class="ops-page">
    <h1>运营管理端</h1>
    <t-loading v-if="loading" />
    <div v-else>
      <!-- 仪表盘 -->
      <t-row :gutter="16" class="stats-row">
        <t-col :span="3"><t-card class="stat-card"><div class="stat-value">{{ stats.tenant_count || 0 }}</div><div class="stat-label">企业数</div></t-card></t-col>
        <t-col :span="3"><t-card class="stat-card"><div class="stat-value">{{ stats.user_count || 0 }}</div><div class="stat-label">用户数</div></t-card></t-col>
        <t-col :span="3"><t-card class="stat-card"><div class="stat-value">{{ stats.document_count || 0 }}</div><div class="stat-label">文档数</div></t-card></t-col>
        <t-col :span="3"><t-card class="stat-card"><div class="stat-value">{{ stats.token_usage_total || 0 }}</div><div class="stat-label">Token总量</div></t-card></t-col>
        <t-col :span="3"><t-card class="stat-card"><div class="stat-value">{{ stats.token_usage_today || 0 }}</div><div class="stat-label">今日Token</div></t-card></t-col>
        <t-col :span="3"><t-card class="stat-card"><div class="stat-value">{{ formatStorage(stats.storage_bytes) }}</div><div class="stat-label">存储用量</div></t-card></t-col>
      </t-row>

      <t-tabs v-model="activeTab" style="margin-top:16px">
        <!-- 企业管理 -->
        <t-tab-panel value="enterprises" label="企业管理">
          <t-card>
            <div style="display:flex;gap:8px;margin-bottom:12px">
              <t-input v-model="entSearch" placeholder="搜索企业名称" @enter="loadEnterprises" clearable style="width:200px" />
              <t-button theme="primary" @click="loadEnterprises">搜索</t-button>
            </div>
            <t-table :data="enterprises" :columns="entColumns" row-key="id" hover stripe size="small" :pagination="{ total: entTotal, current: entPage, pageSize: entPageSize }" @page-change="onEntPageChange">
              <template #auth_status="{ row }">
                <t-tag :theme="row.auth_status === 'active' ? 'success' : row.auth_status === 'suspended' ? 'danger' : 'warning'" size="small">{{ row.auth_status }}</t-tag>
              </template>
              <template #operation="{ row }">
                <t-button size="small" theme="primary" variant="text" @click="toggleEnterprise(row)">{{ row.auth_status === 'suspended' ? '启用' : '暂停' }}</t-button>
              </template>
            </t-table>
          </t-card>
        </t-tab-panel>

        <!-- 用户管理 -->
        <t-tab-panel value="users" label="用户管理">
          <t-card>
            <div style="display:flex;gap:8px;margin-bottom:12px">
              <t-input v-model="userSearch" placeholder="搜索用户名/邮箱/手机号" @enter="loadUsers" clearable style="width:250px" />
              <t-button theme="primary" @click="loadUsers">搜索</t-button>
            </div>
            <t-table :data="users" :columns="userColumns" row-key="id" hover stripe size="small" :pagination="{ total: userTotal, current: userPage, pageSize: userPageSize }" @page-change="onUserPageChange">
              <template #is_active="{ row }">
                <t-tag :theme="row.is_active ? 'success' : 'danger'" size="small">{{ row.is_active ? '正常' : '禁用' }}</t-tag>
              </template>
              <template #is_ops_admin="{ row }">
                <t-tag v-if="row.is_ops_admin" theme="primary" size="small">运营</t-tag>
              </template>
              <template #operation="{ row }">
                <t-button size="small" :theme="row.is_active ? 'danger' : 'success'" variant="text" @click="toggleUser(row)">{{ row.is_active ? '禁用' : '启用' }}</t-button>
              </template>
            </t-table>
          </t-card>
        </t-tab-panel>

        <!-- 敏感词管理 -->
        <t-tab-panel value="filters" label="敏感词管理">
          <t-card>
            <div style="display:flex;gap:8px;margin-bottom:12px">
              <t-input v-model="filterForm.word" placeholder="敏感词" style="width:200px" />
              <t-select v-model="filterForm.category" style="width:150px" placeholder="分类" clearable>
                <t-option value="general" label="通用" />
                <t-option value="political" label="政治敏感" />
                <t-option value="fraud" label="欺诈" />
                <t-option value="brand" label="品牌侵权" />
              </t-select>
              <t-button theme="primary" @click="addWord" :disabled="!filterForm.word">添加</t-button>
            </div>
            <t-table :data="words" :columns="wordColumns" row-key="id" hover stripe size="small" :pagination="{ total: wordTotal, current: wordPage, pageSize: 20 }" @page-change="onWordPageChange">
              <template #category="{ row }">
                <t-tag size="small">{{ row.category }}</t-tag>
              </template>
              <template #operation="{ row }">
                <t-button size="small" theme="danger" variant="text" @click="delWord(row)">删除</t-button>
              </template>
            </t-table>
          </t-card>
        </t-tab-panel>

        <!-- 计费管理 -->
        <t-tab-panel value="billing" label="计费管理">
          <t-card>
            <div style="display:flex;gap:8px;margin-bottom:12px">
              <t-input v-model="planForm.name" placeholder="方案名称" style="width:120px" />
              <t-input v-model="planForm.price" placeholder="价格" type="number" style="width:100px" />
              <t-input v-model="planForm.token_quota" placeholder="Token额度" type="number" style="width:120px" />
              <t-input v-model="planForm.storage_quota" placeholder="存储(MB)" type="number" style="width:120px" />
              <t-button theme="primary" @click="addPlan" :disabled="!planForm.name">添加方案</t-button>
            </div>
            <t-table :data="plans" :columns="planColumns" row-key="id" hover stripe size="small">
              <template #operation="{ row }">
                <t-button size="small" theme="danger" variant="text" @click="delPlan(row)">删除</t-button>
              </template>
            </t-table>
          </t-card>
        </t-tab-panel>

        <!-- 系统配置 -->
        <t-tab-panel value="config" label="系统配置">
          <t-card title="试用期配置">
            <div style="display:flex;gap:16px;align-items:center;margin-bottom:16px">
              <span>试用期天数:</span>
              <t-input v-model="trialForm.trial_days" type="number" style="width:80px" />
              <span>认证后延长:</span>
              <t-input v-model="trialForm.extended_trial_days" type="number" style="width:80px" />
              <span>降级空间限制:</span>
              <t-input v-model="trialForm.downgrade_space_limit" type="number" style="width:80px" />
              <t-button theme="primary" @click="saveTrial">保存</t-button>
            </div>
          </t-card>
          <t-card title="全局配置" style="margin-top:12px">
            <t-table :data="configs" :columns="configColumns" row-key="id" hover stripe size="small">
              <template #operation="{ row }">
                <t-button size="small" theme="primary" variant="text" @click="editConfig(row)">编辑</t-button>
              </template>
            </t-table>
          </t-card>
        </t-tab-panel>

        <!-- 模型管理 -->
        <t-tab-panel value="models" label="模型管理">
          <t-card>
            <div style="display:flex;gap:8px;margin-bottom:12px">
              <t-input v-model="modelForm.name" placeholder="模型名称" style="width:150px" />
              <t-input v-model="modelForm.display_name" placeholder="显示名称" style="width:150px" />
              <t-select v-model="modelForm.type" style="width:120px" placeholder="类型">
                <t-option value="llm" label="LLM" />
                <t-option value="embedding" label="Embedding" />
              </t-select>
              <t-input v-model="modelForm.source" placeholder="来源" style="width:120px" />
              <t-button theme="primary" @click="addModel" :disabled="!modelForm.name">添加</t-button>
            </div>
            <t-table :data="models" :columns="modelColumns" row-key="id" hover stripe size="small">
              <template #is_default="{ row }">
                <t-tag v-if="row.is_default" theme="success" size="small">默认</t-tag>
              </template>
              <template #operation="{ row }">
                <t-button size="small" theme="primary" variant="text" @click="setDefault(row)" :disabled="row.is_default">设为默认</t-button>
                <t-button size="small" theme="danger" variant="text" @click="delModel(row)" :disabled="row.is_builtin">删除</t-button>
              </template>
            </t-table>
          </t-card>
        </t-tab-panel>

        <!-- 审计日志 -->
        <t-tab-panel value="audit" label="审计日志">
          <t-card>
            <div style="display:flex;gap:8px;margin-bottom:12px">
              <t-input v-model="auditSearch.user_id" placeholder="用户ID" style="width:150px" />
              <t-input v-model="auditSearch.action" placeholder="操作类型" style="width:150px" />
              <t-button theme="primary" @click="loadAuditLogs">查询</t-button>
              <t-button theme="default" @click="exportLogs">导出CSV</t-button>
            </div>
            <t-table :data="auditLogs" :columns="auditColumns" row-key="id" hover stripe size="small" :pagination="{ total: auditTotal, current: auditPage, pageSize: 20 }" @page-change="onAuditPageChange">
            </t-table>
          </t-card>
        </t-tab-panel>

        <!-- 公告管理 -->
        <t-tab-panel value="announcements" label="公告管理">
          <t-card>
            <div class="announce-form">
              <t-input v-model="annForm.title" placeholder="公告标题" style="flex:1" />
              <t-button theme="primary" :disabled="!annForm.title || !annForm.content" @click="handleCreateAnn">发布</t-button>
            </div>
            <t-textarea v-model="annForm.content" placeholder="公告内容" :autosize="{ minRows: 2 }" style="margin-top:8px" />
            <t-table v-if="announcements.length > 0" :data="announcements" :columns="annColumns" row-key="id" hover stripe size="small" style="margin-top:12px">
              <template #operation="{ row }">
                <t-button size="small" theme="danger" variant="text" @click="delAnn(row)">删除</t-button>
              </template>
            </t-table>
          </t-card>
        </t-tab-panel>
      </t-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import * as opsApi from '@/api/ops'

const loading = ref(true)
const activeTab = ref('enterprises')
const stats = ref<any>({})

// Enterprises
const enterprises = ref<any[]>([])
const entSearch = ref('')
const entTotal = ref(0)
const entPage = ref(1)
const entPageSize = ref(20)
const entColumns = [
  { colKey: 'name', title: '企业名称', ellipsis: true },
  { colKey: 'member_count', title: '成员数', width: 80 },
  { colKey: 'auth_status', title: '认证状态', width: 100 },
  { colKey: 'subscription_status', title: '订阅', width: 80 },
  { colKey: 'created_at', title: '创建时间', width: 120 },
  { colKey: 'operation', title: '操作', width: 80 },
]

// Users
const users = ref<any[]>([])
const userSearch = ref('')
const userTotal = ref(0)
const userPage = ref(1)
const userPageSize = ref(20)
const userColumns = [
  { colKey: 'username', title: '用户名', width: 120 },
  { colKey: 'email', title: '邮箱', ellipsis: true },
  { colKey: 'phone', title: '手机号', width: 120 },
  { colKey: 'is_active', title: '状态', width: 80 },
  { colKey: 'is_ops_admin', title: '角色', width: 80 },
  { colKey: 'created_at', title: '注册时间', width: 120 },
  { colKey: 'operation', title: '操作', width: 80 },
]

// Filters
const words = ref<any[]>([])
const wordTotal = ref(0)
const wordPage = ref(1)
const filterForm = reactive({ word: '', category: 'general' })
const wordColumns = [
  { colKey: 'word', title: '敏感词', ellipsis: true },
  { colKey: 'category', title: '分类', width: 120 },
  { colKey: 'created_at', title: '添加时间', width: 120 },
  { colKey: 'operation', title: '操作', width: 80 },
]

// Billing
const plans = ref<any[]>([])
const planForm = reactive({ name: '', price: '0', token_quota: '0', storage_quota: '0' })
const planColumns = [
  { colKey: 'name', title: '方案名称', width: 120 },
  { colKey: 'price', title: '价格', width: 80 },
  { colKey: 'token_quota', title: 'Token额度', width: 120 },
  { colKey: 'storage_quota', title: '存储(MB)', width: 100 },
  { colKey: 'status', title: '状态', width: 80 },
  { colKey: 'operation', title: '操作', width: 80 },
]

// Config
const configs = ref<any[]>([])
const trialForm = reactive({ trial_days: '30', extended_trial_days: '90', downgrade_space_limit: '1' })
const configColumns = [
  { colKey: 'key', title: '配置项', width: 200 },
  { colKey: 'value', title: '值', ellipsis: true },
  { colKey: 'description', title: '说明', ellipsis: true },
  { colKey: 'operation', title: '操作', width: 80 },
]

// Models
const models = ref<any[]>([])
const modelForm = reactive({ name: '', display_name: '', type: 'llm', source: '' })
const modelColumns = [
  { colKey: 'name', title: '模型名称', width: 150 },
  { colKey: 'display_name', title: '显示名称', width: 150 },
  { colKey: 'type', title: '类型', width: 100 },
  { colKey: 'source', title: '来源', width: 120 },
  { colKey: 'is_default', title: '默认', width: 80 },
  { colKey: 'status', title: '状态', width: 80 },
  { colKey: 'operation', title: '操作', width: 150 },
]

// Audit
const auditLogs = ref<any[]>([])
const auditTotal = ref(0)
const auditPage = ref(1)
const auditSearch = reactive({ user_id: '', action: '' })
const auditColumns = [
  { colKey: 'created_at', title: '时间', width: 150 },
  { colKey: 'username', title: '用户', width: 100 },
  { colKey: 'action', title: '操作', width: 150 },
  { colKey: 'resource', title: '资源', width: 120 },
  { colKey: 'detail', title: '详情', ellipsis: true },
  { colKey: 'ip', title: 'IP', width: 120 },
]

// Announcements
const announcements = ref<any[]>([])
const annForm = ref({ title: '', content: '' })
const annColumns = [
  { colKey: 'title', title: '标题', ellipsis: true },
  { colKey: 'status', title: '状态', width: 80 },
  { colKey: 'created_at', title: '发布时间', width: 120 },
  { colKey: 'operation', title: '操作', width: 80 },
]

function formatStorage(bytes: number) {
  if (!bytes) return '0'
  if (bytes < 1024) return bytes + 'B'
  if (bytes < 1048576) return (bytes / 1024).toFixed(1) + 'KB'
  if (bytes < 1073741824) return (bytes / 1048576).toFixed(1) + 'MB'
  return (bytes / 1073741824).toFixed(1) + 'GB'
}
function formatDate(t: string) { return t ? new Date(t).toLocaleDateString('zh-CN') : '' }

async function loadAll() {
  loading.value = true
  try {
    const [dashRes] = await Promise.all([opsApi.getOpsDashboard().catch(() => ({ data: {} }))])
    stats.value = (dashRes.data as any) || {}
    await Promise.all([loadEnterprises(), loadUsers(), loadWords(), loadPlans(), loadConfigs(), loadModels(), loadAuditLogs(), loadAnnouncements(), loadTrialConfig()])
  } catch (e) { MessagePlugin.warning('加载失败') }
  finally { loading.value = false }
}

async function loadEnterprises() {
  try {
    const r = await opsApi.listEnterprises({ search: entSearch.value, page: entPage.value, page_size: entPageSize.value })
    enterprises.value = (r.data as any).enterprises || []
    entTotal.value = (r.data as any).total || 0
  } catch {}
}
function onEntPageChange(p: any) { entPage.value = p.current; loadEnterprises() }
async function toggleEnterprise(row: any) {
  try {
    await opsApi.updateEnterpriseStatus(row.id, row.auth_status === 'suspended' ? 'active' : 'suspended')
    MessagePlugin.success('操作成功')
    loadEnterprises()
  } catch (e: any) { MessagePlugin.error(e.response?.data?.error || '操作失败') }
}

async function loadUsers() {
  try {
    const r = await opsApi.listUsers({ search: userSearch.value, page: userPage.value, page_size: userPageSize.value })
    users.value = (r.data as any).users || []
    userTotal.value = (r.data as any).total || 0
  } catch {}
}
function onUserPageChange(p: any) { userPage.value = p.current; loadUsers() }
async function toggleUser(row: any) {
  try {
    await opsApi.updateUserStatus(row.id, !row.is_active)
    MessagePlugin.success('操作成功')
    loadUsers()
  } catch (e: any) { MessagePlugin.error(e.response?.data?.error || '操作失败') }
}

async function loadWords() {
  try {
    const r = await opsApi.listSensitiveWords({ page: wordPage.value })
    words.value = (r.data as any).words || []
    wordTotal.value = (r.data as any).total || 0
  } catch {}
}
function onWordPageChange(p: any) { wordPage.value = p.current; loadWords() }
async function addWord() {
  try {
    await opsApi.createSensitiveWord({ word: filterForm.word, category: filterForm.category })
    MessagePlugin.success('添加成功')
    filterForm.word = ''
    loadWords()
  } catch (e: any) { MessagePlugin.error(e.response?.data?.error || '添加失败') }
}
async function delWord(row: any) {
  try {
    await opsApi.deleteSensitiveWord(row.id)
    MessagePlugin.success('删除成功')
    loadWords()
  } catch (e: any) { MessagePlugin.error(e.response?.data?.error || '删除失败') }
}

async function loadPlans() {
  try {
    const r = await opsApi.listBillingPlans()
    plans.value = (r.data as any).plans || []
  } catch {}
}
async function addPlan() {
  try {
    await opsApi.createBillingPlan({ name: planForm.name, price: parseFloat(planForm.price), token_quota: parseInt(planForm.token_quota), storage_quota: parseInt(planForm.storage_quota) * 1048576 })
    MessagePlugin.success('添加成功')
    planForm.name = ''; planForm.price = '0'; planForm.token_quota = '0'; planForm.storage_quota = '0'
    loadPlans()
  } catch (e: any) { MessagePlugin.error(e.response?.data?.error || '添加失败') }
}
async function delPlan(row: any) {
  try {
    await opsApi.deleteBillingPlan(row.id)
    MessagePlugin.success('删除成功')
    loadPlans()
  } catch (e: any) { MessagePlugin.error(e.response?.data?.error || '删除失败') }
}

async function loadConfigs() {
  try {
    const r = await opsApi.listConfigs()
    configs.value = (r.data as any).configs || []
  } catch {}
}
async function loadTrialConfig() {
  try {
    const r = await opsApi.getTrialConfig()
    const d = r.data as any
    trialForm.trial_days = d.trial_days || '30'
    trialForm.extended_trial_days = d.extended_trial_days || '90'
    trialForm.downgrade_space_limit = d.downgrade_space_limit || '1'
  } catch {}
}
async function saveTrial() {
  try {
    await opsApi.updateTrialConfig({ trial_days: parseInt(trialForm.trial_days), extended_trial_days: parseInt(trialForm.extended_trial_days), downgrade_space_limit: parseInt(trialForm.downgrade_space_limit) })
    MessagePlugin.success('保存成功')
  } catch (e: any) { MessagePlugin.error(e.response?.data?.error || '保存失败') }
}
function editConfig(row: any) {
  const val = prompt('修改配置值：', row.value)
  if (val !== null) {
    opsApi.updateConfig(row.key, val, row.description).then(() => { MessagePlugin.success('更新成功'); loadConfigs() }).catch(() => MessagePlugin.error('更新失败'))
  }
}

async function loadModels() {
  try {
    const r = await opsApi.listModels()
    models.value = (r.data as any).models || []
  } catch {}
}
async function addModel() {
  try {
    await opsApi.createModel({ name: modelForm.name, display_name: modelForm.display_name, type: modelForm.type, source: modelForm.source })
    MessagePlugin.success('添加成功')
    modelForm.name = ''; modelForm.display_name = ''; modelForm.source = ''
    loadModels()
  } catch (e: any) { MessagePlugin.error(e.response?.data?.error || '添加失败') }
}
async function setDefault(row: any) {
  try {
    await opsApi.setDefaultModel(row.id)
    MessagePlugin.success('设置成功')
    loadModels()
  } catch (e: any) { MessagePlugin.error(e.response?.data?.error || '设置失败') }
}
async function delModel(row: any) {
  try {
    await opsApi.deleteModel(row.id)
    MessagePlugin.success('删除成功')
    loadModels()
  } catch (e: any) { MessagePlugin.error(e.response?.data?.error || '删除失败') }
}

async function loadAuditLogs() {
  try {
    const r = await opsApi.getAuditLogs({ user_id: auditSearch.user_id, action: auditSearch.action, page: auditPage.value })
    auditLogs.value = (r.data as any).logs || []
    auditTotal.value = (r.data as any).total || 0
  } catch {}
}
function onAuditPageChange(p: any) { auditPage.value = p.current; loadAuditLogs() }
async function exportLogs() {
  try {
    const r = await opsApi.exportAuditLogs()
    const url = window.URL.createObjectURL(new Blob([r.data as any]))
    const a = document.createElement('a')
    a.href = url; a.download = 'audit_logs.csv'; a.click()
    window.URL.revokeObjectURL(url)
  } catch { MessagePlugin.error('导出失败') }
}

async function loadAnnouncements() {
  try {
    const r = await opsApi.listAnnouncements()
    announcements.value = (r.data as any).announcements || []
  } catch {}
}
async function handleCreateAnn() {
  if (!annForm.value.title) return
  try {
    await opsApi.createAnnouncement(annForm.value)
    MessagePlugin.success('公告已发布')
    annForm.value = { title: '', content: '' }
    loadAnnouncements()
  } catch (e: any) { MessagePlugin.error(e.response?.data?.error || '发布失败') }
}
async function delAnn(row: any) {
  try {
    await opsApi.deleteAnnouncement(row.id)
    MessagePlugin.success('删除成功')
    loadAnnouncements()
  } catch (e: any) { MessagePlugin.error(e.response?.data?.error || '删除失败') }
}

onMounted(loadAll)
</script>

<style scoped>
.ops-page { max-width: 1200px; margin: 0 auto; }
.ops-page h1 { font-size: 24px; font-weight: 600; margin-bottom: 24px; }
.stats-row { margin-bottom: 16px; }
.stat-card { text-align: center; }
.stat-value { font-size: 28px; font-weight: 700; color: #014DB2; }
.stat-label { font-size: 13px; color: #999; margin-top: 4px; }
.announce-form { display: flex; gap: 8px; }
</style>
