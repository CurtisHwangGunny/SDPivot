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
import type { WritingDraft } from '@/api/writing'

defineProps<{
  draft: WritingDraft
  saving: boolean
  generating: boolean
}>()

const emit = defineEmits<{
  'update-title': [value: string]
  'update-content': [value: string]
  save: []
  generate: []
  export: []
}>()
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
  border-radius: 999px;
  font-size: 12px;
  color: var(--brand-primary);
  background: var(--sk-brand-soft);
}

.editor-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
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
  border-radius: 16px;
  background: color-mix(in srgb, var(--surface-elevated) 94%, transparent);
  border-color: var(--border-soft);
  color: var(--text-primary);
  line-height: 1.8;
}

@media (max-width: 900px) {
  .editor-header {
    flex-direction: column;
  }

  .editor-actions {
    width: 100%;
  }
}
</style>
