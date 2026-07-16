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
        <div class="input-tip">支持连续追问，后续可继续接入引用抽屉与模型设置。</div>
        <t-button theme="primary" size="large" :loading="sending" :disabled="!modelValue.trim()" @click="emit('submit')">
          发送问题
        </t-button>
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
defineProps<{
  modelValue: string
  sending: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  submit: []
}>()
</script>

<style scoped>
.input-shell {
  padding: 20px 28px 28px;
}

.input-panel {
  padding: 18px;
  border-radius: 22px;
  background: color-mix(in srgb, var(--surface-elevated) 96%, transparent);
  border: 1px solid var(--border-soft);
  box-shadow: var(--shadow-soft);
}

.input-footer {
  margin-top: 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.input-tip {
  color: var(--text-secondary);
  font-size: 13px;
}

@media (max-width: 900px) {
  .input-footer {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
