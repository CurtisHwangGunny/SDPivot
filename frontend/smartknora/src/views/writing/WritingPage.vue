<template>
  <div class="writing-page">
    <div class="writing-sidebar">
      <div class="sidebar-header">
        <h3>AI 写作</h3>
        <t-button theme="primary" size="small" @click="showNew = true">
          <template #icon><t-icon name="add" /></template>
          新建
        </t-button>
      </div>
      <div class="draft-list">
        <div v-for="d in drafts" :key="d.id" :class="['draft-item', { active: d.id === currentId }]" @click="loadDraft(d.id)">
          <t-icon name="file-paste" size="16px" />
          <div class="draft-info">
            <span class="draft-title">{{ d.title }}</span>
            <span class="draft-cat">{{ categoryLabel(d.category) }}</span>
          </div>
        </div>
        <div v-if="drafts.length === 0" class="no-drafts">暂无草稿</div>
      </div>
    </div>

    <div class="writing-main">
      <div v-if="!currentDraft" class="welcome">
        <t-icon name="edit" size="64px" style="color:#d0d0d0" />
        <h2>AI 写作辅助</h2>
        <p>选择写作类别，输入主题，AI 将基于知识库生成内容</p>
        <t-button theme="primary" size="large" @click="showNew = true">
          <template #icon><t-icon name="add" /></template>
          开始写作
        </t-button>
      </div>

      <template v-else>
        <div class="editor-header">
          <t-input v-model="currentDraft.title" placeholder="文档标题" size="large" @blur="saveDraft" />
          <t-space>
            <t-select v-model="exportFormat" style="width:100px">
              <t-option value="markdown" label="Markdown" />
              <t-option value="docx" label="Word" />
              <t-option value="pdf" label="PDF" />
            </t-select>
            <t-button variant="outline" @click="handleExport">
              <template #icon><t-icon name="download" /></template>
              导出
            </t-button>
          </t-space>
        </div>
        <t-textarea v-model="currentDraft.content" :autosize="{ minRows: 15, maxRows: 30 }" placeholder="在此编辑内容..." class="editor-textarea" @blur="saveDraft" />
      </template>
    </div>

    <t-dialog v-model:visible="showNew" header="AI 写作" :footer="false" width="560px">
      <div class="new-form">
        <t-form label-align="top">
          <t-form-item label="写作类别" name="category">
            <t-select v-model="newForm.category" size="large">
              <t-option v-for="c in CATEGORIES" :key="c.value" :value="c.value" :label="c.label" />
            </t-select>
          </t-form-item>
          <t-form-item label="主题/提示词" name="prompt">
            <t-input v-model="newForm.prompt" placeholder="例如：2026年Q2技术团队工作总结" size="large" />
          </t-form-item>
          <t-form-item label="知识来源" name="source_type">
            <t-radio-group v-model="newForm.source_type">
              <t-radio value="knowledge_base">仅知识库</t-radio>
              <t-radio value="knowledge_plus_web">知识库 + 互联网搜索</t-radio>
            </t-radio-group>
          </t-form-item>
          <t-form-item label="知识空间（选填）" name="space_id">
            <t-select v-model="newForm.space_id" placeholder="选择知识空间（留空则搜索全部）" clearable size="large">
              <t-option v-for="s in spaces" :key="s.id" :value="s.id" :label="s.name" />
            </t-select>
          </t-form-item>
        </t-form>
        <t-button theme="primary" block size="large" :loading="generating" :disabled="!newForm.category || !newForm.prompt" @click="handleGenerate">
          <template #icon><t-icon name="magic" /></template>
          AI 生成
        </t-button>
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { generateContent, createDraft, listDrafts, getDraft, updateDraft, exportDraft, CATEGORIES, type WritingDraft } from '@/api/writing'
import { listSpaces, type Space } from '@/api/spaces'
import { MessagePlugin } from 'tdesign-vue-next'

const drafts = ref<WritingDraft[]>([])
const spaces = ref<Space[]>([])
const currentId = ref('')
const currentDraft = ref<WritingDraft | null>(null)
const showNew = ref(false)
const generating = ref(false)
const newForm = ref({ category: 'notice', prompt: '', space_id: '', source_type: 'knowledge_base', web_search_enabled: false })
const exportFormat = ref('markdown')

function categoryLabel(v: string) { return CATEGORIES.find(c => c.value === v)?.label || v }

async function loadDrafts() {
  try { const res = await listDrafts(); drafts.value = (res.data as any).drafts || [] } catch { drafts.value = [] }
}
async function loadSpaces() {
  try { const res = await listSpaces(); spaces.value = (res.data as any).spaces || [] } catch { spaces.value = [] }
}

async function loadDraft(id: string) {
  currentId.value = id
  try { const res = await getDraft(id); currentDraft.value = (res.data as any).draft } catch { MessagePlugin.error('加载失败') }
}

async function handleGenerate() {
  generating.value = true
  try {
    newForm.value.web_search_enabled = newForm.value.source_type === 'knowledge_plus_web'
    const genRes = await generateContent(newForm.value)
    const content = (genRes.data as any).content
    const sourcesCount = (genRes.data as any).sources_count || 0

    const draftRes = await createDraft({ title: newForm.value.prompt, category: newForm.value.category, space_id: newForm.value.space_id, source_type: newForm.value.source_type, web_search_enabled: newForm.value.web_search_enabled })
    const draft = (draftRes.data as any).draft
    await updateDraft(draft.id, { title: newForm.value.prompt, content, status: 'draft' })
    draft.content = content
    drafts.value.unshift(draft)
    currentId.value = draft.id
    currentDraft.value = draft
    showNew.value = false
    newForm.value = { category: 'notice', prompt: '', space_id: '', source_type: 'knowledge_base', web_search_enabled: false }
    MessagePlugin.success(`生成完成${sourcesCount > 0 ? `（引用 ${sourcesCount} 条知识）` : ''}`)
    loadDrafts()
  } catch (e: any) { MessagePlugin.error(e.response?.data?.error || '生成失败') }
  finally { generating.value = false }
}

async function saveDraft() {
  if (!currentDraft.value) return
  try {
    await updateDraft(currentDraft.value.id, { title: currentDraft.value.title, content: currentDraft.value.content })
  } catch {}
}

async function handleExport() {
  if (!currentDraft.value) return
  try {
    const res = await exportDraft(currentDraft.value.id, exportFormat.value)
    const mimeType = exportFormat.value === 'markdown'
      ? 'text/markdown;charset=utf-8'
      : exportFormat.value === 'pdf'
        ? 'application/pdf'
        : 'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
    const blob = new Blob([res.data], { type: mimeType })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    const safeTitle = (currentDraft.value.title || 'smartknora-draft').replace(/[\/:*?"<>|]/g, '-')
    link.href = url
    link.download = exportFormat.value === 'markdown' ? `${safeTitle}.md` : `${safeTitle}.${exportFormat.value}`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
    MessagePlugin.success('导出完成')
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '导出失败')
  }
}

onMounted(() => { loadDrafts(); loadSpaces() })
</script>

<style scoped>
.writing-page { display: flex; height: calc(100vh - 104px); background: #fff; border-radius: 8px; overflow: hidden; }
.writing-sidebar { width: 260px; border-right: 1px solid #e7e7e7; display: flex; flex-direction: column; }
.sidebar-header { display: flex; justify-content: space-between; align-items: center; padding: 16px; border-bottom: 1px solid #f0f0f0; }
.sidebar-header h3 { font-size: 16px; font-weight: 600; }
.draft-list { flex: 1; overflow-y: auto; padding: 8px; }
.draft-item { display: flex; gap: 8px; padding: 10px 12px; border-radius: 6px; cursor: pointer; margin-bottom: 2px; }
.draft-item:hover { background: #f5f5f5; }
.draft-item.active { background: #e8f0ff; color: #014DB2; }
.draft-info { display: flex; flex-direction: column; }
.draft-title { font-size: 14px; font-weight: 500; }
.draft-cat { font-size: 12px; color: #999; }
.no-drafts { text-align: center; padding: 24px; color: #ccc; font-size: 13px; }
.writing-main { flex: 1; display: flex; flex-direction: column; padding: 24px; overflow-y: auto; }
.welcome { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; color: #999; }
.welcome h2 { font-size: 22px; color: #333; }
.editor-header { display: flex; gap: 12px; margin-bottom: 16px; align-items: center; }
.editor-header :deep(.t-input) { flex: 1; }
.editor-textarea { flex: 1; }
.new-form { padding: 16px 0; display: flex; flex-direction: column; gap: 16px; }
</style>
