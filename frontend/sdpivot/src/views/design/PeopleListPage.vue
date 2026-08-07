<template>
  <SdpSidebarLayout mode="admin">
    <div class="sdp-people-list">
      <div class="sdp-people-list__shell">
        <header class="sdp-people-list__header">
          <div><p>People Directory</p><h1>人员管理</h1><span>统一管理成员身份、角色、部门与账户状态。</span></div>
          <div class="sdp-people-list__header-actions">
            <SdpButton variant="secondary" aria-label="批量导入成员" @click="openImportDialog">批量导入</SdpButton>
            <SdpButton aria-label="配置成员角色" @click="openRoleDialog()">角色配置</SdpButton>
          </div>
        </header>

        <p v-if="showInviteNotice" class="sdp-people-list__notice" role="status">
          邀请入口已准备，请在组织配置中生成邀请链接。
          <button type="button" aria-label="关闭邀请提示" @click="showInviteNotice = false">关闭</button>
        </p>

        <section class="sdp-people-list__stats" aria-label="成员统计">
          <article v-for="item in statItems" :key="item.label"><span>{{ item.label }}</span><strong>{{ loading ? '—' : item.value }}</strong><small>{{ item.note }}</small></article>
        </section>

        <section class="sdp-people-list__panel" aria-labelledby="people-table-title">
          <header class="sdp-people-list__panel-head">
            <div><p>Member Registry</p><h2 id="people-table-title">成员列表</h2></div>
            <SdpButton variant="secondary" size="sm" aria-label="刷新成员列表" :loading="loading" @click="loadMembers">刷新数据</SdpButton>
          </header>

          <div class="sdp-people-list__filters" role="search" aria-label="筛选成员">
            <label class="sdp-people-list__search" for="member-search">
              <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="7" /><path d="m16 16 5 5" /></svg>
              <span class="sr-only">搜索成员</span>
              <input id="member-search" v-model="searchQuery" type="search" placeholder="搜索姓名或邮箱" aria-label="搜索姓名或邮箱">
            </label>
            <label for="role-filter"><span class="sr-only">按角色筛选</span><select id="role-filter" v-model="roleFilter" aria-label="按角色筛选"><option value="all">全部角色</option><option value="admin">管理员</option><option value="member">普通成员</option></select></label>
            <label for="department-filter"><span class="sr-only">按部门筛选</span><select id="department-filter" v-model="departmentFilter" aria-label="按部门筛选"><option value="all">全部部门</option><option v-for="department in departments" :key="department" :value="department">{{ department }}</option></select></label>
            <button type="button" aria-label="清空所有筛选条件" @click="clearFilters">清空筛选</button>
          </div>

          <div v-if="selectedIds.length" class="sdp-people-list__batch" role="status" aria-live="polite">
            <strong>已选择 {{ selectedIds.length }} 名成员</strong>
            <select v-model="bulkAction" aria-label="选择批量操作"><option value="">选择批量操作</option><option value="activate">设为启用</option><option value="disable">设为停用</option><option value="member">设为普通成员</option><option value="admin">设为管理员</option></select>
            <SdpButton size="sm" :disabled="!bulkAction" aria-label="应用批量操作" @click="applyBulkAction">应用</SdpButton>
            <button type="button" aria-label="取消全部选择" @click="selectedIds = []">取消选择</button>
          </div>

          <div v-if="loadError" class="sdp-people-list__error" role="alert"><strong>成员列表加载失败</strong><p>{{ loadError }}</p><button type="button" aria-label="重试加载成员列表" @click="loadMembers">重试</button></div>
          <div v-else class="sdp-people-list__table-wrap">
            <table>
              <caption class="sr-only">平台成员列表，可选择成员并执行批量操作</caption>
              <thead><tr><th scope="col"><label class="sdp-people-list__checkbox"><input type="checkbox" :checked="allPageSelected" :indeterminate="somePageSelected" aria-label="选择当前页全部成员" @change="togglePageSelection"></label></th><th scope="col">姓名 / 邮箱</th><th scope="col">角色</th><th scope="col">部门</th><th scope="col">状态</th><th scope="col">操作</th></tr></thead>
              <tbody>
                <tr v-if="loading"><td colspan="6" class="sdp-people-list__empty">正在加载成员数据...</td></tr>
                <tr v-else-if="pageMembers.length === 0"><td colspan="6" class="sdp-people-list__empty">没有符合筛选条件的成员</td></tr>
                <tr v-for="member in pageMembers" v-else :key="memberKey(member)">
                  <td><label class="sdp-people-list__checkbox"><input type="checkbox" :checked="selectedIds.includes(memberKey(member))" :aria-label="`选择成员 ${displayName(member)}`" @change="toggleMember(member)"></label></td>
                  <td><div class="sdp-people-list__identity"><span aria-hidden="true">{{ displayName(member).charAt(0).toUpperCase() }}</span><div><strong>{{ displayName(member) }}</strong><small>{{ member.email || member.user_id }}</small></div></div></td>
                  <td><span class="sdp-people-list__role" :class="{ 'sdp-people-list__role--admin': isAdmin(member) }">{{ roleLabel(member.access_role || member.role) }}</span></td>
                  <td>{{ member.department || '未分配' }}</td>
                  <td><span class="sdp-people-list__status" :class="{ 'sdp-people-list__status--disabled': !isActive(member) }"><i aria-hidden="true" />{{ isActive(member) ? '启用' : '停用' }}</span></td>
                  <td><div class="sdp-people-list__actions"><button type="button" :aria-label="`配置成员 ${displayName(member)} 的角色`" @click="openRoleDialog(member)">配置角色</button><button type="button" :aria-label="`${isActive(member) ? '停用' : '启用'}成员 ${displayName(member)}`" @click="toggleStatus(member)">{{ isActive(member) ? '停用' : '启用' }}</button></div></td>
                </tr>
              </tbody>
            </table>
          </div>

          <footer class="sdp-people-list__pagination">
            <p>共 {{ filteredMembers.length }} 名成员，第 {{ safePage }} / {{ totalPages }} 页</p>
            <label for="page-size"><span class="sr-only">每页显示数量</span><select id="page-size" v-model.number="pageSize" aria-label="每页显示数量"><option :value="8">每页 8 条</option><option :value="12">每页 12 条</option><option :value="20">每页 20 条</option></select></label>
            <div aria-label="分页导航"><button type="button" aria-label="上一页" :disabled="safePage <= 1" @click="page -= 1">上一页</button><button v-for="pageNumber in visiblePages" :key="pageNumber" type="button" :aria-label="`前往第 ${pageNumber} 页`" :aria-current="safePage === pageNumber ? 'page' : undefined" :class="{ 'sdp-people-list__page--active': safePage === pageNumber }" @click="page = pageNumber">{{ pageNumber }}</button><button type="button" aria-label="下一页" :disabled="safePage >= totalPages" @click="page += 1">下一页</button></div>
          </footer>
          <p class="sr-only" aria-live="polite">{{ actionAnnouncement }}</p>
        </section>
      </div>

      <div v-if="importVisible" class="sdp-people-list__modal" role="presentation" @click.self="importVisible = false">
        <section role="dialog" aria-modal="true" aria-labelledby="people-import-title">
          <header><div><p>Batch Provisioning</p><h2 id="people-import-title">批量导入成员</h2></div><button type="button" aria-label="关闭批量导入" @click="importVisible = false">关闭</button></header>
          <p>支持 CSV 或 JSON。字段：username/name/email/phone/password/department_id；新用户默认角色为知识查阅者，并在首次登录时强制改密。</p>
          <label class="sdp-people-list__file"><input type="file" accept=".csv,.json,application/json,text/csv" :disabled="importing" @change="selectImportFile"><strong>{{ importFile?.name || '选择 CSV / JSON 文件' }}</strong><span>最多 1000 条，文件不超过 10 MB</span></label>
          <div v-if="importResult" class="sdp-people-list__import-result"><strong>成功 {{ importResult.imported }} / {{ importResult.total }}</strong><span v-if="importResult.failed">失败 {{ importResult.failed }} 条</span><ul v-if="importResult.errors.length"><li v-for="(item, index) in importResult.errors.slice(0, 8)" :key="index">第 {{ item.row }} 条：{{ item.message }}</li></ul></div>
          <footer><SdpButton variant="secondary" @click="importVisible = false">取消</SdpButton><SdpButton :loading="importing" :disabled="!importFile" @click="submitImport">开始导入</SdpButton></footer>
        </section>
      </div>

      <div v-if="roleVisible" class="sdp-people-list__modal" role="presentation" @click.self="roleVisible = false">
        <section role="dialog" aria-modal="true" aria-labelledby="people-role-title">
          <header><div><p>RBAC Assignment</p><h2 id="people-role-title">角色配置</h2></div><button type="button" aria-label="关闭角色配置" @click="roleVisible = false">关闭</button></header>
          <label>成员<select v-model="roleForm.userId" aria-label="选择成员" @change="syncRoleForm"><option value="">请选择成员</option><option v-for="member in members" :key="memberKey(member)" :value="member.user_id || memberKey(member)">{{ displayName(member) }}</option></select></label>
          <label>角色<select v-model="roleForm.role" aria-label="选择角色"><option v-for="role in roles" :key="role.code" :value="role.code">{{ role.name }}</option></select></label>
          <label v-if="roleForm.role === 'department_admin'">部门 ID<input v-model="roleForm.departmentId" type="text" placeholder="department_id" aria-label="部门 ID"></label>
          <p class="sdp-people-list__role-note">角色调整立即影响下一次签发的访问令牌，并写入审计日志。</p>
          <footer><SdpButton variant="secondary" @click="roleVisible = false">取消</SdpButton><SdpButton :loading="roleSaving" :disabled="!roleForm.userId" @click="saveRole">保存角色</SdpButton></footer>
        </section>
      </div>
    </div>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { listMembers, type AdminMember } from '@/api/spaces'
import { batchImportAdminUsers, listAdminRoles, updateAdminUserRole, type AdminRole, type AdminUserImportResult } from '@/api/admin'
import { SdpButton } from '@/components/design'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'
import { MessagePlugin } from 'tdesign-vue-next'

const members = ref<AdminMember[]>([])
const loading = ref(true)
const loadError = ref('')
const searchQuery = ref('')
const roleFilter = ref('all')
const departmentFilter = ref('all')
const selectedIds = ref<string[]>([])
const bulkAction = ref('')
const page = ref(1)
const pageSize = ref(8)
const showInviteNotice = ref(false)
const actionAnnouncement = ref('')
const roles = ref<AdminRole[]>([])
const importVisible = ref(false)
const importing = ref(false)
const importFile = ref<File | null>(null)
const importResult = ref<AdminUserImportResult | null>(null)
const roleVisible = ref(false)
const roleSaving = ref(false)
const roleForm = ref<{ userId: string; role: AdminRole['code']; departmentId: string }>({ userId: '', role: 'knowledge_viewer', departmentId: '' })

const departments = computed(() => [...new Set(members.value.map(member => member.department).filter((value): value is string => Boolean(value)))].sort())
const filteredMembers = computed(() => {
  const query = searchQuery.value.trim().toLocaleLowerCase()
  return members.value.filter(member => {
    const matchesQuery = !query || `${displayName(member)} ${member.email || ''} ${member.user_id}`.toLocaleLowerCase().includes(query)
    const matchesRole = roleFilter.value === 'all' || (roleFilter.value === 'admin' ? isAdmin(member) : !isAdmin(member))
    const matchesDepartment = departmentFilter.value === 'all' || member.department === departmentFilter.value
    return matchesQuery && matchesRole && matchesDepartment
  })
})
const totalPages = computed(() => Math.max(1, Math.ceil(filteredMembers.value.length / pageSize.value)))
const safePage = computed(() => Math.min(page.value, totalPages.value))
const pageMembers = computed(() => filteredMembers.value.slice((safePage.value - 1) * pageSize.value, safePage.value * pageSize.value))
const currentPageIds = computed(() => pageMembers.value.map(memberKey))
const allPageSelected = computed(() => currentPageIds.value.length > 0 && currentPageIds.value.every(id => selectedIds.value.includes(id)))
const somePageSelected = computed(() => !allPageSelected.value && currentPageIds.value.some(id => selectedIds.value.includes(id)))
const visiblePages = computed(() => Array.from({ length: totalPages.value }, (_, index) => index + 1).filter(value => Math.abs(value - safePage.value) <= 2))
const statItems = computed(() => [
  { label: '成员总数', value: members.value.length, note: '组织内全部账户' },
  { label: '活跃成员', value: members.value.filter(isActive).length, note: '当前可正常访问' },
  { label: '管理员', value: members.value.filter(isAdmin).length, note: '具备管理权限' },
])

async function loadMembers() {
  loading.value = true
  loadError.value = ''
  try {
    const response = await listMembers()
    members.value = response.data.members || []
  } catch (error: unknown) {
    const value = error as { message?: string; response?: { data?: { error?: string } } }
    loadError.value = value?.response?.data?.error || value?.message || '请检查网络连接与管理员权限。'
  } finally { loading.value = false }
}

function memberKey(member: AdminMember) { return String(member.id || member.user_id) }
function displayName(member: AdminMember) { return member.name || member.nickname || member.username || member.email || `成员 ${member.user_id.slice(0, 8)}` }
function isAdmin(member: AdminMember) { return ['admin', 'owner', 'super_admin', 'department_admin'].includes(String(member.access_role || member.role || '').toLocaleLowerCase()) }
function isActive(member: AdminMember) { return !['disabled', 'inactive', 'blocked'].includes(String(member.status || 'active').toLocaleLowerCase()) }
function roleLabel(role: string) { return roles.value.find(item => item.code === role)?.name || ({ admin: '部门管理员', owner: '超级管理员', member: '知识查阅者' } as Record<string, string>)[role] || role || '知识查阅者' }
function toggleMember(member: AdminMember) { const id = memberKey(member); selectedIds.value = selectedIds.value.includes(id) ? selectedIds.value.filter(value => value !== id) : [...selectedIds.value, id] }
function togglePageSelection() { selectedIds.value = allPageSelected.value ? selectedIds.value.filter(id => !currentPageIds.value.includes(id)) : [...new Set([...selectedIds.value, ...currentPageIds.value])] }
function clearFilters() { searchQuery.value = ''; roleFilter.value = 'all'; departmentFilter.value = 'all' }
function announceAction(message: string) { actionAnnouncement.value = ''; requestAnimationFrame(() => { actionAnnouncement.value = message }) }
function toggleStatus(member: AdminMember) { member.status = isActive(member) ? 'disabled' : 'active'; announceAction(`${displayName(member)} 已${isActive(member) ? '启用' : '停用'}`) }
function applyBulkAction() {
  if (!bulkAction.value) return
  members.value.forEach(member => {
    if (!selectedIds.value.includes(memberKey(member))) return
    if (bulkAction.value === 'activate') member.status = 'active'
    if (bulkAction.value === 'disable') member.status = 'disabled'
    if (bulkAction.value === 'admin') member.role = 'admin'
    if (bulkAction.value === 'member') member.role = 'member'
  })
  announceAction(`已对 ${selectedIds.value.length} 名成员应用批量操作`)
  selectedIds.value = []
  bulkAction.value = ''
}

function openImportDialog() { importFile.value = null; importResult.value = null; importVisible.value = true }
function selectImportFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0] || null
  input.value = ''
  if (!file) return
  if (!/\.(csv|json)$/i.test(file.name) || file.size > 10 * 1024 * 1024) { MessagePlugin.warning('请选择不超过 10 MB 的 CSV 或 JSON 文件'); return }
  importFile.value = file
}
async function fileAsBase64(file: File) { return new Promise<string>((resolve, reject) => { const reader = new FileReader(); reader.onload = () => resolve(String(reader.result).split(',')[1] || ''); reader.onerror = () => reject(reader.error); reader.readAsDataURL(file) }) }
async function submitImport() {
  if (!importFile.value) return
  importing.value = true
  try {
    const payload = /\.json$/i.test(importFile.value.name) ? JSON.parse(await importFile.value.text()) : { csv_base64: await fileAsBase64(importFile.value) }
    const response = await batchImportAdminUsers(payload)
    importResult.value = response.data
    MessagePlugin.success(`成功导入 ${response.data.imported} 名成员`)
    await loadMembers()
  } catch (error: unknown) { const value = error as { response?: { data?: { error?: string } } }; MessagePlugin.error(value.response?.data?.error || '批量导入失败') } finally { importing.value = false }
}
function openRoleDialog(member?: AdminMember) {
  roleForm.value = { userId: member ? (member.user_id || memberKey(member)) : '', role: (member?.access_role as AdminRole['code']) || 'knowledge_viewer', departmentId: member?.department_id || '' }
  roleVisible.value = true
}
function syncRoleForm() {
  const member = members.value.find(item => (item.user_id || memberKey(item)) === roleForm.value.userId)
  if (member) roleForm.value = { userId: roleForm.value.userId, role: (member.access_role as AdminRole['code']) || 'knowledge_viewer', departmentId: member.department_id || '' }
}
async function saveRole() {
  roleSaving.value = true
  try { await updateAdminUserRole(roleForm.value.userId, roleForm.value.role, roleForm.value.departmentId); MessagePlugin.success('角色配置已保存'); roleVisible.value = false; await loadMembers() }
  catch (error: unknown) { const value = error as { response?: { data?: { error?: string } } }; MessagePlugin.error(value.response?.data?.error || '角色配置失败') } finally { roleSaving.value = false }
}

async function loadRoles() { try { roles.value = (await listAdminRoles()).data.roles || [] } catch { roles.value = [] } }

watch([searchQuery, roleFilter, departmentFilter, pageSize], () => { page.value = 1 })
watch(totalPages, value => { if (page.value > value) page.value = value })
onMounted(() => { loadMembers(); loadRoles() })
</script>

<style scoped>
.sdp-people-list { min-height: 100dvh; color: var(--ink-900); background: var(--ink-100); font-family: var(--font-body); }
.sdp-people-list__shell { display: grid; gap: var(--space-6); width: min(100%, var(--content-max-width)); margin-inline: auto; padding: var(--space-8); }
.sdp-people-list__header, .sdp-people-list__header-actions, .sdp-people-list__panel-head, .sdp-people-list__filters, .sdp-people-list__batch, .sdp-people-list__pagination, .sdp-people-list__identity, .sdp-people-list__actions { display: flex; align-items: center; }
.sdp-people-list__header-actions { gap: var(--space-3); }
.sdp-people-list__header { justify-content: space-between; gap: var(--space-6); }
.sdp-people-list__header p, .sdp-people-list__panel-head p { color: var(--brand-700); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); letter-spacing: .08em; text-transform: uppercase; }
.sdp-people-list__header h1 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-4xl); line-height: var(--leading-tight); }
.sdp-people-list__header > div > span { display: block; margin-top: var(--space-2); color: var(--ink-600); font-size: var(--text-base); }
.sdp-people-list__header svg { width: var(--space-4); height: var(--space-4); margin-right: var(--space-2); fill: none; stroke: currentColor; stroke-linecap: round; stroke-linejoin: round; stroke-width: 2; }
.sdp-people-list__notice { display: flex; justify-content: space-between; gap: var(--space-4); padding: var(--space-3) var(--space-4); border: 1px solid var(--brand-300); border-radius: var(--radius-sm); color: var(--brand-900); background: var(--brand-50); font-size: var(--text-sm); }
.sdp-people-list__notice button { border: 0; color: var(--brand-800); background: transparent; font: inherit; font-weight: var(--font-weight-semibold); cursor: pointer; }
.sdp-people-list__stats { display: grid; grid-template-columns: repeat(3, minmax(var(--space-0), 1fr)); gap: var(--space-4); }
.sdp-people-list__stats article { padding: var(--space-5); border: 1px solid var(--ink-200); border-radius: var(--radius-md); background: var(--ink-50); box-shadow: var(--shadow-xs); }
.sdp-people-list__stats span, .sdp-people-list__stats small { display: block; color: var(--ink-600); font-size: var(--text-xs); }
.sdp-people-list__stats strong { display: block; margin: var(--space-2) var(--space-0); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-3xl); line-height: var(--leading-tight); }
.sdp-people-list__stats small { color: var(--ink-500); }
.sdp-people-list__panel { min-width: var(--space-0); overflow: hidden; border: 1px solid var(--ink-200); border-radius: var(--radius-lg); background: var(--ink-50); box-shadow: var(--shadow-xs); }
.sdp-people-list__panel-head { justify-content: space-between; gap: var(--space-4); padding: var(--space-6); border-bottom: 1px solid var(--ink-200); }
.sdp-people-list__panel-head h2 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-xl); }
.sdp-people-list__filters { flex-wrap: wrap; gap: var(--space-3); padding: var(--space-4) var(--space-6); border-bottom: 1px solid var(--ink-200); background: var(--ink-100); }
.sdp-people-list__search { display: flex; min-width: calc(var(--space-24) * 2); flex: 1; align-items: center; gap: var(--space-2); }
.sdp-people-list__search svg { width: var(--space-5); height: var(--space-5); flex: 0 0 var(--space-5); fill: none; stroke: var(--ink-600); stroke-linecap: round; stroke-width: 2; }
.sdp-people-list__filters input, .sdp-people-list__filters select, .sdp-people-list__batch select, .sdp-people-list__pagination select { min-height: var(--space-10); padding: var(--space-2) var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-900); background: var(--ink-50); font-family: var(--font-body); font-size: var(--text-sm); }
.sdp-people-list__search input { width: 100%; }
.sdp-people-list__filters > button, .sdp-people-list__batch > button, .sdp-people-list__actions button, .sdp-people-list__pagination button, .sdp-people-list__error button { min-height: var(--space-8); padding: var(--space-2) var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-800); background: var(--ink-50); font-family: var(--font-body); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); cursor: pointer; }
.sdp-people-list__batch { flex-wrap: wrap; gap: var(--space-3); padding: var(--space-3) var(--space-6); border-bottom: 1px solid var(--brand-200); color: var(--brand-900); background: var(--brand-50); }
.sdp-people-list__batch strong { margin-right: auto; font-size: var(--text-sm); }
.sdp-people-list__table-wrap { overflow-x: auto; }
.sdp-people-list table { width: 100%; border-collapse: collapse; text-align: left; }
.sdp-people-list th, .sdp-people-list td { padding: var(--space-4) var(--space-5); border-bottom: 1px solid var(--ink-200); color: var(--ink-700); font-size: var(--text-sm); white-space: nowrap; }
.sdp-people-list th { color: var(--ink-600); background: var(--ink-100); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); letter-spacing: .04em; }
.sdp-people-list tbody tr:hover { background: var(--brand-50); }
.sdp-people-list__checkbox { width: var(--space-8); height: var(--space-8); display: inline-grid; place-items: center; cursor: pointer; }
.sdp-people-list input[type='checkbox'] { width: var(--space-8); height: var(--space-8); margin: var(--space-0); accent-color: var(--brand-700); cursor: pointer; }
.sdp-people-list__identity { gap: var(--space-3); }
.sdp-people-list__identity > span { width: var(--space-10); height: var(--space-10); display: inline-flex; flex: 0 0 var(--space-10); align-items: center; justify-content: center; border-radius: var(--radius-pill); color: var(--brand-900); background: var(--brand-100); font-weight: var(--font-weight-bold); }
.sdp-people-list__identity strong, .sdp-people-list__identity small { display: block; max-width: calc(var(--space-24) * 2); overflow: hidden; text-overflow: ellipsis; }
.sdp-people-list__identity strong { color: var(--ink-950); }
.sdp-people-list__identity small { margin-top: var(--space-1); color: var(--ink-500); font-size: var(--text-xs); }
.sdp-people-list__role, .sdp-people-list__status { display: inline-flex; align-items: center; gap: var(--space-2); padding: var(--space-1) var(--space-3); border-radius: var(--radius-pill); color: var(--ink-800); background: var(--ink-200); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-people-list__role--admin { color: var(--brand-900); background: var(--brand-100); }
.sdp-people-list__status { color: var(--brand-900); background: var(--brand-50); }
.sdp-people-list__status i { width: var(--space-2); height: var(--space-2); border-radius: var(--radius-pill); background: var(--brand-700); }
.sdp-people-list__status--disabled { color: var(--ink-700); background: var(--ink-200); }
.sdp-people-list__status--disabled i { background: var(--ink-600); }
.sdp-people-list__actions { gap: var(--space-2); }
.sdp-people-list__actions button { border-color: transparent; color: var(--brand-800); background: transparent; }
.sdp-people-list__empty { height: var(--space-24); color: var(--ink-500); text-align: center; }
.sdp-people-list__error { margin: var(--space-6); padding: var(--space-6); border: 1px solid var(--ink-300); border-radius: var(--radius-md); background: var(--ink-100); text-align: center; }
.sdp-people-list__error p { margin: var(--space-2) var(--space-0) var(--space-4); color: var(--ink-600); }
.sdp-people-list__pagination { flex-wrap: wrap; justify-content: flex-end; gap: var(--space-3); padding: var(--space-4) var(--space-6); }
.sdp-people-list__pagination p { margin-right: auto; color: var(--ink-600); font-size: var(--text-xs); }
.sdp-people-list__pagination div { display: flex; gap: var(--space-1); }
.sdp-people-list__pagination button:disabled { cursor: not-allowed; opacity: .5; }
.sdp-people-list__pagination .sdp-people-list__page--active { border-color: var(--brand-600); color: var(--ink-50); background: var(--brand-700); }
.sdp-people-list button:focus-visible, .sdp-people-list input:focus-visible, .sdp-people-list select:focus-visible { outline: 2px solid var(--brand-500); outline-offset: 2px; }
.sdp-people-list__modal { position: fixed; z-index: 80; inset: 0; display: grid; place-items: center; padding: var(--space-5); background: color-mix(in srgb, var(--ink-950) 58%, transparent); }
.sdp-people-list__modal > section { width: min(100%, 38rem); max-height: 90dvh; overflow-y: auto; padding: var(--space-6); border: 1px solid var(--ink-300); border-radius: var(--radius-lg); background: var(--ink-50); box-shadow: var(--shadow-lg); }
.sdp-people-list__modal header, .sdp-people-list__modal footer { display: flex; align-items: center; justify-content: space-between; gap: var(--space-3); }
.sdp-people-list__modal header { margin-bottom: var(--space-5); }
.sdp-people-list__modal header p { color: var(--brand-700); font: var(--font-weight-semibold) var(--text-xs) var(--font-mono); letter-spacing: .08em; text-transform: uppercase; }
.sdp-people-list__modal header button { border: 0; color: var(--ink-600); background: transparent; cursor: pointer; }
.sdp-people-list__modal > section > p, .sdp-people-list__role-note { margin-bottom: var(--space-4); color: var(--ink-600); font-size: var(--text-sm); line-height: var(--leading-relaxed); }
.sdp-people-list__modal label:not(.sdp-people-list__file) { display: grid; gap: var(--space-2); margin-top: var(--space-4); color: var(--ink-700); font-size: var(--text-sm); font-weight: var(--font-weight-semibold); }
.sdp-people-list__modal select, .sdp-people-list__modal input { min-height: var(--space-10); padding: var(--space-2) var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); background: var(--ink-50); }
.sdp-people-list__file { display: grid; gap: var(--space-2); padding: var(--space-8); border: 1px dashed var(--brand-500); border-radius: var(--radius-md); color: var(--ink-700); background: var(--brand-50); text-align: center; cursor: pointer; }
.sdp-people-list__file input { position: absolute; opacity: 0; pointer-events: none; }
.sdp-people-list__file span { color: var(--ink-500); font-size: var(--text-xs); }
.sdp-people-list__modal footer { justify-content: flex-end; margin-top: var(--space-6); }
.sdp-people-list__import-result { display: grid; gap: var(--space-2); margin-top: var(--space-4); padding: var(--space-4); border-radius: var(--radius-sm); background: var(--ink-100); font-size: var(--text-sm); }
.sdp-people-list__import-result ul { padding-left: var(--space-5); color: var(--ink-600); }
@media (max-width: 48rem) { .sdp-people-list__shell { padding: var(--space-6); } .sdp-people-list__stats { grid-template-columns: 1fr; } .sdp-people-list__filters { align-items: stretch; flex-direction: column; } .sdp-people-list__search { min-width: var(--space-0); } .sdp-people-list__filters label, .sdp-people-list__filters input, .sdp-people-list__filters select, .sdp-people-list__filters > button { width: 100%; } }
@media (max-width: 40rem) { .sdp-people-list__shell { padding: var(--space-4); } .sdp-people-list__header { align-items: flex-start; flex-direction: column; } .sdp-people-list__header h1 { font-size: var(--text-3xl); } .sdp-people-list__header :deep(.sdp-button) { width: 100%; } .sdp-people-list__panel-head, .sdp-people-list__filters, .sdp-people-list__batch, .sdp-people-list__pagination { padding-inline: var(--space-4); } .sdp-people-list__pagination { align-items: stretch; flex-direction: column; } .sdp-people-list__pagination p { margin-right: var(--space-0); } }
</style>
