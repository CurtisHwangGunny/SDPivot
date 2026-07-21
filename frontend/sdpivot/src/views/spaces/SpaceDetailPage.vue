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
          <t-icon name="file-paste" size="48px" class="muted-empty-icon" />
          <p>此空间暂无文档</p>
          <div class="empty-actions">
            <t-button theme="primary" variant="outline" @click="showUpload = true">导入文档</t-button>
            <t-button variant="outline" @click="$router.push(`/spaces/${spaceId}/documents`)">
              <template #icon><t-icon name="file-paste" /></template>
              文档管理
            </t-button>
          </div>
        </div>
      </t-card>
    </template>

    <SpaceMembersDialog
      v-if="showMembers"
      v-model:visible="showMembers"
      :space-id="spaceId"
      @updated="loadMembers"
    />

    <SpaceUploadDialog
      v-if="showUpload"
      v-model:visible="showUpload"
      :space-id="spaceId"
      @success="loadDocuments"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { getSpace, listSpaceMembers, type Space } from '@/api/spaces'
import client from '@/api/client'

const SpaceMembersDialog = defineAsyncComponent(() => import('@/components/spaces/SpaceMembersDialog.vue'))
const SpaceUploadDialog = defineAsyncComponent(() => import('@/components/spaces/SpaceUploadDialog.vue'))

const route = useRoute()
const spaceId = route.params.id as string

const space = ref<Space | null>(null)
const documents = ref<any[]>([])
const members = ref<any[]>([])
const loading = ref(true)
const showMembers = ref(false)
const showUpload = ref(false)

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

function visibilityLabel(v: string) { return { private: '私密', team: '团队可见', enterprise: '企业可见' }[v] || v }
function statusLabel(s: string) { return { pending: '等待中', processing: '解析中', completed: '已完成', failed: '失败' }[s] || s }
function statusTheme(s: string): any { return { pending: 'default', processing: 'warning', completed: 'success', failed: 'danger' }[s] || 'default' }
function formatDate(t: string) { return t ? new Date(t).toLocaleDateString('zh-CN') : '' }
function formatSize(bytes: number) { if (!bytes) return '-'; if (bytes < 1024) return bytes + 'B'; if (bytes < 1024*1024) return (bytes/1024).toFixed(1)+'KB'; return (bytes/1024/1024).toFixed(1)+'MB' }

async function loadSpace() { try { space.value = (await getSpace(spaceId)).data } catch {} }
async function loadDocuments() { try { documents.value = ((await client.get('/documents', { params: { space_id: spaceId } })).data as any).documents || [] } catch { documents.value = [] } }
async function loadMembers() { try { members.value = ((await listSpaceMembers(spaceId)).data as any).members || [] } catch { members.value = [] } }

onMounted(async () => { loading.value = true; await Promise.all([loadSpace(), loadDocuments(), loadMembers()]); loading.value = false })
</script>

<style scoped>
.detail-page { max-width: 1000px; margin: 0 auto; }
.detail-header { display: flex; align-items: flex-start; gap: 16px; margin-bottom: 24px; flex-wrap: wrap; }
.header-info { flex: 1; }
.header-info h1 { font-size: 24px; font-weight: 600; color: var(--text-primary); }
.header-desc { font-size: 14px; color: var(--text-secondary); margin: 4px 0 8px; }
.header-meta { display: flex; align-items: center; gap: 12px; }
.meta-text { font-size: 12px; color: var(--text-muted); }
.stats-card { margin-bottom: 16px; }
.stat { text-align: center; }
.stat-num { font-size: 28px; font-weight: 700; color: var(--brand-primary); display: block; }
.stat-label { font-size: 13px; color: var(--text-secondary); }
.empty-docs { text-align: center; padding: 40px; color: var(--text-secondary); }
.empty-docs p { margin: 12px 0; }
.empty-actions { display: flex; justify-content: center; gap: 12px; flex-wrap: wrap; }
.muted-empty-icon { color: var(--text-muted); }
</style>
