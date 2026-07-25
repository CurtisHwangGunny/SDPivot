<template>
  <t-card>
    <div style="display:flex;gap:8px;margin-bottom:12px">
      <t-input v-model="entSearch" placeholder="搜索企业名称" @enter="loadEnterprises" clearable style="width:200px" />
      <t-button theme="primary" @click="loadEnterprises">搜索</t-button>
    </div>
    <t-table :data="enterprises" :columns="entColumns" row-key="id" hover stripe size="small" :pagination="{ total: entTotal, current: entPage, pageSize: entPageSize }" @page-change="onEntPageChange">
      <template #auth_status="{ row }">
        <t-tag :theme="row.auth_status === 'active' ? 'success' : row.auth_status === 'suspended' ? 'danger' : 'warning'" size="small">{{ row.auth_status }}</t-tag>
      </template>
      <template #operation="{ row }">
        <t-button size="small" theme="primary" variant="text" @click="toggleEnterprise(row)">{{ row.auth_status === 'suspended' ? '启用' : '暂停' }}</t-button>
      </template>
    </t-table>
  </t-card>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import * as opsApi from '@/api/ops'

const enterprises = ref<any[]>([])
const entSearch = ref('')
const entTotal = ref(0)
const entPage = ref(1)
const entPageSize = ref(20)
const entColumns = [
  { colKey: 'name', title: '企业名称', ellipsis: true },
  { colKey: 'member_count', title: '成员数', width: 80 },
  { colKey: 'auth_status', title: '认证状态', width: 100 },
  { colKey: 'created_at', title: '创建时间', width: 120 },
  { colKey: 'operation', title: '操作', width: 80 },
]

async function loadEnterprises() {
  try {
    const r = await opsApi.listEnterprises({ search: entSearch.value, page: entPage.value, page_size: entPageSize.value })
    enterprises.value = (r.data as any).enterprises || []
    entTotal.value = (r.data as any).total || 0
  } catch {}
}

function onEntPageChange(p: any) {
  entPage.value = p.current
  loadEnterprises()
}

async function toggleEnterprise(row: any) {
  try {
    await opsApi.updateEnterpriseStatus(row.id, row.auth_status === 'suspended' ? 'active' : 'suspended')
    MessagePlugin.success('操作成功')
    loadEnterprises()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '操作失败')
  }
}

onMounted(loadEnterprises)
</script>
