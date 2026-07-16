<template>
  <t-dialog
    :visible="visible"
    header="创建企业"
    @update:visible="emit('update:visible', $event)"
    @confirm="emit('confirm')"
    :confirm-btn="{ loading: loading }"
  >
    <t-form>
      <t-form-item label="企业名称" name="name">
        <t-input :model-value="form.name" placeholder="请输入企业名称" @update:model-value="updateField('name', String($event || ''))" />
      </t-form-item>
      <t-form-item label="企业描述" name="description">
        <t-textarea :model-value="form.description" placeholder="选填" :autosize="{ minRows: 2 }" @update:model-value="updateField('description', String($event || ''))" />
      </t-form-item>
    </t-form>
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
