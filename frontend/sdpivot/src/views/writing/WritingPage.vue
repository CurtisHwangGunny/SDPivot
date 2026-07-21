<template>
  <div class="writing-page page-section-card">
    <aside class="writing-sidebar">
      <div class="sidebar-head">
        <p class="sidebar-tag">AI Writing</p>
        <h2>写作工作台</h2>
        <p>管理草稿、快速生成内容，并统一导出你的写作成果。</p>
      </div>

      <t-button theme="primary" block size="large" @click="openCreateDialog">
        <template #icon><t-icon name="add" /></template>
        新建草稿
      </t-button>

      <div class="draft-list">
        <template v-if="drafts.length">
          <button
            v-for="draft in drafts"
            :key="draft.id"
            type="button"
            class="draft-item"
            :class="{ active: currentDraft?.id === draft.id }"
            @click="selectDraft(draft.id)"
          >
            <div class="draft-icon"><t-icon name="edit-1" /></div>
            <div class="draft-info">
              <strong>{{ draft.title || '未命名草稿' }}</strong>
              <span>{{ draft.updated_at ? formatDate(draft.updated_at) : '刚刚创建' }}</span>
            </div>
          </button>
        </template>
        <div v-else class="no-drafts">还没有草稿，先创建一篇新的内容吧。</div>
      </div>
    </aside>

    <section class="writing-main page-section-card">
      <div v-if="!currentDraft" class="welcome">
        <div class="welcome-mark"><t-icon name="edit-1" size="30px" /></div>
        <h3>开始你的下一篇内容</h3>
        <p>创建草稿后即可在右侧统一编辑，并保留后续生成、润色、导出等扩展能力。</p>
        <t-button theme="primary" size="large" @click="openCreateDialog">新建草稿</t-button>
      </div>

      <Suspense v-else>
        <WritingEditorPane
          :draft="currentDraft"
          :saving="saving"
          :generating="generating"
          :source-type="sourceType"
          :generation-meta="generationMeta"
          @update-title="updateDraftTitle"
          @update-content="updateDraftContent"
          @save="saveDraft"
          @update:source-type="sourceType = $event"
          @generate="generateFromPrompt"
          @export="exportCurrent"
        />
        <template #fallback>
          <div class="editor-loading">
            <t-loading size="small" text="正在加载编辑器..." />
          </div>
        </template>
      </Suspense>
    </section>

    <Suspense v-if="showCreateDialog">
      <WritingCreateDraftDialog
        :visible="showCreateDialog"
        :title="newDraft.title"
        :creating="creating"
        @update:visible="showCreateDialog = $event"
        @update:title="newDraft.title = $event"
        @confirm="createDraftItem"
      />
      <template #fallback>
        <div class="dialog-loading">
          <t-loading size="small" text="正在加载草稿弹窗..." />
        </div>
      </template>
    </Suspense>
  </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent, onMounted, ref } from 'vue'
import { createDraft, listDrafts, getDraft, updateDraft, exportDraft, generateContent, type WritingDraft, type WritingSourceType } from '@/api/writing'
import { MessagePlugin } from 'tdesign-vue-next'

const WritingEditorPane = defineAsyncComponent(() => import('@/components/writing/WritingEditorPane.vue'))
const WritingCreateDraftDialog = defineAsyncComponent(() => import('@/components/writing/WritingCreateDraftDialog.vue'))

const drafts = ref<WritingDraft[]>([])
const currentDraft = ref<WritingDraft | null>(null)
const showCreateDialog = ref(false)
const newDraft = ref({ title: '' })
const creating = ref(false)
const saving = ref(false)
const generating = ref(false)
const sourceType = ref<WritingSourceType>('knowledge_base')
const generationMeta = ref<{ knowledge: number; web: number; model: string } | null>(null)

function formatDate(value: string) {
  return value ? new Date(value).toLocaleDateString('zh-CN') : ''
}

async function loadDrafts() {
  try {
    const res = await listDrafts()
    drafts.value = res.data.drafts || []
    if (!currentDraft.value && drafts.value.length) {
      await selectDraft(drafts.value[0].id)
    }
  } catch {
    drafts.value = []
  }
}

async function selectDraft(id: string) {
  try {
    const res = await getDraft(id)
    currentDraft.value = res.data.draft
    sourceType.value = res.data.draft.source_type === 'knowledge_plus_web' ? 'knowledge_plus_web' : 'knowledge_base'
    generationMeta.value = null
  } catch {
    MessagePlugin.error('读取草稿失败')
  }
}

function openCreateDialog() {
  showCreateDialog.value = true
}

function updateDraftTitle(title: string) {
  if (currentDraft.value) currentDraft.value.title = title
}

function updateDraftContent(content: string) {
  if (currentDraft.value) currentDraft.value.content = content
}

async function createDraftItem() {
  if (!newDraft.value.title.trim()) {
    MessagePlugin.warning('请输入草稿标题')
    return
  }
  creating.value = true
  try {
    const res = await createDraft({
      title: newDraft.value.title,
      category: 'report',
      source_type: sourceType.value,
      web_search_enabled: sourceType.value === 'knowledge_plus_web',
    })
    const draft = res.data.draft
    showCreateDialog.value = false
    newDraft.value.title = ''
    await loadDrafts()
    if (draft?.id) {
      await selectDraft(draft.id)
    }
  } catch {
    MessagePlugin.error('创建草稿失败')
  } finally {
    creating.value = false
  }
}

async function saveDraft() {
  if (!currentDraft.value?.id) return
  saving.value = true
  try {
    await updateDraft(currentDraft.value.id, {
      title: currentDraft.value.title,
      content: currentDraft.value.content,
    })
    await loadDrafts()
  } catch {
    MessagePlugin.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function generateFromPrompt() {
  if (!currentDraft.value?.id) return
  generating.value = true
  try {
    const res = await generateContent({
      category: currentDraft.value.category || 'report',
      prompt: currentDraft.value.content || currentDraft.value.title,
      space_id: currentDraft.value.space_id || undefined,
      source_type: sourceType.value,
      web_search_enabled: sourceType.value === 'knowledge_plus_web',
    })
    currentDraft.value.source_type = res.data.source_type
    currentDraft.value.web_search_enabled = res.data.web_search_enabled
    generationMeta.value = {
      knowledge: res.data.knowledge_sources_count,
      web: res.data.web_sources_count,
      model: res.data.model,
    }
    currentDraft.value.content = res.data.content || currentDraft.value.content
    await saveDraft()
  } catch {
    MessagePlugin.error('生成失败')
  } finally {
    generating.value = false
  }
}

async function exportCurrent() {
  if (!currentDraft.value?.id) return
  try {
    const res = await exportDraft(currentDraft.value.id, 'docx')
    const url = window.URL.createObjectURL(new Blob([res.data as BlobPart]))
    const link = document.createElement('a')
    link.href = url
    link.download = `${currentDraft.value.title || 'draft'}.docx`
    link.click()
    window.URL.revokeObjectURL(url)
  } catch {
    MessagePlugin.error('导出失败')
  }
}

onMounted(loadDrafts)
</script>

<style scoped>
.writing-page {
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  gap: 18px;
  padding: 18px;
  background: transparent;
}

.writing-sidebar {
  padding: 20px;
  border-radius: 12px;
  background: linear-gradient(180deg, color-mix(in srgb, var(--surface-elevated) 92%, transparent) 0%, color-mix(in srgb, var(--sdp-surface-soft) 86%, transparent) 100%);
  border: 1px solid var(--border-soft);
}

.sidebar-tag {
  display: inline-block;
  font-size: 12px;
  color: var(--brand-primary);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.sidebar-head h2 {
  margin: 8px 0 0;
  font-size: 24px;
  color: var(--text-primary);
}

.sidebar-head p {
  margin: 8px 0 0;
  color: var(--text-secondary);
  font-size: 14px;
}

.draft-list {
  margin-top: 20px;
  flex: 1;
  overflow-y: auto;
}

.draft-item {
  width: 100%;
  display: flex;
  gap: 12px;
  padding: 12px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  transition: background 0.24s ease, transform 0.24s cubic-bezier(0.16, 1, 0.3, 1);
}

.draft-item + .draft-item {
  margin-top: 8px;
}

.draft-item:hover {
  background: rgba(0, 185, 107, 0.08);
}

.draft-item.active {
  background: rgba(0, 185, 107, 0.14);
  transform: translateX(2px);
}

.draft-icon {
  width: 36px;
  height: 36px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  background: color-mix(in srgb, var(--surface-elevated) 92%, transparent);
  border: 1px solid var(--border-soft);
  color: var(--brand-primary);
  flex-shrink: 0;
}

.draft-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.draft-info strong {
  font-size: 14px;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.draft-info span {
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-secondary);
}

.no-drafts {
  padding: 18px 10px;
  color: var(--text-secondary);
  font-size: 13px;
}

.writing-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: linear-gradient(180deg, color-mix(in srgb, var(--surface-elevated) 92%, transparent) 0%, color-mix(in srgb, var(--sdp-surface-soft) 88%, transparent) 100%);
}

.welcome {
  flex: 1;
  padding: 28px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.welcome-mark {
  width: 72px;
  height: 72px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  background: var(--sdp-brand-soft);
  color: var(--brand-primary);
}

.welcome h3 {
  margin: 20px 0 0;
  font-size: 28px;
  color: var(--text-primary);
}

.welcome p {
  max-width: 520px;
  margin: 14px auto 0;
  color: var(--text-secondary);
  line-height: 1.7;
}

.editor-loading,
.dialog-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 160px;
  color: var(--text-secondary);
}

@media (max-width: 1080px) {
  .writing-page {
    grid-template-columns: 1fr;
  }
}
</style>
