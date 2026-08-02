<template>
  <t-card class="users-card">
    <div class="toolbar">
      <div class="search-controls">
        <t-input v-model="userSearch" placeholder="搜索用户名/邮箱/手机号" @enter="searchUsers" clearable class="search-input" />
        <t-select v-model="activeFilter" placeholder="全部状态" clearable class="status-select" @change="searchUsers">
          <t-option value="true" label="正常" />
          <t-option value="false" label="禁用" />
        </t-select>
        <t-button theme="primary" :loading="loading" @click="searchUsers">搜索</t-button>
      </div>
      <div class="toolbar-actions">
        <t-button variant="outline" @click="openImportDialog">批量导入</t-button>
        <t-button theme="primary" @click="openCreateDialog">创建用户</t-button>
      </div>
    </div>
    <t-table :data="users" :columns="userColumns" :loading="loading" row-key="id" hover stripe size="small" :pagination="{ total: userTotal, current: userPage, pageSize: userPageSize }" @page-change="onUserPageChange">
      <template #is_active="{ row }">
        <t-tag :theme="row.is_active ? 'success' : 'danger'" size="small">{{ row.is_active ? '正常' : '禁用' }}</t-tag>
      </template>
      <template #access_role="{ row }">
        <t-tag :theme="isAdminRole(row.access_role) ? 'primary' : 'default'" size="small">{{ roleLabel(row.access_role) }}</t-tag>
      </template>
      <template #created_at="{ row }">
        {{ formatDate(row.created_at) }}
      </template>
      <template #operation="{ row }">
        <t-button size="small" :theme="row.is_active ? 'danger' : 'success'" variant="text" @click="toggleUser(row)">{{ row.is_active ? '禁用' : '启用' }}</t-button>
      </template>
    </t-table>
  </t-card>

  <t-dialog v-model:visible="createVisible" header="创建用户" width="520px" :confirm-btn="{ loading: creating }" @confirm="submitCreate">
    <div class="dialog-intro">创建后将自动生成个人工作区和默认组织，用户首次登录需要修改初始密码。</div>
    <t-form label-align="top">
      <div class="identity-grid">
        <t-form-item label="手机号">
          <t-input v-model="createForm.phone" placeholder="手机号和邮箱至少填写一项" />
        </t-form-item>
        <t-form-item label="邮箱">
          <t-input v-model="createForm.email" placeholder="name@example.com" />
        </t-form-item>
      </div>
      <t-form-item label="昵称">
        <t-input v-model="createForm.nickname" placeholder="选填，默认使用手机号或邮箱" maxlength="100" />
      </t-form-item>
      <t-form-item label="初始密码" help="至少 8 个字符">
        <t-input v-model="createForm.password" type="password" placeholder="请输入初始密码" @enter="submitCreate" />
      </t-form-item>
    </t-form>
  </t-dialog>

  <t-dialog v-model:visible="importVisible" header="批量导入用户" width="680px" :footer="false">
    <div class="import-panel">
      <div class="import-guide">
        <strong>上传 CSV 或 XLSX 文件</strong>
        <span>最多 1000 条、文件不超过 10 MB。必需列：password，以及 phone 或 email；nickname 可选。</span>
      </div>
      <label class="file-picker" :class="{ disabled: importing }">
        <input type="file" accept=".csv,.xlsx" :disabled="importing" @change="selectImportFile" />
        <span class="file-picker-title">{{ importFile ? importFile.name : '选择导入文件' }}</span>
        <span class="file-picker-meta">{{ importFile ? formatFileSize(importFile.size) : '支持中英文表头' }}</span>
      </label>
      <div class="import-actions">
        <t-button variant="outline" :disabled="importing" @click="importVisible = false">取消</t-button>
        <t-button theme="primary" :loading="importing" :disabled="!importFile" @click="submitImport">开始导入</t-button>
      </div>
      <div v-if="importResult" class="import-result">
        <div class="result-metrics">
          <div><strong>{{ importResult.total }}</strong><span>总计</span></div>
          <div class="success"><strong>{{ importResult.imported }}</strong><span>成功</span></div>
          <div :class="{ danger: importResult.failed > 0 }"><strong>{{ importResult.failed }}</strong><span>失败</span></div>
        </div>
        <div v-if="importResult.errors.length" class="error-list">
          <div v-for="(error, index) in importResult.errors" :key="`${error.row}-${error.field}-${index}`" class="error-row">
            <span>第 {{ error.row }} 行</span>
            <span>{{ error.field || '记录' }}</span>
            <span>{{ error.message }}</span>
          </div>
        </div>
      </div>
    </div>
  </t-dialog>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import * as opsApi from '@/api/ops'

interface UserRow {
  id: string
  username: string
  email: string
  phone: string
  nickname: string
  is_active: boolean
  access_role: string
  created_at: string
}

interface ImportResult {
  total: number
  imported: number
  failed: number
  errors: Array<{ row: number; field?: string; value?: string; message: string }>
}

const users = ref<UserRow[]>([])
const userSearch = ref('')
const activeFilter = ref('')
const userTotal = ref(0)
const userPage = ref(1)
const userPageSize = ref(20)
const loading = ref(false)
const createVisible = ref(false)
const creating = ref(false)
const createForm = ref({ phone: '', email: '', nickname: '', password: '' })
const importVisible = ref(false)
const importing = ref(false)
const importFile = ref<File | null>(null)
const importResult = ref<ImportResult | null>(null)
const userColumns = [
  { colKey: 'username', title: '用户名', width: 170, ellipsis: true },
  { colKey: 'nickname', title: '昵称', width: 120, ellipsis: true },
  { colKey: 'email', title: '邮箱', width: 200, ellipsis: true },
  { colKey: 'phone', title: '手机号', width: 140 },
  { colKey: 'is_active', title: '状态', width: 80 },
  { colKey: 'access_role', title: '角色', width: 110 },
  { colKey: 'created_at', title: '注册时间', width: 170 },
  { colKey: 'operation', title: '操作', width: 80 },
]

async function loadUsers() {
  loading.value = true
  try {
    const r = await opsApi.listUsers({ search: userSearch.value, is_active: activeFilter.value || undefined, page: userPage.value, page_size: userPageSize.value })
    users.value = r.data.users || []
    userTotal.value = r.data.total || 0
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '用户列表加载失败')
  } finally {
    loading.value = false
  }
}

function searchUsers() {
  userPage.value = 1
  loadUsers()
}

function onUserPageChange(p: any) {
  userPage.value = p.current
  loadUsers()
}

async function toggleUser(row: any) {
  try {
    await opsApi.updateUserStatus(row.id, !row.is_active)
    MessagePlugin.success('操作成功')
    loadUsers()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '操作失败')
  }
}

function openCreateDialog() {
  createForm.value = { phone: '', email: '', nickname: '', password: '' }
  createVisible.value = true
}

async function submitCreate() {
  const form = createForm.value
  if (!form.phone.trim() && !form.email.trim()) {
    MessagePlugin.warning('手机号和邮箱至少填写一项')
    return
  }
  if (form.password.length < 8) {
    MessagePlugin.warning('初始密码至少 8 个字符')
    return
  }
  creating.value = true
  try {
    await opsApi.createUser({ phone: form.phone.trim(), email: form.email.trim(), nickname: form.nickname.trim(), password: form.password })
    MessagePlugin.success('用户创建成功')
    createVisible.value = false
    userPage.value = 1
    loadUsers()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '用户创建失败')
  } finally {
    creating.value = false
  }
}

function openImportDialog() {
  importFile.value = null
  importResult.value = null
  importVisible.value = true
}

function selectImportFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0] || null
  input.value = ''
  if (!file) return
  if (!/\.(csv|xlsx)$/i.test(file.name)) {
    MessagePlugin.warning('请选择 CSV 或 XLSX 文件')
    return
  }
  if (file.size > 10 * 1024 * 1024) {
    MessagePlugin.warning('文件不能超过 10 MB')
    return
  }
  importFile.value = file
  importResult.value = null
}

async function submitImport() {
  if (!importFile.value) return
  importing.value = true
  try {
    const response = await opsApi.importUsers(importFile.value)
    importResult.value = response.data
    if (response.data.imported > 0) {
      MessagePlugin.success(`成功导入 ${response.data.imported} 个用户`)
      userPage.value = 1
      loadUsers()
    } else {
      MessagePlugin.warning('没有用户导入成功，请检查错误明细')
    }
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '批量导入失败')
  } finally {
    importing.value = false
  }
}

function formatDate(value: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

function isAdminRole(role: string) {
  return role === 'super_admin' || role === 'department_admin'
}

function roleLabel(role: string) {
  if (role === 'super_admin') return '超级管理员'
  if (role === 'department_admin') return '部门管理员'
  return '成员'
}

function formatFileSize(bytes: number) {
  return bytes < 1024 * 1024 ? `${Math.ceil(bytes / 1024)} KB` : `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

onMounted(loadUsers)
</script>

<style scoped>
.users-card {
  min-height: 420px;
}

.toolbar,
.search-controls,
.toolbar-actions,
.import-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.toolbar {
  justify-content: space-between;
  margin-bottom: 16px;
}

.search-input {
  width: 280px;
}

.status-select {
  width: 120px;
}

.dialog-intro,
.import-guide {
  margin-bottom: 18px;
  padding: 13px 15px;
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  background: var(--sdp-brand-soft);
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.identity-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.import-guide {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.import-guide strong {
  color: var(--text-primary);
}

.file-picker {
  display: flex;
  min-height: 120px;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 7px;
  border: 1px dashed var(--brand-primary);
  border-radius: 12px;
  background: var(--surface-elevated);
  cursor: pointer;
}

.file-picker:hover {
  background: var(--sdp-brand-soft);
}

.file-picker.disabled {
  cursor: wait;
  opacity: 0.65;
}

.file-picker input {
  display: none;
}

.file-picker-title {
  color: var(--text-primary);
  font-weight: 600;
}

.file-picker-meta {
  color: var(--text-secondary);
  font-size: 12px;
}

.import-actions {
  justify-content: flex-end;
  margin-top: 16px;
}

.import-result {
  margin-top: 20px;
  padding-top: 18px;
  border-top: 1px solid var(--border-soft);
}

.result-metrics {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.result-metrics > div {
  display: flex;
  align-items: baseline;
  justify-content: center;
  gap: 8px;
  padding: 12px;
  border-radius: 10px;
  background: var(--surface-elevated);
}

.result-metrics strong {
  color: var(--text-primary);
  font-size: 22px;
}

.result-metrics span {
  color: var(--text-secondary);
  font-size: 12px;
}

.result-metrics .success strong {
  color: #2f8f61;
}

.result-metrics .danger strong {
  color: #d54941;
}

.error-list {
  max-height: 220px;
  margin-top: 12px;
  overflow: auto;
  border: 1px solid var(--border-soft);
  border-radius: 10px;
}

.error-row {
  display: grid;
  grid-template-columns: 80px 100px 1fr;
  gap: 10px;
  padding: 9px 12px;
  color: var(--text-secondary);
  font-size: 12px;
}

.error-row + .error-row {
  border-top: 1px solid var(--border-soft);
}

@media (max-width: 760px) {
  .toolbar,
  .search-controls {
    align-items: stretch;
    flex-direction: column;
  }

  .toolbar-actions {
    align-self: stretch;
  }

  .toolbar-actions :deep(.t-button) {
    flex: 1;
  }

  .search-input,
  .status-select {
    width: 100%;
  }

  .identity-grid {
    grid-template-columns: 1fr;
    gap: 0;
  }
}
</style>
