<template>
  <SdpSidebarLayout>
    <main class="sdp-global-search">
      <div class="sdp-global-search__shell">
        <header class="sdp-global-search__header">
          <p>Knowledge Discovery</p>
          <h1>全局搜索</h1>
          <span>跨知识空间检索你有权访问的文档内容。</span>
        </header>

        <form class="sdp-global-search__form" role="search" @submit.prevent="runSearch()">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <circle cx="11" cy="11" r="7" />
            <path d="m16 16 5 5" />
          </svg>
          <input
            ref="searchInput"
            v-model="query"
            type="search"
            placeholder="输入关键词，搜索全部可见空间"
            aria-label="搜索全部可见空间"
          >
          <SdpButton :loading="loading" :disabled="!query.trim()" aria-label="开始全局搜索" @click="runSearch()">搜索</SdpButton>
        </form>

        <section v-if="history.length && !hasSearched" class="sdp-global-search__history" aria-labelledby="recent-search-title">
          <div>
            <p>Recent Queries</p>
            <h2 id="recent-search-title">最近搜索</h2>
          </div>
          <div class="sdp-global-search__history-list">
            <button v-for="item in history" :key="`${item.query}-${item.searched_at}`" type="button" @click="useHistory(item.query)">
              {{ item.query }}
            </button>
          </div>
        </section>

        <SdpSkeleton v-if="loading" class="sdp-global-search__state" variant="list" :count="5" />
        <SdpErrorState v-else-if="error" class="sdp-global-search__state" type="network" title="搜索失败" :description="error" retryable @retry="runSearch()" />
        <SdpEmptyState v-else-if="hasSearched && results.length === 0" class="sdp-global-search__state" title="没有找到相关内容" description="尝试使用更短的关键词，或确认目标空间和文档对你可见。" />

        <section v-else-if="hasSearched" class="sdp-global-search__results" aria-labelledby="search-results-title">
          <header>
            <div>
              <p>Search Results</p>
              <h2 id="search-results-title">“{{ searchedQuery }}”的结果</h2>
            </div>
            <span>共 {{ total }} 条</span>
          </header>

          <article v-for="result in results" :key="result.chunk_id" class="sdp-global-search__result">
            <div class="sdp-global-search__result-meta">
              <span>{{ result.space_name }}</span>
              <span>片段 {{ result.chunk_index + 1 }}</span>
            </div>
            <h3>{{ result.document_title }}</h3>
            <p>{{ result.content }}</p>
            <RouterLink :to="`/spaces/${result.space_id}/documents`" :aria-label="`查看 ${result.document_title} 所在空间`">
              查看空间文档
              <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m9 5 7 7-7 7" /></svg>
            </RouterLink>
          </article>
        </section>
      </div>
    </main>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { getSearchHistory, globalSearch, type GlobalSearchResult, type SearchHistoryEntry } from '@/api/search'
import { SdpButton, SdpEmptyState, SdpErrorState, SdpSkeleton } from '@/components/design'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'

const route = useRoute()
const router = useRouter()
const query = ref(typeof route.query.q === 'string' ? route.query.q : '')
const searchedQuery = ref('')
const results = ref<GlobalSearchResult[]>([])
const history = ref<SearchHistoryEntry[]>([])
const total = ref(0)
const loading = ref(false)
const hasSearched = ref(false)
const error = ref('')
const searchInput = ref<HTMLInputElement | null>(null)

function errorMessage(value: unknown) {
  if (typeof value !== 'object' || value === null) return '请稍后重试。'
  const requestError = value as { message?: string; response?: { data?: { error?: string; message?: string } } }
  return requestError.response?.data?.error || requestError.response?.data?.message || requestError.message || '请稍后重试。'
}

async function loadHistory() {
  try {
    const response = await getSearchHistory()
    history.value = response.data.history || []
  } catch {
    history.value = []
  }
}

async function runSearch(nextQuery = query.value) {
  const normalized = nextQuery.trim()
  if (!normalized || loading.value) return
  query.value = normalized
  searchedQuery.value = normalized
  hasSearched.value = true
  loading.value = true
  error.value = ''
  await router.replace({ path: '/search', query: { q: normalized } })
  try {
    const response = await globalSearch(normalized)
    results.value = response.data.results || []
    total.value = response.data.total || 0
    await loadHistory()
  } catch (requestError: unknown) {
    results.value = []
    total.value = 0
    error.value = errorMessage(requestError)
  } finally {
    loading.value = false
  }
}

function useHistory(value: string) {
  query.value = value
  void runSearch(value)
}

onMounted(async () => {
  await loadHistory()
  if (query.value.trim()) await runSearch()
  else await nextTick(() => searchInput.value?.focus())
})
</script>

<style scoped>
.sdp-global-search { min-height: 100dvh; color: var(--ink-900); background: var(--ink-100); font-family: var(--font-body); }
.sdp-global-search__shell { width: min(100%, var(--content-max-width)); display: grid; gap: var(--space-6); margin-inline: auto; padding: var(--space-8); }
.sdp-global-search__header p, .sdp-global-search__history p, .sdp-global-search__results header p { margin: 0 0 var(--space-2); color: var(--brand-700); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); letter-spacing: .12em; text-transform: uppercase; }
.sdp-global-search__header h1, .sdp-global-search__history h2, .sdp-global-search__results h2 { margin: 0; color: var(--ink-950); font-family: var(--font-display); }
.sdp-global-search__header h1 { font-size: clamp(2rem, 5vw, 3.5rem); }
.sdp-global-search__header span { display: block; margin-top: var(--space-3); color: var(--ink-600); }
.sdp-global-search__form { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: var(--space-3); padding: var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-lg); background: var(--ink-50); box-shadow: var(--shadow-sm); }
.sdp-global-search__form svg { width: var(--space-6); height: var(--space-6); fill: none; stroke: var(--ink-500); stroke-width: 1.8; stroke-linecap: round; }
.sdp-global-search__form input { width: 100%; min-height: var(--space-10); border: 0; outline: 0; color: var(--ink-950); background: transparent; font: inherit; font-size: var(--text-lg); }
.sdp-global-search__history, .sdp-global-search__results { display: grid; gap: var(--space-4); }
.sdp-global-search__history-list { display: flex; flex-wrap: wrap; gap: var(--space-2); }
.sdp-global-search__history-list button { padding: var(--space-2) var(--space-4); border: 1px solid var(--ink-300); border-radius: 999px; color: var(--ink-700); background: var(--ink-50); cursor: pointer; }
.sdp-global-search__history-list button:hover { border-color: var(--brand-500); color: var(--brand-700); }
.sdp-global-search__results > header { display: flex; align-items: end; justify-content: space-between; gap: var(--space-4); }
.sdp-global-search__results > header > span { color: var(--ink-500); font-size: var(--text-sm); }
.sdp-global-search__result { display: grid; gap: var(--space-3); padding: var(--space-5); border: 1px solid var(--ink-300); border-left: 4px solid var(--brand-500); border-radius: var(--radius-md); background: var(--ink-50); }
.sdp-global-search__result-meta { display: flex; gap: var(--space-2); color: var(--ink-500); font-size: var(--text-xs); }
.sdp-global-search__result-meta span + span::before { content: '·'; margin-right: var(--space-2); }
.sdp-global-search__result h3, .sdp-global-search__result p { margin: 0; }
.sdp-global-search__result h3 { color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-xl); }
.sdp-global-search__result p { display: -webkit-box; overflow: hidden; color: var(--ink-700); line-height: var(--leading-relaxed); -webkit-box-orient: vertical; -webkit-line-clamp: 4; }
.sdp-global-search__result a { width: fit-content; display: inline-flex; align-items: center; gap: var(--space-1); color: var(--brand-700); font-weight: var(--font-weight-semibold); text-decoration: none; }
.sdp-global-search__result a svg { width: var(--space-4); height: var(--space-4); fill: none; stroke: currentColor; stroke-width: 2; }
.sdp-global-search__state { min-height: 18rem; }
@media (max-width: 720px) {
  .sdp-global-search__shell { padding: var(--space-5); }
  .sdp-global-search__form { grid-template-columns: auto minmax(0, 1fr); }
  .sdp-global-search__form :deep(button) { grid-column: 1 / -1; width: 100%; }
  .sdp-global-search__results > header { align-items: start; flex-direction: column; }
}
</style>
