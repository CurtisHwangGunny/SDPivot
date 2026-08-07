<template>
  <SdpSidebarLayout>
    <div class="sdp-writing-workspace">
      <aside class="sdp-writing-workspace__drafts" aria-labelledby="draft-list-title">
        <header>
          <div>
            <p>AI Writing</p>
            <h1 id="draft-list-title">写作草稿</h1>
          </div>
          <button type="button" aria-label="新建写作草稿" :disabled="creating" @click="createNewDraft">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg>
            {{ creating ? '创建中' : '新建' }}
          </button>
        </header>

        <p v-if="loadError" class="sdp-writing-workspace__error" role="alert">{{ loadError }}</p>
        <div v-if="loading" class="sdp-writing-workspace__loading" role="status">正在加载草稿...</div>
        <div v-else-if="groupedDrafts.length" class="sdp-writing-workspace__draft-scroll" @keydown="handleDraftKeydown">
          <section v-for="group in groupedDrafts" :key="group.status">
            <div class="sdp-writing-workspace__group-heading">
              <h2>{{ statusLabel(group.status) }}</h2>
              <span>{{ group.items.length }}</span>
            </div>
            <div role="list" :aria-label="`${statusLabel(group.status)}草稿`">
              <button
                v-for="draft in group.items"
                :key="draft.id"
                class="sdp-writing-workspace__draft-item"
                :class="{ 'sdp-writing-workspace__draft-item--active': currentDraft?.id === draft.id }"
                type="button"
                :aria-current="currentDraft?.id === draft.id ? 'true' : undefined"
                :aria-label="`打开草稿 ${draft.title || '未命名草稿'}`"
                @click="selectDraft(draft.id)"
              >
                <span class="sdp-writing-workspace__draft-icon" aria-hidden="true">
                  <svg viewBox="0 0 24 24"><path d="m4 20 4.5-1 10-10a2.12 2.12 0 0 0-3-3l-10 10L4 20Zm10-12 3 3M4 20h16" /></svg>
                </span>
                <span class="sdp-writing-workspace__draft-copy">
                  <strong>{{ draft.title || '未命名草稿' }}</strong>
                  <small>{{ categoryLabel(draft.category) }} · {{ formatDate(draft.updated_at) }}</small>
                </span>
                <span class="sdp-writing-workspace__state">{{ statusLabel(draft.status) }}</span>
              </button>
            </div>
          </section>
        </div>
        <div v-else class="sdp-writing-workspace__empty">
          <strong>还没有写作草稿</strong>
          <span>创建第一篇内容，开始组织你的写作链路。</span>
        </div>
      </aside>

      <main class="sdp-writing-workspace__editor" aria-labelledby="writing-editor-title">
        <header class="sdp-writing-workspace__topbar">
          <div>
            <p>Writing Studio</p>
            <h2 id="writing-editor-title">{{ currentDraft?.title || 'AI 写作工作台' }}</h2>
          </div>
          <div class="sdp-writing-workspace__tools" role="toolbar" aria-label="写作工具">
            <label for="writing-template">
              <span>模板</span>
              <select id="writing-template" v-model="selectedTemplate" aria-label="选择写作模板">
                <option v-for="template in templates" :key="template.value" :value="template.value">{{ template.label }}</option>
              </select>
            </label>
            <label for="writing-source">
              <span>知识来源</span>
              <select id="writing-source" v-model="knowledgeSource" aria-label="选择写作知识来源">
                <option value="knowledge_base">企业知识库</option>
                <option value="knowledge_plus_web">知识库 + 互联网</option>
              </select>
            </label>
            <button type="button" aria-label="导出当前草稿为文本文件" :disabled="!currentDraft" @click="exportCurrentDraft">
              <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3v12m-4-4 4 4 4-4M5 20h14" /></svg>
              导出
            </button>
            <button type="button" aria-label="管理写作模板和优先级" @click="managementVisible = true">
              <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 6h16M7 12h10M10 18h4" /></svg>
              模板/优先级
            </button>
          </div>
        </header>

        <div v-if="!currentDraft" class="sdp-writing-workspace__welcome">
          <span aria-hidden="true"><svg viewBox="0 0 24 24"><path d="m4 20 4.5-1 10-10a2.12 2.12 0 0 0-3-3l-10 10L4 20Zm10-12 3 3M4 20h16" /></svg></span>
          <h3>让写作从清晰的优先级开始</h3>
          <p>模板约束结构，自学习补充语气，自定义要求完成最后校准。</p>
          <button type="button" aria-label="创建第一篇写作草稿" @click="createNewDraft">创建草稿</button>
        </div>

        <div v-else class="sdp-writing-workspace__canvas">
          <section class="sdp-writing-workspace__priority" aria-labelledby="priority-chain-title">
            <div>
              <p>Generation Priority</p>
              <h3 id="priority-chain-title">生成优先级链</h3>
            </div>
            <ol>
              <li><span>P1</span><strong>模板规则</strong><small>{{ templateLabel }}</small></li>
              <li aria-hidden="true"><svg viewBox="0 0 24 24"><path d="m9 5 7 7-7 7" /></svg></li>
              <li><span>P2</span><strong>自学习风格</strong><small>团队表达偏好</small></li>
              <li aria-hidden="true"><svg viewBox="0 0 24 24"><path d="m9 5 7 7-7 7" /></svg></li>
              <li><span>P3</span><strong>自定义要求</strong><small>当前草稿指令</small></li>
            </ol>
          </section>

          <section class="sdp-writing-workspace__paper" aria-label="草稿编辑器">
            <div class="sdp-writing-workspace__metadata">
              <label for="draft-category">
                <span>文档类型</span>
                <select id="draft-category" v-model="currentDraft.category" aria-label="选择文档类型">
                  <option v-for="category in CATEGORIES" :key="category.value" :value="category.value">{{ category.label }}</option>
                </select>
              </label>
              <label for="draft-status">
                <span>草稿状态</span>
                <select id="draft-status" v-model="currentDraft.status" aria-label="选择草稿状态" @change="saveDraft">
                  <option value="draft">草稿</option>
                  <option value="review">待审核</option>
                  <option value="completed">已完成</option>
                </select>
              </label>
              <div>
                <span>保存状态</span>
                <strong role="status">{{ saving ? '正在保存...' : saveState }}</strong>
              </div>
            </div>

            <label class="sdp-writing-workspace__title" for="draft-title">
              <span class="sr-only">草稿标题</span>
              <input id="draft-title" v-model="currentDraft.title" type="text" placeholder="输入文档标题" aria-label="草稿标题" @blur="saveDraft">
            </label>
            <div class="sdp-writing-workspace__rule" />
            <label class="sdp-writing-workspace__content" for="draft-content">
              <span class="sr-only">草稿正文</span>
              <textarea id="draft-content" v-model="currentDraft.content" placeholder="从这里开始写作..." aria-label="草稿正文" @blur="saveDraft" />
            </label>
          </section>

          <div class="sdp-writing-workspace__ai-bar" role="status" aria-label="AI 写作建议占位区">
            <span aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M12 3 9.8 8.8 4 11l5.8 2.2L12 19l2.2-5.8L20 11l-5.8-2.2L12 3Z" /></svg></span>
            <div><strong>AI 建议</strong><small>选中文本后，可在此获得续写、润色与改写建议。</small></div>
            <button type="button" aria-label="查看 AI 写作建议" disabled>即将开放</button>
          </div>
          <p v-if="saveError" class="sdp-writing-workspace__save-error" role="alert">{{ saveError }}</p>
        </div>
      </main>

      <div v-if="managementVisible" class="sdp-writing-workspace__management-backdrop" role="presentation" @click.self="managementVisible = false">
        <aside class="sdp-writing-workspace__management" role="dialog" aria-modal="true" aria-labelledby="writing-management-title">
          <header>
            <div><p>Template Governance</p><h2 id="writing-management-title">模板 / 优先级管理</h2></div>
            <button type="button" aria-label="关闭模板管理" @click="managementVisible = false">关闭</button>
          </header>

          <section class="sdp-writing-workspace__priority-guide">
            <strong>固定生成链</strong>
            <ol><li><span>P1</span>标准 / 自定义模板结构</li><li><span>P2</span>同类型标签文档自学习</li><li><span>P3</span>当前草稿补充要求</li></ol>
          </section>

          <section v-if="isAdmin" class="sdp-writing-workspace__management-section">
            <div class="sdp-writing-workspace__section-title"><div><span>Categories</span><h3>写作类别</h3></div><button type="button" @click="editCategory()">新增类别</button></div>
            <div class="sdp-writing-workspace__management-list">
              <article v-for="category in managedCategories" :key="category.id"><div><strong>{{ category.name }}</strong><small>{{ category.description || '暂无说明' }} · 排序 {{ category.sort }}</small></div><div><button type="button" @click="editCategory(category)">编辑</button><button type="button" @click="removeCategory(category)">删除</button></div></article>
              <p v-if="!managedCategories.length">暂无自定义写作类别。</p>
            </div>
          </section>

          <section class="sdp-writing-workspace__management-section">
            <div class="sdp-writing-workspace__section-title"><div><span>Templates</span><h3>模板库</h3></div><button v-if="canManageTemplates" type="button" :disabled="!managedCategories.length" @click="editTemplate()">新增模板</button></div>
            <div class="sdp-writing-workspace__management-list">
              <article v-for="template in managedTemplates" :key="template.id"><div><strong>{{ template.name }} <i v-if="template.is_builtin">内置</i></strong><small>{{ categoryName(template.category_id) }} · 排序 {{ template.sort }}</small><p>{{ template.content }}</p></div><div v-if="canEditTemplate(template)"><button type="button" @click="editTemplate(template)">编辑</button><button type="button" @click="removeTemplate(template)">删除</button></div></article>
              <p v-if="!managedTemplates.length">暂无可用模板。管理员先创建类别后，编辑者即可维护自定义模板。</p>
            </div>
          </section>

          <form v-if="categoryEditing" class="sdp-writing-workspace__management-form" @submit.prevent="saveCategory">
            <h3>{{ categoryForm.id ? '编辑类别' : '新增类别' }}</h3>
            <label>名称<input v-model="categoryForm.name" required maxlength="100"></label>
            <label>说明<textarea v-model="categoryForm.description" rows="3" /></label>
            <label>排序<input v-model.number="categoryForm.sort" type="number"></label>
            <footer><button type="button" @click="categoryEditing = false">取消</button><button type="submit" :disabled="managementSaving">保存</button></footer>
          </form>

          <form v-if="templateEditing" class="sdp-writing-workspace__management-form" @submit.prevent="saveTemplate">
            <h3>{{ templateForm.id ? '编辑模板' : '新增模板' }}</h3>
            <label>类别<select v-model="templateForm.category_id" required><option v-for="category in managedCategories" :key="category.id" :value="category.id">{{ category.name }}</option></select></label>
            <label>名称<input v-model="templateForm.name" required maxlength="100"></label>
            <label>模板内容<textarea v-model="templateForm.content" required rows="8" /></label>
            <label>排序<input v-model.number="templateForm.sort" type="number"></label>
            <label v-if="isAdmin" class="sdp-writing-workspace__builtin"><input v-model="templateForm.is_builtin" type="checkbox">设为内置标准模板</label>
            <footer><button type="button" @click="templateEditing = false">取消</button><button type="submit" :disabled="managementSaving">保存</button></footer>
          </form>
          <p v-if="managementError" class="sdp-writing-workspace__save-error" role="alert">{{ managementError }}</p>
        </aside>
      </div>
    </div>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { CATEGORIES, createDraft, getDraft, listDrafts, updateDraft, listWritingCategories, createWritingCategory, updateWritingCategory, deleteWritingCategory, listWritingTemplates, createWritingTemplate, updateWritingTemplate, deleteWritingTemplate, type WritingCategory, type WritingDraft, type WritingTemplate } from '@/api/writing'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'
import { getRoleFromToken } from '@/utils/jwt'

const templates = [
  { value: 'business-report', label: '商务报告' },
  { value: 'project-brief', label: '项目简报' },
  { value: 'meeting-notes', label: '会议纪要' },
]

const drafts = ref<WritingDraft[]>([])
const currentDraft = ref<WritingDraft | null>(null)
const selectedTemplate = ref('business-report')
const knowledgeSource = ref('knowledge_base')
const loading = ref(true)
const creating = ref(false)
const saving = ref(false)
const loadError = ref('')
const saveError = ref('')
const saveState = ref('已同步')
const managementVisible = ref(false)
const managementSaving = ref(false)
const managementError = ref('')
const managedCategories = ref<WritingCategory[]>([])
const managedTemplates = ref<WritingTemplate[]>([])
const categoryEditing = ref(false)
const templateEditing = ref(false)
const categoryForm = ref({ id: '', name: '', description: '', sort: 0 })
const templateForm = ref({ id: '', category_id: '', name: '', content: '', is_builtin: false, sort: 0 })
const currentRole = getRoleFromToken()
const isAdmin = ['super_admin', 'department_admin'].includes(currentRole)
const canManageTemplates = ['super_admin', 'department_admin', 'knowledge_editor'].includes(currentRole)

const groupedDrafts = computed(() => {
  const order = ['draft', 'review', 'completed']
  return order.map(status => ({ status, items: drafts.value.filter(draft => normalizedStatus(draft.status) === status) })).filter(group => group.items.length)
})
const templateLabel = computed(() => templates.find(template => template.value === selectedTemplate.value)?.label || '商务报告')

function normalizedStatus(status: string) {
  if (['review', 'pending', 'pending_review'].includes(status)) return 'review'
  if (['completed', 'published', 'done'].includes(status)) return 'completed'
  return 'draft'
}

function statusLabel(status: string) {
  const normalized = normalizedStatus(status)
  if (normalized === 'review') return '待审核'
  if (normalized === 'completed') return '已完成'
  return '草稿'
}

function categoryLabel(category: string) {
  return CATEGORIES.find(item => item.value === category)?.label || '文档'
}

function formatDate(value: string) {
  return new Date(value).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

function errorMessage(error: unknown, fallback: string) {
  if (typeof error !== 'object' || error === null) return fallback
  const requestError = error as { message?: string; response?: { data?: { error?: string; message?: string } } }
  return requestError.response?.data?.error || requestError.response?.data?.message || requestError.message || fallback
}

async function loadDrafts() {
  loading.value = true
  loadError.value = ''
  try {
    const response = await listDrafts()
    drafts.value = response.data.drafts || []
    if (!currentDraft.value && drafts.value[0]) await selectDraft(drafts.value[0].id)
  } catch (error: unknown) {
    loadError.value = errorMessage(error, '草稿列表加载失败，请稍后重试。')
  } finally {
    loading.value = false
  }
}

async function selectDraft(id: string) {
  loadError.value = ''
  try {
    const response = await getDraft(id)
    currentDraft.value = response.data.draft
    currentDraft.value.status = normalizedStatus(currentDraft.value.status)
    knowledgeSource.value = currentDraft.value.source_type === 'knowledge_plus_web' ? 'knowledge_plus_web' : 'knowledge_base'
    saveState.value = '已同步'
  } catch (error: unknown) {
    loadError.value = errorMessage(error, '草稿读取失败，请重试。')
  }
}

async function createNewDraft() {
  if (creating.value) return
  creating.value = true
  loadError.value = ''
  try {
    const response = await createDraft({
      title: '未命名草稿',
      category: 'report',
      source_type: knowledgeSource.value,
      web_search_enabled: knowledgeSource.value === 'knowledge_plus_web',
    })
    const draft = response.data.draft
    drafts.value = [draft, ...drafts.value.filter(item => item.id !== draft.id)]
    await selectDraft(draft.id)
  } catch (error: unknown) {
    loadError.value = errorMessage(error, '新建草稿失败，请稍后重试。')
  } finally {
    creating.value = false
  }
}

async function saveDraft() {
  if (!currentDraft.value || saving.value) return
  saving.value = true
  saveError.value = ''
  try {
    await updateDraft(currentDraft.value.id, {
      title: currentDraft.value.title,
      content: currentDraft.value.content,
      status: currentDraft.value.status,
    })
    const index = drafts.value.findIndex(draft => draft.id === currentDraft.value?.id)
    if (index >= 0) drafts.value[index] = { ...drafts.value[index], ...currentDraft.value, updated_at: new Date().toISOString() }
    saveState.value = '刚刚保存'
  } catch (error: unknown) {
    saveState.value = '保存失败'
    saveError.value = errorMessage(error, '草稿保存失败，请重试。')
  } finally {
    saving.value = false
  }
}

function exportCurrentDraft() {
  if (!currentDraft.value) return
  const blob = new Blob([`${currentDraft.value.title}\n\n${currentDraft.value.content || ''}`], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `${currentDraft.value.title || 'draft'}.txt`
  link.click()
  URL.revokeObjectURL(url)
}

function handleDraftKeydown(event: KeyboardEvent) {
  if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return
  const buttons = Array.from(document.querySelectorAll<HTMLButtonElement>('.sdp-writing-workspace__draft-item'))
  const currentIndex = buttons.indexOf(document.activeElement as HTMLButtonElement)
  if (currentIndex < 0 || !buttons.length) return
  event.preventDefault()
  let nextIndex = currentIndex
  if (event.key === 'Home') nextIndex = 0
  if (event.key === 'End') nextIndex = buttons.length - 1
  if (event.key === 'ArrowDown') nextIndex = (currentIndex + 1) % buttons.length
  if (event.key === 'ArrowUp') nextIndex = (currentIndex - 1 + buttons.length) % buttons.length
  buttons[nextIndex]?.focus()
}

function categoryName(id: string) { return managedCategories.value.find(item => item.id === id)?.name || '未分类' }
function canEditTemplate(template: WritingTemplate) { return canManageTemplates && (!template.is_builtin || isAdmin) }
async function loadWritingManagement() {
  managementError.value = ''
  try {
    const [categoriesResponse, templatesResponse] = await Promise.all([listWritingCategories(), listWritingTemplates()])
    managedCategories.value = categoriesResponse.data.categories || []
    managedTemplates.value = templatesResponse.data.templates || []
  } catch (error: unknown) { managementError.value = errorMessage(error, '模板与类别加载失败。') }
}
function editCategory(category?: WritingCategory) {
  categoryForm.value = category ? { id: category.id, name: category.name, description: category.description, sort: category.sort } : { id: '', name: '', description: '', sort: managedCategories.value.length * 10 }
  categoryEditing.value = true
  templateEditing.value = false
}
async function saveCategory() {
  managementSaving.value = true
  try {
    const payload = { name: categoryForm.value.name.trim(), description: categoryForm.value.description.trim(), sort: categoryForm.value.sort }
    if (categoryForm.value.id) await updateWritingCategory(categoryForm.value.id, payload); else await createWritingCategory(payload)
    categoryEditing.value = false
    await loadWritingManagement()
  } catch (error: unknown) { managementError.value = errorMessage(error, '写作类别保存失败。') } finally { managementSaving.value = false }
}
async function removeCategory(category: WritingCategory) {
  if (!window.confirm(`确认删除类别“${category.name}”？`)) return
  try { await deleteWritingCategory(category.id); await loadWritingManagement() } catch (error: unknown) { managementError.value = errorMessage(error, '写作类别删除失败。') }
}
function editTemplate(template?: WritingTemplate) {
  templateForm.value = template ? { id: template.id, category_id: template.category_id, name: template.name, content: template.content, is_builtin: template.is_builtin, sort: template.sort } : { id: '', category_id: managedCategories.value[0]?.id || '', name: '', content: '', is_builtin: false, sort: managedTemplates.value.length * 10 }
  templateEditing.value = true
  categoryEditing.value = false
}
async function saveTemplate() {
  managementSaving.value = true
  try {
    const payload = { category_id: templateForm.value.category_id, name: templateForm.value.name.trim(), content: templateForm.value.content.trim(), is_builtin: templateForm.value.is_builtin, sort: templateForm.value.sort }
    if (templateForm.value.id) await updateWritingTemplate(templateForm.value.id, payload); else await createWritingTemplate(payload)
    templateEditing.value = false
    await loadWritingManagement()
  } catch (error: unknown) { managementError.value = errorMessage(error, '写作模板保存失败。') } finally { managementSaving.value = false }
}
async function removeTemplate(template: WritingTemplate) {
  if (!window.confirm(`确认删除模板“${template.name}”？`)) return
  try { await deleteWritingTemplate(template.id); await loadWritingManagement() } catch (error: unknown) { managementError.value = errorMessage(error, '写作模板删除失败。') }
}

onMounted(() => { loadDrafts(); loadWritingManagement() })
</script>

<style scoped>
.sdp-writing-workspace { min-height: 100dvh; display: grid; grid-template-columns: var(--draft-pane-width) minmax(var(--space-0), 1fr); color: var(--ink-900); background: var(--ink-100); font-family: var(--font-body); }
.sdp-writing-workspace__drafts { height: 100dvh; display: flex; flex-direction: column; gap: var(--space-5); padding: var(--space-6) var(--space-4); overflow: hidden; border-right: 1px solid var(--ink-200); background: var(--ink-50); }
.sdp-writing-workspace__drafts header, .sdp-writing-workspace__drafts header button, .sdp-writing-workspace__draft-item, .sdp-writing-workspace__group-heading, .sdp-writing-workspace__topbar, .sdp-writing-workspace__tools, .sdp-writing-workspace__tools label, .sdp-writing-workspace__tools button, .sdp-writing-workspace__metadata, .sdp-writing-workspace__ai-bar { display: flex; align-items: center; }
.sdp-writing-workspace__drafts header, .sdp-writing-workspace__group-heading, .sdp-writing-workspace__topbar { justify-content: space-between; gap: var(--space-4); }
.sdp-writing-workspace__drafts header p, .sdp-writing-workspace__topbar p, .sdp-writing-workspace__priority > div p { color: var(--brand-700); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); letter-spacing: .08em; text-transform: uppercase; }
.sdp-writing-workspace__drafts h1, .sdp-writing-workspace__topbar h2, .sdp-writing-workspace__priority h3 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-weight: var(--font-weight-bold); line-height: var(--leading-tight); }
.sdp-writing-workspace__drafts h1 { font-size: var(--text-xl); }
.sdp-writing-workspace__topbar h2 { font-size: var(--text-2xl); }
.sdp-writing-workspace__drafts header button { min-height: var(--space-10); gap: var(--space-2); padding: var(--space-2) var(--space-3); border: 1px solid var(--brand-600); border-radius: var(--radius-sm); color: var(--ink-950); background: var(--brand-500); font: var(--font-weight-semibold) var(--text-sm)/var(--leading-tight) var(--font-body); cursor: pointer; }
.sdp-writing-workspace__drafts header svg, .sdp-writing-workspace__draft-icon svg, .sdp-writing-workspace__tools button svg, .sdp-writing-workspace__priority li svg, .sdp-writing-workspace__ai-bar svg, .sdp-writing-workspace__welcome svg { width: var(--space-5); height: var(--space-5); fill: none; stroke: currentColor; stroke-linecap: round; stroke-linejoin: round; stroke-width: 1.8; }
.sdp-writing-workspace__draft-scroll { min-height: var(--space-0); flex: 1; overflow-y: auto; }
.sdp-writing-workspace__draft-scroll section + section { margin-top: var(--space-6); }
.sdp-writing-workspace__group-heading { padding-inline: var(--space-2); }
.sdp-writing-workspace__group-heading h2, .sdp-writing-workspace__group-heading span { color: var(--ink-500); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-writing-workspace__group-heading span { min-width: var(--space-5); padding-inline: var(--space-1); border-radius: var(--radius-pill); color: var(--ink-700); background: var(--ink-200); text-align: center; }
.sdp-writing-workspace__draft-scroll section > div:last-child { display: grid; gap: var(--space-1); margin-top: var(--space-2); }
.sdp-writing-workspace__draft-item { width: 100%; min-width: var(--space-0); gap: var(--space-3); padding: var(--space-3); border: 1px solid transparent; border-radius: var(--radius-sm); color: var(--ink-800); background: transparent; text-align: left; cursor: pointer; }
.sdp-writing-workspace__draft-item:hover { background: var(--ink-100); }
.sdp-writing-workspace__draft-item--active { border-color: var(--brand-200); color: var(--brand-900); background: var(--brand-50); }
.sdp-writing-workspace__draft-icon { width: var(--space-8); height: var(--space-8); flex: 0 0 var(--space-8); display: grid; place-items: center; border-radius: var(--radius-sm); color: var(--brand-800); background: var(--brand-100); }
.sdp-writing-workspace__draft-icon svg { width: var(--space-4); height: var(--space-4); }
.sdp-writing-workspace__draft-copy { min-width: var(--space-0); flex: 1; display: flex; flex-direction: column; }
.sdp-writing-workspace__draft-copy strong, .sdp-writing-workspace__draft-copy small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sdp-writing-workspace__draft-copy strong { font-size: var(--text-sm); font-weight: var(--font-weight-semibold); }
.sdp-writing-workspace__draft-copy small { margin-top: var(--space-1); color: var(--ink-500); font-size: var(--text-xs); }
.sdp-writing-workspace__state { padding: var(--space-1) var(--space-2); border: 1px solid var(--ink-300); border-radius: var(--radius-pill); color: var(--ink-700); background: var(--ink-50); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-writing-workspace__loading, .sdp-writing-workspace__empty { padding: var(--space-6) var(--space-3); color: var(--ink-600); font-size: var(--text-sm); text-align: center; }
.sdp-writing-workspace__empty { display: flex; flex-direction: column; gap: var(--space-2); }
.sdp-writing-workspace__empty strong { color: var(--ink-800); }
.sdp-writing-workspace__error, .sdp-writing-workspace__save-error { padding: var(--space-3); border: 1px solid var(--brand-300); border-radius: var(--radius-sm); color: var(--ink-900); background: var(--brand-50); font-size: var(--text-xs); }
.sdp-writing-workspace__editor { height: 100dvh; min-width: var(--space-0); display: grid; grid-template-rows: auto minmax(var(--space-0), 1fr); overflow: hidden; }
.sdp-writing-workspace__topbar { min-height: var(--header-height); padding: var(--space-4) var(--space-8); border-bottom: 1px solid var(--ink-200); background: var(--ink-50); }
.sdp-writing-workspace__tools { flex-wrap: wrap; justify-content: flex-end; gap: var(--space-3); }
.sdp-writing-workspace__tools label { gap: var(--space-2); color: var(--ink-600); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-writing-workspace__tools select, .sdp-writing-workspace__tools button { min-height: var(--space-10); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-900); background: var(--ink-50); font: var(--text-sm) var(--font-body); }
.sdp-writing-workspace__tools select { padding: var(--space-2) var(--space-3); }
.sdp-writing-workspace__tools button { gap: var(--space-2); padding: var(--space-2) var(--space-3); font-weight: var(--font-weight-semibold); cursor: pointer; }
.sdp-writing-workspace__tools button:disabled, .sdp-writing-workspace__drafts header button:disabled { cursor: not-allowed; opacity: .6; }
.sdp-writing-workspace__welcome { margin: auto; max-width: calc(var(--space-24) * 5); padding: var(--space-8); text-align: center; }
.sdp-writing-workspace__welcome > span { width: var(--space-16); height: var(--space-16); display: grid; place-items: center; margin-inline: auto; border-radius: var(--radius-lg); color: var(--brand-800); background: var(--brand-100); }
.sdp-writing-workspace__welcome svg { width: var(--space-8); height: var(--space-8); }
.sdp-writing-workspace__welcome h3 { margin-top: var(--space-5); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-2xl); }
.sdp-writing-workspace__welcome p { margin-top: var(--space-2); color: var(--ink-600); line-height: var(--leading-relaxed); }
.sdp-writing-workspace__welcome button { margin-top: var(--space-5); padding: var(--space-3) var(--space-5); border: 1px solid var(--brand-600); border-radius: var(--radius-sm); color: var(--ink-950); background: var(--brand-500); font: var(--font-weight-semibold) var(--text-sm) var(--font-body); cursor: pointer; }
.sdp-writing-workspace__canvas { min-height: var(--space-0); overflow-y: auto; padding: var(--space-6) max(var(--space-6), calc((100% - 58rem) / 2)) var(--space-20); }
.sdp-writing-workspace__priority { display: flex; align-items: center; justify-content: space-between; gap: var(--space-6); padding: var(--space-4) var(--space-5); border: 1px solid var(--ink-200); border-radius: var(--radius-md); background: var(--ink-50); box-shadow: var(--shadow-xs); }
.sdp-writing-workspace__priority h3 { font-size: var(--text-lg); }
.sdp-writing-workspace__priority ol { display: flex; align-items: center; gap: var(--space-2); list-style: none; }
.sdp-writing-workspace__priority li:not([aria-hidden]) { display: grid; grid-template-columns: auto 1fr; column-gap: var(--space-2); }
.sdp-writing-workspace__priority li > span { grid-row: 1 / 3; align-self: center; padding: var(--space-1) var(--space-2); border-radius: var(--radius-xs); color: var(--brand-900); background: var(--brand-100); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-bold); }
.sdp-writing-workspace__priority li strong { color: var(--ink-800); font-size: var(--text-xs); }
.sdp-writing-workspace__priority li small { color: var(--ink-500); font-size: var(--text-xs); }
.sdp-writing-workspace__priority li[aria-hidden] { color: var(--ink-400); }
.sdp-writing-workspace__priority li svg { width: var(--space-4); height: var(--space-4); }
.sdp-writing-workspace__paper { min-height: calc(var(--space-24) * 6); margin-top: var(--space-5); padding: var(--space-8) var(--space-10); border: 1px solid var(--ink-200); border-radius: var(--radius-md); background: var(--ink-50); box-shadow: var(--shadow-md); }
.sdp-writing-workspace__metadata { flex-wrap: wrap; gap: var(--space-6); }
.sdp-writing-workspace__metadata label, .sdp-writing-workspace__metadata > div { display: flex; align-items: center; gap: var(--space-2); }
.sdp-writing-workspace__metadata span { color: var(--ink-500); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-writing-workspace__metadata select { min-height: var(--space-8); padding: var(--space-1) var(--space-2); border: 1px solid var(--ink-300); border-radius: var(--radius-xs); color: var(--ink-800); background: var(--ink-50); font: var(--text-xs) var(--font-body); }
.sdp-writing-workspace__metadata strong { color: var(--brand-800); font-size: var(--text-xs); }
.sdp-writing-workspace__title { display: block; margin-top: var(--space-8); }
.sdp-writing-workspace__title input { width: 100%; border: 0; outline: 0; color: var(--ink-950); background: transparent; font: var(--font-weight-bold) var(--text-3xl)/var(--leading-tight) var(--font-display); }
.sdp-writing-workspace__title input::placeholder, .sdp-writing-workspace__content textarea::placeholder { color: var(--ink-400); }
.sdp-writing-workspace__rule { height: 1px; margin-block: var(--space-5); background: var(--ink-200); }
.sdp-writing-workspace__content { display: block; min-height: calc(var(--space-24) * 4); }
.sdp-writing-workspace__content textarea { width: 100%; min-height: calc(var(--space-24) * 4); resize: vertical; border: 0; outline: 0; color: var(--ink-800); background: transparent; font: var(--text-base)/var(--leading-loose) var(--font-body); }
.sdp-writing-workspace__ai-bar { position: sticky; bottom: var(--space-4); max-width: calc(var(--space-24) * 7); gap: var(--space-3); margin: var(--space-5) auto var(--space-0); padding: var(--space-3) var(--space-4); border: 1px solid var(--brand-300); border-radius: var(--radius-pill); color: var(--ink-100); background: var(--ink-950); box-shadow: var(--shadow-lg); }
.sdp-writing-workspace__ai-bar > span { width: var(--space-10); height: var(--space-10); flex: 0 0 var(--space-10); display: grid; place-items: center; border-radius: var(--radius-pill); color: var(--ink-950); background: var(--brand-500); }
.sdp-writing-workspace__ai-bar > div { min-width: var(--space-0); flex: 1; display: flex; flex-direction: column; }
.sdp-writing-workspace__ai-bar strong { color: var(--ink-50); font-size: var(--text-sm); }
.sdp-writing-workspace__ai-bar small { color: var(--ink-300); font-size: var(--text-xs); }
.sdp-writing-workspace__ai-bar button { padding: var(--space-2) var(--space-3); border: 1px solid var(--ink-700); border-radius: var(--radius-pill); color: var(--ink-300); background: var(--ink-900); font: var(--text-xs) var(--font-body); }
.sdp-writing-workspace__save-error { max-width: calc(var(--space-24) * 7); margin: var(--space-3) auto var(--space-0); }
.sdp-writing-workspace button:focus-visible, .sdp-writing-workspace select:focus-visible, .sdp-writing-workspace input:focus-visible, .sdp-writing-workspace textarea:focus-visible { outline: 2px solid var(--brand-500); outline-offset: 2px; }
.sdp-writing-workspace__management-backdrop { position: fixed; z-index: 90; inset: 0; display: flex; justify-content: flex-end; background: color-mix(in srgb, var(--ink-950) 54%, transparent); }
.sdp-writing-workspace__management { width: min(100%, 42rem); height: 100dvh; overflow-y: auto; padding: var(--space-6); color: var(--ink-900); background: var(--ink-50); box-shadow: var(--shadow-lg); }
.sdp-writing-workspace__management > header, .sdp-writing-workspace__section-title, .sdp-writing-workspace__management-list article, .sdp-writing-workspace__management-form footer { display: flex; align-items: center; justify-content: space-between; gap: var(--space-4); }
.sdp-writing-workspace__management > header { padding-bottom: var(--space-5); border-bottom: 1px solid var(--ink-200); }
.sdp-writing-workspace__management > header p, .sdp-writing-workspace__section-title span { color: var(--brand-700); font: var(--font-weight-semibold) var(--text-xs) var(--font-mono); letter-spacing: .08em; text-transform: uppercase; }
.sdp-writing-workspace__management > header button, .sdp-writing-workspace__section-title button, .sdp-writing-workspace__management-list article button, .sdp-writing-workspace__management-form button { padding: var(--space-2) var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-800); background: var(--ink-50); font: var(--font-weight-semibold) var(--text-xs) var(--font-body); cursor: pointer; }
.sdp-writing-workspace__priority-guide { margin-top: var(--space-5); padding: var(--space-5); border: 1px solid var(--brand-300); border-radius: var(--radius-md); background: var(--brand-50); }
.sdp-writing-workspace__priority-guide ol { display: grid; gap: var(--space-2); margin-top: var(--space-3); padding: 0; list-style: none; }
.sdp-writing-workspace__priority-guide li { display: flex; align-items: center; gap: var(--space-3); color: var(--ink-700); font-size: var(--text-sm); }
.sdp-writing-workspace__priority-guide li span { padding: var(--space-1) var(--space-2); border-radius: var(--radius-xs); color: var(--brand-900); background: var(--brand-100); font-family: var(--font-mono); font-weight: var(--font-weight-bold); }
.sdp-writing-workspace__management-section { margin-top: var(--space-7); }
.sdp-writing-workspace__section-title h3 { margin-top: var(--space-1); font-family: var(--font-display); font-size: var(--text-xl); }
.sdp-writing-workspace__management-list { display: grid; gap: var(--space-3); margin-top: var(--space-4); }
.sdp-writing-workspace__management-list article { align-items: flex-start; padding: var(--space-4); border: 1px solid var(--ink-200); border-radius: var(--radius-sm); background: var(--ink-100); }
.sdp-writing-workspace__management-list article > div:first-child { min-width: 0; display: grid; gap: var(--space-1); }
.sdp-writing-workspace__management-list article > div:last-child { display: flex; flex: 0 0 auto; gap: var(--space-1); }
.sdp-writing-workspace__management-list small, .sdp-writing-workspace__management-list p { color: var(--ink-500); font-size: var(--text-xs); }
.sdp-writing-workspace__management-list article p { max-height: 3.4em; overflow: hidden; line-height: 1.7; }
.sdp-writing-workspace__management-list i { padding: var(--space-1) var(--space-2); border-radius: var(--radius-pill); color: var(--brand-900); background: var(--brand-100); font-size: var(--text-xs); font-style: normal; }
.sdp-writing-workspace__management-form { display: grid; gap: var(--space-4); margin-top: var(--space-6); padding: var(--space-5); border: 1px solid var(--ink-300); border-radius: var(--radius-md); background: var(--ink-100); }
.sdp-writing-workspace__management-form label { display: grid; gap: var(--space-2); color: var(--ink-700); font-size: var(--text-sm); font-weight: var(--font-weight-semibold); }
.sdp-writing-workspace__management-form input:not([type='checkbox']), .sdp-writing-workspace__management-form select, .sdp-writing-workspace__management-form textarea { width: 100%; padding: var(--space-2) var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-900); background: var(--ink-50); font: var(--text-sm) var(--font-body); }
.sdp-writing-workspace__management-form textarea { resize: vertical; }
.sdp-writing-workspace__management-form .sdp-writing-workspace__builtin { display: flex; align-items: center; grid-template-columns: none; }
.sdp-writing-workspace__management-form footer { justify-content: flex-end; }
@media (max-width: 72rem) { .sdp-writing-workspace__topbar { align-items: flex-start; flex-direction: column; } .sdp-writing-workspace__tools { justify-content: flex-start; } .sdp-writing-workspace__priority { align-items: flex-start; flex-direction: column; } }
@media (max-width: 56rem) { .sdp-writing-workspace { min-height: calc(100dvh - var(--header-height)); grid-template-columns: 1fr; } .sdp-writing-workspace__drafts { height: auto; max-height: calc(var(--space-24) * 3); border-right: 0; border-bottom: 1px solid var(--ink-200); } .sdp-writing-workspace__editor { height: auto; min-height: calc(100dvh - var(--header-height)); overflow: visible; } .sdp-writing-workspace__canvas { padding-inline: var(--space-4); } .sdp-writing-workspace__paper { padding: var(--space-6); } .sdp-writing-workspace__priority ol { align-items: stretch; flex-direction: column; } .sdp-writing-workspace__priority li[aria-hidden] { transform: rotate(90deg); align-self: center; } }
@media (max-width: 40rem) { .sdp-writing-workspace__topbar { padding: var(--space-4); } .sdp-writing-workspace__tools, .sdp-writing-workspace__tools label, .sdp-writing-workspace__tools select, .sdp-writing-workspace__tools button { width: 100%; } .sdp-writing-workspace__tools label { align-items: stretch; flex-direction: column; } .sdp-writing-workspace__tools button { justify-content: center; } .sdp-writing-workspace__paper { padding: var(--space-5); } .sdp-writing-workspace__title input { font-size: var(--text-2xl); } .sdp-writing-workspace__ai-bar { align-items: flex-start; border-radius: var(--radius-md); } .sdp-writing-workspace__ai-bar button { display: none; } }
</style>
