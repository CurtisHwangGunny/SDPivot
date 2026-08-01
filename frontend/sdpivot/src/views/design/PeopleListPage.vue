<template>
  <SdpSidebarLayout mode="admin">
    <div class="sdp-people-list">
      <div class="sdp-people-list__shell">
        <header class="sdp-people-list__header">
          <div><p>People Directory</p><h1>人员管理</h1><span>统一管理成员身份、角色、部门与账户状态。</span></div>
          <SdpButton aria-label="邀请新成员" @click="showInviteNotice = true">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M15 19v-1.5a4.5 4.5 0 0 0-4.5-4.5h-5A4.5 4.5 0 0 0 1 17.5V19m7-10a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm11-1v6m-3-3h6" /></svg>
            邀请成员
          </SdpButton>
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
              <thead><tr><th scope="col"><input type="checkbox" :checked="allPageSelected" :indeterminate="somePageSelected" aria-label="选择当前页全部成员" @change="togglePageSelection"></th><th scope="col">姓名 / 邮箱</th><th scope="col">角色</th><th scope="col">部门</th><th scope="col">状态</th><th scope="col">操作</th></tr></thead>
              <tbody>
                <tr v-if="loading"><td colspan="6" class="sdp-people-list__empty">正在加载成员数据...</td></tr>
                <tr v-else-if="pageMembers.length === 0"><td colspan="6" class="sdp-people-list__empty">没有符合筛选条件的成员</td></tr>
                <tr v-for="member in pageMembers" v-else :key="memberKey(member)">
                  <td><input type="checkbox" :checked="selectedIds.includes(memberKey(member))" :aria-label="`选择成员 ${displayName(member)}`" @change="toggleMember(member)"></td>
                  <td><div class="sdp-people-list__identity"><span aria-hidden="true">{{ displayName(member).charAt(0).toUpperCase() }}</span><div><strong>{{ displayName(member) }}</strong><small>{{ member.email || member.user_id }}</small></div></div></td>
                  <td><span class="sdp-people-list__role" :class="{ 'sdp-people-list__role--admin': isAdmin(member) }">{{ roleLabel(member.role) }}</span></td>
                  <td>{{ member.department || '未分配' }}</td>
                  <td><span class="sdp-people-list__status" :class="{ 'sdp-people-list__status--disabled': !isActive(member) }"><i aria-hidden="true" />{{ isActive(member) ? '启用' : '停用' }}</span></td>
                  <td><div class="sdp-people-list__actions"><button type="button" :aria-label="`编辑成员 ${displayName(member)}`" @click="announceAction(`已选择编辑 ${displayName(member)}`)">编辑</button><button type="button" :aria-label="`${isActive(member) ? '停用' : '启用'}成员 ${displayName(member)}`" @click="toggleStatus(member)">{{ isActive(member) ? '停用' : '启用' }}</button></div></td>
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
    </div>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { listMembers, type AdminMember } from '@/api/spaces'
import { SdpButton } from '@/components/design'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'

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
function isAdmin(member: AdminMember) { return ['admin', 'owner', 'super_admin'].includes(String(member.role || '').toLocaleLowerCase()) }
function isActive(member: AdminMember) { return !['disabled', 'inactive', 'blocked'].includes(String(member.status || 'active').toLocaleLowerCase()) }
function roleLabel(role: string) { return ['admin', 'owner', 'super_admin'].includes(String(role).toLocaleLowerCase()) ? '管理员' : '普通成员' }
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

watch([searchQuery, roleFilter, departmentFilter, pageSize], () => { page.value = 1 })
watch(totalPages, value => { if (page.value > value) page.value = value })
onMounted(loadMembers)
</script>

<style scoped>
.sdp-people-list { min-height: 100dvh; color: var(--ink-900); background: var(--ink-100); font-family: var(--font-body); }
.sdp-people-list__shell { display: grid; gap: var(--space-6); width: min(100%, var(--content-max-width)); margin-inline: auto; padding: var(--space-8); }
.sdp-people-list__header, .sdp-people-list__panel-head, .sdp-people-list__filters, .sdp-people-list__batch, .sdp-people-list__pagination, .sdp-people-list__identity, .sdp-people-list__actions { display: flex; align-items: center; }
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
.sdp-people-list input[type='checkbox'] { width: var(--space-4); height: var(--space-4); accent-color: var(--brand-700); }
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
@media (max-width: 48rem) { .sdp-people-list__shell { padding: var(--space-6); } .sdp-people-list__stats { grid-template-columns: 1fr; } .sdp-people-list__filters { align-items: stretch; flex-direction: column; } .sdp-people-list__search { min-width: var(--space-0); } .sdp-people-list__filters label, .sdp-people-list__filters input, .sdp-people-list__filters select, .sdp-people-list__filters > button { width: 100%; } }
@media (max-width: 40rem) { .sdp-people-list__shell { padding: var(--space-4); } .sdp-people-list__header { align-items: flex-start; flex-direction: column; } .sdp-people-list__header h1 { font-size: var(--text-3xl); } .sdp-people-list__header :deep(.sdp-button) { width: 100%; } .sdp-people-list__panel-head, .sdp-people-list__filters, .sdp-people-list__batch, .sdp-people-list__pagination { padding-inline: var(--space-4); } .sdp-people-list__pagination { align-items: stretch; flex-direction: column; } .sdp-people-list__pagination p { margin-right: var(--space-0); } }
</style>
