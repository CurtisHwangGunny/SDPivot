<template>
  <footer class="input-shell">
    <div class="input-panel">
      <t-textarea
        :model-value="modelValue"
        :autosize="{ minRows: 3, maxRows: 8 }"
        placeholder="输入你的问题，系统将结合知识内容进行回答..."
        @update:model-value="emit('update:modelValue', String($event ?? ''))"
      />
      <div class="input-footer">
        <div class="model-control">
          <span class="input-tip">回答模型</span>
          <t-select
            :model-value="modelId"
            :options="modelOptions"
            :disabled="sending || !models.length"
            placeholder="选择模型"
            @update:model-value="emit('update:modelId', String($event ?? ''))"
          />
        </div>
        <t-button theme="primary" size="large" :loading="sending" :disabled="!modelValue.trim() || !modelId" @click="emit('submit')">
          发送问题
        </t-button>
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { QAAvailableModel } from '@/api/qa'

const props = defineProps<{
  modelValue: string
  modelId: string
  models: QAAvailableModel[]
  sending: boolean
}>()

const modelOptions = computed(() => props.models.map(model => ({
  label: model.display_name || model.name,
  value: model.id,
})))

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'update:modelId': [value: string]
  submit: []
}>()
</script>

<style scoped>
.input-shell {
  padding: 18px 24px 24px;
}

.input-panel {
  padding: 16px;
  border-radius: 12px;
  background: color-mix(in srgb, var(--surface-elevated) 96%, transparent);
  border: 1px solid var(--border-soft);
  box-shadow: var(--shadow-soft);
}

.input-footer {
  margin-top: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.model-control {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 260px;
}

.model-control :deep(.t-select__wrap) {
  min-width: 190px;
}

.input-tip {
  flex: 0 0 auto;
  color: var(--text-secondary);
  font-size: 13px;
}

@media (max-width: 900px) {
  .input-footer {
    flex-direction: column;
    align-items: stretch;
  }

  .model-control,
  .model-control :deep(.t-select__wrap) {
    width: 100%;
    min-width: 0;
  }
}
</style>
