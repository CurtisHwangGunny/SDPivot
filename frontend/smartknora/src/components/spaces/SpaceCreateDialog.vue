<template>
  <t-dialog
    :visible="visible"
    header="创建知识空间"
    width="640px"
    @update:visible="emit('update:visible', $event)"
    @confirm="emit('confirm')"
    :confirm-btn="{ loading: loading, content: '创建并继续' }"
    :cancel-btn="{ content: '取消' }"
  >
    <div class="create-dialog">
      <div class="dialog-tip">
        先建立空间，再继续导入文档、绑定问答与写作来源。当前仍沿用现有接口，后续可升级为多步骤向导。
      </div>
      <t-form label-align="top">
        <t-form-item label="空间名称" name="name">
          <t-input :model-value="form.name" placeholder="例如：售前知识库 / 产品规范库" :maxlength="30" @update:model-value="updateField('name', String($event || ''))" />
        </t-form-item>
        <t-form-item label="空间描述" name="description">
          <t-textarea :model-value="form.description" placeholder="建议写明空间用途、覆盖资料范围和使用对象" :autosize="{ minRows: 3 }" @update:model-value="updateField('description', String($event || ''))" />
        </t-form-item>
        <t-form-item label="可见性" name="visibility">
          <t-select :model-value="form.visibility" @update:model-value="updateField('visibility', String($event || 'private'))">
            <t-option value="private" label="私密" />
            <t-option value="team" label="团队可见" />
            <t-option value="enterprise" label="企业可见" />
          </t-select>
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
    visibility: string
  }
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'update:form', value: { name: string; description: string; visibility: string }): void
  (e: 'confirm'): void
}>()

function updateField(field: 'name' | 'description' | 'visibility', value: string) {
  emit('update:form', {
    ...props.form,
    [field]: value,
  })
}
</script>

<style scoped>
.create-dialog {
  padding-top: 8px;
}

.dialog-tip {
  margin-bottom: 18px;
  padding: 14px 16px;
  border-radius: 16px;
  background: var(--sk-brand-soft);
  color: var(--sk-brand-deep);
  font-size: 13px;
  line-height: 1.6;
}
</style>
