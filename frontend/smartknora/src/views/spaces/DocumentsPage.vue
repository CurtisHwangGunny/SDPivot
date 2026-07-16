<template>
  <div class="docs-page">
    <div class="docs-header">
      <div class="header-left">
        <t-button variant="text" @click="$router.push('/spaces')">
          <template #icon><t-icon name="chevron-left" /></template>
          知识空间
        </t-button>
        <h2 v-if="spaceId">空间文档</h2>
      </div>
      <t-space>
        <t-input v-model="searchQuery" placeholder="搜索文档..." clearable style="width:200px" @enter="loadDocuments">
          <template #prefix-icon><t-icon name="search" /></template>
        </t-input>
        <t-select v-model="statusFilter" placeholder="状态" clearable style="width:120px" @change="loadDocuments">
          <t-option value="pending" label="等待中" />
          <t-option value="parsing" label="解析中" />
          <t-option value="completed" label="已完成" />
          <t-option value="failed" label="失败" />
        </t-select>
        <t-button theme="primary" @click="showImport = true">
          <template #icon><t-icon name="upload" /></template>
          导入文档
        </t-button>
      </t-space>
    </div>

    <t-loading v-if="loading" />
    <div v-else-if="documents.length === 0" class="empty">
      <t-icon name="file-paste" size="64px" class="muted-empty-icon" />
      <p class="empty-title">暂无文档</p>
      <p class="empty-desc">上传文件、导入网页或手动录入知识内容</p>
      <t-button theme="primary" @click="showImport = true">导入文档</t-button>
    </div>
    <div v-else class="doc-list">
      <t-table :data="documents" :columns="columns" row-key="id" hover stripe @row-click="onRowClick">
        <template #file_type="{ row }">
          <t-tag theme="default" variant="light">{{ row.file_type || '—' }}</t-tag>
        </template>
        <template #file_size="{ row }">
          {{ formatSize(row.file_size) }}
        </template>
        <template #parse_status="{ row }">
          <t-tag :theme="statusTheme(row.parse_status)" variant="light">
            <template #icon v-if="row.parse_status === 'parsing' || row.parse_status === 'pending'">
              <t-icon name="loading" />
            </template>
            {{ statusLabel(row.parse_status) }}
          </t-tag>
        </template>
        <template #chunk_count="{ row }">
          <t-tag theme="primary" variant="light" size="small">{{ row.chunk_count }} 块</t-tag>
        </template>
        <template #created_at="{ row }">
          {{ formatDate(row.created_at) }}
        </template>
        <template #operation="{ row }">
          <t-space size="small">
            <t-button variant="text" size="small" @click.stop="viewChunks(row)">
              <t-icon name="view-list" /> 分块
            </t-button>
            <t-button variant="text" size="small" theme="warning" @click.stop="handleReparse(row)">
              <t-icon name="refresh" /> 重新解析
            </t-button>
            <t-popconfirm content="确认删除此文档？" @confirm="handleDelete(row)">
              <t-button variant="text" size="small" theme="danger" @click.stop>
                <t-icon name="delete" /> 删除
              </t-button>
            </t-popconfirm>
          </t-space>
        </template>
      </t-table>
      <div class="pagination" v-if="total > pageSize">
        <t-pagination
          v-model="currentPage"
          :total="total"
          :page-size="pageSize"
          show-jumper
          @change="loadDocuments"
        />
      </div>
    </div>

    <DocumentImportDialog
      v-if="showImport"
      v-model:visible="showImport"
      :space-id="spaceId"
      @success="loadDocuments"
    />

    <DocumentChunksDialog
      v-if="showChunks"
      v-model:visible="showChunks"
      :loading="chunksLoading"
      :chunks="chunks"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { listDocuments, deleteDocument, reparseDocument, getDocumentChunks, type Document, type DocumentChunk } from '@/api/documents'
import { MessagePlugin } from 'tdesign-vue-next'

const DocumentImportDialog = defineAsyncComponent(() => import('@/components/documents/DocumentImportDialog.vue'))
const DocumentChunksDialog = defineAsyncComponent(() => import('@/components/documents/DocumentChunksDialog.vue'))

const route = useRoute()
const spaceId = computed(() => (route.params.id as string) || '')

const documents = ref<Document[]>([])
const loading = ref(true)
const searchQuery = ref('')
const statusFilter = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

const showImport = ref(false)
const showChunks = ref(false)
const chunks = ref<DocumentChunk[]>([])
const chunksLoading = ref(false)

const columns = [
  { colKey: 'title', title: '文档名称', ellipsis: true, minWidth: 200 },
  { colKey: 'file_type', title: '类型', width: 80 },
  { colKey: 'file_size', title: '大小', width: 100 },
  { colKey: 'parse_status', title: '状态', width: 100 },
  { colKey: 'chunk_count', title: '分块', width: 80 },
  { colKey: 'created_at', title: '上传时间', width: 120 },
  { colKey: 'operation', title: '操作', width: 220, fixed: 'right' as const },
]

function statusLabel(s: string) {
  return { pending: '等待中', parsing: '解析中', completed: '已完成', failed: '失败' }[s] || s
}
function statusTheme(s: string): any {
  return { pending: 'default', parsing: 'warning', completed: 'success', failed: 'danger' }[s] || 'default'
}
function formatDate(t: string) { return t ? new Date(t).toLocaleDateString('zh-CN') : '' }
function formatSize(bytes: number) {
  if (!bytes) return '-'
  if (bytes < 1024) return bytes + 'B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + 'KB'
  return (bytes / 1024 / 1024).toFixed(1) + 'MB'
}

async function loadDocuments() {
  loading.value = true
  try {
    const res = await listDocuments({
      space_id: spaceId.value || undefined,
      search: searchQuery.value || undefined,
      parse_status: statusFilter.value || undefined,
      page: currentPage.value,
      page_size: pageSize.value,
    })
    documents.value = (res.data as any).documents || []
    total.value = (res.data as any).total || 0
  } catch {
    documents.value = []
  } finally {
    loading.value = false
  }
}

async function handleDelete(row: Document) {
  try {
    await deleteDocument(row.id)
    MessagePlugin.success('文档已删除')
    loadDocuments()
  } catch {
    MessagePlugin.error('删除失败')
  }
}

async function handleReparse(row: Document) {
  try {
    await reparseDocument(row.id)
    MessagePlugin.success('已触发重新解析')
    loadDocuments()
  } catch {
    MessagePlugin.error('操作失败')
  }
}

async function viewChunks(row: Document) {
  showChunks.value = true
  chunksLoading.value = true
  try {
    const res = await getDocumentChunks(row.id)
    chunks.value = (res.data as any).chunks || []
  } catch {
    chunks.value = []
  } finally {
    chunksLoading.value = false
  }
}

function onRowClick(row: any) {
  viewChunks(row)
}

watch(spaceId, () => {
  if (spaceId.value) loadDocuments()
})

onMounted(() => {
  if (spaceId.value) loadDocuments()
})
</script>

<style scoped>
.docs-page { max-width: 1200px; margin: 0 auto; }
.docs-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; flex-wrap: wrap; gap: 12px; }
.header-left { display: flex; align-items: center; gap: 12px; }
.header-left h2 { font-size: 20px; font-weight: 600; color: var(--text-primary); }
.empty { text-align: center; padding: 80px 20px; }
.empty-title { font-size: 18px; font-weight: 500; margin: 16px 0 8px; color: var(--text-primary); }
.empty-desc { color: var(--text-secondary); margin-bottom: 24px; }
.pagination { margin-top: 16px; display: flex; justify-content: center; }
.muted-empty-icon { color: var(--text-muted); }
</style>
