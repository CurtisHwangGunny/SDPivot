<template>
  <t-card>
    <div style="display:flex;gap:8px;margin-bottom:12px;flex-wrap:wrap">
      <t-input v-model="modelForm.name" placeholder="模型名称" style="width:150px" />
      <t-input v-model="modelForm.display_name" placeholder="显示名称" style="width:150px" />
      <t-select v-model="modelForm.type" style="width:120px" placeholder="类型">
        <t-option value="llm" label="LLM" />
        <t-option value="embedding" label="Embedding" />
      </t-select>
      <t-input v-model="modelForm.source" placeholder="来源" style="width:120px" />
      <t-button theme="primary" @click="addModel" :disabled="!modelForm.name">添加</t-button>
    </div>
    <t-table :data="models" :columns="modelColumns" row-key="id" hover stripe size="small">
      <template #is_default="{ row }">
        <t-tag v-if="row.is_default" theme="success" size="small">默认</t-tag>
      </template>
      <template #operation="{ row }">
        <t-button size="small" theme="primary" variant="text" @click="setDefault(row)" :disabled="row.is_default">设为默认</t-button>
        <t-button size="small" theme="danger" variant="text" @click="delModel(row)" :disabled="row.is_builtin">删除</t-button>
      </template>
    </t-table>
  </t-card>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import * as opsApi from '@/api/ops'

const models = ref<any[]>([])
const modelForm = reactive({ name: '', display_name: '', type: 'llm', source: '' })
const modelColumns = [
  { colKey: 'name', title: '模型名称', width: 150 },
  { colKey: 'display_name', title: '显示名称', width: 150 },
  { colKey: 'type', title: '类型', width: 100 },
  { colKey: 'source', title: '来源', width: 120 },
  { colKey: 'is_default', title: '默认', width: 80 },
  { colKey: 'status', title: '状态', width: 80 },
  { colKey: 'operation', title: '操作', width: 150 },
]

async function loadModels() {
  try {
    const r = await opsApi.listModels()
    models.value = (r.data as any).models || []
  } catch {}
}

async function addModel() {
  try {
    await opsApi.createModel({ name: modelForm.name, display_name: modelForm.display_name, type: modelForm.type, source: modelForm.source })
    MessagePlugin.success('添加成功')
    modelForm.name = ''
    modelForm.display_name = ''
    modelForm.source = ''
    loadModels()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '添加失败')
  }
}

async function setDefault(row: any) {
  try {
    await opsApi.setDefaultModel(row.id)
    MessagePlugin.success('设置成功')
    loadModels()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '设置失败')
  }
}

async function delModel(row: any) {
  try {
    await opsApi.deleteModel(row.id)
    MessagePlugin.success('删除成功')
    loadModels()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '删除失败')
  }
}

onMounted(loadModels)
</script>
