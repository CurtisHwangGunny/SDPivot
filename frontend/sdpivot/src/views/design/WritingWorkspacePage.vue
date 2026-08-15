<template>
  <SdpSidebarLayout>
    <div class="sdp-writing-workspace">
      <aside class="sdp-writing-workspace__drafts" aria-labelledby="draft-list-title">
        <header><div><p>AI Writing</p><h1 id="draft-list-title">写作草稿</h1></div><button type="button" :disabled="creating" @click="createNewDraft"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg>{{ creating ? '创建中' : '新建' }}</button></header>
        <p v-if="loadError" class="sdp-writing-workspace__error" role="alert">{{ loadError }}</p>
        <div v-if="loading" class="sdp-writing-workspace__loading">正在加载草稿...</div>
        <div v-else-if="groupedDrafts.length" class="sdp-writing-workspace__draft-scroll" @keydown="handleDraftKeydown">
          <section v-for="group in groupedDrafts" :key="group.status">
            <div class="sdp-writing-workspace__group-heading"><h2>{{ statusLabel(group.status) }}</h2><span>{{ group.items.length }}</span></div>
            <div role="list">
              <button v-for="draft in group.items" :key="draft.id" class="sdp-writing-workspace__draft-item" :class="{ 'sdp-writing-workspace__draft-item--active': currentDraft?.id === draft.id }" type="button" @click="selectDraft(draft.id)">
                <span class="sdp-writing-workspace__draft-icon"><svg viewBox="0 0 24 24"><path d="m4 20 4.5-1 10-10a2.12 2.12 0 0 0-3-3l-10 10L4 20Zm10-12 3 3M4 20h16" /></svg></span>
                <span class="sdp-writing-workspace__draft-copy"><strong>{{ draft.title || '未命名草稿' }}</strong><small>{{ categoryLabel(draft.category) }} · {{ formatDate(draft.updated_at) }}</small></span>
                <span class="sdp-writing-workspace__state">{{ statusLabel(draft.status) }}</span>
              </button>
            </div>
          </section>
        </div>
        <div v-else class="sdp-writing-workspace__empty"><strong>还没有写作草稿</strong><span>创建第一篇内容，开始组织写作链路。</span></div>
      </aside>

      <main class="sdp-writing-workspace__editor" aria-labelledby="writing-editor-title">
        <header class="sdp-writing-workspace__topbar">
          <div><p>Writing Studio</p><h2 id="writing-editor-title">{{ currentDraft?.title || 'AI 写作工作台' }}</h2></div>
          <div class="sdp-writing-workspace__tools">
            <label for="writing-source"><span>知识来源</span><select id="writing-source" v-model="knowledgeSource"><option value="knowledge_base">企业知识库</option><option value="knowledge_plus_web">知识库 + 互联网</option></select></label>
            <button type="button" @click="openManagement('personal')">管理我的模板</button>
            <button v-if="isAdmin" type="button" @click="openManagement('admin')">写作模板库</button>
          </div>
        </header>

        <div v-if="!currentDraft" class="sdp-writing-workspace__welcome"><span><svg viewBox="0 0 24 24"><path d="m4 20 4.5-1 10-10a2.12 2.12 0 0 0-3-3l-10 10L4 20Zm10-12 3 3M4 20h16" /></svg></span><h3>从类型开始，四步完成一篇文稿</h3><p>先确定内容骨架，再选择格式风格，补充关键事实后生成。</p><button type="button" @click="createNewDraft">创建草稿</button></div>

        <div v-else class="sdp-writing-workspace__canvas">
          <ol class="writing-steps" aria-label="AI 写作步骤">
            <li v-for="(step, index) in writingSteps" :key="step" :class="{ active: currentStep === index + 1, done: currentStep > index + 1 }"><span>{{ index + 1 }}</span><strong>{{ step }}</strong></li>
          </ol>

          <section v-if="showGuide" class="writing-guide" aria-labelledby="writing-guide-title">
            <div><p>FIRST RUN GUIDE</p><h3 id="writing-guide-title">如何开始写作</h3><span>先选文档类型和模板，再用一行一个要点提供事实。信息越具体，结果越可用。</span></div>
            <pre>客户：华东区重点客户
目标：本季度续约率提升至 85%
数据：已触达 120 家，意向 76 家
要求：先结论后数据，列出下周行动</pre>
            <button type="button" aria-label="关闭新手指引" @click="dismissGuide">关闭</button>
          </section>

          <section class="writing-config" aria-labelledby="writing-type-title">
            <div class="writing-section-head"><div><span>STEP 01</span><h3 id="writing-type-title">选择文档类型</h3><p>类型决定生成内容的顶层骨架。</p></div></div>
            <div class="writing-type-grid">
              <button v-for="category in displayCategories" :key="category.id" type="button" :class="{ active: selectedCategoryId === category.id }" @click="selectCategory(category)"><strong>{{ category.name }}</strong><span>{{ category.description || categoryHint(category.name) }}</span></button>
            </div>
          </section>

          <section class="writing-config" aria-labelledby="writing-template-title">
            <div class="writing-section-head"><div><span>STEP 02</span><h3 id="writing-template-title">选择模板 <small>可选</small></h3><p>仅展示“{{ selectedCategory?.name || '当前类型' }}”下可用的格式与风格。</p></div><button v-if="selectedTemplateId" type="button" @click="selectedTemplateId = ''">不使用模板</button></div>
            <div v-if="filteredTemplates.length" class="writing-template-grid">
              <button v-for="template in filteredTemplates" :key="`${template.source}-${template.id}`" type="button" :class="{ active: selectedTemplateId === template.id }" @click="selectedTemplateId = template.id"><span class="template-source" :class="`source-${template.source}`">{{ sourceLabel(template.source) }}</span><strong>{{ template.name }}</strong><p>{{ template.content }}</p></button>
            </div>
            <div v-else class="writing-inline-empty">该类型暂无模板，可直接写要点生成，或前往“管理我的模板”新增。</div>
          </section>

          <section class="writing-config writing-points" aria-labelledby="writing-points-title">
            <div class="writing-section-head"><div><span>STEP 03</span><h3 id="writing-points-title">写下关键要点</h3><p>提供客户、目标、事实数据、约束与期望输出。</p></div></div>
            <label><span class="sr-only">写作要点</span><textarea v-model="writingPoints" rows="7" placeholder="例如：&#10;客户：华东区重点客户&#10;目标：输出季度复盘并提出下季度行动&#10;数据：营收同比增长 18%，新增客户 32 家" /></label>
            <div class="writing-generate-row"><span>{{ selectedTemplate ? `将采用「${selectedTemplate.name}」模板` : '将采用所选类型的系统骨架' }}</span><button class="generate-button" type="button" :disabled="generating || !writingPoints.trim()" @click="generateDraft"><svg viewBox="0 0 24 24"><path d="M12 3 9.8 8.8 4 11l5.8 2.2L12 19l2.2-5.8L20 11l-5.8-2.2L12 3Z" /></svg>{{ generating ? '正在生成...' : '生成' }}</button></div>
          </section>

          <section v-if="hasResult" class="sdp-writing-workspace__paper writing-result" aria-labelledby="writing-result-title">
            <div class="writing-result-head"><div><span>STEP 04</span><h3 id="writing-result-title">生成结果</h3></div><div><button type="button" :disabled="saving" @click="saveDraft(true)">保存到空间</button><button type="button" :disabled="exporting" @click="downloadWord">{{ exporting ? '下载中' : '下载 Word' }}</button></div></div>
            <label class="sdp-writing-workspace__title" for="draft-title"><span class="sr-only">草稿标题</span><input id="draft-title" v-model="currentDraft.title" type="text" placeholder="输入文档标题" @blur="saveDraft()"></label>
            <div class="sdp-writing-workspace__rule" />
            <label class="sdp-writing-workspace__content" for="draft-content"><span class="sr-only">生成正文</span><textarea id="draft-content" v-model="currentDraft.content" aria-label="生成结果，可继续编辑" @blur="saveDraft()" /></label>
            <footer><span role="status">{{ saving ? '正在保存...' : saveState }}</span><span v-if="generationMeta">引用知识 {{ generationMeta.knowledge }} 条<span v-if="generationMeta.web"> · 互联网 {{ generationMeta.web }} 条</span> · {{ generationMeta.model }}</span></footer>
          </section>
          <p v-if="saveError" class="sdp-writing-workspace__save-error" role="alert">{{ saveError }}</p>
        </div>
      </main>

      <div v-if="managementVisible" class="sdp-writing-workspace__management-backdrop" role="presentation" @click.self="managementVisible = false">
        <aside class="sdp-writing-workspace__management" role="dialog" aria-modal="true" aria-labelledby="writing-management-title">
          <header><div><p>Template Library</p><h2 id="writing-management-title">{{ managementMode === 'admin' ? '写作模板库' : '管理我的模板' }}</h2></div><button type="button" @click="managementVisible = false">关闭</button></header>
          <section class="sdp-writing-workspace__management-section">
            <div class="sdp-writing-workspace__section-title"><div><span>{{ managementMode }}</span><h3>{{ managementMode === 'admin' ? '管理员默认模板' : '个人模板库' }}</h3></div><button type="button" :disabled="!managedCategories.length" @click="editManagedTemplate()">新增模板</button></div>
            <div class="sdp-writing-workspace__management-list">
              <article v-for="template in managementTemplates" :key="template.id"><div><strong>{{ template.name }}</strong><small>{{ categoryName(template.category_id) }} · {{ managementMode === 'admin' ? (isDefaultTemplate(template) ? '管理员默认' : '管理员模板') : '个人模板' }}</small><p>{{ template.content }}</p></div><div><button type="button" @click="editManagedTemplate(template)">编辑</button><button type="button" @click="removeManagedTemplate(template)">删除</button></div></article>
              <p v-if="!managementTemplates.length">暂无模板，点击“新增模板”创建。</p>
            </div>
          </section>
          <p v-if="managementError" class="sdp-writing-workspace__save-error" role="alert">{{ managementError }}</p>
        </aside>
      </div>

      <div v-if="templateEditing" class="template-modal-backdrop" role="presentation" @click.self="templateEditing = false">
        <form class="template-modal" role="dialog" aria-modal="true" aria-labelledby="template-editor-title" @submit.prevent="saveManagedTemplate">
          <header><div><p>{{ managementMode === 'admin' ? 'ADMIN TEMPLATE' : 'PERSONAL TEMPLATE' }}</p><h2 id="template-editor-title">{{ templateForm.id ? '编辑模板' : '新增模板' }}</h2></div><button type="button" @click="templateEditing = false">关闭</button></header>
          <label>模板名称 <strong>*</strong><input v-model.trim="templateForm.name" required maxlength="100" placeholder="例如：季度经营复盘"></label>
          <label>文档类型 <strong>*</strong><select v-model="templateForm.category_id" required><option value="" disabled>请选择类型</option><option v-for="category in managedCategories" :key="category.id" :value="category.id">{{ category.name }}</option></select></label>
          <label>模板内容 <strong>*</strong><textarea v-model.trim="templateForm.content" required rows="9" placeholder="描述章节结构、语气、格式和必须包含的信息"></textarea></label>
          <label v-if="managementMode === 'admin'" class="template-checkbox"><input v-model="templateForm.is_default" type="checkbox">设为该类型的管理员默认模板</label>
          <p v-if="templateFormError" class="sdp-writing-workspace__save-error">{{ templateFormError }}</p>
          <footer><button type="button" @click="templateEditing = false">取消</button><button class="primary" type="submit" :disabled="managementSaving">{{ managementSaving ? '保存中...' : '保存模板' }}</button></footer>
        </form>
      </div>
    </div>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { CATEGORIES, createAdminWritingTemplate, createDraft, createMyWritingTemplate, deleteAdminWritingTemplate, deleteMyWritingTemplate, exportDraft, generateContent, getDraft, getMyWritingTemplates, listAdminWritingTemplates, listDrafts, listWritingCategories, listWritingTemplates, updateAdminWritingTemplate, updateDraft, updateMyWritingTemplate, type PersonalWritingTemplate, type ResolvedWritingTemplate, type WritingCategory, type WritingDraft, type WritingTemplate, type WritingTemplateSource } from '@/api/writing'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'
import { getRoleFromToken } from '@/utils/jwt'

type DisplayCategory = WritingCategory & { code: string }
type DisplayTemplate = { id: string; category_id: string; name: string; content: string; source: WritingTemplateSource }
type ManagedTemplate = PersonalWritingTemplate | WritingTemplate

const GUIDE_KEY = 'sdpivot.writing.guide.dismissed'
const drafts = ref<WritingDraft[]>([])
const currentDraft = ref<WritingDraft | null>(null)
const managedCategories = ref<WritingCategory[]>([])
const baseTemplates = ref<WritingTemplate[]>([])
const resolvedTemplates = ref<ResolvedWritingTemplate[]>([])
const personalTemplates = ref<PersonalWritingTemplate[]>([])
const adminTemplates = ref<WritingTemplate[]>([])
const selectedCategoryId = ref('')
const selectedTemplateId = ref('')
const writingPoints = ref('')
const knowledgeSource = ref<'knowledge_base' | 'knowledge_plus_web'>('knowledge_base')
const generationMeta = ref<{ knowledge: number; web: number; model: string } | null>(null)
const generated = ref(false)
const loading = ref(true)
const creating = ref(false)
const saving = ref(false)
const generating = ref(false)
const exporting = ref(false)
const loadError = ref('')
const saveError = ref('')
const saveState = ref('已同步')
const showGuide = ref(localStorage.getItem(GUIDE_KEY) !== '1')
const managementVisible = ref(false)
const managementMode = ref<'personal' | 'admin'>('personal')
const managementSaving = ref(false)
const managementError = ref('')
const templateEditing = ref(false)
const templateFormError = ref('')
const templateForm = ref({ id: '', category_id: '', name: '', content: '', sort: 0, is_default: false })
const isAdmin = ['super_admin', 'department_admin'].includes(getRoleFromToken())
const writingSteps = ['选类型', '选模板', '写要点', '生成']

const displayCategories = computed<DisplayCategory[]>(() => {
  if (managedCategories.value.length) return managedCategories.value.map((item, index) => ({ ...item, code: categoryCode(item, index) }))
  return CATEGORIES.map((item, index) => ({ id: item.value, name: item.label, description: categoryHint(item.label), sort: index * 10, created_at: '', updated_at: '', code: item.value }))
})
const selectedCategory = computed(() => displayCategories.value.find(item => item.id === selectedCategoryId.value))
const allTemplates = computed<DisplayTemplate[]>(() => {
  const values: DisplayTemplate[] = [
    ...baseTemplates.value.map(item => ({ id: item.id, category_id: item.category_id, name: item.name, content: item.content, source: item.is_builtin ? 'builtin' as const : 'system' as const })),
    ...resolvedTemplates.value.map(item => ({ ...item })),
    ...personalTemplates.value.map(item => ({ id: item.id, category_id: item.category_id, name: item.name, content: item.content, source: 'personal' as const })),
  ]
  const unique = new Map<string, DisplayTemplate>()
  values.forEach(item => unique.set(item.id, item))
  return [...unique.values()]
})
const filteredTemplates = computed(() => allTemplates.value.filter(item => item.category_id === selectedCategoryId.value))
const selectedTemplate = computed(() => filteredTemplates.value.find(item => item.id === selectedTemplateId.value))
const currentStep = computed(() => generated.value || Boolean(currentDraft.value?.content) ? 4 : writingPoints.value.trim() ? 3 : selectedTemplateId.value ? 2 : 1)
const hasResult = computed(() => generated.value || Boolean(currentDraft.value?.content))
const managementTemplates = computed<ManagedTemplate[]>(() => managementMode.value === 'admin' ? adminTemplates.value : personalTemplates.value)
const groupedDrafts = computed(() => ['draft', 'review', 'completed'].map(status => ({ status, items: drafts.value.filter(draft => normalizedStatus(draft.status) === status) })).filter(group => group.items.length))

watch(selectedCategoryId, () => {
  if (!filteredTemplates.value.some(item => item.id === selectedTemplateId.value)) selectedTemplateId.value = filteredTemplates.value[0]?.id || ''
  if (currentDraft.value && selectedCategory.value) currentDraft.value.category = selectedCategory.value.code
})

function normalizedStatus(status: string) { return ['review', 'pending', 'pending_review'].includes(status) ? 'review' : ['completed', 'published', 'done'].includes(status) ? 'completed' : 'draft' }
function statusLabel(status: string) { return normalizedStatus(status) === 'review' ? '待审核' : normalizedStatus(status) === 'completed' ? '已完成' : '草稿' }
function categoryLabel(category: string) { return CATEGORIES.find(item => item.value === category)?.label || displayCategories.value.find(item => item.code === category)?.name || '文档' }
function formatDate(value: string) { return new Date(value).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' }) }
function categoryName(id: string) { return managedCategories.value.find(item => item.id === id)?.name || '未分类' }
function isDefaultTemplate(template: ManagedTemplate) { return 'is_default' in template && Boolean(template.is_default) }
function categoryCode(category: WritingCategory, index: number) {
  const normalized = category.name.replace(/\s+/g, '')
  const byLabel = CATEGORIES.find(item => item.label === normalized)
  return byLabel?.value || CATEGORIES[index]?.value || 'report'
}
function categoryHint(name: string) { return ({ 通知: '事项、对象、时间与执行要求', 公告: '公开发布、范围与正式说明', 技术文档: '背景、方案、接口与验证', 会议纪要: '议题、决议、负责人和期限', 制度解读: '制度背景、条款与执行口径', 报告: '事实数据、分析、结论与建议', 工作总结: '成果、问题、经验和计划', 研究报告: '课题、方法、发现与洞察' } as Record<string, string>)[name] || '确定文章结构与核心表达目标' }
function sourceLabel(source: WritingTemplateSource) { return ({ personal: '个人', admin: '管理员默认', builtin: '系统内置', system: '系统模板' } as Record<WritingTemplateSource, string>)[source] }
function errorMessage(error: unknown, fallback: string) { if (typeof error !== 'object' || !error) return fallback; const value = error as { message?: string; response?: { data?: { error?: string; message?: string } } }; return value.response?.data?.error || value.response?.data?.message || value.message || fallback }

async function loadDrafts() {
  loading.value = true
  try { const response = await listDrafts(); drafts.value = response.data.drafts || []; if (!currentDraft.value && drafts.value[0]) await selectDraft(drafts.value[0].id) }
  catch (error: unknown) { loadError.value = errorMessage(error, '草稿列表加载失败，请稍后重试。') }
  finally { loading.value = false }
}
async function loadWritingData() {
  managementError.value = ''
  try {
    const [categoriesResponse, templatesResponse, myResponse] = await Promise.all([listWritingCategories(), listWritingTemplates(), getMyWritingTemplates(true)])
    managedCategories.value = categoriesResponse.data.categories || []
    baseTemplates.value = templatesResponse.data.templates || []
    personalTemplates.value = myResponse.data.preferences?.templates || []
    resolvedTemplates.value = myResponse.data.resolved_templates || []
    if (!selectedCategoryId.value) selectedCategoryId.value = displayCategories.value[0]?.id || ''
  } catch (error: unknown) { managementError.value = errorMessage(error, '写作类型与模板加载失败。') }
}
async function selectDraft(id: string) {
  try {
    const response = await getDraft(id)
    currentDraft.value = response.data.draft
    currentDraft.value.status = normalizedStatus(currentDraft.value.status)
    knowledgeSource.value = currentDraft.value.source_type === 'knowledge_plus_web' ? 'knowledge_plus_web' : 'knowledge_base'
    selectedCategoryId.value = displayCategories.value.find(item => item.code === currentDraft.value?.category)?.id || displayCategories.value[0]?.id || ''
    writingPoints.value = ''
    generated.value = Boolean(currentDraft.value.content)
    generationMeta.value = null
    saveState.value = '已同步'
  } catch (error: unknown) { loadError.value = errorMessage(error, '草稿读取失败，请重试。') }
}
async function createNewDraft() {
  if (creating.value) return
  creating.value = true
  try {
    const category = selectedCategory.value?.code || 'report'
    const response = await createDraft({ title: '未命名草稿', category, source_type: knowledgeSource.value, web_search_enabled: knowledgeSource.value === 'knowledge_plus_web' })
    drafts.value = [response.data.draft, ...drafts.value.filter(item => item.id !== response.data.draft.id)]
    await selectDraft(response.data.draft.id)
  } catch (error: unknown) { loadError.value = errorMessage(error, '新建草稿失败，请稍后重试。') }
  finally { creating.value = false }
}
function selectCategory(category: DisplayCategory) { selectedCategoryId.value = category.id; generated.value = false }
function dismissGuide() { showGuide.value = false; localStorage.setItem(GUIDE_KEY, '1') }

async function generateDraft() {
  if (!currentDraft.value || !writingPoints.value.trim() || generating.value) return
  generating.value = true
  saveError.value = ''
  try {
    const response = await generateContent({ category: selectedCategory.value?.code || currentDraft.value.category || 'report', prompt: writingPoints.value.trim(), space_id: currentDraft.value.space_id || undefined, source_type: knowledgeSource.value, web_search_enabled: knowledgeSource.value === 'knowledge_plus_web', custom_template: selectedTemplate.value?.content })
    currentDraft.value.category = response.data.category
    currentDraft.value.content = response.data.content
    currentDraft.value.source_type = response.data.source_type
    currentDraft.value.web_search_enabled = response.data.web_search_enabled
    generationMeta.value = { knowledge: response.data.knowledge_sources_count, web: response.data.web_sources_count, model: response.data.model }
    generated.value = true
    await saveDraft()
    MessagePlugin.success('内容已生成，可继续编辑')
  } catch (error: unknown) { saveError.value = errorMessage(error, '生成失败，请检查模型配置后重试。'); MessagePlugin.error(saveError.value) }
  finally { generating.value = false }
}
async function saveDraft(notify = false) {
  if (!currentDraft.value || saving.value) return
  saving.value = true
  try {
    await updateDraft(currentDraft.value.id, { title: currentDraft.value.title, content: currentDraft.value.content, status: currentDraft.value.status })
    const index = drafts.value.findIndex(item => item.id === currentDraft.value?.id)
    if (index >= 0) drafts.value[index] = { ...drafts.value[index], ...currentDraft.value, updated_at: new Date().toISOString() }
    saveState.value = '刚刚保存'
    if (notify) MessagePlugin.success('草稿已保存到空间')
  } catch (error: unknown) { saveState.value = '保存失败'; saveError.value = errorMessage(error, '草稿保存失败，请重试。') }
  finally { saving.value = false }
}
async function downloadWord() {
  if (!currentDraft.value || exporting.value) return
  exporting.value = true
  try {
    await saveDraft()
    const response = await exportDraft(currentDraft.value.id, 'docx')
    const url = URL.createObjectURL(new Blob([response.data as BlobPart]))
    const link = document.createElement('a'); link.href = url; link.download = `${currentDraft.value.title || 'draft'}.docx`; link.click(); URL.revokeObjectURL(url)
  } catch (error: unknown) { MessagePlugin.error(errorMessage(error, 'Word 下载失败')) }
  finally { exporting.value = false }
}

async function openManagement(mode: 'personal' | 'admin') {
  managementMode.value = mode
  managementVisible.value = true
  managementError.value = ''
  if (mode === 'admin') {
    try { const response = await listAdminWritingTemplates(); adminTemplates.value = response.data.templates || [] }
    catch (error: unknown) { managementError.value = errorMessage(error, '管理员模板加载失败。') }
  }
}
function editManagedTemplate(template?: ManagedTemplate) {
  templateForm.value = template ? { id: template.id, category_id: template.category_id, name: template.name, content: template.content, sort: template.sort, is_default: 'is_default' in template ? Boolean(template.is_default) : false } : { id: '', category_id: selectedCategoryId.value || managedCategories.value[0]?.id || '', name: '', content: '', sort: managementTemplates.value.length * 10, is_default: false }
  templateFormError.value = ''
  templateEditing.value = true
}
async function saveManagedTemplate() {
  if (!templateForm.value.name.trim() || !templateForm.value.category_id || !templateForm.value.content.trim()) { templateFormError.value = '模板名称、文档类型和模板内容均为必填项。'; return }
  managementSaving.value = true
  templateFormError.value = ''
  try {
    const payload = { category_id: templateForm.value.category_id, name: templateForm.value.name.trim(), content: templateForm.value.content.trim(), sort: templateForm.value.sort }
    if (managementMode.value === 'admin') {
      const adminPayload = { ...payload, is_default: templateForm.value.is_default }
      if (templateForm.value.id) await updateAdminWritingTemplate(templateForm.value.id, adminPayload); else await createAdminWritingTemplate(adminPayload)
      const response = await listAdminWritingTemplates(); adminTemplates.value = response.data.templates || []
    } else {
      if (templateForm.value.id) await updateMyWritingTemplate(templateForm.value.id, payload); else await createMyWritingTemplate(payload)
    }
    templateEditing.value = false
    await loadWritingData()
    MessagePlugin.success('模板已保存')
  } catch (error: unknown) { templateFormError.value = errorMessage(error, '模板保存失败。') }
  finally { managementSaving.value = false }
}
async function removeManagedTemplate(template: ManagedTemplate) {
  if (!window.confirm(`确认删除模板“${template.name}”？`)) return
  try {
    if (managementMode.value === 'admin') { await deleteAdminWritingTemplate(template.id); adminTemplates.value = adminTemplates.value.filter(item => item.id !== template.id) }
    else { await deleteMyWritingTemplate(template.id); personalTemplates.value = personalTemplates.value.filter(item => item.id !== template.id) }
    await loadWritingData()
  } catch (error: unknown) { managementError.value = errorMessage(error, '模板删除失败。') }
}
function handleDraftKeydown(event: KeyboardEvent) {
  if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return
  const buttons = Array.from(document.querySelectorAll<HTMLButtonElement>('.sdp-writing-workspace__draft-item')); const index = buttons.indexOf(document.activeElement as HTMLButtonElement)
  if (index < 0 || !buttons.length) return
  event.preventDefault(); const next = event.key === 'Home' ? 0 : event.key === 'End' ? buttons.length - 1 : event.key === 'ArrowDown' ? (index + 1) % buttons.length : (index - 1 + buttons.length) % buttons.length; buttons[next]?.focus()
}

onMounted(async () => { await loadWritingData(); await loadDrafts() })
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
.writing-steps { display: grid; grid-template-columns: repeat(4, 1fr); gap: var(--space-3); margin: 0 0 var(--space-5); padding: 0; list-style: none; }
.writing-steps li { position: relative; min-height: 58px; display: flex; align-items: center; gap: var(--space-3); padding: var(--space-3) var(--space-4); border: 1px solid var(--ink-200); border-radius: var(--radius-md); color: var(--ink-500); background: var(--ink-50); }
.writing-steps li:not(:last-child)::after { content: ''; position: absolute; z-index: 2; right: calc(var(--space-3) * -1); width: var(--space-3); height: 1px; background: var(--ink-300); }
.writing-steps li span { width: var(--space-7); height: var(--space-7); display: grid; place-items: center; flex: 0 0 var(--space-7); border-radius: 50%; color: var(--ink-600); background: var(--ink-100); font: var(--font-weight-bold) var(--text-xs) var(--font-mono); }
.writing-steps li strong { font-size: var(--text-sm); }
.writing-steps li.active { border-color: var(--brand-500); color: var(--brand-950); background: var(--brand-50); box-shadow: 0 0 0 2px var(--brand-100); }
.writing-steps li.active span, .writing-steps li.done span { color: var(--ink-950); background: var(--brand-500); }
.writing-steps li.done { color: var(--ink-800); border-color: var(--brand-200); }
.writing-guide { position: relative; display: grid; grid-template-columns: minmax(0, 1fr) minmax(16rem, .8fr); gap: var(--space-6); margin-bottom: var(--space-5); padding: var(--space-5); border: 1px solid var(--brand-300); border-radius: var(--radius-lg); background: linear-gradient(135deg, var(--brand-50), var(--ink-50) 72%); box-shadow: var(--shadow-sm); }
.writing-guide p, .writing-section-head > div > span, .writing-result-head > div > span, .template-modal header p { color: var(--brand-700); font: var(--font-weight-semibold) var(--text-xs) var(--font-mono); letter-spacing: .08em; }
.writing-guide h3, .writing-section-head h3, .writing-result-head h3 { margin-top: var(--space-1); font-family: var(--font-display); font-size: var(--text-xl); }
.writing-guide div > span, .writing-section-head p { display: block; margin-top: var(--space-2); color: var(--ink-600); font-size: var(--text-sm); line-height: var(--leading-relaxed); }
.writing-guide pre { margin: 0; padding: var(--space-4); overflow-x: auto; border: 1px solid var(--ink-200); border-radius: var(--radius-sm); color: var(--ink-700); background: var(--ink-100); font: var(--text-xs)/1.8 var(--font-mono); white-space: pre-wrap; }
.writing-guide > button { position: absolute; top: var(--space-3); right: var(--space-3); border: 0; color: var(--ink-500); background: transparent; font-size: var(--text-xs); cursor: pointer; }
.writing-config { margin-top: var(--space-5); padding: var(--space-5); border: 1px solid var(--ink-200); border-radius: var(--radius-lg); background: var(--ink-50); box-shadow: var(--shadow-xs); }
.writing-section-head, .writing-result-head, .writing-generate-row, .writing-result footer { display: flex; align-items: center; justify-content: space-between; gap: var(--space-4); }
.writing-section-head h3 small { color: var(--ink-500); font-family: var(--font-body); font-size: var(--text-xs); font-weight: var(--font-weight-normal); }
.writing-section-head > button, .writing-result-head button { padding: var(--space-2) var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-700); background: var(--ink-50); font: var(--font-weight-semibold) var(--text-xs) var(--font-body); cursor: pointer; }
.writing-type-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: var(--space-3); margin-top: var(--space-4); }
.writing-type-grid button { min-height: 96px; display: flex; flex-direction: column; gap: var(--space-2); padding: var(--space-4); border: 1px solid var(--ink-200); border-radius: var(--radius-sm); color: var(--ink-800); background: var(--ink-100); text-align: left; cursor: pointer; }
.writing-type-grid button strong { font-family: var(--font-display); font-size: var(--text-base); }
.writing-type-grid button span { color: var(--ink-500); font-size: var(--text-xs); line-height: var(--leading-relaxed); }
.writing-type-grid button.active { border-color: var(--brand-500); color: var(--brand-950); background: var(--brand-50); box-shadow: inset 3px 0 var(--brand-500); }
.writing-template-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: var(--space-3); margin-top: var(--space-4); }
.writing-template-grid > button { min-height: 138px; display: flex; flex-direction: column; align-items: flex-start; gap: var(--space-2); padding: var(--space-4); overflow: hidden; border: 1px solid var(--ink-200); border-radius: var(--radius-sm); color: var(--ink-800); background: var(--ink-100); text-align: left; cursor: pointer; }
.writing-template-grid > button.active { border-color: var(--brand-500); background: var(--brand-50); box-shadow: 0 0 0 2px var(--brand-100); }
.writing-template-grid p { display: -webkit-box; margin-top: var(--space-1); overflow: hidden; color: var(--ink-500); font-size: var(--text-xs); line-height: 1.7; -webkit-box-orient: vertical; -webkit-line-clamp: 3; }
.template-source { padding: var(--space-1) var(--space-2); border-radius: var(--radius-pill); color: var(--ink-700); background: var(--ink-200); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.template-source.source-personal { color: var(--brand-900); background: var(--brand-100); }
.template-source.source-admin { color: oklch(0.42 0.12 230); background: var(--info-50); }
.writing-inline-empty { margin-top: var(--space-4); padding: var(--space-5); border: 1px dashed var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-500); background: var(--ink-100); font-size: var(--text-sm); text-align: center; }
.writing-points textarea { width: 100%; margin-top: var(--space-4); padding: var(--space-4); resize: vertical; border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-900); background: white; font: var(--text-sm)/var(--leading-loose) var(--font-body); }
.writing-generate-row { margin-top: var(--space-4); }
.writing-generate-row > span { color: var(--ink-500); font-size: var(--text-xs); }
.generate-button { min-width: 132px; min-height: 48px; display: inline-flex; align-items: center; justify-content: center; gap: var(--space-2); border: 1px solid var(--brand-700); border-radius: var(--radius-sm); color: var(--ink-950); background: var(--brand-500); box-shadow: 0 8px 22px color-mix(in srgb, var(--brand-700) 22%, transparent); font: var(--font-weight-bold) var(--text-base) var(--font-body); cursor: pointer; }
.generate-button:hover { background: var(--brand-400); transform: translateY(-1px); }
.generate-button:disabled { cursor: not-allowed; opacity: .5; transform: none; }
.generate-button svg { width: var(--space-5); height: var(--space-5); fill: none; stroke: currentColor; stroke-width: 1.8; }
.writing-result { border-color: var(--brand-300); }
.writing-result-head > div:last-child { display: flex; gap: var(--space-2); }
.writing-result footer { margin-top: var(--space-4); padding-top: var(--space-4); border-top: 1px solid var(--ink-200); color: var(--ink-500); font-size: var(--text-xs); }
.template-modal-backdrop { position: fixed; z-index: 110; inset: 0; display: grid; place-items: center; padding: var(--space-4); background: color-mix(in srgb, var(--ink-950) 62%, transparent); }
.template-modal { width: min(100%, 38rem); max-height: calc(100dvh - var(--space-8)); display: grid; gap: var(--space-4); padding: var(--space-6); overflow-y: auto; border: 1px solid var(--ink-300); border-radius: var(--radius-lg); background: var(--ink-50); box-shadow: var(--shadow-lg); }
.template-modal header, .template-modal footer { display: flex; align-items: center; justify-content: space-between; gap: var(--space-4); }
.template-modal header { padding-bottom: var(--space-4); border-bottom: 1px solid var(--ink-200); }
.template-modal header button, .template-modal footer button { padding: var(--space-2) var(--space-4); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-800); background: var(--ink-50); font: var(--font-weight-semibold) var(--text-sm) var(--font-body); cursor: pointer; }
.template-modal label { display: grid; gap: var(--space-2); color: var(--ink-700); font-size: var(--text-sm); font-weight: var(--font-weight-semibold); }
.template-modal label strong { color: var(--danger-700, #b42318); }
.template-modal input:not([type='checkbox']), .template-modal select, .template-modal textarea { width: 100%; padding: var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-900); background: white; font: var(--text-sm) var(--font-body); }
.template-modal textarea { resize: vertical; line-height: var(--leading-relaxed); }
.template-modal .template-checkbox { display: flex; align-items: center; grid-template-columns: none; }
.template-modal footer { justify-content: flex-end; padding-top: var(--space-2); }
.template-modal footer .primary { border-color: var(--brand-700); color: var(--ink-950); background: var(--brand-500); }
@media (max-width: 72rem) { .sdp-writing-workspace__topbar { align-items: flex-start; flex-direction: column; } .sdp-writing-workspace__tools { justify-content: flex-start; } .sdp-writing-workspace__priority { align-items: flex-start; flex-direction: column; } }
@media (max-width: 72rem) { .writing-type-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } .writing-template-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 56rem) { .sdp-writing-workspace { min-height: calc(100dvh - var(--header-height)); grid-template-columns: 1fr; } .sdp-writing-workspace__drafts { height: auto; max-height: calc(var(--space-24) * 3); border-right: 0; border-bottom: 1px solid var(--ink-200); } .sdp-writing-workspace__editor { height: auto; min-height: calc(100dvh - var(--header-height)); overflow: visible; } .sdp-writing-workspace__canvas { padding-inline: var(--space-4); } .sdp-writing-workspace__paper { padding: var(--space-6); } .writing-guide { grid-template-columns: 1fr; } .writing-steps { grid-template-columns: repeat(2, 1fr); } .writing-steps li::after { display: none; } }
@media (max-width: 40rem) { .sdp-writing-workspace__topbar { padding: var(--space-4); } .sdp-writing-workspace__tools, .sdp-writing-workspace__tools label, .sdp-writing-workspace__tools select, .sdp-writing-workspace__tools button { width: 100%; } .sdp-writing-workspace__tools label { align-items: stretch; flex-direction: column; } .sdp-writing-workspace__tools button { justify-content: center; } .sdp-writing-workspace__paper { padding: var(--space-5); } .sdp-writing-workspace__title input { font-size: var(--text-2xl); } .writing-steps, .writing-type-grid, .writing-template-grid { grid-template-columns: 1fr; } .writing-section-head, .writing-generate-row, .writing-result-head, .writing-result footer { align-items: stretch; flex-direction: column; } .generate-button { width: 100%; } .writing-result-head > div:last-child { display: grid; } }
</style>
