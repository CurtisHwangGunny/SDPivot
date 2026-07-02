<template>
  <div class="admin-page">
    <h1>管理后台</h1>
    <t-loading v-if="loading" />
    <div v-else class="admin-content">
      <t-row :gutter="16" class="stats-row">
        <t-col :span="4">
          <t-card class="stat-card">
            <div class="stat-value">{{ stats.space_count || 0 }}</div>
            <div class="stat-label">知识空间</div>
          </t-card>
        </t-col>
        <t-col :span="4">
          <t-card class="stat-card">
            <div class="stat-value">{{ stats.document_count || 0 }}</div>
            <div class="stat-label">文档总数</div>
          </t-card>
        </t-col>
        <t-col :span="4">
          <t-card class="stat-card">
            <div class="stat-value">{{ stats.member_count || 0 }}</div>
            <div class="stat-label">成员数</div>
          </t-card>
        </t-col>
      </t-row>

      <t-card title="空间管理" style="margin-top:16px">
        <t-table v-if="spaces.length > 0" :data="spaces" :columns="spaceColumns" row-key="id" hover stripe size="small">
          <template #visibility="{ row }">
            <t-tag theme="primary" variant="light" size="small">{{ row.visibility }}</t-tag>
          </template>
          <template #created_at="{ row }">{{ formatDate(row.created_at) }}</template>
        </t-table>
        <div v-else class="empty">暂无空间</div>
      </t-card>

      <t-card title="成员管理" style="margin-top:16px">
        <t-table v-if="members.length > 0" :data="members" :columns="memberColumns" row-key="id" hover stripe size="small">
          <template #role="{ row }">
            <t-tag :theme="row.role === 'admin' ? 'primary' : 'default'" variant="light" size="small">{{ row.role }}</t-tag>
          </template>
        </t-table>
        <div v-else class="empty">暂无成员</div>
      </t-card>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getAdminStats, getAdminMembers, getAdminSpaces } from '@/api/qa'
import { MessagePlugin } from 'tdesign-vue-next'

const loading = ref(true)
const stats = ref<any>({})
const spaces = ref<any[]>([])
const members = ref<any[]>([])

const spaceColumns = [
  { colKey: 'name', title: '空间名称', ellipsis: true },
  { colKey: 'visibility', title: '可见性', width: 100 },
  { colKey: 'created_at', title: '创建时间', width: 120 },
]
const memberColumns = [
  { colKey: 'user_id', title: '用户ID', ellipsis: true },
  { colKey: 'role', title: '角色', width: 100 },
  { colKey: 'status', title: '状态', width: 80 },
]

function formatDate(t: string) { return t ? new Date(t).toLocaleDateString('zh-CN') : '' }

async function load() {
  loading.value = true
  try {
    const [statsRes, spacesRes, membersRes] = await Promise.all([
      getAdminStats().catch(() => ({ data: {} })),
      getAdminSpaces().catch(() => ({ data: { spaces: [] } })),
      getAdminMembers().catch(() => ({ data: { members: [] } })),
    ])
    stats.value = (statsRes.data as any) || {}
    spaces.value = (spacesRes.data as any).spaces || []
    members.value = (membersRes.data as any).members || []
  } catch {
    MessagePlugin.warning('需要管理员权限')
  } finally { loading.value = false }
}
onMounted(load)
</script>
<style scoped>
.admin-page { max-width: 1000px; margin: 0 auto; }
.admin-page h1 { font-size: 24px; font-weight: 600; margin-bottom: 24px; }
.stats-row { margin-bottom: 16px; }
.stat-card { text-align: center; }
.stat-value { font-size: 32px; font-weight: 700; color: #014DB2; }
.stat-label { font-size: 14px; color: #999; margin-top: 4px; }
.empty { text-align: center; padding: 24px; color: #999; }
</style>
