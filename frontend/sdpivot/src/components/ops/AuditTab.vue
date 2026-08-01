<template>
  <t-card>
    <div style="display:flex;gap:8px;margin-bottom:12px">
      <t-input v-model="auditSearch.user_id" placeholder="用户ID" style="width:150px" />
      <t-input v-model="auditSearch.action" placeholder="操作类型" style="width:150px" />
      <t-button theme="primary" @click="loadAuditLogs">查询</t-button>
      <t-button theme="default" @click="exportLogs">导出CSV</t-button>
    </div>
    <t-table :data="auditLogs" :columns="auditColumns" row-key="id" hover stripe size="small" :pagination="{ total: auditTotal, current: auditPage, pageSize: 20 }" @page-change="onAuditPageChange" />
  </t-card>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import * as opsApi from '@/api/ops'

const auditLogs = ref<any[]>([])
const auditTotal = ref(0)
const auditPage = ref(1)
const auditSearch = reactive({ user_id: '', action: '' })
const auditColumns = [
  { colKey: 'created_at', title: '时间', width: 150 },
  { colKey: 'username', title: '用户', width: 100 },
  { colKey: 'action', title: '操作', width: 150 },
  { colKey: 'resource', title: '资源', width: 120 },
  { colKey: 'detail', title: '详情', ellipsis: true },
  { colKey: 'ip', title: 'IP', width: 120 },
]

async function loadAuditLogs() {
  try {
    const r = await opsApi.getAuditLogs({ user_id: auditSearch.user_id, action: auditSearch.action, page: auditPage.value })
    auditLogs.value = (r.data as any).logs || []
    auditTotal.value = (r.data as any).total || 0
  } catch {
    MessagePlugin.error('操作失败')
  }
}

function onAuditPageChange(p: any) {
  auditPage.value = p.current
  loadAuditLogs()
}

async function exportLogs() {
  try {
    const r = await opsApi.exportAuditLogs()
    const url = window.URL.createObjectURL(new Blob([r.data as any]))
    const a = document.createElement('a')
    a.href = url
    a.download = 'audit_logs.csv'
    a.click()
    window.URL.revokeObjectURL(url)
  } catch {
    MessagePlugin.error('导出失败')
  }
}

onMounted(loadAuditLogs)
</script>
