<template>
  <t-dialog :visible="visible" header="导入文档" :footer="false" width="480px" @update:visible="emit('update:visible', $event)">
    <t-tabs v-model="uploadTab">
      <t-tab-panel value="file" label="文件上传">
        <t-upload v-model="fileList" :action="`/api/v1/smartknora/documents/upload`" :data="{ space_id: spaceId }" :headers="uploadHeaders" :max="20" :size-limit="{ size: 50, unit: 'MB' }" :accept="acceptFormats" multiple auto-upload @success="onUploadSuccess" @fail="onUploadFail">
          <t-button theme="primary"><template #icon><t-icon name="upload" /></template>选择文件</t-button>
          <template #tips>支持 PDF、Word、Excel、PPT、Markdown、TXT、图片，单文件 ≤ 50MB</template>
        </t-upload>
      </t-tab-panel>
      <t-tab-panel value="manual" label="手动录入">
        <div class="manual-form">
          <t-input v-model="manualForm.title" placeholder="文档标题" />
          <t-textarea v-model="manualForm.content" placeholder="Markdown 内容" :autosize="{ minRows: 6 }" />
          <t-button theme="primary" @click="handleManualCreate">保存</t-button>
        </div>
      </t-tab-panel>
    </t-tabs>
  </t-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import client from '@/api/client'
import { MessagePlugin } from 'tdesign-vue-next'

const props = defineProps<{
  visible: boolean
  spaceId: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  success: []
}>()

const uploadTab = ref('file')
const fileList = ref([])
const manualForm = ref({ title: '', content: '' })

const uploadHeaders = computed(() => ({ Authorization: `Bearer ${localStorage.getItem('access_token')}` }))
const acceptFormats = '.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.md,.txt,.png,.jpg,.jpeg,.mp3'

function closeDialog() {
  emit('update:visible', false)
}

function onUploadSuccess() {
  MessagePlugin.success('上传成功')
  emit('success')
  closeDialog()
}

function onUploadFail() {
  MessagePlugin.error('上传失败')
}

async function handleManualCreate() {
  if (!manualForm.value.title) {
    MessagePlugin.warning('请输入标题')
    return
  }
  try {
    await client.post('/documents/manual', { space_id: props.spaceId, title: manualForm.value.title, content: manualForm.value.content })
    MessagePlugin.success('保存成功')
    manualForm.value = { title: '', content: '' }
    emit('success')
    closeDialog()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '保存失败')
  }
}
</script>

<style scoped>
.manual-form { display: flex; flex-direction: column; gap: 10px; padding: 14px 0; }
</style>
