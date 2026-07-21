<template>
  <t-card>
    <div style="display:flex;gap:8px;margin-bottom:12px">
      <t-input v-model="userSearch" placeholder="搜索用户名/邮箱/手机号" @enter="loadUsers" clearable style="width:250px" />
      <t-button theme="primary" @click="loadUsers">搜索</t-button>
    </div>
    <t-table :data="users" :columns="userColumns" row-key="id" hover stripe size="small" :pagination="{ total: userTotal, current: userPage, pageSize: userPageSize }" @page-change="onUserPageChange">
      <template #is_active="{ row }">
        <t-tag :theme="row.is_active ? 'success' : 'danger'" size="small">{{ row.is_active ? '正常' : '禁用' }}</t-tag>
      </template>
      <template #is_ops_admin="{ row }">
        <t-tag v-if="row.is_ops_admin" theme="primary" size="small">运营</t-tag>
      </template>
      <template #operation="{ row }">
        <t-button size="small" :theme="row.is_active ? 'danger' : 'success'" variant="text" @click="toggleUser(row)">{{ row.is_active ? '禁用' : '启用' }}</t-button>
      </template>
    </t-table>
  </t-card>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import * as opsApi from '@/api/ops'

const users = ref<any[]>([])
const userSearch = ref('')
const userTotal = ref(0)
const userPage = ref(1)
const userPageSize = ref(20)
const userColumns = [
  { colKey: 'username', title: '用户名', width: 120 },
  { colKey: 'email', title: '邮箱', ellipsis: true },
  { colKey: 'phone', title: '手机号', width: 120 },
  { colKey: 'is_active', title: '状态', width: 80 },
  { colKey: 'is_ops_admin', title: '角色', width: 80 },
  { colKey: 'created_at', title: '注册时间', width: 120 },
  { colKey: 'operation', title: '操作', width: 80 },
]

async function loadUsers() {
  try {
    const r = await opsApi.listUsers({ search: userSearch.value, page: userPage.value, page_size: userPageSize.value })
    users.value = (r.data as any).users || []
    userTotal.value = (r.data as any).total || 0
  } catch {}
}

function onUserPageChange(p: any) {
  userPage.value = p.current
  loadUsers()
}

async function toggleUser(row: any) {
  try {
    await opsApi.updateUserStatus(row.id, !row.is_active)
    MessagePlugin.success('操作成功')
    loadUsers()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '操作失败')
  }
}

onMounted(loadUsers)
</script>
