<template>
  <t-dialog :visible="visible" header="文档分块" :footer="false" width="640px" @update:visible="emit('update:visible', $event)">
    <div class="chunks-container">
      <div v-if="!loading && chunks.length > 0" class="chunks-summary">共 {{ chunks.length }} 个分块，以下内容按切分顺序展示。</div>
      <t-loading v-if="loading" />
      <div v-else-if="chunks.length === 0" class="empty-chunks">暂无分块数据</div>
      <div v-else class="chunk-list">
        <t-card v-for="chunk in chunks" :key="chunk.id" class="chunk-card" :bordered="false">
          <div class="chunk-header">
            <t-tag theme="primary" variant="light" size="small">#{{ chunk.chunk_index }}</t-tag>
            <span class="chunk-tokens">{{ chunk.token_count }} tokens</span>
          </div>
          <div class="chunk-content">{{ chunk.content }}</div>
        </t-card>
      </div>
    </div>
  </t-dialog>
</template>

<script setup lang="ts">
import type { DocumentChunk } from '@/api/documents'

defineProps<{
  visible: boolean
  loading: boolean
  chunks: DocumentChunk[]
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()
</script>

<style scoped>
.chunks-container { max-height: 480px; overflow-y: auto; padding-top: 8px; }
.chunks-summary { margin-bottom: 14px; padding: 12px 14px; border-radius: 12px; background: var(--sk-brand-soft); color: var(--sk-brand-deep); font-size: 12px; line-height: 1.6; }
.empty-chunks { text-align: center; padding: 32px 16px; border-radius: 12px; background: var(--td-bg-color-container-hover); color: var(--text-secondary); }
.chunk-list { display: flex; flex-direction: column; gap: 10px; }
.chunk-card { border-radius: 16px; background: var(--td-bg-color-container); box-shadow: var(--sk-shadow-sm); }
.chunk-header { display: flex; align-items: center; gap: 8px; margin-bottom: 10px; }
.chunk-tokens { font-size: 12px; color: var(--text-secondary); }
.chunk-content { font-size: 14px; line-height: 1.7; color: var(--text-primary); white-space: pre-wrap; max-height: 188px; overflow-y: auto; }
</style>
