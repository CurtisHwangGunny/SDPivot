<template>
  <SdpSidebarLayout>
    <div class="sdp-space-detail">
      <div class="sdp-space-detail__shell">
        <SdpErrorState v-if="loadError" type="network" title="空间加载失败" :description="loadError" retryable @retry="loadData" />
        <SdpSkeleton v-else-if="loading" variant="card" :count="4" />
        <template v-else-if="space">
          <section class="sdp-space-detail__hero" aria-labelledby="space-detail-title">
            <div><p>Knowledge Workspace</p><h1 id="space-detail-title">{{ space.name }}</h1><span>{{ space.description || '这个知识空间还没有补充描述。' }}</span></div>
            <div class="sdp-space-detail__hero-actions"><span>{{ visibilityLabel(space.visibility) }}</span><SdpButton variant="secondary" aria-label="打开空间设置" @click="activeTab = 'settings'">空间设置</SdpButton><SdpButton :aria-label="`管理 ${space.name} 的文档`" @click="goDocuments">文档管理</SdpButton></div>
          </section>

          <section class="sdp-space-detail__facts" aria-label="空间概览">
            <article><span>空间成员</span><strong>{{ members.length }}</strong><small>当前协作者</small></article>
            <article><span>创建时间</span><strong>{{ formatDate(space.created_at) }}</strong><small>空间建立日期</small></article>
            <article><span>最近更新</span><strong>{{ formatDate(space.updated_at) }}</strong><small>配置更新时间</small></article>
          </section>

          <section class="sdp-space-detail__panel">
            <div ref="tabListRef" class="sdp-space-detail__tabs" role="tablist" aria-label="空间详情栏目" @keydown="handleTabKeydown">
              <button v-for="tab in tabs" :id="`space-detail-tab-${tab.value}`" :key="tab.value" type="button" role="tab" :tabindex="activeTab === tab.value ? 0 : -1" :aria-selected="activeTab === tab.value" :aria-controls="`space-detail-panel-${tab.value}`" :class="{ 'sdp-space-detail__tab--active': activeTab === tab.value }" @click="activeTab = tab.value">{{ tab.label }}</button>
            </div>

            <div v-if="activeTab === 'overview'" id="space-detail-panel-overview" class="sdp-space-detail__tab-panel" role="tabpanel" aria-labelledby="space-detail-tab-overview" tabindex="0">
              <div class="sdp-space-detail__overview-grid"><article><p>空间标识</p><strong>{{ space.id }}</strong><span>用于 API、链接和系统内关联。</span></article><article><p>访问范围</p><strong>{{ visibilityLabel(space.visibility) }}</strong><span>{{ visibilityDescription(space.visibility) }}</span></article><article><p>所有者</p><strong>{{ space.owner_id || '未指定' }}</strong><span>负责空间配置与成员权限。</span></article></div>
            </div>

            <div v-else-if="activeTab === 'members'" id="space-detail-panel-members" class="sdp-space-detail__tab-panel" role="tabpanel" aria-labelledby="space-detail-tab-members" tabindex="0">
              <div v-if="members.length" class="sdp-space-detail__table-wrap"><table><caption class="sr-only">空间成员列表</caption><thead><tr><th scope="col">成员</th><th scope="col">角色</th><th scope="col">加入时间</th></tr></thead><tbody><tr v-for="member in members" :key="member.id || member.user_id"><td><div class="sdp-space-detail__member"><span aria-hidden="true">{{ memberInitial(member) }}</span><div><strong>{{ memberName(member) }}</strong><small>{{ member.user_id }}</small></div></div></td><td><span class="sdp-space-detail__role">{{ roleLabel(member.role) }}</span></td><td>{{ formatDate(member.created_at) }}</td></tr></tbody></table></div>
              <SdpEmptyState v-else variant="compact" title="暂无成员数据" description="当前空间尚未返回成员记录。" />
            </div>

            <div v-else id="space-detail-panel-settings" class="sdp-space-detail__tab-panel" role="tabpanel" aria-labelledby="space-detail-tab-settings" tabindex="0">
              <form class="sdp-space-detail__form" @submit.prevent="saveSettings">
                <label for="space-detail-name"><span>空间名称</span><input id="space-detail-name" v-model.trim="settingsForm.name" type="text" maxlength="255" required></label>
                <label for="space-detail-description"><span>空间描述</span><textarea id="space-detail-description" v-model.trim="settingsForm.description" rows="4" placeholder="说明空间用途、资料范围和使用对象" /></label>
                <label for="space-detail-visibility"><span>可见性</span><select id="space-detail-visibility" v-model="settingsForm.visibility"><option value="private">私密</option><option value="team">团队可见</option><option value="org">组织可见</option></select></label>
                <p v-if="saveError" class="sdp-space-detail__form-error" role="alert">{{ saveError }}</p>
                <p v-if="saveSuccess" class="sdp-space-detail__form-success" role="status">空间设置已保存。</p>
                <div class="sdp-space-detail__form-actions"><SdpButton variant="secondary" :disabled="saving" aria-label="还原空间设置" @click="resetSettings">还原</SdpButton><SdpButton :loading="saving" :disabled="!settingsForm.name" aria-label="保存空间设置" @click="saveSettings">保存设置</SdpButton></div>
              </form>
            </div>
          </section>
        </template>
      </div>
    </div>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getSpace, updateSpace, listSpaceMembers, type Space } from '@/api/spaces'
import { SdpButton, SdpEmptyState, SdpErrorState, SdpSkeleton } from '@/components/design'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'

type DetailTab = 'overview' | 'members' | 'settings'
type SpaceMember = { id?: string; user_id: string; role: string; created_at?: string; name?: string; nickname?: string; username?: string; email?: string }
const tabs: Array<{ label: string; value: DetailTab }> = [{ label: '空间信息', value: 'overview' }, { label: '成员列表', value: 'members' }, { label: '空间设置', value: 'settings' }]
const route = useRoute()
const router = useRouter()
const spaceId = computed(() => String(route.params.id || ''))
const space = ref<Space | null>(null)
const members = ref<SpaceMember[]>([])
const loading = ref(false)
const loadError = ref('')
const activeTab = ref<DetailTab>('overview')
const tabListRef = ref<HTMLElement | null>(null)
const saving = ref(false)
const saveError = ref('')
const saveSuccess = ref(false)
const settingsForm = ref({ name: '', description: '', visibility: 'private' })

async function loadData() {
  if (!spaceId.value) return
  loading.value = true
  loadError.value = ''
  try {
    const [spaceResponse, membersResponse] = await Promise.all([getSpace(spaceId.value), listSpaceMembers(spaceId.value)])
    const payload = spaceResponse.data as Space | { space: Space }
    space.value = 'space' in payload ? payload.space : payload
    members.value = ((membersResponse.data as { members?: SpaceMember[] }).members || [])
    resetSettings()
  } catch (error: unknown) {
    space.value = null
    members.value = []
    loadError.value = errorMessage(error, '请检查网络连接和空间访问权限。')
  } finally { loading.value = false }
}

function resetSettings() { if (!space.value) return; settingsForm.value = { name: space.value.name, description: space.value.description || '', visibility: normalizeVisibility(space.value.visibility) }; saveError.value = ''; saveSuccess.value = false }
async function saveSettings() {
  if (!space.value || !settingsForm.value.name || saving.value) return
  saving.value = true
  saveError.value = ''
  saveSuccess.value = false
  try {
    await updateSpace(spaceId.value, settingsForm.value)
    space.value = { ...space.value, ...settingsForm.value, updated_at: new Date().toISOString() }
    saveSuccess.value = true
  } catch (error: unknown) { saveError.value = errorMessage(error, '保存失败，请稍后重试。') } finally { saving.value = false }
}

function goDocuments() { router.push(`/spaces/${spaceId.value}/documents`) }
function handleTabKeydown(event: KeyboardEvent) { if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return; event.preventDefault(); const buttons = Array.from(tabListRef.value?.querySelectorAll<HTMLButtonElement>('[role="tab"]') || []); const current = tabs.findIndex(tab => tab.value === activeTab.value); let next = current; if (event.key === 'Home') next = 0; if (event.key === 'End') next = tabs.length - 1; if (event.key === 'ArrowRight') next = (current + 1) % tabs.length; if (event.key === 'ArrowLeft') next = (current - 1 + tabs.length) % tabs.length; activeTab.value = tabs[next].value; buttons[next]?.focus() }
function normalizeVisibility(value: string) { return value === 'enterprise' ? 'org' : value || 'private' }
function visibilityLabel(value: string) { if (value === 'team') return '团队可见'; if (value === 'org' || value === 'enterprise') return '组织可见'; return '私密' }
function visibilityDescription(value: string) { if (value === 'team') return '空间成员和团队协作者可以访问。'; if (value === 'org' || value === 'enterprise') return '组织内具备权限的成员可以访问。'; return '仅受邀成员可以访问。' }
function memberName(member: SpaceMember) { return member.name || member.nickname || member.username || member.email || `成员 ${member.user_id.slice(0, 8)}` }
function memberInitial(member: SpaceMember) { return memberName(member).trim().charAt(0).toUpperCase() || 'U' }
function roleLabel(role: string) { return ({ owner: '所有者', editor: '编辑者', viewer: '查看者' } as Record<string, string>)[role] || role || '成员' }
function formatDate(value?: string) { return value ? new Date(value).toLocaleDateString('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' }) : '—' }
function errorMessage(error: unknown, fallback: string) { if (typeof error !== 'object' || !error) return fallback; const value = error as { message?: string; response?: { data?: { error?: string; message?: string } } }; return value.response?.data?.error || value.response?.data?.message || value.message || fallback }

watch(spaceId, loadData, { immediate: true })
</script>

<style scoped>
.sdp-space-detail { min-height: 100dvh; color: var(--ink-900); background: var(--ink-100); font-family: var(--font-body); }
.sdp-space-detail__shell { display: grid; gap: var(--space-6); width: min(100%, var(--content-max-width)); margin-inline: auto; padding: var(--space-8); }
.sdp-space-detail__hero { display: flex; align-items: flex-end; justify-content: space-between; gap: var(--space-8); padding: var(--space-8); border: 1px solid var(--ink-800); border-radius: var(--radius-xl); color: var(--ink-50); background: var(--ink-950); box-shadow: var(--shadow-md); }
.sdp-space-detail__hero p { color: var(--brand-400); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); letter-spacing: .08em; text-transform: uppercase; }
.sdp-space-detail__hero h1 { margin-top: var(--space-2); font-family: var(--font-display); font-size: var(--text-4xl); line-height: var(--leading-tight); }
.sdp-space-detail__hero > div > span { display: block; max-width: 60ch; margin-top: var(--space-3); color: var(--ink-300); font-size: var(--text-base); line-height: var(--leading-relaxed); }
.sdp-space-detail__hero-actions { display: flex; flex: 0 0 auto; align-items: center; gap: var(--space-3); }
.sdp-space-detail__hero-actions > span { margin: var(--space-0); padding: var(--space-2) var(--space-3); border: 1px solid var(--brand-700); border-radius: var(--radius-pill); color: var(--brand-100); background: var(--brand-900); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-space-detail__facts { display: grid; grid-template-columns: repeat(3, minmax(var(--space-0), 1fr)); gap: var(--space-4); }
.sdp-space-detail__facts article { padding: var(--space-5); border: 1px solid var(--ink-200); border-radius: var(--radius-md); background: var(--ink-50); box-shadow: var(--shadow-xs); }
.sdp-space-detail__facts span, .sdp-space-detail__facts small { display: block; color: var(--ink-600); font-size: var(--text-xs); }
.sdp-space-detail__facts strong { display: block; margin: var(--space-2) var(--space-0); overflow-wrap: anywhere; color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-2xl); line-height: var(--leading-tight); }
.sdp-space-detail__facts small { color: var(--ink-500); }
.sdp-space-detail__panel { min-width: var(--space-0); overflow: hidden; border: 1px solid var(--ink-200); border-radius: var(--radius-lg); background: var(--ink-50); box-shadow: var(--shadow-xs); }
.sdp-space-detail__tabs { display: flex; gap: var(--space-1); padding: var(--space-3) var(--space-6) var(--space-0); border-bottom: 1px solid var(--ink-200); }
.sdp-space-detail__tabs button { min-height: var(--space-12); padding: var(--space-2) var(--space-4); border: 0; border-bottom: 3px solid transparent; color: var(--ink-600); background: transparent; font-family: var(--font-body); font-size: var(--text-sm); font-weight: var(--font-weight-semibold); cursor: pointer; }
.sdp-space-detail__tabs .sdp-space-detail__tab--active { border-bottom-color: var(--brand-700); color: var(--brand-900); }
.sdp-space-detail__tab-panel { min-height: calc(var(--space-24) * 2.5); padding: var(--space-6); }
.sdp-space-detail__overview-grid { display: grid; grid-template-columns: repeat(3, minmax(var(--space-0), 1fr)); gap: var(--space-4); }
.sdp-space-detail__overview-grid article { padding: var(--space-5); border: 1px solid var(--ink-200); border-radius: var(--radius-md); background: var(--ink-100); }
.sdp-space-detail__overview-grid p { color: var(--brand-700); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); text-transform: uppercase; }
.sdp-space-detail__overview-grid strong { display: block; margin: var(--space-3) var(--space-0); overflow-wrap: anywhere; color: var(--ink-950); font-size: var(--text-lg); }
.sdp-space-detail__overview-grid span { color: var(--ink-600); font-size: var(--text-sm); line-height: var(--leading-relaxed); }
.sdp-space-detail__table-wrap { overflow-x: auto; }
.sdp-space-detail table { width: 100%; border-collapse: collapse; text-align: left; }
.sdp-space-detail th, .sdp-space-detail td { padding: var(--space-4) var(--space-5); border-bottom: 1px solid var(--ink-200); color: var(--ink-700); font-size: var(--text-sm); white-space: nowrap; }
.sdp-space-detail th { color: var(--ink-600); background: var(--ink-100); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-space-detail__member { display: flex; align-items: center; gap: var(--space-3); }
.sdp-space-detail__member > span { width: var(--space-10); height: var(--space-10); display: inline-flex; flex: 0 0 var(--space-10); align-items: center; justify-content: center; border-radius: var(--radius-pill); color: var(--brand-900); background: var(--brand-100); font-weight: var(--font-weight-bold); }
.sdp-space-detail__member strong, .sdp-space-detail__member small { display: block; }
.sdp-space-detail__member strong { color: var(--ink-950); }
.sdp-space-detail__member small { margin-top: var(--space-1); color: var(--ink-500); font-size: var(--text-xs); }
.sdp-space-detail__role { display: inline-flex; padding: var(--space-1) var(--space-3); border-radius: var(--radius-pill); color: var(--brand-900); background: var(--brand-100); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-space-detail__form { display: grid; gap: var(--space-5); max-width: calc(var(--space-24) * 7); }
.sdp-space-detail__form label { display: grid; gap: var(--space-2); }
.sdp-space-detail__form label span { color: var(--ink-800); font-size: var(--text-sm); font-weight: var(--font-weight-semibold); }
.sdp-space-detail__form input, .sdp-space-detail__form textarea, .sdp-space-detail__form select { width: 100%; min-height: var(--space-12); padding: var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-900); background: var(--ink-50); font-family: var(--font-body); font-size: var(--text-sm); }
.sdp-space-detail__form textarea { min-height: var(--space-24); resize: vertical; }
.sdp-space-detail__form-actions { display: flex; justify-content: flex-end; gap: var(--space-3); }
.sdp-space-detail__form-error, .sdp-space-detail__form-success { padding: var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-950); background: var(--ink-100); font-size: var(--text-sm); }
.sdp-space-detail__form-success { border-color: var(--brand-300); color: var(--brand-900); background: var(--brand-50); }
.sdp-space-detail button:focus-visible, .sdp-space-detail input:focus-visible, .sdp-space-detail textarea:focus-visible, .sdp-space-detail select:focus-visible, .sdp-space-detail__tab-panel:focus-visible { outline: 2px solid var(--brand-500); outline-offset: 2px; }
@media (max-width: 64rem) { .sdp-space-detail__hero { align-items: flex-start; flex-direction: column; } .sdp-space-detail__hero-actions { flex-wrap: wrap; } .sdp-space-detail__overview-grid { grid-template-columns: 1fr; } }
@media (max-width: 48rem) { .sdp-space-detail__shell { padding: var(--space-6); } .sdp-space-detail__facts { grid-template-columns: 1fr; } }
@media (max-width: 40rem) { .sdp-space-detail__shell { padding: var(--space-4); } .sdp-space-detail__hero { padding: var(--space-6); } .sdp-space-detail__hero h1 { font-size: var(--text-3xl); } .sdp-space-detail__hero-actions { width: 100%; align-items: stretch; flex-direction: column; } .sdp-space-detail__hero-actions :deep(.sdp-button) { width: 100%; } .sdp-space-detail__tabs { padding-inline: var(--space-3); overflow-x: auto; } .sdp-space-detail__tab-panel { padding: var(--space-4); } .sdp-space-detail__form-actions { align-items: stretch; flex-direction: column-reverse; } .sdp-space-detail__form-actions :deep(.sdp-button) { width: 100%; } }
</style>
