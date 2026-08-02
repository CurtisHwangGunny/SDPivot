<template>
  <SdpSidebarLayout>
    <div class="sdp-space-home">
      <div class="sdp-space-home__shell">
        <section class="sdp-space-home__hero" aria-labelledby="space-home-title">
          <div class="sdp-space-home__hero-copy">
            <p>Knowledge Workspace</p>
            <h1 id="space-home-title">知识空间</h1>
            <span>将分散的资料整理成清晰、可协作、可持续复用的知识结构。</span>
          </div>
          <SdpButton size="lg" aria-label="新建知识空间" @click="openCreateDialog">
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M12 5v14M5 12h14" />
            </svg>
            新建空间
          </SdpButton>
        </section>

        <section class="sdp-space-home__kpis" aria-label="知识空间数据概览">
          <article v-for="item in kpis" :key="item.label" class="sdp-space-home__kpi">
            <span class="sdp-space-home__kpi-icon" aria-hidden="true">
              <svg viewBox="0 0 24 24"><path :d="item.icon" /></svg>
            </span>
            <div>
              <p>{{ item.label }}</p>
              <strong>{{ loading ? '—' : item.value }}</strong>
            </div>
          </article>
        </section>

        <SdpNotice
          v-model:visible="tipVisible"
          title="建议先创建知识空间结构"
          type="info"
          closable
        >
          按部门、项目或业务场景拆分空间，再导入对应文档，后续检索和协作会更清晰。
        </SdpNotice>

        <section class="sdp-space-home__spaces" aria-labelledby="space-list-title">
          <header class="sdp-space-home__section-header">
            <div>
              <p>Workspace Directory</p>
              <h2 id="space-list-title">空间列表</h2>
            </div>

            <div class="sdp-space-home__controls">
              <label class="sdp-space-home__search" for="space-search">
                <svg viewBox="0 0 24 24" aria-hidden="true">
                  <circle cx="11" cy="11" r="7" />
                  <path d="m16 16 5 5" />
                </svg>
                <span class="sr-only">搜索知识空间</span>
                <input
                  id="space-search"
                  v-model="searchQuery"
                  type="search"
                  placeholder="搜索空间名称或描述"
                  aria-label="搜索知识空间"
                >
              </label>

              <div
                ref="tabListRef"
                class="sdp-space-home__tabs"
                role="tablist"
                aria-label="筛选知识空间"
                @keydown="handleTabKeydown"
              >
                <button
                  v-for="tab in tabs"
                  :id="`space-tab-${tab.value}`"
                  :key="tab.value"
                  type="button"
                  role="tab"
                  :tabindex="activeTab === tab.value ? 0 : -1"
                  :aria-selected="activeTab === tab.value"
                  :aria-controls="'space-grid-panel'"
                  :class="{ 'sdp-space-home__tab--active': activeTab === tab.value }"
                  @click="activeTab = tab.value"
                >
                  {{ tab.label }}
                </button>
              </div>
            </div>
          </header>

          <div
            id="space-grid-panel"
            role="tabpanel"
            :aria-labelledby="`space-tab-${activeTab}`"
          >
            <SdpSkeleton v-if="loading" variant="grid" :count="4" />

            <SdpErrorState
              v-else-if="loadError"
              type="network"
              title="空间列表加载失败"
              :description="loadError"
              retryable
              @retry="loadSpaces"
            />

            <SdpEmptyState
              v-else-if="filteredSpaces.length === 0"
              :title="spaces.length === 0 ? '还没有知识空间' : '没有匹配的空间'"
              :description="spaces.length === 0 ? '创建第一个空间，开始整理团队知识。' : '尝试调整搜索词或切换筛选条件。'"
            >
              <template v-if="spaces.length === 0" #actions>
                <SdpButton aria-label="创建第一个知识空间" @click="openCreateDialog">创建空间</SdpButton>
              </template>
            </SdpEmptyState>

            <div
              v-else
              ref="gridRef"
              class="sdp-space-home__grid"
              role="list"
              aria-label="知识空间列表"
              @keydown="handleGridKeydown"
            >
              <article
                v-for="space in visibleSpaces"
                :key="space.id"
                class="sdp-space-home__card"
                role="listitem"
                tabindex="0"
                :aria-label="`${space.name}，${visibilityLabel(space.visibility)}`"
                @keydown.enter.self="goDetail(space.id)"
                @keydown.space.self.prevent="goDetail(space.id)"
              >
                <div class="sdp-space-home__card-head">
                  <span class="sdp-space-home__space-icon" aria-hidden="true">
                    <svg viewBox="0 0 24 24">
                      <path d="M3 6.75A1.75 1.75 0 0 1 4.75 5h5l2 2h7.5A1.75 1.75 0 0 1 21 8.75v8.5A1.75 1.75 0 0 1 19.25 19H4.75A1.75 1.75 0 0 1 3 17.25V6.75Z" />
                    </svg>
                  </span>
                  <span class="sdp-space-home__badge">{{ visibilityLabel(space.visibility) }}</span>
                </div>

                <div class="sdp-space-home__card-copy">
                  <h3>{{ space.name }}</h3>
                  <p>{{ space.description || '这个知识空间还没有补充描述，可进入空间继续完善。' }}</p>
                </div>

                <footer class="sdp-space-home__card-footer">
                  <span>ID · {{ space.id }}</span>
                  <SdpButton
                    variant="ghost"
                    size="sm"
                    :aria-label="`进入空间 ${space.name}`"
                    @click="goDetail(space.id)"
                  >
                    进入
                    <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m9 5 7 7-7 7" /></svg>
                  </SdpButton>
                </footer>
              </article>
            </div>
          </div>

          <button
            v-if="filteredSpaces.length > collapsedCount"
            class="sdp-space-home__show-more"
            type="button"
            :aria-expanded="showAll"
            aria-controls="space-grid-panel"
            :aria-label="showAll ? '收起空间列表' : '展开全部空间'"
            @click="showAll = !showAll"
          >
            {{ showAll ? '收起列表' : '展开全部' }}
            <svg :class="{ 'sdp-space-home__show-more-icon--open': showAll }" viewBox="0 0 24 24" aria-hidden="true">
              <path d="m6 9 6 6 6-6" />
            </svg>
          </button>
        </section>
      </div>

      <div
        v-if="showCreate"
        class="sdp-space-home__dialog-backdrop"
        role="presentation"
        @mousedown.self="closeCreateDialog"
      >
        <section
          ref="dialogRef"
          class="sdp-space-home__dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby="create-space-title"
          aria-describedby="create-space-description"
          @keydown.esc="closeCreateDialog"
          @keydown.tab="trapDialogFocus"
        >
          <header class="sdp-space-home__dialog-header">
            <div>
              <p>New Workspace</p>
              <h2 id="create-space-title">新建知识空间</h2>
            </div>
            <button type="button" aria-label="关闭新建空间对话框" @click="closeCreateDialog">
              <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18" /></svg>
            </button>
          </header>

          <p id="create-space-description" class="sdp-space-home__dialog-description">
            建立清晰的知识边界，创建后即可继续导入文档并邀请成员。
          </p>

          <form class="sdp-space-home__form" @submit.prevent="handleCreate">
            <label for="space-name">
              <span>空间名称</span>
              <input
                id="space-name"
                ref="nameInputRef"
                v-model.trim="form.name"
                type="text"
                maxlength="30"
                required
                autocomplete="off"
                placeholder="例如：产品规范库"
                aria-label="空间名称"
              >
            </label>

            <label for="space-description">
              <span>空间描述</span>
              <textarea
                id="space-description"
                v-model.trim="form.description"
                rows="3"
                placeholder="说明空间用途、资料范围和使用对象"
                aria-label="空间描述"
              />
            </label>

            <label for="space-visibility">
              <span>可见性</span>
              <select id="space-visibility" v-model="form.visibility" aria-label="空间可见性">
                <option value="private">私密</option>
                <option value="team">团队可见</option>
                <option value="enterprise">企业可见</option>
              </select>
            </label>

            <p v-if="createError" class="sdp-space-home__form-error" role="alert">{{ createError }}</p>

            <div class="sdp-space-home__dialog-actions">
              <SdpButton variant="secondary" aria-label="取消创建空间" @click="closeCreateDialog">取消</SdpButton>
              <SdpButton
                :loading="creating"
                :disabled="!form.name"
                aria-label="确认创建知识空间"
                @click="handleCreate"
              >
                创建空间
              </SdpButton>
            </div>
          </form>
        </section>
      </div>
    </div>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { createSpace, listSpaces, type Space } from '@/api/spaces'
import { SdpButton, SdpEmptyState, SdpErrorState, SdpNotice, SdpSkeleton } from '@/components/design'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'
import { useAuthStore } from '@/stores/auth'

type SpaceFilter = 'all' | 'mine' | 'team'
type SpaceWithCounts = Space & {
  document_count?: number
  member_count?: number
}

const collapsedCount = 4
const tabs: Array<{ label: string; value: SpaceFilter }> = [
  { label: '全部', value: 'all' },
  { label: '我的空间', value: 'mine' },
  { label: '团队空间', value: 'team' },
]

const router = useRouter()
const authStore = useAuthStore()
const spaces = ref<SpaceWithCounts[]>([])
const loading = ref(true)
const loadError = ref('')
const createError = ref('')
const creating = ref(false)
const showCreate = ref(false)
const showAll = ref(false)
const tipVisible = ref(true)
const searchQuery = ref('')
const activeTab = ref<SpaceFilter>('all')
const tabListRef = ref<HTMLElement | null>(null)
const gridRef = ref<HTMLElement | null>(null)
const dialogRef = ref<HTMLElement | null>(null)
const nameInputRef = ref<HTMLInputElement | null>(null)
const form = ref({ name: '', description: '', visibility: 'private' })

const currentUserId = computed(() => String(authStore.user?.id || authStore.user?.user_id || ''))
const filteredSpaces = computed(() => {
  const query = searchQuery.value.trim().toLocaleLowerCase()
  return spaces.value.filter((space) => {
    const matchesSearch = !query || `${space.name} ${space.description || ''}`.toLocaleLowerCase().includes(query)
    const matchesTab = activeTab.value === 'all'
      || (activeTab.value === 'mine' && Boolean(currentUserId.value) && space.owner_id === currentUserId.value)
      || (activeTab.value === 'team' && space.visibility !== 'private')
    return matchesSearch && matchesTab
  })
})
const visibleSpaces = computed(() => showAll.value ? filteredSpaces.value : filteredSpaces.value.slice(0, collapsedCount))
const documentCount = computed(() => spaces.value.reduce((total, space) => total + (space.document_count || 0), 0))
const memberCount = computed(() => spaces.value.reduce((total, space) => total + (space.member_count || 0), 0))
const weeklyNewCount = computed(() => {
  const start = Date.now() - 7 * 24 * 60 * 60 * 1000
  return spaces.value.filter((space) => new Date(space.created_at).getTime() >= start).length
})
const kpis = computed(() => [
  { label: '空间总数', value: spaces.value.length, icon: 'M3 6.75A1.75 1.75 0 0 1 4.75 5h5l2 2h7.5A1.75 1.75 0 0 1 21 8.75v8.5A1.75 1.75 0 0 1 19.25 19H4.75A1.75 1.75 0 0 1 3 17.25V6.75Z' },
  { label: '文档总数', value: documentCount.value, icon: 'M6 3h8l4 4v14H6V3Zm8 0v5h5M9 13h6M9 17h6' },
  { label: '协作成员', value: memberCount.value, icon: 'M16 20v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2m7-10a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm13 10v-2a4 4 0 0 0-3-3.87m-2-11.96a4 4 0 0 1 0 7.75' },
  { label: '本周新增', value: weeklyNewCount.value, icon: 'M12 2v20M2 12h20M5 5l14 14M19 5 5 19' },
])

async function loadSpaces() {
  loading.value = true
  loadError.value = ''
  try {
    const response = await listSpaces()
    spaces.value = response.data.spaces || []
  } catch (error: unknown) {
    spaces.value = []
    loadError.value = errorMessage(error, '请检查网络连接后重试。')
  } finally {
    loading.value = false
  }
}

function errorMessage(error: unknown, fallback: string) {
  if (typeof error !== 'object' || error === null) return fallback
  const requestError = error as { message?: string; response?: { data?: { error?: string; message?: string } } }
  return requestError.response?.data?.error || requestError.response?.data?.message || requestError.message || fallback
}

function visibilityLabel(value?: string) {
  if (value === 'team') return '团队可见'
  if (value === 'enterprise') return '企业可见'
  return '私密'
}

function goDetail(id: string) {
  router.push(`/spaces/${id}`)
}

async function openCreateDialog() {
  createError.value = ''
  showCreate.value = true
  await nextTick()
  nameInputRef.value?.focus()
}

function closeCreateDialog() {
  if (creating.value) return
  showCreate.value = false
}

async function handleCreate() {
  if (!form.value.name || creating.value) return
  creating.value = true
  createError.value = ''
  try {
    await createSpace(form.value)
    form.value = { name: '', description: '', visibility: 'private' }
    showCreate.value = false
    await loadSpaces()
  } catch (error: unknown) {
    createError.value = errorMessage(error, '创建失败，请稍后重试。')
  } finally {
    creating.value = false
  }
}

function handleTabKeydown(event: KeyboardEvent) {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  const buttons = Array.from(tabListRef.value?.querySelectorAll<HTMLButtonElement>('[role="tab"]') || [])
  const currentIndex = buttons.indexOf(document.activeElement as HTMLButtonElement)
  if (currentIndex < 0) return
  event.preventDefault()
  let nextIndex = currentIndex
  if (event.key === 'Home') nextIndex = 0
  if (event.key === 'End') nextIndex = buttons.length - 1
  if (event.key === 'ArrowRight') nextIndex = (currentIndex + 1) % buttons.length
  if (event.key === 'ArrowLeft') nextIndex = (currentIndex - 1 + buttons.length) % buttons.length
  activeTab.value = tabs[nextIndex].value
  buttons[nextIndex]?.focus()
}

function handleGridKeydown(event: KeyboardEvent) {
  if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key)) return
  const cards = Array.from(gridRef.value?.querySelectorAll<HTMLElement>('.sdp-space-home__card') || [])
  const currentIndex = cards.indexOf(document.activeElement as HTMLElement)
  if (currentIndex < 0) return
  event.preventDefault()
  let nextIndex = currentIndex
  if (event.key === 'Home') nextIndex = 0
  if (event.key === 'End') nextIndex = cards.length - 1
  if (event.key === 'ArrowLeft') nextIndex = Math.max(0, currentIndex - 1)
  if (event.key === 'ArrowRight') nextIndex = Math.min(cards.length - 1, currentIndex + 1)
  if (event.key === 'ArrowUp') nextIndex = Math.max(0, currentIndex - 2)
  if (event.key === 'ArrowDown') nextIndex = Math.min(cards.length - 1, currentIndex + 2)
  cards[nextIndex]?.focus()
}

function trapDialogFocus(event: KeyboardEvent) {
  const focusable = Array.from(dialogRef.value?.querySelectorAll<HTMLElement>(
    'button:not(:disabled), input:not(:disabled), textarea:not(:disabled), select:not(:disabled)',
  ) || [])
  if (focusable.length === 0) return
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

watch([searchQuery, activeTab], () => {
  showAll.value = false
})

onMounted(loadSpaces)
</script>

<style scoped>
.sdp-space-home {
  min-height: 100dvh;
  color: var(--ink-900);
  background: var(--ink-100);
  font-family: var(--font-body);
}

.sdp-space-home__shell {
  display: grid;
  gap: var(--space-6);
  width: min(100%, var(--content-max-width));
  margin-inline: auto;
  padding: var(--space-8);
}

.sdp-space-home__hero {
  position: relative;
  display: flex;
  min-height: var(--space-24);
  align-items: center;
  justify-content: space-between;
  gap: var(--space-8);
  overflow: hidden;
  padding: var(--space-10);
  border-radius: var(--radius-xl);
  color: var(--ink-50);
  background:
    radial-gradient(circle at top right, var(--brand-500), transparent 42%),
    linear-gradient(135deg, var(--brand-900), var(--brand-700));
  box-shadow: var(--shadow-md);
}

.sdp-space-home__hero::after {
  position: absolute;
  inset: auto calc(var(--space-12) * -1) calc(var(--space-16) * -1) auto;
  width: var(--space-24);
  height: var(--space-24);
  border: var(--space-6) solid var(--brand-600);
  border-radius: var(--radius-pill);
  content: '';
  opacity: 0.65;
}

.sdp-space-home__hero-copy,
.sdp-space-home__hero :deep(.sdp-button) {
  position: relative;
  z-index: var(--z-base);
}

.sdp-space-home__hero-copy p,
.sdp-space-home__section-header > div:first-child p,
.sdp-space-home__dialog-header p {
  color: var(--brand-200);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  font-weight: var(--font-weight-semibold);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.sdp-space-home__hero h1 {
  margin-top: var(--space-2);
  font-family: var(--font-display);
  font-size: var(--text-4xl);
  font-weight: var(--font-weight-bold);
  line-height: var(--leading-tight);
}

.sdp-space-home__hero-copy > span {
  display: block;
  max-width: 40rem;
  margin-top: var(--space-3);
  color: var(--brand-100);
  font-size: var(--text-base);
  line-height: var(--leading-relaxed);
}

.sdp-space-home__hero :deep(.sdp-button) svg,
.sdp-space-home__card-footer :deep(.sdp-button) svg {
  width: var(--space-5);
  height: var(--space-5);
  fill: none;
  stroke: currentColor;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 2;
}

.sdp-space-home__kpis {
  display: grid;
  grid-template-columns: repeat(4, minmax(var(--space-0), 1fr));
  gap: var(--space-4);
}

.sdp-space-home__kpi {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-5);
  border: 1px solid var(--ink-200);
  border-radius: var(--radius-md);
  background: var(--ink-50);
  box-shadow: var(--shadow-xs);
}

.sdp-space-home__kpi-icon,
.sdp-space-home__space-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--brand-800);
  background: var(--brand-100);
}

.sdp-space-home__kpi-icon {
  width: var(--space-10);
  height: var(--space-10);
  flex: 0 0 var(--space-10);
  border-radius: var(--radius-sm);
}

.sdp-space-home__kpi-icon svg {
  width: var(--space-5);
  height: var(--space-5);
  fill: none;
  stroke: currentColor;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 1.8;
}

.sdp-space-home__kpi p {
  color: var(--ink-600);
  font-size: var(--text-xs);
  font-weight: var(--font-weight-medium);
}

.sdp-space-home__kpi strong {
  display: block;
  margin-top: var(--space-1);
  color: var(--ink-950);
  font-family: var(--font-display);
  font-size: var(--text-2xl);
  font-weight: var(--font-weight-bold);
  line-height: var(--leading-tight);
}

.sdp-space-home__spaces {
  display: grid;
  gap: var(--space-6);
}

.sdp-space-home__section-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--space-6);
}

.sdp-space-home__section-header > div:first-child p,
.sdp-space-home__dialog-header p {
  color: var(--brand-700);
}

.sdp-space-home__section-header h2,
.sdp-space-home__dialog-header h2 {
  margin-top: var(--space-1);
  color: var(--ink-950);
  font-family: var(--font-display);
  font-size: var(--text-2xl);
  font-weight: var(--font-weight-bold);
  line-height: var(--leading-tight);
}

.sdp-space-home__controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-3);
}

.sdp-space-home__search {
  display: flex;
  min-width: calc(var(--space-24) * 2);
  min-height: var(--space-10);
  align-items: center;
  gap: var(--space-2);
  padding-inline: var(--space-3);
  border: 1px solid var(--ink-300);
  border-radius: var(--radius-sm);
  color: var(--ink-600);
  background: var(--ink-50);
}

.sdp-space-home__search:focus-within {
  border-color: var(--brand-600);
  outline: 2px solid var(--brand-100);
  outline-offset: 2px;
}

.sdp-space-home__search svg {
  width: var(--space-5);
  height: var(--space-5);
  flex: 0 0 var(--space-5);
  fill: none;
  stroke: currentColor;
  stroke-linecap: round;
  stroke-width: 1.8;
}

.sdp-space-home__search input {
  width: 100%;
  min-height: var(--space-8);
  border: 0;
  outline: 0;
  color: var(--ink-900);
  background: transparent;
  font-family: var(--font-body);
  font-size: var(--text-sm);
}

.sdp-space-home__search input::placeholder {
  color: var(--ink-500);
}

.sdp-space-home__tabs {
  display: inline-flex;
  gap: var(--space-1);
  padding: var(--space-1);
  border: 1px solid var(--ink-200);
  border-radius: var(--radius-sm);
  background: var(--ink-200);
}

.sdp-space-home__tabs button {
  min-height: var(--space-8);
  padding: var(--space-2) var(--space-3);
  border: 0;
  border-radius: var(--radius-xs);
  color: var(--ink-700);
  background: transparent;
  font-family: var(--font-body);
  font-size: var(--text-sm);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
}

.sdp-space-home__tabs button:hover {
  color: var(--ink-950);
}

.sdp-space-home__tabs .sdp-space-home__tab--active {
  color: var(--brand-900);
  background: var(--ink-50);
  box-shadow: var(--shadow-xs);
}

.sdp-space-home__tabs button:focus-visible,
.sdp-space-home__card:focus-visible,
.sdp-space-home__show-more:focus-visible,
.sdp-space-home__dialog-header button:focus-visible,
.sdp-space-home__form input:focus-visible,
.sdp-space-home__form textarea:focus-visible,
.sdp-space-home__form select:focus-visible {
  outline: 2px solid var(--brand-500);
  outline-offset: 2px;
}

.sdp-space-home__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(var(--space-0), 1fr));
  gap: var(--space-4);
}

.sdp-space-home__card {
  display: grid;
  min-width: var(--space-0);
  gap: var(--space-5);
  padding: var(--space-6);
  border: 1px solid var(--ink-200);
  border-radius: var(--radius-lg);
  background: var(--ink-50);
  box-shadow: var(--shadow-xs);
  transition:
    border-color var(--duration-normal) var(--ease-out-quart),
    box-shadow var(--duration-normal) var(--ease-out-quart),
    transform var(--duration-normal) var(--ease-out-quart);
}

.sdp-space-home__card:hover,
.sdp-space-home__card:focus-visible {
  border-color: var(--brand-300);
  box-shadow: var(--shadow-md);
  transform: translateY(calc(var(--space-1) * -1));
}

.sdp-space-home__card-head,
.sdp-space-home__card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
}

.sdp-space-home__space-icon {
  width: var(--space-12);
  height: var(--space-12);
  border-radius: var(--radius-md);
}

.sdp-space-home__space-icon svg {
  width: var(--space-6);
  height: var(--space-6);
  fill: none;
  stroke: currentColor;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 1.8;
}

.sdp-space-home__badge {
  padding: var(--space-1) var(--space-3);
  border: 1px solid var(--brand-200);
  border-radius: var(--radius-pill);
  color: var(--brand-800);
  background: var(--brand-50);
  font-size: var(--text-xs);
  font-weight: var(--font-weight-semibold);
}

.sdp-space-home__card-copy {
  min-height: var(--space-20);
}

.sdp-space-home__card-copy h3 {
  overflow: hidden;
  color: var(--ink-950);
  font-family: var(--font-display);
  font-size: var(--text-xl);
  font-weight: var(--font-weight-semibold);
  line-height: var(--leading-tight);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sdp-space-home__card-copy p {
  display: -webkit-box;
  overflow: hidden;
  margin-top: var(--space-3);
  color: var(--ink-600);
  font-size: var(--text-sm);
  line-height: var(--leading-relaxed);
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.sdp-space-home__card-footer {
  padding-top: var(--space-4);
  border-top: 1px solid var(--ink-200);
}

.sdp-space-home__card-footer > span {
  overflow: hidden;
  color: var(--ink-500);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sdp-space-home__card-footer :deep(.sdp-button--ghost) {
  border-color: transparent;
  color: var(--brand-800);
  background: transparent;
}

.sdp-space-home__show-more {
  display: inline-flex;
  min-height: var(--space-10);
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  justify-self: center;
  padding: var(--space-2) var(--space-5);
  border: 1px solid var(--ink-300);
  border-radius: var(--radius-pill);
  color: var(--ink-800);
  background: var(--ink-50);
  font-family: var(--font-body);
  font-size: var(--text-sm);
  font-weight: var(--font-weight-semibold);
  cursor: pointer;
}

.sdp-space-home__show-more:hover {
  border-color: var(--brand-500);
  color: var(--brand-800);
  background: var(--brand-50);
}

.sdp-space-home__show-more svg,
.sdp-space-home__dialog-header button svg {
  width: var(--space-4);
  height: var(--space-4);
  fill: none;
  stroke: currentColor;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 2;
  transition: transform var(--duration-normal) var(--ease-out-quart);
}

.sdp-space-home__show-more-icon--open {
  transform: rotate(180deg);
}

.sdp-space-home__dialog-backdrop {
  position: fixed;
  inset: var(--space-0);
  z-index: var(--z-modal);
  display: grid;
  place-items: center;
  padding: var(--space-6);
  background: color-mix(in oklch, var(--ink-950) 72%, transparent);
}

.sdp-space-home__dialog {
  width: min(100%, calc(var(--space-24) * 6));
  max-height: calc(100dvh - var(--space-12));
  overflow-y: auto;
  padding: var(--space-8);
  border: 1px solid var(--ink-200);
  border-radius: var(--radius-lg);
  background: var(--ink-50);
  box-shadow: var(--shadow-xl);
}

.sdp-space-home__dialog-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-4);
}

.sdp-space-home__dialog-header button {
  display: inline-flex;
  width: var(--space-10);
  height: var(--space-10);
  flex: 0 0 var(--space-10);
  align-items: center;
  justify-content: center;
  border: 1px solid var(--ink-300);
  border-radius: var(--radius-sm);
  color: var(--ink-700);
  background: var(--ink-50);
  cursor: pointer;
}

.sdp-space-home__dialog-header button:hover {
  color: var(--ink-950);
  background: var(--ink-100);
}

.sdp-space-home__dialog-description {
  margin-top: var(--space-4);
  color: var(--ink-600);
  font-size: var(--text-sm);
  line-height: var(--leading-relaxed);
}

.sdp-space-home__form {
  display: grid;
  gap: var(--space-5);
  margin-top: var(--space-6);
}

.sdp-space-home__form label {
  display: grid;
  gap: var(--space-2);
}

.sdp-space-home__form label > span {
  color: var(--ink-800);
  font-size: var(--text-sm);
  font-weight: var(--font-weight-semibold);
}

.sdp-space-home__form input,
.sdp-space-home__form textarea,
.sdp-space-home__form select {
  width: 100%;
  min-height: var(--space-12);
  padding: var(--space-3);
  border: 1px solid var(--ink-300);
  border-radius: var(--radius-sm);
  color: var(--ink-900);
  background: var(--ink-50);
  font-family: var(--font-body);
  font-size: var(--text-sm);
}

.sdp-space-home__form textarea {
  min-height: var(--space-24);
  resize: vertical;
}

.sdp-space-home__form input:hover,
.sdp-space-home__form textarea:hover,
.sdp-space-home__form select:hover {
  border-color: var(--ink-500);
}

.sdp-space-home__form-error {
  padding: var(--space-3);
  border: 1px solid var(--danger-300);
  border-radius: var(--radius-sm);
  color: var(--ink-900);
  background: var(--danger-50);
  font-size: var(--text-sm);
}

.sdp-space-home__dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  padding-top: var(--space-3);
}

@media (max-width: 64rem) {
  .sdp-space-home__kpis {
    grid-template-columns: repeat(2, minmax(var(--space-0), 1fr));
  }

  .sdp-space-home__section-header {
    align-items: stretch;
    flex-direction: column;
  }

  .sdp-space-home__controls {
    justify-content: space-between;
  }
}

@media (max-width: 48rem) {
  .sdp-space-home__shell {
    padding: var(--space-6);
  }

  .sdp-space-home__hero {
    align-items: flex-start;
    flex-direction: column;
    padding: var(--space-8);
  }

  .sdp-space-home__hero h1 {
    font-size: var(--text-3xl);
  }

  .sdp-space-home__grid {
    grid-template-columns: 1fr;
  }

  .sdp-space-home__controls,
  .sdp-space-home__search {
    width: 100%;
  }

  .sdp-space-home__controls {
    align-items: stretch;
    flex-direction: column;
  }

  .sdp-space-home__tabs {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 40rem) {
  .sdp-space-home__shell {
    gap: var(--space-5);
    padding: var(--space-4);
  }

  .sdp-space-home__hero {
    padding: var(--space-6);
    border-radius: var(--radius-lg);
  }

  .sdp-space-home__hero :deep(.sdp-button),
  .sdp-space-home__dialog-actions :deep(.sdp-button) {
    width: 100%;
  }

  .sdp-space-home__kpis {
    grid-template-columns: 1fr;
  }

  .sdp-space-home__card {
    padding: var(--space-5);
  }

  .sdp-space-home__dialog-backdrop {
    align-items: end;
    padding: var(--space-0);
  }

  .sdp-space-home__dialog {
    width: 100%;
    max-height: calc(100dvh - var(--space-6));
    padding: var(--space-6);
    border-radius: var(--radius-lg) var(--radius-lg) var(--radius-none) var(--radius-none);
  }

  .sdp-space-home__dialog-actions {
    flex-direction: column-reverse;
  }
}
</style>
