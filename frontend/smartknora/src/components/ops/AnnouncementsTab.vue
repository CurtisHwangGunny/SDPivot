<template>
  <t-card>
    <div class="announce-form">
      <t-input v-model="annForm.title" placeholder="公告标题" style="flex:1" />
      <t-button theme="primary" :disabled="!annForm.title || !annForm.content" @click="handleCreateAnn">发布</t-button>
    </div>
    <t-textarea v-model="annForm.content" placeholder="公告内容" :autosize="{ minRows: 2 }" style="margin-top:8px" />
    <t-table v-if="announcements.length > 0" :data="announcements" :columns="annColumns" row-key="id" hover stripe size="small" style="margin-top:12px">
      <template #operation="{ row }">
        <t-button size="small" theme="danger" variant="text" @click="delAnn(row)">删除</t-button>
      </template>
    </t-table>
  </t-card>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import * as opsApi from '@/api/ops'

const announcements = ref<any[]>([])
const annForm = ref({ title: '', content: '' })
const annColumns = [
  { colKey: 'title', title: '标题', ellipsis: true },
  { colKey: 'status', title: '状态', width: 80 },
  { colKey: 'created_at', title: '发布时间', width: 120 },
  { colKey: 'operation', title: '操作', width: 80 },
]

async function loadAnnouncements() {
  try {
    const r = await opsApi.listAnnouncements()
    announcements.value = (r.data as any).announcements || []
  } catch {}
}

async function handleCreateAnn() {
  if (!annForm.value.title) return
  try {
    await opsApi.createAnnouncement(annForm.value)
    MessagePlugin.success('公告已发布')
    annForm.value = { title: '', content: '' }
    loadAnnouncements()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '发布失败')
  }
}

async function delAnn(row: any) {
  try {
    await opsApi.deleteAnnouncement(row.id)
    MessagePlugin.success('删除成功')
    loadAnnouncements()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '删除失败')
  }
}

onMounted(loadAnnouncements)
</script>

<style scoped>
.announce-form {
  display: flex;
  gap: 8px;
}
</style>
