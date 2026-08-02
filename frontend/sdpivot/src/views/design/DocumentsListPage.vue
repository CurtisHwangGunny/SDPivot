<template>
  <SdpSidebarLayout>
    <div class="sdp-documents-list">
      <div class="sdp-documents-list__shell">
        <header class="sdp-documents-list__header">
          <div>
            <p>Space Documents</p>
            <h1>文档列表</h1>
            <span>搜索、筛选和管理当前知识空间的文档。</span>
          </div>
          <SdpButton aria-label="上传文档" @click="openUploadDialog">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 16V4m0 0L7 9m5-5 5 5M4 15v4h16v-4" /></svg>
            上传文档
          </SdpButton>
        </header>

        <section class="sdp-documents-list__panel" aria-labelledby="documents-title">
          <header class="sdp-documents-list__panel-head">
            <div>
              <p>Document Registry</p>
              <h2 id="documents-title">空间文档</h2>
            </div>
            <span aria-live="polite">{{ loading ? '正在加载' : `共 ${total} 个文档` }}</span>
          </header>

          <div class="sdp-documents-list__filters" role="search" aria-label="筛选文档">
            <label class="sdp-documents-list__search" for="document-search">
              <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="7" /><path d="m16 16 5 5" /></svg>
              <span class="sr-only">搜索文档</span>
              <input id="document-search" v-model="searchQuery" type="search" placeholder="搜索文档名称" aria-label="搜索文档名称">
            </label>
            <label for="document-status"><span class="sr-only">按解析状态筛选</span><select id="document-status" v-model="statusFilter" aria-label="按解析状态筛选"><option value="">全部状态</option><option value="pending">等待中</option><option value="parsing">解析中</option><option value="completed">已完成</option><option value="failed">失败</option></select></label>
            <SdpButton variant="secondary" size="sm" :loading="loading" aria-label="刷新文档列表" @click="loadDocuments">刷新</SdpButton>
          </div>

          <SdpErrorState v-if="loadError" type="network" title="文档列表加载失败" :description="loadError" retryable @retry="loadDocuments" />
          <SdpSkeleton v-else-if="loading" class="sdp-documents-list__state" variant="list" :count="5" />
          <SdpEmptyState v-else-if="documents.length === 0" class="sdp-documents-list__state" :title="searchQuery || statusFilter ? '没有匹配的文档' : '空间内还没有文档'" :description="searchQuery || statusFilter ? '尝试调整搜索词或解析状态。' : '上传第一个文件，开始构建空间知识。'">
            <template v-if="!searchQuery && !statusFilter" #actions><SdpButton aria-label="上传第一个文档" @click="openUploadDialog">上传文档</SdpButton></template>
          </SdpEmptyState>

          <div v-else class="sdp-documents-list__table-wrap">
            <table>
              <caption class="sr-only">当前空间文档列表</caption>
              <thead><tr><th scope="col">文档名称</th><th scope="col">类型</th><th scope="col">大小</th><th scope="col">解析状态</th><th scope="col">更新时间</th><th scope="col">操作</th></tr></thead>
              <tbody>
                <tr v-for="document in documents" :key="document.id">
                  <td><strong>{{ document.title || document.file_name }}</strong><small>{{ document.chunk_count }} 个分块</small></td>
                  <td>{{ document.file_type || '未知' }}</td>
                  <td>{{ formatSize(document.file_size) }}</td>
                  <td><span class="sdp-documents-list__status" :class="`sdp-documents-list__status--${statusTone(document.parse_status)}`"><i aria-hidden="true" />{{ statusLabel(document.parse_status) }}</span></td>
                  <td>{{ formatDate(document.updated_at || document.created_at) }}</td>
                  <td><SdpButton variant="ghost" size="sm" :aria-label="`删除文档 ${document.title || document.file_name}`" @click="requestDelete(document)">删除</SdpButton></td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>

      <div v-if="showUpload" class="sdp-documents-list__backdrop" role="presentation" @mousedown.self="closeUploadDialog">
        <section ref="dialogRef" class="sdp-documents-list__dialog" role="dialog" aria-modal="true" aria-labelledby="upload-document-title" aria-describedby="upload-document-description" @keydown.esc="closeUploadDialog" @keydown.tab="trapDialogFocus">
          <header><div><p>Upload Document</p><h2 id="upload-document-title">上传文档</h2></div><button type="button" aria-label="关闭上传文档对话框" :disabled="uploading" @click="closeUploadDialog">关闭</button></header>
          <p id="upload-document-description">选择一个不超过 50MB 的文件，上传后系统将自动解析并建立索引。</p>
          <form @submit.prevent="handleUpload">
            <label for="document-file"><span>选择文件</span><input id="document-file" ref="fileInputRef" type="file" required :disabled="uploading" @change="selectFile"></label>
            <label for="document-tags"><span>标签（选填）</span><input id="document-tags" v-model.trim="uploadTags" type="text" :disabled="uploading" placeholder="例如：合同, 客户案例"></label>
            <p v-if="uploadError" class="sdp-documents-list__form-error" role="alert">{{ uploadError }}</p>
            <div class="sdp-documents-list__dialog-actions"><SdpButton variant="secondary" :disabled="uploading" aria-label="取消上传" @click="closeUploadDialog">取消</SdpButton><SdpButton :loading="uploading" :disabled="!uploadFile" aria-label="确认上传文档" @click="handleUpload">上传</SdpButton></div>
          </form>
        </section>
      </div>

      <SdpConfirmDialog v-model:visible="showDeleteConfirm" type="danger" title="删除文档" :message="`确定删除“${pendingDelete?.title || pendingDelete?.file_name || ''}”吗？此操作无法撤销。`" confirm-text="确认删除" cancel-text="取消" :loading="deleting" @confirm="handleDelete" />
      <p class="sr-only" aria-live="polite">{{ announcement }}</p>
    </div>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { listDocuments, uploadDocument, deleteDocument, type Document } from '@/api/documents'
import { SdpButton, SdpConfirmDialog, SdpEmptyState, SdpErrorState, SdpSkeleton } from '@/components/design'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'

const route = useRoute()
const spaceId = computed(() => String(route.params.id || ''))
const documents = ref<Document[]>([])
const total = ref(0)
const loading = ref(false)
const loadError = ref('')
const searchQuery = ref('')
const statusFilter = ref('')
const showUpload = ref(false)
const uploadFile = ref<File | null>(null)
const uploadTags = ref('')
const uploading = ref(false)
const uploadError = ref('')
const dialogRef = ref<HTMLElement | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)
const showDeleteConfirm = ref(false)
const pendingDelete = ref<Document | null>(null)
const deleting = ref(false)
const announcement = ref('')
let searchTimer: ReturnType<typeof setTimeout> | undefined

async function loadDocuments() {
  if (!spaceId.value) return
  loading.value = true
  loadError.value = ''
  try {
    const response = await listDocuments({ space_id: spaceId.value, search: searchQuery.value.trim() || undefined, parse_status: statusFilter.value || undefined, page: 1, page_size: 100 })
    documents.value = response.data.documents || []
    total.value = response.data.total || documents.value.length
  } catch (error: unknown) {
    documents.value = []
    total.value = 0
    loadError.value = errorMessage(error, '请检查网络连接后重试。')
  } finally { loading.value = false }
}

async function openUploadDialog() { uploadError.value = ''; showUpload.value = true; await nextTick(); fileInputRef.value?.focus() }
function closeUploadDialog() { if (uploading.value) return; showUpload.value = false; uploadFile.value = null; uploadTags.value = '' }
function selectFile(event: Event) { uploadFile.value = (event.target as HTMLInputElement).files?.[0] || null }
async function handleUpload() {
  if (!uploadFile.value || !spaceId.value || uploading.value) return
  uploading.value = true
  uploadError.value = ''
  try {
    await uploadDocument({ space_id: spaceId.value, file: uploadFile.value, tags: uploadTags.value || undefined })
    uploading.value = false
    closeUploadDialog()
    announcement.value = '文档上传成功'
    await loadDocuments()
  } catch (error: unknown) { uploadError.value = errorMessage(error, '上传失败，请稍后重试。') } finally { uploading.value = false }
}

function requestDelete(document: Document) { pendingDelete.value = document; showDeleteConfirm.value = true }
async function handleDelete() {
  if (!pendingDelete.value || deleting.value) return
  deleting.value = true
  try {
    await deleteDocument(pendingDelete.value.id)
    announcement.value = '文档已删除'
    showDeleteConfirm.value = false
    pendingDelete.value = null
    await loadDocuments()
  } catch (error: unknown) { announcement.value = errorMessage(error, '删除失败，请稍后重试。') } finally { deleting.value = false }
}

function statusLabel(status: string) { return ({ pending: '等待中', parsing: '解析中', processing: '解析中', completed: '已完成', failed: '失败' } as Record<string, string>)[status] || status || '未知' }
function statusTone(status: string) { if (status === 'completed') return 'complete'; if (status === 'failed') return 'failed'; if (status === 'parsing' || status === 'processing') return 'active'; return 'pending' }
function formatSize(bytes: number) { if (!bytes) return '—'; if (bytes < 1024) return `${bytes} B`; if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`; return `${(bytes / 1024 / 1024).toFixed(1)} MB` }
function formatDate(value: string) { return value ? new Date(value).toLocaleString('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }) : '—' }
function errorMessage(error: unknown, fallback: string) { if (typeof error !== 'object' || !error) return fallback; const value = error as { message?: string; response?: { data?: { error?: string; message?: string } } }; return value.response?.data?.error || value.response?.data?.message || value.message || fallback }
function trapDialogFocus(event: KeyboardEvent) { const elements = Array.from(dialogRef.value?.querySelectorAll<HTMLElement>('button:not(:disabled), input:not(:disabled)') || []); if (!elements.length) return; const first = elements[0]; const last = elements[elements.length - 1]; if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus() } else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus() } }

watch([spaceId, statusFilter], loadDocuments, { immediate: true })
watch(searchQuery, () => { clearTimeout(searchTimer); searchTimer = setTimeout(loadDocuments, 300) })
onBeforeUnmount(() => clearTimeout(searchTimer))
</script>

<style scoped>
.sdp-documents-list { min-height: 100dvh; color: var(--ink-900); background: var(--ink-100); font-family: var(--font-body); }
.sdp-documents-list__shell { display: grid; gap: var(--space-6); width: min(100%, var(--content-max-width)); margin-inline: auto; padding: var(--space-8); }
.sdp-documents-list__header, .sdp-documents-list__panel-head, .sdp-documents-list__filters, .sdp-documents-list__dialog header, .sdp-documents-list__dialog-actions { display: flex; align-items: center; }
.sdp-documents-list__header, .sdp-documents-list__panel-head { justify-content: space-between; gap: var(--space-6); }
.sdp-documents-list__header p, .sdp-documents-list__panel-head p, .sdp-documents-list__dialog header p { color: var(--brand-700); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); letter-spacing: .08em; text-transform: uppercase; }
.sdp-documents-list__header h1 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-4xl); line-height: var(--leading-tight); }
.sdp-documents-list__header > div > span { display: block; margin-top: var(--space-2); color: var(--ink-600); font-size: var(--text-base); }
.sdp-documents-list__header svg { width: var(--space-5); height: var(--space-5); margin-right: var(--space-2); fill: none; stroke: currentColor; stroke-linecap: round; stroke-linejoin: round; stroke-width: 2; }
.sdp-documents-list__panel { min-width: var(--space-0); overflow: hidden; border: 1px solid var(--ink-200); border-radius: var(--radius-lg); background: var(--ink-50); box-shadow: var(--shadow-xs); }
.sdp-documents-list__panel-head { padding: var(--space-6); border-bottom: 1px solid var(--ink-200); }
.sdp-documents-list__panel-head h2 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-xl); }
.sdp-documents-list__panel-head > span { color: var(--ink-600); font-size: var(--text-xs); }
.sdp-documents-list__filters { flex-wrap: wrap; gap: var(--space-3); padding: var(--space-4) var(--space-6); border-bottom: 1px solid var(--ink-200); background: var(--ink-100); }
.sdp-documents-list__search { display: flex; min-width: calc(var(--space-24) * 2); flex: 1; align-items: center; gap: var(--space-2); }
.sdp-documents-list__search svg { width: var(--space-5); height: var(--space-5); flex: 0 0 var(--space-5); fill: none; stroke: var(--ink-600); stroke-linecap: round; stroke-width: 2; }
.sdp-documents-list__filters input, .sdp-documents-list__filters select, .sdp-documents-list__dialog input { width: 100%; min-height: var(--space-10); padding: var(--space-2) var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-900); background: var(--ink-50); font-family: var(--font-body); font-size: var(--text-sm); }
.sdp-documents-list__filters label:not(.sdp-documents-list__search) { min-width: calc(var(--space-24) * 1.5); }
.sdp-documents-list__state { margin: var(--space-6); }
.sdp-documents-list__table-wrap { overflow-x: auto; }
.sdp-documents-list table { width: 100%; border-collapse: collapse; text-align: left; }
.sdp-documents-list th, .sdp-documents-list td { padding: var(--space-4) var(--space-5); border-bottom: 1px solid var(--ink-200); color: var(--ink-700); font-size: var(--text-sm); white-space: nowrap; }
.sdp-documents-list th { color: var(--ink-600); background: var(--ink-100); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); letter-spacing: .04em; }
.sdp-documents-list tbody tr:hover { background: var(--brand-50); }
.sdp-documents-list td strong, .sdp-documents-list td small { display: block; max-width: calc(var(--space-24) * 2.5); overflow: hidden; text-overflow: ellipsis; }
.sdp-documents-list td strong { color: var(--ink-950); }
.sdp-documents-list td small { margin-top: var(--space-1); color: var(--ink-500); font-size: var(--text-xs); }
.sdp-documents-list__status { display: inline-flex; align-items: center; gap: var(--space-2); padding: var(--space-1) var(--space-3); border-radius: var(--radius-pill); color: var(--ink-800); background: var(--ink-200); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-documents-list__status i { width: var(--space-2); height: var(--space-2); border-radius: var(--radius-pill); background: var(--ink-600); }
.sdp-documents-list__status--complete { color: var(--brand-900); background: var(--brand-100); }
.sdp-documents-list__status--complete i, .sdp-documents-list__status--active i { background: var(--brand-700); }
.sdp-documents-list__status--active { color: var(--brand-900); background: var(--brand-50); }
.sdp-documents-list__status--failed { color: var(--ink-950); background: var(--ink-300); }
.sdp-documents-list__backdrop { position: fixed; z-index: var(--z-modal); inset: var(--space-0); display: flex; align-items: center; justify-content: center; padding: var(--space-4); background: var(--ink-950); }
.sdp-documents-list__dialog { width: min(100%, calc(var(--space-24) * 5)); padding: var(--space-6); border: 1px solid var(--ink-200); border-radius: var(--radius-lg); color: var(--ink-900); background: var(--ink-50); box-shadow: var(--shadow-xl); }
.sdp-documents-list__dialog header { align-items: flex-start; justify-content: space-between; gap: var(--space-4); }
.sdp-documents-list__dialog h2 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-2xl); }
.sdp-documents-list__dialog header button { min-height: var(--space-10); padding: var(--space-2) var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-800); background: var(--ink-50); font: inherit; font-size: var(--text-xs); font-weight: var(--font-weight-semibold); cursor: pointer; }
.sdp-documents-list__dialog > p { margin-top: var(--space-4); color: var(--ink-600); font-size: var(--text-sm); line-height: var(--leading-relaxed); }
.sdp-documents-list__dialog form { display: grid; gap: var(--space-5); margin-top: var(--space-6); }
.sdp-documents-list__dialog label { display: grid; gap: var(--space-2); }
.sdp-documents-list__dialog label span { color: var(--ink-800); font-size: var(--text-sm); font-weight: var(--font-weight-semibold); }
.sdp-documents-list__dialog-actions { justify-content: flex-end; gap: var(--space-3); }
.sdp-documents-list__form-error { padding: var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-950); background: var(--ink-100); font-size: var(--text-sm); }
.sdp-documents-list button:focus-visible, .sdp-documents-list input:focus-visible, .sdp-documents-list select:focus-visible { outline: 2px solid var(--brand-500); outline-offset: 2px; }
@media (max-width: 48rem) { .sdp-documents-list__shell { padding: var(--space-6); } .sdp-documents-list__filters { align-items: stretch; flex-direction: column; } .sdp-documents-list__search { min-width: var(--space-0); } .sdp-documents-list__filters label, .sdp-documents-list__filters input, .sdp-documents-list__filters select { width: 100%; } }
@media (max-width: 40rem) { .sdp-documents-list__shell { padding: var(--space-4); } .sdp-documents-list__header { align-items: flex-start; flex-direction: column; } .sdp-documents-list__header h1 { font-size: var(--text-3xl); } .sdp-documents-list__header :deep(.sdp-button) { width: 100%; } .sdp-documents-list__panel-head, .sdp-documents-list__filters { padding-inline: var(--space-4); } .sdp-documents-list__backdrop { align-items: end; padding: var(--space-0); } .sdp-documents-list__dialog { border-radius: var(--radius-lg) var(--radius-lg) var(--radius-none) var(--radius-none); } .sdp-documents-list__dialog-actions { align-items: stretch; flex-direction: column-reverse; } .sdp-documents-list__dialog-actions :deep(.sdp-button) { width: 100%; } }
</style>
