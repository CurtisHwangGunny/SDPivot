<template>
  <t-dialog :visible="visible" header="新建草稿" :confirm-btn="{ loading: creating }" @update:visible="emit('update:visible', $event)" @confirm="emit('confirm')">
    <div class="new-form">
      <div class="dialog-tip">先创建一个标题清晰的草稿，后续即可继续 AI 生成、手工编辑与导出。</div>
      <t-input :model-value="title" placeholder="请输入草稿标题" @update:model-value="emit('update:title', String($event ?? ''))" />
    </div>
  </t-dialog>
</template>

<script setup lang="ts">
defineProps<{
  visible: boolean
  title: string
  creating: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'update:title': [value: string]
  confirm: []
}>()
</script>

<style scoped>
.new-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.dialog-tip {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.6;
}
</style>
