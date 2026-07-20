<template>
  <div>
    <header class="editor-header">
      <div class="editor-title-wrap">
        <span class="editor-tag">Current draft</span>
        <t-input :model-value="draft.title" size="large" @update:model-value="emit('update-title', String($event ?? ''))" @blur="emit('save')" />
      </div>
      <div class="editor-actions">
        <t-button variant="outline" :loading="generating" @click="emit('generate')">AI 生成</t-button>
        <t-button variant="outline" :loading="saving" @click="emit('save')">保存</t-button>
        <t-button theme="primary" @click="emit('export')">导出</t-button>
      </div>
    </header>

    <div class="source-toolbar">
      <div>
        <strong>生成知识来源</strong>
        <p>互联网搜索会补充公开信息，知识库内容仍作为内部事实的优先来源。</p>
      </div>
      <t-radio-group :value="sourceType" variant="default-filled" @change="onSourceTypeChange">
        <t-radio-button value="knowledge_base">仅知识库</t-radio-button>
        <t-radio-button value="knowledge_plus_web">知识库 + 互联网</t-radio-button>
      </t-radio-group>
    </div>

    <div v-if="generationMeta" class="generation-meta">
      <span>知识库来源 {{ generationMeta.knowledge }}</span>
      <span>互联网来源 {{ generationMeta.web }}</span>
      <span>模型 {{ generationMeta.model || '未知' }}</span>
    </div>

    <div class="editor-stage">
      <t-textarea
        :model-value="draft.content"
        class="editor-textarea"
        :autosize="false"
        @update:model-value="emit('update-content', String($event ?? ''))"
        @blur="emit('save')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { WritingDraft, WritingSourceType } from '@/api/writing'

defineProps<{
  draft: WritingDraft
  saving: boolean
  generating: boolean
  sourceType: WritingSourceType
  generationMeta: { knowledge: number; web: number; model: string } | null
}>()

const emit = defineEmits<{
  'update-title': [value: string]
  'update-content': [value: string]
  'update:source-type': [value: WritingSourceType]
  save: []
  generate: []
  export: []
}>()

function onSourceTypeChange(value: unknown) {
  emit('update:source-type', value === 'knowledge_plus_web' ? 'knowledge_plus_web' : 'knowledge_base')
}
</script>

<style scoped>
.editor-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 22px 24px 0;
}

.editor-title-wrap {
  min-width: 0;
  flex: 1;
}

.editor-tag {
  display: inline-flex;
  padding: 5px 9px;
  border-radius: var(--sk-radius-md);
  font-size: 12px;
  color: var(--brand-primary);
  background: var(--sk-brand-soft);
}

.editor-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.source-toolbar {
  margin: 16px 24px 0;
  padding: 12px 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border: 1px solid var(--border-soft);
  border-radius: var(--sk-radius-md);
  background: var(--sk-surface-soft);
}

.source-toolbar strong {
  color: var(--text-primary);
  font-size: 14px;
}

.source-toolbar p {
  margin: 3px 0 0;
  color: var(--text-secondary);
  font-size: 12px;
}

.generation-meta {
  margin: 10px 24px 0;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.generation-meta span {
  padding: 4px 8px;
  border-radius: var(--sk-radius-sm);
  color: var(--sk-brand-deep);
  background: var(--sk-brand-soft);
  font-size: 12px;
}

.editor-stage {
  padding: 18px 24px 24px;
  flex: 1;
  min-height: 520px;
}

:deep(.editor-textarea) {
  height: 100%;
}

:deep(.editor-textarea .t-textarea__inner) {
  min-height: 520px;
  padding: 16px;
  border-radius: var(--sk-radius-md);
  background: color-mix(in srgb, var(--surface-elevated) 94%, transparent);
  border-color: var(--border-soft);
  color: var(--text-primary);
  line-height: 1.8;
}

@media (max-width: 900px) {
  .editor-header,
  .source-toolbar {
    flex-direction: column;
  }

  .editor-actions {
    width: 100%;
  }
}
</style>
