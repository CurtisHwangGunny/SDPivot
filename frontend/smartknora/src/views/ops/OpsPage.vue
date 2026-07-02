<template>
  <div class="ops-page">
    <h1>运营管理端</h1>
    <t-loading v-if="loading" />
    <div v-else>
      <t-row :gutter="16" class="stats-row">
        <t-col :span="4"><t-card class="stat-card"><div class="stat-value">{{ stats.tenant_count || 0 }}</div><div class="stat-label">企业数</div></t-card></t-col>
        <t-col :span="4"><t-card class="stat-card"><div class="stat-value">{{ stats.user_count || 0 }}</div><div class="stat-label">用户数</div></t-card></t-col>
        <t-col :span="4"><t-card class="stat-card"><div class="stat-value">{{ stats.document_count || 0 }}</div><div class="stat-label">文档数</div></t-card></t-col>
      </t-row>

      <t-card title="系统公告" style="margin-top:16px">
        <div class="announce-form">
          <t-input v-model="annForm.title" placeholder="公告标题" style="flex:1" />
          <t-button theme="primary" :disabled="!annForm.title || !annForm.content" @click="handleCreateAnn">发布</t-button>
        </div>
        <t-textarea v-model="annForm.content" placeholder="公告内容" :autosize="{ minRows: 2 }" style="margin-top:8px" />
        <t-table v-if="announcements.length > 0" :data="announcements" :columns="annColumns" row-key="id" hover stripe size="small" style="margin-top:12px">
          <template #created_at="{ row }">{{ formatDate(row.created_at) }}</template>
        </t-table>
      </t-card>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getOpsDashboard, listAnnouncements, createAnnouncement } from '@/api/writing'
import { MessagePlugin } from 'tdesign-vue-next'

const loading = ref(true)
const stats = ref<any>({})
const announcements = ref<any[]>([])
const annForm = ref({ title: '', content: '' })
const annColumns = [
  { colKey: 'title', title: '标题', ellipsis: true },
  { colKey: 'status', title: '状态', width: 80 },
  { colKey: 'created_at', title: '发布时间', width: 120 },
]

function formatDate(t: string) { return t ? new Date(t).toLocaleDateString('zh-CN') : '' }

async function load() {
  loading.value = true
  try {
    const [dashRes, annRes] = await Promise.all([
      getOpsDashboard().catch(() => ({ data: {} })),
      listAnnouncements().catch(() => ({ data: { announcements: [] } })),
    ])
    stats.value = (dashRes.data as any) || {}
    announcements.value = (annRes.data as any).announcements || []
  } catch { MessagePlugin.warning('需要管理员权限') }
  finally { loading.value = false }
}

async function handleCreateAnn() {
  if (!annForm.value.title) return
  try {
    await createAnnouncement(annForm.value)
    MessagePlugin.success('公告已发布')
    annForm.value = { title: '', content: '' }
    load()
  } catch (e: any) { MessagePlugin.error(e.response?.data?.error || '发布失败') }
}

onMounted(load)
</script>
<style scoped>
.ops-page { max-width: 1000px; margin: 0 auto; }
.ops-page h1 { font-size: 24px; font-weight: 600; margin-bottom: 24px; }
.stats-row { margin-bottom: 16px; }
.stat-card { text-align: center; }
.stat-value { font-size: 32px; font-weight: 700; color: #014DB2; }
.stat-label { font-size: 14px; color: #999; margin-top: 4px; }
.announce-form { display: flex; gap: 8px; }
</style>
