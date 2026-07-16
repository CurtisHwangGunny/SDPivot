<template>
  <t-dialog :visible="visible" header="导入文档" :footer="false" width="560px" @update:visible="emit('update:visible', $event)">
    <t-tabs v-model="localImportTab">
      <t-tab-panel value="file" label="文件上传">
        <div class="upload-area">
          <t-upload
            v-model="localFileList"
            :action="`/api/v1/smartknora/documents/upload`"
            :data="{ space_id: spaceId }"
            :headers="uploadHeaders"
            :max="20"
            :size-limit="{ size: 50, unit: 'MB' }"
            :accept="acceptFormats"
            multiple
            auto-upload
            theme="file"
            @success="onUploadSuccess"
            @fail="onUploadFail"
          >
            <t-button theme="primary" block size="large">
              <template #icon><t-icon name="upload" /></template>
              点击或拖拽文件到此处上传
            </t-button>
            <template #tips>
              <div class="upload-tips">
                支持 PDF、Word、Excel、PPT、Markdown、TXT、图片、音频<br/>
                单文件 ≤ 50MB，一次最多 20 个文件
              </div>
            </template>
          </t-upload>
        </div>
      </t-tab-panel>

      <t-tab-panel value="url" label="网页链接">
        <div class="url-form">
          <t-input v-model="urlForm.url" placeholder="https://example.com/article" size="large">
            <template #prefix-icon><t-icon name="link" /></template>
          </t-input>
          <t-input v-model="urlForm.tags" placeholder="标签（选填，逗号分隔）" />
          <t-button theme="primary" block size="large" :loading="urlLoading" :disabled="!urlForm.url" @click="handleUrlImport">
            <template #icon><t-icon name="download" /></template>
            导入网页
          </t-button>
          <p class="form-tip">系统将自动抓取网页正文内容并导入</p>
        </div>
      </t-tab-panel>

      <t-tab-panel value="manual" label="手动录入">
        <div class="manual-form">
          <t-input v-model="manualForm.title" placeholder="文档标题（必填）" size="large" />
          <t-textarea
            v-model="manualForm.content"
            placeholder="输入 Markdown 格式内容..."
            :autosize="{ minRows: 8, maxRows: 20 }"
          />
          <t-input v-model="manualForm.tags" placeholder="标签（选填，逗号分隔）" />
          <t-button theme="primary" block size="large" :loading="manualLoading" :disabled="!manualForm.title || !manualForm.content" @click="handleManualCreate">
            <template #icon><t-icon name="save" /></template>
            保存
          </t-button>
        </div>
      </t-tab-panel>
    </t-tabs>
  </t-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { uploadManual, uploadFromURL } from '@/api/documents'

const props = defineProps<{
  visible: boolean
  spaceId: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  success: []
}>()

const localImportTab = ref('file')
const localFileList = ref([])
const urlForm = ref({ url: '', tags: '' })
const urlLoading = ref(false)
const manualForm = ref({ title: '', content: '', tags: '' })
const manualLoading = ref(false)

const uploadHeaders = computed(() => ({ Authorization: `Bearer ${localStorage.getItem('access_token')}` }))
const acceptFormats = '.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.md,.txt,.png,.jpg,.jpeg,.mp3'

watch(() => props.visible, (visible) => {
  if (!visible) {
    localImportTab.value = 'file'
  }
})

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

async function handleUrlImport() {
  if (!urlForm.value.url) return
  urlLoading.value = true
  try {
    await uploadFromURL({ space_id: props.spaceId, url: urlForm.value.url, tags: urlForm.value.tags })
    MessagePlugin.success('网页导入成功')
    urlForm.value = { url: '', tags: '' }
    emit('success')
    closeDialog()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '导入失败')
  } finally {
    urlLoading.value = false
  }
}

async function handleManualCreate() {
  if (!manualForm.value.title || !manualForm.value.content) return
  manualLoading.value = true
  try {
    await uploadManual({ space_id: props.spaceId, title: manualForm.value.title, content: manualForm.value.content, tags: manualForm.value.tags })
    MessagePlugin.success('文档已保存')
    manualForm.value = { title: '', content: '', tags: '' }
    emit('success')
    closeDialog()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '保存失败')
  } finally {
    manualLoading.value = false
  }
}
</script>

<style scoped>
.upload-area { padding: 20px 0; }
.upload-tips { font-size: 12px; color: var(--text-secondary); text-align: center; margin-top: 8px; line-height: 1.6; }
.url-form { display: flex; flex-direction: column; gap: 12px; padding: 20px 0; }
.form-tip { font-size: 12px; color: var(--text-secondary); text-align: center; }
.manual-form { display: flex; flex-direction: column; gap: 12px; padding: 20px 0; }
</style>
