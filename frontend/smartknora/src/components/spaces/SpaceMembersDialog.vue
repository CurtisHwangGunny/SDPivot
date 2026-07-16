<template>
  <t-dialog :visible="visible" header="成员管理" :footer="false" width="600px" @update:visible="emit('update:visible', $event)">
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
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import client from '@/api/client'
import { listSpaceMembers } from '@/api/spaces'
import { MessagePlugin } from 'tdesign-vue-next'

const props = defineProps<{
  visible: boolean
  spaceId: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  updated: []
}>()

const members = ref<any[]>([])
const memberForm = ref({ user_id: '', role: 'viewer' })

const memberColumns = [
  { colKey: 'user_id', title: '用户ID', ellipsis: true },
  { colKey: 'role', title: '角色', width: 100 },
  { colKey: 'operation', title: '操作', width: 80 },
]

function roleLabel(r: string) { return { admin: '管理员', editor: '编辑者', viewer: '查阅者' }[r] || r }
function roleTheme(r: string): any { return { admin: 'primary', editor: 'warning', viewer: 'default' }[r] || 'default' }

async function loadMembers() {
  try {
    members.value = ((await listSpaceMembers(props.spaceId)).data as any).members || []
  } catch {
    members.value = []
  }
}

async function handleAddMember() {
  if (!memberForm.value.user_id) {
    MessagePlugin.warning('请输入用户ID')
    return
  }
  try {
    await client.post(`/spaces/${props.spaceId}/members`, { user_id: memberForm.value.user_id, role: memberForm.value.role })
    MessagePlugin.success('成员已添加')
    memberForm.value.user_id = ''
    await loadMembers()
    emit('updated')
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '添加失败')
  }
}

async function handleRemoveMember(row: any) {
  try {
    await client.delete(`/spaces/${props.spaceId}/members/${row.user_id}`)
    MessagePlugin.success('成员已移除')
    await loadMembers()
    emit('updated')
  } catch {
    MessagePlugin.error('移除失败')
  }
}

watch(() => props.visible, (visible) => {
  if (visible && props.spaceId) {
    loadMembers()
  }
})
</script>

<style scoped>
.member-dialog { padding: 8px 0; }
.add-member-row { display: flex; gap: 8px; }
.empty-members { text-align: center; padding: 24px; color: var(--text-secondary); }
</style>
