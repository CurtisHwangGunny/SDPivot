<template>
  <t-dialog
    :visible="visible"
    header="加入企业"
    width="520px"
    @update:visible="emit('update:visible', $event)"
    @confirm="emit('confirm')"
    :confirm-btn="{ loading: loading }"
  >
    <div class="org-dialog">
      <div class="dialog-tip">输入企业 ID 后即可申请加入，适合快速接入已有企业工作区。</div>
      <t-form label-align="top">
      <t-form-item label="企业ID" name="org_id">
        <t-input :model-value="form.org_id" placeholder="请输入企业ID" @update:model-value="updateField(String($event || ''))" />
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
