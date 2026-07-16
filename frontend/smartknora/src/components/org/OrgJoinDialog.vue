<template>
  <t-dialog
    :visible="visible"
    header="加入企业"
    @update:visible="emit('update:visible', $event)"
    @confirm="emit('confirm')"
    :confirm-btn="{ loading: loading }"
  >
    <t-form>
      <t-form-item label="企业ID" name="org_id">
        <t-input :model-value="form.org_id" placeholder="请输入企业ID" @update:model-value="updateField(String($event || ''))" />
      </t-form-item>
    </t-form>
  </t-dialog>
</template>

<script setup lang="ts">
const props = defineProps<{
  visible: boolean
  loading: boolean
  form: {
    org_id: string
  }
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'update:form', value: { org_id: string }): void
  (e: 'confirm'): void
}>()

function updateField(value: string) {
  emit('update:form', {
    ...props.form,
    org_id: value,
  })
}
</script>
