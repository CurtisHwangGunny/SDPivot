<template>
  <t-card>
    <div style="display:flex;gap:8px;margin-bottom:12px">
      <t-input v-model="filterForm.word" placeholder="敏感词" style="width:200px" />
      <t-select v-model="filterForm.category" style="width:150px" placeholder="分类" clearable>
        <t-option value="general" label="通用" />
        <t-option value="political" label="政治敏感" />
        <t-option value="fraud" label="欺诈" />
        <t-option value="brand" label="品牌侵权" />
      </t-select>
      <t-button theme="primary" @click="addWord" :disabled="!filterForm.word">添加</t-button>
    </div>
    <t-table :data="words" :columns="wordColumns" row-key="id" hover stripe size="small" :pagination="{ total: wordTotal, current: wordPage, pageSize: 20 }" @page-change="onWordPageChange">
      <template #category="{ row }">
        <t-tag size="small">{{ row.category }}</t-tag>
      </template>
      <template #operation="{ row }">
        <t-button size="small" theme="danger" variant="text" @click="delWord(row)">删除</t-button>
      </template>
    </t-table>
  </t-card>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import * as opsApi from '@/api/ops'

const words = ref<any[]>([])
const wordTotal = ref(0)
const wordPage = ref(1)
const filterForm = reactive({ word: '', category: 'general' })
const wordColumns = [
  { colKey: 'word', title: '敏感词', ellipsis: true },
  { colKey: 'category', title: '分类', width: 120 },
  { colKey: 'created_at', title: '添加时间', width: 120 },
  { colKey: 'operation', title: '操作', width: 80 },
]

async function loadWords() {
  try {
    const r = await opsApi.listSensitiveWords({ page: wordPage.value })
    words.value = (r.data as any).words || []
    wordTotal.value = (r.data as any).total || 0
  } catch {}
}

function onWordPageChange(p: any) {
  wordPage.value = p.current
  loadWords()
}

async function addWord() {
  try {
    await opsApi.createSensitiveWord({ word: filterForm.word, category: filterForm.category })
    MessagePlugin.success('添加成功')
    filterForm.word = ''
    loadWords()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '添加失败')
  }
}

async function delWord(row: any) {
  try {
    await opsApi.deleteSensitiveWord(row.id)
    MessagePlugin.success('删除成功')
    loadWords()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '删除失败')
  }
}

onMounted(loadWords)
</script>
