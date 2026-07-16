<template>
  <t-dialog :visible="visible" header="文档分块" :footer="false" width="700px" @update:visible="emit('update:visible', $event)">
    <div class="chunks-container">
      <t-loading v-if="loading" />
      <div v-else-if="chunks.length === 0" class="empty-chunks">暂无分块数据</div>
      <div v-else class="chunk-list">
        <t-card v-for="chunk in chunks" :key="chunk.id" class="chunk-card">
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
.chunks-container { max-height: 500px; overflow-y: auto; }
.empty-chunks { text-align: center; padding: 40px; color: var(--text-secondary); }
.chunk-list { display: flex; flex-direction: column; gap: 12px; }
.chunk-card { }
.chunk-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.chunk-tokens { font-size: 12px; color: var(--text-secondary); }
.chunk-content { font-size: 14px; line-height: 1.6; color: var(--text-primary); white-space: pre-wrap; max-height: 200px; overflow-y: auto; }
</style>
