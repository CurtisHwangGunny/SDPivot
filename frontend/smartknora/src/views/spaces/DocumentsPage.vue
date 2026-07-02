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
      <t-icon name="file-paste" size="64px" style="color:#d0d0d0" />
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

    <t-dialog v-model:visible="showImport" header="导入文档" :footer="false" width="560px">
      <t-tabs v-model="importTab">
        <t-tab-panel value="file" label="文件上传">
          <div class="upload-area">
            <t-upload
              v-model="fileList"
              :action="`/api/v1/smartknora/documents/upload`"
              :data="{ space_id: spaceId }"
              :headers="uploadHeaders"
              :max="20"
              :size-limit="{ size: 50, unit: 'MB' }"
              :accept="acceptFormats"
              multiple
              auto-upload
              theme="file"
              @success="onUploadSuccess"
              @fail="onUploadFail"
            >
              <t-button theme="primary" block size="large">
                <template #icon><t-icon name="upload" /></template>
                点击或拖拽文件到此处上传
              </t-button>
              <template #tips>
                <div class="upload-tips">
                  支持 PDF、Word、Excel、PPT、Markdown、TXT、图片、音频<br/>
                  单文件 ≤ 50MB，一次最多 20 个文件
                </div>
              </template>
            </t-upload>
          </div>
        </t-tab-panel>

        <t-tab-panel value="url" label="网页链接">
          <div class="url-form">
            <t-input v-model="urlForm.url" placeholder="https://example.com/article" size="large">
              <template #prefix-icon><t-icon name="link" /></template>
            </t-input>
            <t-input v-model="urlForm.tags" placeholder="标签（选填，逗号分隔）" />
            <t-button theme="primary" block size="large" :loading="urlLoading" :disabled="!urlForm.url" @click="handleUrlImport">
              <template #icon><t-icon name="download" /></template>
              导入网页
            </t-button>
            <p class="form-tip">系统将自动抓取网页正文内容并导入</p>
          </div>
        </t-tab-panel>

        <t-tab-panel value="manual" label="手动录入">
          <div class="manual-form">
            <t-input v-model="manualForm.title" placeholder="文档标题（必填）" size="large" />
            <t-textarea
              v-model="manualForm.content"
              placeholder="输入 Markdown 格式内容..."
              :autosize="{ minRows: 8, maxRows: 20 }"
            />
            <t-input v-model="manualForm.tags" placeholder="标签（选填，逗号分隔）" />
            <t-button theme="primary" block size="large" :loading="manualLoading" :disabled="!manualForm.title || !manualForm.content" @click="handleManualCreate">
              <template #icon><t-icon name="save" /></template>
              保存
            </t-button>
          </div>
        </t-tab-panel>
      </t-tabs>
    </t-dialog>

    <t-dialog v-model:visible="showChunks" header="文档分块" :footer="false" width="700px">
      <div class="chunks-container">
        <t-loading v-if="chunksLoading" />
        <div v-else-if="chunks.length === 0" class="empty-chunks">暂无分块数据</div>
        <div v-else class="chunk-list">
          <t-card v-for="(chunk, i) in chunks" :key="chunk.id" class="chunk-card">
            <div class="chunk-header">
              <t-tag theme="primary" variant="light" size="small">#{{ chunk.chunk_index }}</t-tag>
              <span class="chunk-tokens">{{ chunk.token_count }} tokens</span>
            </div>
            <div class="chunk-content">{{ chunk.content }}</div>
          </t-card>
        </div>
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { listDocuments, deleteDocument, reparseDocument, getDocumentChunks, uploadManual, uploadFromURL, type Document, type DocumentChunk } from '@/api/documents'
import { MessagePlugin } from 'tdesign-vue-next'

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
const importTab = ref('file')
const fileList = ref([])
const urlForm = ref({ url: '', tags: '' })
const urlLoading = ref(false)
const manualForm = ref({ title: '', content: '', tags: '' })
const manualLoading = ref(false)

const showChunks = ref(false)
const chunks = ref<DocumentChunk[]>([])
const chunksLoading = ref(false)

const uploadHeaders = computed(() => ({ Authorization: `Bearer ${localStorage.getItem('access_token')}` }))
const acceptFormats = '.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.md,.txt,.png,.jpg,.jpeg,.mp3'

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
  } catch { documents.value = [] }
  finally { loading.value = false }
}

async function handleDelete(row: Document) {
  try {
    await deleteDocument(row.id)
    MessagePlugin.success('文档已删除')
    loadDocuments()
  } catch { MessagePlugin.error('删除失败') }
}

async function handleReparse(row: Document) {
  try {
    await reparseDocument(row.id)
    MessagePlugin.success('已触发重新解析')
    loadDocuments()
  } catch { MessagePlugin.error('操作失败') }
}

async function viewChunks(row: Document) {
  showChunks.value = true
  chunksLoading.value = true
  try {
    const res = await getDocumentChunks(row.id)
    chunks.value = (res.data as any).chunks || []
  } catch { chunks.value = [] }
  finally { chunksLoading.value = false }
}

function onUploadSuccess() {
  MessagePlugin.success('上传成功')
  loadDocuments()
}
function onUploadFail() {
  MessagePlugin.error('上传失败')
}

async function handleUrlImport() {
  if (!urlForm.value.url) return
  urlLoading.value = true
  try {
    await uploadFromURL({ space_id: spaceId.value, url: urlForm.value.url, tags: urlForm.value.tags })
    MessagePlugin.success('网页导入成功')
    showImport.value = false
    urlForm.value = { url: '', tags: '' }
    loadDocuments()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '导入失败')
  } finally { urlLoading.value = false }
}

async function handleManualCreate() {
  if (!manualForm.value.title || !manualForm.value.content) return
  manualLoading.value = true
  try {
    await uploadManual({ space_id: spaceId.value, title: manualForm.value.title, content: manualForm.value.content, tags: manualForm.value.tags })
    MessagePlugin.success('文档已保存')
    showImport.value = false
    manualForm.value = { title: '', content: '', tags: '' }
    loadDocuments()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '保存失败')
  } finally { manualLoading.value = false }
}

function onRowClick(row: any) {
  viewChunks(row)
}

watch(spaceId, () => { if (spaceId.value) loadDocuments() })
onMounted(() => { if (spaceId.value) loadDocuments() })
</script>

<style scoped>
.docs-page { max-width: 1200px; margin: 0 auto; }
.docs-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; flex-wrap: wrap; gap: 12px; }
.header-left { display: flex; align-items: center; gap: 12px; }
.header-left h2 { font-size: 20px; font-weight: 600; }
.empty { text-align: center; padding: 80px 20px; }
.empty-title { font-size: 18px; font-weight: 500; margin: 16px 0 8px; }
.empty-desc { color: #999; margin-bottom: 24px; }
.pagination { margin-top: 16px; display: flex; justify-content: center; }
.upload-area { padding: 20px 0; }
.upload-tips { font-size: 12px; color: #999; text-align: center; margin-top: 8px; line-height: 1.6; }
.url-form { display: flex; flex-direction: column; gap: 12px; padding: 20px 0; }
.form-tip { font-size: 12px; color: #999; text-align: center; }
.manual-form { display: flex; flex-direction: column; gap: 12px; padding: 20px 0; }
.chunks-container { max-height: 500px; overflow-y: auto; }
.empty-chunks { text-align: center; padding: 40px; color: #999; }
.chunk-list { display: flex; flex-direction: column; gap: 12px; }
.chunk-card { }
.chunk-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.chunk-tokens { font-size: 12px; color: #999; }
.chunk-content { font-size: 14px; line-height: 1.6; color: #333; white-space: pre-wrap; max-height: 200px; overflow-y: auto; }
</style>
