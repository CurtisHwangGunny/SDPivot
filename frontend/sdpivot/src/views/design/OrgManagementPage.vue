<template>
  <SdpSidebarLayout mode="admin">
    <div class="sdp-org-management">
      <div class="sdp-org-management__shell">
        <header class="sdp-org-management__header">
          <div>
            <p>Organization Directory</p>
            <h1>组织管理</h1>
            <span>维护组织资料、成员归属与协作边界。</span>
          </div>
          <SdpButton aria-label="创建组织" @click="openCreateDialog">创建组织</SdpButton>
        </header>

        <p v-if="message" class="sdp-org-management__message" :class="{ 'sdp-org-management__message--error': messageType === 'error' }" role="status" aria-live="polite">{{ message }}</p>

        <section class="sdp-org-management__stats" aria-label="组织统计">
          <article v-for="stat in stats" :key="stat.label">
            <span>{{ stat.label }}</span>
            <strong>{{ loading ? '—' : stat.value }}</strong>
            <small>{{ stat.note }}</small>
          </article>
        </section>

        <section class="sdp-org-management__workspace" aria-label="组织与成员管理">
          <aside class="sdp-org-management__panel" aria-labelledby="org-list-title">
            <header class="sdp-org-management__panel-head">
              <div><p>Organizations</p><h2 id="org-list-title">组织列表</h2></div>
              <SdpButton variant="secondary" size="sm" aria-label="刷新组织列表" :loading="loading" @click="loadOrganizations">刷新</SdpButton>
            </header>

            <div v-if="loadError" class="sdp-org-management__state" role="alert">
              <strong>组织列表加载失败</strong>
              <span>{{ loadError }}</span>
              <button type="button" @click="loadOrganizations">重试</button>
            </div>
            <div v-else-if="loading" class="sdp-org-management__state" role="status">正在加载组织数据...</div>
            <div v-else-if="organizations.length === 0" class="sdp-org-management__state">
              <strong>尚未创建组织</strong>
              <span>创建第一个组织以开始管理成员。</span>
            </div>
            <ul v-else class="sdp-org-management__org-list">
              <li v-for="org in organizations" :key="org.id">
                <button type="button" :class="{ 'sdp-org-management__org-button--active': selectedOrgId === org.id }" :aria-current="selectedOrgId === org.id ? 'true' : undefined" @click="selectOrganization(org.id)">
                  <span class="sdp-org-management__org-mark" aria-hidden="true">{{ org.name.charAt(0).toUpperCase() }}</span>
                  <span><strong>{{ org.name }}</strong><small>{{ memberCounts[org.id] ?? org.member_count ?? 0 }} 名成员</small></span>
                </button>
              </li>
            </ul>
          </aside>

          <section class="sdp-org-management__panel sdp-org-management__members" aria-labelledby="member-list-title">
            <header class="sdp-org-management__panel-head">
              <div><p>Member Registry</p><h2 id="member-list-title">{{ selectedOrg?.name || '组织成员' }}</h2><span v-if="selectedOrg">{{ selectedOrg.description || '暂无组织说明' }}</span></div>
              <div v-if="selectedOrg" class="sdp-org-management__panel-actions">
                <SdpButton variant="secondary" size="sm" :aria-label="`编辑组织 ${selectedOrg.name}`" @click="openEditDialog">编辑</SdpButton>
                <SdpButton v-if="canDeleteSelectedOrg" variant="danger" size="sm" :aria-label="`删除组织 ${selectedOrg.name}`" @click="deleteDialogOpen = true">删除</SdpButton>
              </div>
            </header>

            <div v-if="!selectedOrg && !loading" class="sdp-org-management__state">请选择或创建一个组织。</div>
            <div v-else-if="membersLoading" class="sdp-org-management__state" role="status">正在加载成员...</div>
            <div v-else-if="membersError" class="sdp-org-management__state" role="alert">
              <strong>成员列表加载失败</strong>
              <span>{{ membersError }}</span>
              <button type="button" @click="loadOrganizationMembers">重试</button>
            </div>
            <div v-else class="sdp-org-management__table-wrap">
              <table>
                <caption class="sr-only">{{ selectedOrg?.name }} 的组织成员列表</caption>
                <thead><tr><th scope="col">成员</th><th scope="col">角色</th><th scope="col">状态</th><th scope="col">加入时间</th></tr></thead>
                <tbody>
                  <tr v-if="members.length === 0"><td colspan="4" class="sdp-org-management__empty">当前组织暂无成员</td></tr>
                  <tr v-for="member in members" v-else :key="memberKey(member)">
                    <td><div class="sdp-org-management__identity"><span aria-hidden="true">{{ memberName(member).charAt(0).toUpperCase() }}</span><div><strong>{{ memberName(member) }}</strong><small>{{ member.email || member.user_id || '未提供账户信息' }}</small></div></div></td>
                    <td><span class="sdp-org-management__role">{{ roleLabel(member.role) }}</span></td>
                    <td><span class="sdp-org-management__status"><i aria-hidden="true" />{{ statusLabel(member.status) }}</span></td>
                    <td>{{ formatDate(member.joined_at || member.created_at) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>
        </section>
      </div>

      <div v-if="editorOpen" class="sdp-org-management__backdrop" role="presentation" @mousedown.self="closeEditor">
        <section ref="dialogRef" class="sdp-org-management__dialog" role="dialog" aria-modal="true" aria-labelledby="org-editor-title" @keydown.esc="closeEditor" @keydown.tab="trapFocus">
          <header><div><p>{{ editingOrg ? 'Edit Organization' : 'New Organization' }}</p><h2 id="org-editor-title">{{ editingOrg ? '编辑组织' : '创建组织' }}</h2></div><button type="button" aria-label="关闭组织编辑对话框" @click="closeEditor">关闭</button></header>
          <form @submit.prevent="saveOrganization">
            <label for="org-name"><span>组织名称</span><input id="org-name" ref="nameInputRef" v-model.trim="form.name" type="text" maxlength="255" autocomplete="organization" required aria-describedby="org-name-help"><small id="org-name-help">用于成员识别和协作空间展示。</small></label>
            <label for="org-description"><span>组织说明</span><textarea id="org-description" v-model.trim="form.description" maxlength="1000" rows="4" placeholder="描述组织职责或协作范围" /></label>
            <div class="sdp-org-management__dialog-actions"><SdpButton variant="secondary" :disabled="saving" aria-label="取消编辑组织" @click="closeEditor">取消</SdpButton><SdpButton :loading="saving" :disabled="!form.name" aria-label="保存组织" @click="saveOrganization">保存组织</SdpButton></div>
          </form>
        </section>
      </div>

      <SdpConfirmDialog v-model:visible="deleteDialogOpen" type="danger" title="删除组织" :message="`确定删除“${selectedOrg?.name || ''}”吗？此操作无法撤销。`" confirm-text="删除组织" cancel-text="取消" :loading="deleting" @confirm="removeOrganization" />
    </div>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { createOrg, deleteOrg, listMembers as listOrgMembers, listOrgs, updateOrg, type OrganizationListItem } from '@/api/org'
import type { AdminMember } from '@/api/spaces'
import { SdpButton, SdpConfirmDialog } from '@/components/design'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'
import { useAuthStore } from '@/stores/auth'

interface OrganizationMember extends Partial<AdminMember> {
  id?: string
  user_id?: string
  username?: string
  email?: string
  role?: string
  status?: string
  joined_at?: string
  created_at?: string
}

const authStore = useAuthStore()
const organizations = ref<OrganizationListItem[]>([])
const membersByOrg = ref<Record<string, OrganizationMember[]>>({})
const selectedOrgId = ref('')
const loading = ref(true)
const membersLoading = ref(false)
const saving = ref(false)
const deleting = ref(false)
const loadError = ref('')
const membersError = ref('')
const message = ref('')
const messageType = ref<'success' | 'error'>('success')
const editorOpen = ref(false)
const deleteDialogOpen = ref(false)
const editingOrg = ref<OrganizationListItem | null>(null)
const form = ref({ name: '', description: '' })
const dialogRef = ref<HTMLElement | null>(null)
const nameInputRef = ref<HTMLInputElement | null>(null)

const selectedOrg = computed(() => organizations.value.find(org => org.id === selectedOrgId.value) || null)
const members = computed(() => selectedOrgId.value ? membersByOrg.value[selectedOrgId.value] || [] : [])
const memberCounts = computed(() => Object.fromEntries(Object.entries(membersByOrg.value).map(([id, values]) => [id, values.length])))
const totalMembers = computed(() => organizations.value.reduce((total, org) => total + (memberCounts.value[org.id] ?? org.member_count ?? 0), 0))
const currentUserId = computed(() => String(authStore.user?.id || authStore.user?.user_id || ''))
const canDeleteSelectedOrg = computed(() => Boolean(selectedOrg.value && currentUserId.value && selectedOrg.value.owner_id === currentUserId.value))
const stats = computed(() => [
  { label: '组织数量', value: organizations.value.length, note: '当前账户可访问' },
  { label: '成员总数', value: totalMembers.value, note: '按组织成员合计' },
  { label: '我创建的', value: organizations.value.filter(org => org.owner_id === currentUserId.value).length, note: '可编辑与删除' },
])

async function loadOrganizations() {
  loading.value = true
  loadError.value = ''
  try {
    const response = await listOrgs()
    organizations.value = normalizeOrganizations(response.data.organizations || [])
    if (!organizations.value.some(org => org.id === selectedOrgId.value)) selectedOrgId.value = organizations.value[0]?.id || ''
    await loadAllOrganizationMembers()
  } catch (error: unknown) {
    loadError.value = errorMessage(error, '请检查网络连接与组织访问权限。')
  } finally { loading.value = false }
}

async function loadAllOrganizationMembers() {
  membersError.value = ''
  const results = await Promise.allSettled(organizations.value.map(async org => {
    const response = await listOrgMembers(org.id)
    return [org.id, normalizeMembers(response.data)] as const
  }))
  const nextMembers = { ...membersByOrg.value }
  results.forEach((result, index) => {
    if (result.status === 'fulfilled') nextMembers[result.value[0]] = result.value[1]
    else if (organizations.value[index]?.id === selectedOrgId.value) membersError.value = errorMessage(result.reason, '无法读取当前组织成员。')
  })
  membersByOrg.value = nextMembers
}

async function loadOrganizationMembers() {
  if (!selectedOrgId.value) return
  membersLoading.value = true
  membersError.value = ''
  try {
    const response = await listOrgMembers(selectedOrgId.value)
    membersByOrg.value = { ...membersByOrg.value, [selectedOrgId.value]: normalizeMembers(response.data) }
  } catch (error: unknown) {
    membersError.value = errorMessage(error, '无法读取当前组织成员。')
  } finally { membersLoading.value = false }
}

async function selectOrganization(id: string) {
  selectedOrgId.value = id
  if (!membersByOrg.value[id]) await loadOrganizationMembers()
}

async function openCreateDialog() {
  editingOrg.value = null
  form.value = { name: '', description: '' }
  await showEditor()
}

async function openEditDialog() {
  if (!selectedOrg.value) return
  editingOrg.value = selectedOrg.value
  form.value = { name: selectedOrg.value.name, description: selectedOrg.value.description || '' }
  await showEditor()
}

async function showEditor() {
  editorOpen.value = true
  await nextTick()
  nameInputRef.value?.focus()
}

function closeEditor() {
  if (!saving.value) editorOpen.value = false
}

async function saveOrganization() {
  if (!form.value.name || saving.value) return
  saving.value = true
  try {
    const payload = { name: form.value.name, description: form.value.description }
    const response = editingOrg.value ? await updateOrg(editingOrg.value.id, payload) : await createOrg(payload)
    const saved = normalizeOrganization(response.data)
    editorOpen.value = false
    showMessage(editingOrg.value ? '组织资料已更新。' : '组织已创建。', 'success')
    await loadOrganizations()
    if (saved.id) selectedOrgId.value = saved.id
  } catch (error: unknown) {
    showMessage(errorMessage(error, '组织保存失败。'), 'error')
  } finally { saving.value = false }
}

async function removeOrganization() {
  if (!selectedOrg.value || deleting.value) return
  deleting.value = true
  const name = selectedOrg.value.name
  try {
    await deleteOrg(selectedOrg.value.id)
    deleteDialogOpen.value = false
    showMessage(`组织“${name}”已删除。`, 'success')
    await loadOrganizations()
  } catch (error: unknown) {
    showMessage(errorMessage(error, '组织删除失败。'), 'error')
  } finally { deleting.value = false }
}

function normalizeOrganizations(values: unknown[]): OrganizationListItem[] {
  return values.map(value => normalizeOrganization(value)).filter(org => Boolean(org.id))
}

function normalizeOrganization(value: unknown): OrganizationListItem {
  const item = value as { organization?: OrganizationListItem; auth_status?: string; days_remaining?: number; data?: OrganizationListItem }
  const org = item.organization || item.data || item as OrganizationListItem
  return { ...org, auth_status: item.auth_status || org.auth_status, days_remaining: item.days_remaining ?? org.days_remaining }
}

function normalizeMembers(value: unknown): OrganizationMember[] {
  if (Array.isArray(value)) return value
  const payload = value as { members?: OrganizationMember[]; data?: { members?: OrganizationMember[] } }
  return payload?.members || payload?.data?.members || []
}

function memberKey(member: OrganizationMember) { return String(member.id || member.user_id || member.email || member.username) }
function memberName(member: OrganizationMember) { return member.name || member.nickname || member.username || member.email || `成员 ${String(member.user_id || '').slice(0, 8)}` }
function roleLabel(role?: string) { return ({ owner: '所有者', admin: '管理员', editor: '编辑者', viewer: '查看者', member: '成员' } as Record<string, string>)[String(role || '').toLocaleLowerCase()] || role || '成员' }
function statusLabel(status?: string) { return ['disabled', 'inactive', 'blocked'].includes(String(status || '').toLocaleLowerCase()) ? '停用' : '启用' }
function formatDate(value?: string) { if (!value) return '—'; const date = new Date(value); return Number.isNaN(date.getTime()) ? '—' : new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' }).format(date) }
function showMessage(value: string, type: 'success' | 'error') { message.value = value; messageType.value = type }
function errorMessage(error: unknown, fallback: string) { const value = error as { message?: string; response?: { data?: { error?: string; message?: string } } }; return value?.response?.data?.error || value?.response?.data?.message || value?.message || fallback }
function trapFocus(event: KeyboardEvent) { const elements = Array.from(dialogRef.value?.querySelectorAll<HTMLElement>('button:not(:disabled), input:not(:disabled), textarea:not(:disabled)') || []); if (!elements.length) return; const first = elements[0]; const last = elements[elements.length - 1]; if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus() } else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus() } }

onMounted(loadOrganizations)
</script>

<style scoped>
.sdp-org-management { min-height: 100dvh; color: var(--ink-900); background: var(--ink-100); font-family: var(--font-body); }
.sdp-org-management__shell { display: grid; gap: var(--space-6); width: min(100%, var(--content-max-width)); margin-inline: auto; padding: var(--space-8); }
.sdp-org-management__header, .sdp-org-management__panel-head, .sdp-org-management__panel-actions, .sdp-org-management__identity, .sdp-org-management__dialog header, .sdp-org-management__dialog-actions { display: flex; align-items: center; }
.sdp-org-management__header { justify-content: space-between; gap: var(--space-6); }
.sdp-org-management__header p, .sdp-org-management__panel-head p, .sdp-org-management__dialog header p { color: var(--brand-700); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); letter-spacing: .08em; text-transform: uppercase; }
.sdp-org-management__header h1 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-4xl); line-height: var(--leading-tight); }
.sdp-org-management__header > div > span { display: block; margin-top: var(--space-2); color: var(--ink-600); font-size: var(--text-base); }
.sdp-org-management__message { padding: var(--space-3) var(--space-4); border: 1px solid var(--brand-300); border-radius: var(--radius-sm); color: var(--brand-900); background: var(--brand-50); font-size: var(--text-sm); }
.sdp-org-management__message--error { border-color: var(--ink-400); color: var(--ink-900); background: var(--ink-200); }
.sdp-org-management__stats { display: grid; grid-template-columns: repeat(3, minmax(var(--space-0), 1fr)); gap: var(--space-4); }
.sdp-org-management__stats article { padding: var(--space-5); border: 1px solid var(--ink-200); border-radius: var(--radius-md); background: var(--ink-50); box-shadow: var(--shadow-xs); }
.sdp-org-management__stats span, .sdp-org-management__stats small { display: block; color: var(--ink-600); font-size: var(--text-xs); }
.sdp-org-management__stats strong { display: block; margin: var(--space-2) var(--space-0); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-3xl); line-height: var(--leading-tight); }
.sdp-org-management__stats small { color: var(--ink-500); }
.sdp-org-management__workspace { display: grid; grid-template-columns: calc(var(--space-24) * 3) minmax(var(--space-0), 1fr); gap: var(--space-5); align-items: start; }
.sdp-org-management__panel { min-width: var(--space-0); overflow: hidden; border: 1px solid var(--ink-200); border-radius: var(--radius-lg); background: var(--ink-50); box-shadow: var(--shadow-xs); }
.sdp-org-management__panel-head { min-height: calc(var(--space-24) + var(--space-2)); justify-content: space-between; gap: var(--space-4); padding: var(--space-5); border-bottom: 1px solid var(--ink-200); }
.sdp-org-management__panel-head h2 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-xl); }
.sdp-org-management__panel-head > div > span { display: block; max-width: calc(var(--space-24) * 5); margin-top: var(--space-1); overflow: hidden; color: var(--ink-600); font-size: var(--text-xs); text-overflow: ellipsis; white-space: nowrap; }
.sdp-org-management__panel-actions { flex-wrap: wrap; justify-content: flex-end; gap: var(--space-2); }
.sdp-org-management__org-list { padding: var(--space-3); list-style: none; }
.sdp-org-management__org-list li + li { margin-top: var(--space-1); }
.sdp-org-management__org-list button { width: 100%; display: flex; align-items: center; gap: var(--space-3); padding: var(--space-3); border: 1px solid transparent; border-radius: var(--radius-sm); color: var(--ink-800); background: transparent; font-family: var(--font-body); text-align: left; cursor: pointer; }
.sdp-org-management__org-list button:hover { background: var(--ink-100); }
.sdp-org-management__org-list .sdp-org-management__org-button--active { border-color: var(--brand-300); color: var(--brand-900); background: var(--brand-50); }
.sdp-org-management__org-mark { width: var(--space-10); height: var(--space-10); display: inline-flex; flex: 0 0 var(--space-10); align-items: center; justify-content: center; border-radius: var(--radius-md); color: var(--ink-950); background: var(--brand-300); font-family: var(--font-display); font-weight: var(--font-weight-bold); }
.sdp-org-management__org-list button > span:last-child, .sdp-org-management__identity div { min-width: var(--space-0); }
.sdp-org-management__org-list strong, .sdp-org-management__org-list small, .sdp-org-management__identity strong, .sdp-org-management__identity small { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sdp-org-management__org-list strong { color: inherit; font-size: var(--text-sm); }
.sdp-org-management__org-list small, .sdp-org-management__identity small { margin-top: var(--space-1); color: var(--ink-500); font-size: var(--text-xs); }
.sdp-org-management__state { min-height: calc(var(--space-24) * 2); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: var(--space-2); padding: var(--space-6); color: var(--ink-600); font-size: var(--text-sm); text-align: center; }
.sdp-org-management__state strong { color: var(--ink-900); }
.sdp-org-management__state button { padding: var(--space-2) var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--brand-800); background: var(--ink-50); font: inherit; font-weight: var(--font-weight-semibold); cursor: pointer; }
.sdp-org-management__table-wrap { overflow-x: auto; }
.sdp-org-management table { width: 100%; border-collapse: collapse; text-align: left; }
.sdp-org-management th, .sdp-org-management td { padding: var(--space-4) var(--space-5); border-bottom: 1px solid var(--ink-200); color: var(--ink-700); font-size: var(--text-sm); white-space: nowrap; }
.sdp-org-management th { color: var(--ink-600); background: var(--ink-100); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); letter-spacing: .04em; }
.sdp-org-management tbody tr:hover { background: var(--brand-50); }
.sdp-org-management__identity { gap: var(--space-3); }
.sdp-org-management__identity > span { width: var(--space-10); height: var(--space-10); display: inline-flex; flex: 0 0 var(--space-10); align-items: center; justify-content: center; border-radius: var(--radius-pill); color: var(--brand-900); background: var(--brand-100); font-weight: var(--font-weight-bold); }
.sdp-org-management__identity strong { max-width: calc(var(--space-24) * 2); color: var(--ink-950); }
.sdp-org-management__role, .sdp-org-management__status { display: inline-flex; align-items: center; gap: var(--space-2); padding: var(--space-1) var(--space-3); border-radius: var(--radius-pill); color: var(--brand-900); background: var(--brand-100); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-org-management__status { background: var(--brand-50); }
.sdp-org-management__status i { width: var(--space-2); height: var(--space-2); border-radius: var(--radius-pill); background: var(--brand-700); }
.sdp-org-management__empty { height: calc(var(--space-24) * 2); color: var(--ink-500); text-align: center; }
.sdp-org-management__backdrop { position: fixed; inset: var(--space-0); z-index: var(--z-modal); display: grid; place-items: center; padding: var(--space-6); background: color-mix(in oklch, var(--ink-950) 72%, transparent); }
.sdp-org-management__dialog { width: min(100%, calc(var(--space-24) * 5)); padding: var(--space-6); border: 1px solid var(--ink-200); border-radius: var(--radius-lg); background: var(--ink-50); box-shadow: var(--shadow-xl); }
.sdp-org-management__dialog header { align-items: flex-start; justify-content: space-between; gap: var(--space-4); }
.sdp-org-management__dialog h2 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-2xl); }
.sdp-org-management__dialog header button { min-height: var(--space-10); padding: var(--space-2) var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-800); background: var(--ink-50); font-family: var(--font-body); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); cursor: pointer; }
.sdp-org-management__dialog form { display: grid; gap: var(--space-5); margin-top: var(--space-6); }
.sdp-org-management__dialog label { display: grid; gap: var(--space-2); }
.sdp-org-management__dialog label > span { color: var(--ink-800); font-size: var(--text-sm); font-weight: var(--font-weight-semibold); }
.sdp-org-management__dialog label small { color: var(--ink-600); font-size: var(--text-xs); }
.sdp-org-management__dialog input, .sdp-org-management__dialog textarea { width: 100%; min-height: var(--space-12); padding: var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-900); background: var(--ink-50); font-family: var(--font-body); font-size: var(--text-sm); }
.sdp-org-management__dialog textarea { min-height: var(--space-24); resize: vertical; }
.sdp-org-management__dialog-actions { justify-content: flex-end; gap: var(--space-3); padding-top: var(--space-2); }
.sdp-org-management button:focus-visible, .sdp-org-management input:focus-visible, .sdp-org-management textarea:focus-visible { outline: 2px solid var(--brand-500); outline-offset: 2px; }
@media (max-width: 64rem) { .sdp-org-management__workspace { grid-template-columns: 1fr; } .sdp-org-management__org-list { display: grid; grid-template-columns: repeat(2, minmax(var(--space-0), 1fr)); gap: var(--space-2); } .sdp-org-management__org-list li + li { margin-top: var(--space-0); } }
@media (max-width: 48rem) { .sdp-org-management__shell { padding: var(--space-6); } .sdp-org-management__stats { grid-template-columns: 1fr; } .sdp-org-management__org-list { grid-template-columns: 1fr; } }
@media (max-width: 40rem) { .sdp-org-management__shell { padding: var(--space-4); } .sdp-org-management__header, .sdp-org-management__panel-head { align-items: flex-start; flex-direction: column; } .sdp-org-management__header h1 { font-size: var(--text-3xl); } .sdp-org-management__header :deep(.sdp-button), .sdp-org-management__panel-actions { width: 100%; } .sdp-org-management__panel-actions :deep(.sdp-button) { flex: 1; } .sdp-org-management__backdrop { align-items: end; padding: var(--space-0); } .sdp-org-management__dialog { border-radius: var(--radius-lg) var(--radius-lg) var(--radius-none) var(--radius-none); } .sdp-org-management__dialog-actions { align-items: stretch; flex-direction: column-reverse; } .sdp-org-management__dialog-actions :deep(.sdp-button) { width: 100%; } }
</style>
