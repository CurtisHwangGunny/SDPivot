<template>
  <t-dialog
    :visible="visible"
    header="创建企业"
    width="560px"
    @update:visible="emit('update:visible', $event)"
    @confirm="emit('confirm')"
    :confirm-btn="{ loading: loading }"
  >
    <div class="org-dialog">
      <div class="dialog-tip">创建企业后即可统一管理成员、空间和后续企业级配置。</div>
      <t-form label-align="top">
      <t-form-item label="企业名称" name="name">
        <t-input :model-value="form.name" placeholder="请输入企业名称" @update:model-value="updateField('name', String($event || ''))" />
      </t-form-item>
      <t-form-item label="企业描述" name="description">
        <t-textarea :model-value="form.description" placeholder="选填" :autosize="{ minRows: 2 }" @update:model-value="updateField('description', String($event || ''))" />
      </t-form-item>
      </t-form>
    </div>
  </t-dialog>
</template>

<script setup lang="ts">
const props = defineProps<{
  visible: boolean
  loading: boolean
  form: {
    name: string
    description: string
  }
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'update:form', value: { name: string; description: string }): void
  (e: 'confirm'): void
}>()

function updateField(field: 'name' | 'description', value: string) {
  emit('update:form', {
    ...props.form,
    [field]: value,
  })
}
</script>

<style scoped>
.org-dialog {
  padding-top: 6px;
}

.dialog-tip {
  margin-bottom: 14px;
  padding: 12px 14px;
  border-radius: 12px;
  background: var(--sk-brand-soft);
  color: var(--sk-brand-deep);
  font-size: 12px;
  line-height: 1.6;
}
</style>
