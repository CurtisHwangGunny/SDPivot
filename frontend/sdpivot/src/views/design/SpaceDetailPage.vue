<template>
  <SdpSidebarLayout>
    <div class="sdp-space-detail">
      <div class="sdp-space-detail__shell">
        <SdpErrorState v-if="loadError" type="network" title="空间加载失败" :description="loadError" retryable @retry="loadData" />
        <SdpSkeleton v-else-if="loading" variant="card" :count="4" />
        <template v-else-if="space">
          <header class="sdp-space-detail__hero">
            <div>
              <p>Workspace Detail</p>
              <h1>{{ space.name }}</h1>
              <span>{{ space.description || '这个知识空间还没有补充描述。' }}</span>
            </div>
            <div class="sdp-space-detail__hero-actions">
              <span>{{ visibilityLabel(space.visibility) }}</span>
              <SdpButton variant="secondary" aria-label="管理空间成员" @click="showMembers = true">成员管理</SdpButton>
              <SdpButton aria-label="导入文档" @click="showImport = true">导入文档</SdpButton>
            </div>
          </header>

          <section class="sdp-space-detail__metrics" aria-label="空间使用概览">
            <article>
              <span>空间成员</span>
              <strong>{{ members.length }}</strong>
              <small>当前协作者</small>
            </article>
            <article>
              <span>空间文档</span>
              <strong>{{ documents.length }}</strong>
              <small>{{ completedDocuments }} 份已完成解析</small>
            </article>
            <article>
              <span>存储占用</span>
              <strong>{{ formatSize(storageBytes) }}</strong>
              <small>来自当前空间文档</small>
            </article>
            <article>
              <span>问答调用</span>
              <strong>{{ qaCount }}</strong>
              <small>关联当前空间的会话</small>
            </article>
          </section>

          <div class="sdp-space-detail__content-grid">
            <section class="sdp-space-detail__panel" aria-labelledby="space-documents-title">
              <header class="sdp-space-detail__panel-head">
                <div><p>Knowledge Assets</p><h2 id="space-documents-title">最近文档</h2></div>
                <SdpButton variant="ghost" size="sm" aria-label="查看全部空间文档" @click="goDocuments">全部文档</SdpButton>
              </header>
              <div v-if="documents.length" class="sdp-space-detail__document-list">
                <article v-for="document in documents.slice(0, 5)" :key="document.id">
                  <span aria-hidden="true">{{ fileCode(document.file_type) }}</span>
                  <div><strong>{{ document.title || document.file_name }}</strong><small>{{ statusLabel(document.parse_status) }} · {{ formatSize(document.file_size) }}</small></div>
                  <time>{{ formatDate(document.updated_at || document.created_at) }}</time>
                </article>
              </div>
              <SdpEmptyState v-else variant="compact" title="暂无空间文档" description="导入第一份文档，开始构建可检索知识。">
                <template #actions><SdpButton size="sm" aria-label="导入第一份空间文档" @click="showImport = true">导入文档</SdpButton></template>
              </SdpEmptyState>
            </section>

            <aside class="sdp-space-detail__side">
              <section class="sdp-space-detail__panel" aria-labelledby="space-settings-title">
                <header class="sdp-space-detail__panel-head">
                  <div><p>Configuration</p><h2 id="space-settings-title">空间设置摘要</h2></div>
                  <button type="button" aria-label="编辑空间设置" @click="editingSettings = !editingSettings">{{ editingSettings ? '收起' : '编辑' }}</button>
                </header>
                <dl class="sdp-space-detail__summary">
                  <div><dt>访问范围</dt><dd>{{ visibilityLabel(space.visibility) }}</dd></div>
                  <div><dt>空间所有者</dt><dd>{{ space.owner_id || '未指定' }}</dd></div>
                  <div><dt>创建时间</dt><dd>{{ formatDate(space.created_at) }}</dd></div>
                  <div><dt>最近更新</dt><dd>{{ formatDate(space.updated_at) }}</dd></div>
                </dl>
                <form v-if="editingSettings" class="sdp-space-detail__form" @submit.prevent="saveSettings">
                  <label for="space-detail-name"><span>空间名称</span><input id="space-detail-name" v-model.trim="settingsForm.name" type="text" maxlength="255" required></label>
                  <label for="space-detail-description"><span>空间描述</span><textarea id="space-detail-description" v-model.trim="settingsForm.description" rows="3" /></label>
                  <label for="space-detail-visibility"><span>可见性</span><select id="space-detail-visibility" v-model="settingsForm.visibility"><option value="private">私密</option><option value="team">团队可见</option><option value="org">组织可见</option></select></label>
                  <p v-if="saveError" class="sdp-space-detail__form-message" role="alert">{{ saveError }}</p>
                  <p v-if="saveSuccess" class="sdp-space-detail__form-message" role="status">空间设置已保存。</p>
                  <div class="sdp-space-detail__form-actions"><SdpButton variant="secondary" size="sm" :disabled="saving" aria-label="还原空间设置" @click="resetSettings">还原</SdpButton><SdpButton size="sm" :loading="saving" :disabled="!settingsForm.name" aria-label="保存空间设置" @click="saveSettings">保存设置</SdpButton></div>
                </form>
              </section>

              <section class="sdp-space-detail__panel" aria-labelledby="space-members-title">
                <header class="sdp-space-detail__panel-head">
                  <div><p>Collaboration</p><h2 id="space-members-title">空间成员</h2></div>
                  <button type="button" aria-label="打开空间成员管理" @click="showMembers = true">管理</button>
                </header>
                <ul v-if="members.length" class="sdp-space-detail__members">
                  <li v-for="member in members.slice(0, 5)" :key="member.id || member.user_id">
                    <span aria-hidden="true">{{ memberInitial(member) }}</span>
                    <div><strong>{{ memberName(member) }}</strong><small>{{ roleLabel(member.role) }}</small></div>
                  </li>
                </ul>
                <SdpEmptyState v-else variant="compact" title="暂无成员数据" description="可通过成员管理邀请协作者。" />
              </section>
            </aside>
          </div>
        </template>
      </div>

      <SpaceDocumentImportDrawer v-model:open="showImport" :space-id="spaceId" @import="loadDocuments" />
      <SpaceMemberManagementDrawer v-model:open="showMembers" :space-id="spaceId" :members="members" />
    </div>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { listDocuments, type Document } from '@/api/documents'
import { listSessions } from '@/api/qa'
import { getSpace, listSpaceMembers, updateSpace, type Space } from '@/api/spaces'
import SpaceDocumentImportDrawer from '@/components/design/SpaceDocumentImportDrawer.vue'
import { SdpButton, SdpEmptyState, SdpErrorState, SdpSkeleton } from '@/components/design'
import SpaceMemberManagementDrawer from '@/components/design/SpaceMemberManagementDrawer.vue'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'

type SpaceMember = { id?: string; user_id: string; role: string; name?: string; nickname?: string; username?: string; email?: string }

const route = useRoute()
const router = useRouter()
const spaceId = computed(() => String(route.params.id || ''))
const space = ref<Space | null>(null)
const members = ref<SpaceMember[]>([])
const documents = ref<Document[]>([])
const qaCount = ref(0)
const loading = ref(true)
const loadError = ref('')
const showImport = ref(false)
const showMembers = ref(false)
const editingSettings = ref(false)
const saving = ref(false)
const saveError = ref('')
const saveSuccess = ref(false)
const settingsForm = ref({ name: '', description: '', visibility: 'private' })

const storageBytes = computed(() => documents.value.reduce((total, document) => total + (Number(document.file_size) || 0), 0))
const completedDocuments = computed(() => documents.value.filter(document => document.parse_status === 'completed').length)

async function loadMembers() {
  const response = await listSpaceMembers(spaceId.value)
  members.value = ((response.data as { members?: SpaceMember[] }).members || [])
}

async function loadDocuments() {
  const response = await listDocuments({ space_id: spaceId.value, page: 1, page_size: 100 })
  documents.value = response.data.documents || []
}

async function loadData() {
  if (!spaceId.value) return
  loading.value = true
  loadError.value = ''
  try {
    const [spaceResponse, membersResult, documentsResult, sessionsResult] = await Promise.all([
      getSpace(spaceId.value),
      listSpaceMembers(spaceId.value),
      listDocuments({ space_id: spaceId.value, page: 1, page_size: 100 }),
      listSessions().catch(() => ({ data: { sessions: [] } })),
    ])
    const payload = spaceResponse.data as Space | { space: Space }
    space.value = 'space' in payload ? payload.space : payload
    members.value = ((membersResult.data as { members?: SpaceMember[] }).members || [])
    documents.value = documentsResult.data.documents || []
    qaCount.value = (sessionsResult.data.sessions || []).filter(session => session.space_id === spaceId.value).length
    resetSettings()
  } catch (error: unknown) {
    space.value = null
    members.value = []
    documents.value = []
    qaCount.value = 0
    loadError.value = errorMessage(error, '请检查网络连接和空间访问权限。')
  } finally {
    loading.value = false
  }
}

function resetSettings() {
  if (!space.value) return
  settingsForm.value = { name: space.value.name, description: space.value.description || '', visibility: normalizeVisibility(space.value.visibility) }
  saveError.value = ''
  saveSuccess.value = false
}

async function saveSettings() {
  if (!space.value || !settingsForm.value.name || saving.value) return
  saving.value = true
  saveError.value = ''
  saveSuccess.value = false
  try {
    await updateSpace(spaceId.value, settingsForm.value)
    space.value = { ...space.value, ...settingsForm.value, updated_at: new Date().toISOString() }
    saveSuccess.value = true
  } catch (error: unknown) {
    saveError.value = errorMessage(error, '保存失败，请稍后重试。')
  } finally {
    saving.value = false
  }
}

function goDocuments() { void router.push(`/spaces/${spaceId.value}/documents`) }
function normalizeVisibility(value: string) { return value === 'enterprise' ? 'org' : value || 'private' }
function visibilityLabel(value: string) { if (value === 'team') return '团队可见'; if (value === 'org' || value === 'enterprise') return '组织可见'; return '私密' }
function memberName(member: SpaceMember) { return member.name || member.nickname || member.username || member.email || `成员 ${member.user_id.slice(0, 8)}` }
function memberInitial(member: SpaceMember) { return memberName(member).trim().charAt(0).toUpperCase() || 'U' }
function roleLabel(role: string) { return ({ owner: '所有者', admin: '管理员', editor: '编辑者', viewer: '查看者' } as Record<string, string>)[role] || role || '成员' }
function statusLabel(status: string) { return ({ pending: '等待解析', parsing: '解析中', processing: '解析中', completed: '解析完成', failed: '解析失败' } as Record<string, string>)[status] || status || '未知状态' }
function fileCode(type: string) { return String(type || 'DOC').replace('.', '').slice(0, 3).toUpperCase() }
function formatSize(bytes: number) { if (!bytes) return '0 B'; if (bytes < 1024) return `${bytes} B`; if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`; if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`; return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB` }
function formatDate(value?: string) { return value ? new Date(value).toLocaleDateString('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' }) : '—' }
function errorMessage(error: unknown, fallback: string) { if (typeof error !== 'object' || !error) return fallback; const value = error as { message?: string; response?: { data?: { error?: string; message?: string } } }; return value.response?.data?.error || value.response?.data?.message || value.message || fallback }

watch(spaceId, loadData, { immediate: true })
</script>

<style scoped>
.sdp-space-detail { min-height: 100dvh; color: var(--ink-900); background: var(--ink-100); font-family: var(--font-body); }
.sdp-space-detail__shell { display: grid; gap: var(--space-6); width: min(100%, var(--content-max-width)); margin-inline: auto; padding: var(--space-8); }
.sdp-space-detail__hero, .sdp-space-detail__panel-head, .sdp-space-detail__hero-actions, .sdp-space-detail__document-list article, .sdp-space-detail__members li, .sdp-space-detail__form-actions { display: flex; align-items: center; }
.sdp-space-detail__hero { justify-content: space-between; gap: var(--space-8); padding: var(--space-8); border: 1px solid var(--ink-800); border-radius: var(--radius-xl); color: var(--ink-50); background: radial-gradient(circle at 86% 20%, var(--brand-400), transparent 28%), linear-gradient(135deg, var(--ink-950), var(--brand-900)); box-shadow: var(--shadow-md); }
.sdp-space-detail__panel-head p { color: var(--brand-400); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); letter-spacing: .08em; text-transform: uppercase; }
.sdp-space-detail__hero p { color: var(--brand-200); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); letter-spacing: .08em; text-transform: uppercase; }
.sdp-space-detail__hero h1 { margin-top: var(--space-2); font-family: var(--font-display); font-size: var(--text-4xl); line-height: var(--leading-tight); }
.sdp-space-detail__hero > div:first-child > span { display: block; max-width: 60ch; margin-top: var(--space-3); color: var(--ink-200); line-height: var(--leading-relaxed); }
.sdp-space-detail__hero-actions { flex: 0 0 auto; flex-wrap: wrap; justify-content: flex-end; gap: var(--space-3); }
.sdp-space-detail__hero-actions > span { padding: var(--space-2) var(--space-3); border: 1px solid var(--brand-700); border-radius: var(--radius-pill); color: var(--brand-100); background: var(--brand-900); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-space-detail__metrics { display: grid; grid-template-columns: repeat(4, minmax(var(--space-0), 1fr)); gap: var(--space-4); }
.sdp-space-detail__metrics article { padding: var(--space-5); border: 1px solid var(--ink-200); border-radius: var(--radius-md); background: var(--ink-50); box-shadow: var(--shadow-xs); }
.sdp-space-detail__metrics span { display: block; color: var(--ink-600); font-size: var(--text-xs); }
.sdp-space-detail__metrics small { display: block; color: var(--brand-700); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-space-detail__metrics strong { display: block; margin: var(--space-2) var(--space-0); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-2xl); }
.sdp-space-detail__content-grid { display: grid; grid-template-columns: minmax(var(--space-0), 1.35fr) minmax(20rem, .65fr); gap: var(--space-4); align-items: start; }
.sdp-space-detail__side { display: grid; gap: var(--space-4); }
.sdp-space-detail__panel { min-width: var(--space-0); overflow: hidden; border: 1px solid var(--ink-200); border-radius: var(--radius-lg); background: var(--ink-50); box-shadow: var(--shadow-xs); }
.sdp-space-detail__panel-head { justify-content: space-between; gap: var(--space-4); padding: var(--space-5); border-bottom: 1px solid var(--ink-200); }
.sdp-space-detail__panel-head h2 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-xl); }
.sdp-space-detail__panel-head > button { min-width: var(--space-8); min-height: var(--space-8); padding: var(--space-2) var(--space-3); border: 0; border-radius: var(--radius-sm); color: var(--brand-800); background: transparent; font: var(--font-weight-semibold) var(--text-xs)/var(--leading-tight) var(--font-body); cursor: pointer; }
.sdp-space-detail__document-list { padding: var(--space-2) var(--space-5); }
.sdp-space-detail__document-list article { gap: var(--space-3); padding: var(--space-4) var(--space-0); border-bottom: 1px solid var(--ink-200); }
.sdp-space-detail__document-list article:last-child { border-bottom: 0; }
.sdp-space-detail__document-list article > span { width: var(--space-10); height: var(--space-10); display: grid; flex: 0 0 var(--space-10); place-items: center; border-radius: var(--radius-sm); color: var(--brand-900); background: var(--brand-100); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-bold); }
.sdp-space-detail__document-list article div { min-width: var(--space-0); flex: 1; }
.sdp-space-detail__document-list strong, .sdp-space-detail__document-list small { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sdp-space-detail__document-list strong { color: var(--ink-900); font-size: var(--text-sm); }
.sdp-space-detail__document-list small, .sdp-space-detail__document-list time { color: var(--ink-500); font-size: var(--text-xs); }
.sdp-space-detail__summary { padding: var(--space-3) var(--space-5); }
.sdp-space-detail__summary div { display: flex; justify-content: space-between; gap: var(--space-4); padding: var(--space-3) var(--space-0); border-bottom: 1px solid var(--ink-200); }
.sdp-space-detail__summary div:last-child { border-bottom: 0; }
.sdp-space-detail__summary dt { color: var(--ink-600); font-size: var(--text-xs); }
.sdp-space-detail__summary dd { max-width: 65%; overflow-wrap: anywhere; color: var(--ink-900); font-size: var(--text-sm); font-weight: var(--font-weight-semibold); text-align: right; }
.sdp-space-detail__form { display: grid; gap: var(--space-4); padding: var(--space-5); border-top: 1px solid var(--ink-200); background: var(--ink-100); }
.sdp-space-detail__form label { display: grid; gap: var(--space-2); }
.sdp-space-detail__form label span { color: var(--ink-800); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-space-detail__form input, .sdp-space-detail__form textarea, .sdp-space-detail__form select { width: 100%; min-height: var(--space-10); padding: var(--space-2) var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-900); background: var(--ink-50); font: var(--text-sm) var(--font-body); }
.sdp-space-detail__form textarea { min-height: var(--space-20); resize: vertical; }
.sdp-space-detail__form-actions { justify-content: flex-end; gap: var(--space-3); }
.sdp-space-detail__form-message { padding: var(--space-3); border-radius: var(--radius-sm); color: var(--ink-800); background: var(--ink-200); font-size: var(--text-xs); }
.sdp-space-detail__members { padding: var(--space-2) var(--space-5); list-style: none; }
.sdp-space-detail__members li { gap: var(--space-3); padding: var(--space-3) var(--space-0); border-bottom: 1px solid var(--ink-200); }
.sdp-space-detail__members li:last-child { border-bottom: 0; }
.sdp-space-detail__members li > span { width: var(--space-8); height: var(--space-8); display: grid; flex: 0 0 var(--space-8); place-items: center; border-radius: var(--radius-pill); color: var(--brand-900); background: var(--brand-100); font-weight: var(--font-weight-bold); }
.sdp-space-detail__members strong, .sdp-space-detail__members small { display: block; }
.sdp-space-detail__members strong { color: var(--ink-900); font-size: var(--text-sm); }
.sdp-space-detail__members small { margin-top: var(--space-1); color: var(--ink-500); font-size: var(--text-xs); }
.sdp-space-detail button:focus-visible, .sdp-space-detail input:focus-visible, .sdp-space-detail textarea:focus-visible, .sdp-space-detail select:focus-visible { outline: 2px solid var(--brand-500); outline-offset: 2px; }
@media (max-width: 64rem) { .sdp-space-detail__hero { align-items: flex-start; flex-direction: column; } .sdp-space-detail__hero-actions { justify-content: flex-start; } .sdp-space-detail__metrics { grid-template-columns: repeat(2, 1fr); } .sdp-space-detail__content-grid { grid-template-columns: 1fr; } }
@media (max-width: 48rem) { .sdp-space-detail__shell { padding: var(--space-6); } }
@media (max-width: 40rem) { .sdp-space-detail__shell { padding: var(--space-4); } .sdp-space-detail__hero { padding: var(--space-6); } .sdp-space-detail__hero h1 { font-size: var(--text-3xl); } .sdp-space-detail__hero-actions, .sdp-space-detail__hero-actions :deep(.sdp-button) { width: 100%; } .sdp-space-detail__hero-actions { align-items: stretch; flex-direction: column; } .sdp-space-detail__metrics { grid-template-columns: 1fr; } .sdp-space-detail__document-list article { align-items: flex-start; } .sdp-space-detail__document-list time { display: none; } .sdp-space-detail__form-actions { align-items: stretch; flex-direction: column-reverse; } .sdp-space-detail__form-actions :deep(.sdp-button) { width: 100%; } }
</style>
