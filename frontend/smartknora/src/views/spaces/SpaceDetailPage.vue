<template>
  <div class="detail-page">
    <div class="detail-header">
      <t-button variant="text" @click="$router.push('/spaces')">
        <template #icon><t-icon name="chevron-left" /></template>
        返回
      </t-button>
      <div class="header-info" v-if="space">
        <h1>{{ space.name }}</h1>
        <p class="header-desc">{{ space.description || '暂无描述' }}</p>
        <div class="header-meta">
          <t-tag theme="primary" variant="light">{{ visibilityLabel(space.visibility) }}</t-tag>
          <span class="meta-text">创建于 {{ formatDate(space.created_at) }}</span>
        </div>
      </div>
      <t-space>
        <t-button variant="outline" @click="showMembers = true">
          <template #icon><t-icon name="user-group" /></template>
          成员管理
        </t-button>
        <t-button theme="primary" @click="showUpload = true">
          <template #icon><t-icon name="upload" /></template>
          导入文档
        </t-button>
      </t-space>
    </div>

    <t-loading v-if="loading" />
    <template v-else>
      <t-card class="stats-card">
        <t-row :gutter="16">
          <t-col :span="3">
            <div class="stat"><span class="stat-num">{{ documents.length }}</span><span class="stat-label">文档总数</span></div>
          </t-col>
          <t-col :span="3">
            <div class="stat"><span class="stat-num">{{ completedCount }}</span><span class="stat-label">已解析</span></div>
          </t-col>
          <t-col :span="3">
            <div class="stat"><span class="stat-num">{{ processingCount }}</span><span class="stat-label">解析中</span></div>
          </t-col>
          <t-col :span="3">
            <div class="stat"><span class="stat-num">{{ memberCount }}</span><span class="stat-label">成员数</span></div>
          </t-col>
        </t-row>
      </t-card>

      <t-card class="doc-card" title="文档列表" style="margin-top:16px">
        <t-table v-if="documents.length > 0" :data="documents" :columns="docColumns" row-key="id" hover stripe>
          <template #parse_status="{ row }">
            <t-tag :theme="statusTheme(row.parse_status)" variant="light">{{ statusLabel(row.parse_status) }}</t-tag>
          </template>
          <template #file_size="{ row }">{{ formatSize(row.file_size) }}</template>
          <template #created_at="{ row }">{{ formatDate(row.created_at) }}</template>
        </t-table>
        <div v-else class="empty-docs">
          <t-icon name="file-paste" size="48px" style="color:#d0d0d0" />
          <p>此空间暂无文档</p>
          <t-button theme="primary" variant="outline" @click="showUpload = true">导入文档</t-button>
        <t-button variant="outline" @click="$router.push(`/spaces/${spaceId}/documents`)">
          <template #icon><t-icon name="file-paste" /></template>
          文档管理
        </t-button>
        </div>
      </t-card>
    </template>

    <t-dialog v-model:visible="showMembers" header="成员管理" :footer="false" width="600px">
      <div class="member-dialog">
        <div class="add-member-row">
          <t-input v-model="memberForm.user_id" placeholder="用户ID" style="flex:1" />
          <t-select v-model="memberForm.role" style="width:120px">
            <t-option value="viewer" label="查阅者" />
            <t-option value="editor" label="编辑者" />
            <t-option value="admin" label="管理员" />
          </t-select>
          <t-button theme="primary" @click="handleAddMember">添加</t-button>
        </div>
        <t-table v-if="members.length > 0" :data="members" :columns="memberColumns" row-key="id" hover stripe style="margin-top:12px">
          <template #role="{ row }"><t-tag :theme="roleTheme(row.role)" variant="light">{{ roleLabel(row.role) }}</t-tag></template>
          <template #operation="{ row }"><t-button variant="text" theme="danger" size="small" @click="handleRemoveMember(row)">移除</t-button></template>
        </t-table>
        <div v-else class="empty-members">暂无成员</div>
      </div>
    </t-dialog>

    <t-dialog v-model:visible="showUpload" header="导入文档" :footer="false" width="500px">
      <t-tabs v-model="uploadTab">
        <t-tab-panel value="file" label="文件上传">
          <t-upload v-model="fileList" :action="`/api/v1/smartknora/documents/upload`" :data="{ space_id: spaceId }" :headers="uploadHeaders" :max="20" :size-limit="{ size: 50, unit: 'MB' }" :accept="acceptFormats" multiple auto-upload @success="onUploadSuccess" @fail="onUploadFail">
            <t-button theme="primary"><template #icon><t-icon name="upload" /></template>选择文件</t-button>
            <template #tips>支持 PDF、Word、Excel、PPT、Markdown、TXT、图片，单文件 ≤ 50MB</template>
          </t-upload>
        </t-tab-panel>
        <t-tab-panel value="manual" label="手动录入">
          <div class="manual-form">
            <t-input v-model="manualForm.title" placeholder="文档标题" />
            <t-textarea v-model="manualForm.content" placeholder="Markdown 内容" :autosize="{ minRows: 6 }" />
            <t-button theme="primary" @click="handleManualCreate">保存</t-button>
          </div>
        </t-tab-panel>
      </t-tabs>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { getSpace, listSpaceMembers, type Space } from '@/api/spaces'
import client from '@/api/client'
import { MessagePlugin } from 'tdesign-vue-next'

const route = useRoute()
const spaceId = route.params.id as string

const space = ref<Space | null>(null)
const documents = ref<any[]>([])
const members = ref<any[]>([])
const loading = ref(true)
const showMembers = ref(false)
const showUpload = ref(false)
const uploadTab = ref('file')
const fileList = ref([])
const memberForm = ref({ user_id: '', role: 'viewer' })
const manualForm = ref({ title: '', content: '' })

const uploadHeaders = computed(() => ({ Authorization: `Bearer ${localStorage.getItem('access_token')}` }))
const acceptFormats = '.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.md,.txt,.png,.jpg,.jpeg,.mp3'

const completedCount = computed(() => documents.value.filter(d => d.parse_status === 'completed').length)
const processingCount = computed(() => documents.value.filter(d => d.parse_status === 'processing' || d.parse_status === 'pending').length)
const memberCount = computed(() => members.value.length)

const docColumns = [
  { colKey: 'title', title: '文档名称', ellipsis: true },
  { colKey: 'file_type', title: '类型', width: 80 },
  { colKey: 'file_size', title: '大小', width: 100 },
  { colKey: 'parse_status', title: '状态', width: 100 },
  { colKey: 'created_at', title: '上传时间', width: 120 },
]
const memberColumns = [
  { colKey: 'user_id', title: '用户ID', ellipsis: true },
  { colKey: 'role', title: '角色', width: 100 },
  { colKey: 'operation', title: '操作', width: 80 },
]

function visibilityLabel(v: string) { return { private: '私密', team: '团队可见', enterprise: '企业可见' }[v] || v }
function statusLabel(s: string) { return { pending: '等待中', processing: '解析中', completed: '已完成', failed: '失败' }[s] || s }
function statusTheme(s: string): any { return { pending: 'default', processing: 'warning', completed: 'success', failed: 'danger' }[s] || 'default' }
function roleLabel(r: string) { return { admin: '管理员', editor: '编辑者', viewer: '查阅者' }[r] || r }
function roleTheme(r: string): any { return { admin: 'primary', editor: 'warning', viewer: 'default' }[r] || 'default' }
function formatDate(t: string) { return t ? new Date(t).toLocaleDateString('zh-CN') : '' }
function formatSize(bytes: number) { if (!bytes) return '-'; if (bytes < 1024) return bytes + 'B'; if (bytes < 1024*1024) return (bytes/1024).toFixed(1)+'KB'; return (bytes/1024/1024).toFixed(1)+'MB' }

async function loadSpace() { try { space.value = (await getSpace(spaceId)).data } catch {} }
async function loadDocuments() { try { documents.value = ((await client.get('/documents', { params: { space_id: spaceId } })).data as any).documents || [] } catch { documents.value = [] } }
async function loadMembers() { try { members.value = ((await listSpaceMembers(spaceId)).data as any).members || [] } catch { members.value = [] } }

async function handleAddMember() {
  if (!memberForm.value.user_id) { MessagePlugin.warning('请输入用户ID'); return }
  try { await client.post(`/spaces/${spaceId}/members`, { user_id: memberForm.value.user_id, role: memberForm.value.role }); MessagePlugin.success('成员已添加'); memberForm.value.user_id = ''; loadMembers() }
  catch (e: any) { MessagePlugin.error(e.response?.data?.error || '添加失败') }
}
async function handleRemoveMember(row: any) {
  try { await client.delete(`/spaces/${spaceId}/members/${row.user_id}`); MessagePlugin.success('成员已移除'); loadMembers() } catch { MessagePlugin.error('移除失败') }
}
function onUploadSuccess() { MessagePlugin.success('上传成功'); loadDocuments() }
function onUploadFail() { MessagePlugin.error('上传失败') }
async function handleManualCreate() {
  if (!manualForm.value.title) { MessagePlugin.warning('请输入标题'); return }
  try { await client.post('/documents/manual', { space_id: spaceId, title: manualForm.value.title, content: manualForm.value.content }); MessagePlugin.success('保存成功'); showUpload.value = false; manualForm.value = { title: '', content: '' }; loadDocuments() }
  catch (e: any) { MessagePlugin.error(e.response?.data?.error || '保存失败') }
}

onMounted(async () => { loading.value = true; await Promise.all([loadSpace(), loadDocuments(), loadMembers()]); loading.value = false })
</script>

<style scoped>
.detail-page { max-width: 1000px; margin: 0 auto; }
.detail-header { display: flex; align-items: flex-start; gap: 16px; margin-bottom: 24px; flex-wrap: wrap; }
.header-info { flex: 1; }
.header-info h1 { font-size: 24px; font-weight: 600; }
.header-desc { font-size: 14px; color: #999; margin: 4px 0 8px; }
.header-meta { display: flex; align-items: center; gap: 12px; }
.meta-text { font-size: 12px; color: #bbb; }
.stats-card { margin-bottom: 16px; }
.stat { text-align: center; }
.stat-num { font-size: 28px; font-weight: 700; color: #014DB2; display: block; }
.stat-label { font-size: 13px; color: #999; }
.empty-docs { text-align: center; padding: 40px; color: #999; }
.empty-docs p { margin: 12px 0; }
.member-dialog { padding: 8px 0; }
.add-member-row { display: flex; gap: 8px; }
.empty-members { text-align: center; padding: 24px; color: #999; }
.manual-form { display: flex; flex-direction: column; gap: 12px; padding: 16px 0; }
</style>
