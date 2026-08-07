<template>
  <SdpSidebarLayout mode="admin">
    <div class="sdp-tag-dictionary">
      <div class="sdp-tag-dictionary__shell">
        <header class="sdp-tag-dictionary__header">
          <div><p>Knowledge Taxonomy</p><h1>标签字典</h1><span>维护平台统一分类语言，让自动标注、检索与权限治理保持一致。</span></div>
          <SdpButton variant="secondary" aria-label="刷新标签字典" :loading="loading" @click="loadDictionary">刷新数据</SdpButton>
        </header>

        <section class="sdp-tag-dictionary__accuracy" aria-labelledby="accuracy-title">
          <div><p>Auto-tag Quality</p><h2 id="accuracy-title">自动标注准确率</h2><span>最近 30 天模型自动标注与人工确认的一致率</span></div>
          <div class="sdp-tag-dictionary__score"><strong>{{ accuracy }}%</strong><span>较上期 +2.4%</span></div>
          <div class="sdp-tag-dictionary__meter" role="progressbar" aria-label="自动标注准确率" aria-valuemin="0" aria-valuemax="100" :aria-valuenow="accuracy"><span :style="{ width: `${accuracy}%` }" /></div>
        </section>

        <p v-if="message" class="sdp-tag-dictionary__message" :class="{ 'sdp-tag-dictionary__message--error': messageType === 'error' }" role="status" aria-live="polite">{{ message }}</p>

        <nav class="sdp-tag-dictionary__tabs" aria-label="标签管理视图"><button type="button" :class="{ active: activeTab === 'dictionary' }" @click="activeTab = 'dictionary'">标签维度与字典</button><button v-if="canReview" type="button" :class="{ active: activeTab === 'feedback' }" @click="openFeedbackTab">待确认队列 <span v-if="feedbackItems.length">{{ feedbackItems.length }}</span></button></nav>

        <section v-if="activeTab === 'dictionary'" class="sdp-tag-dictionary__grid" aria-label="多维标签字典">
          <article v-for="dimension in dimensionCards" :key="dimension.code" class="sdp-tag-dictionary__card">
            <header>
              <div><span>{{ dimension.code }}</span><h2>{{ dimension.name }}</h2><p>{{ dimension.description }}</p></div>
              <div class="sdp-tag-dictionary__dimension-actions"><button v-if="canManage" type="button" :aria-label="`编辑维度 ${dimension.name}`" @click="openDimensionEditor(dimension)">维度</button><button v-if="canManage" type="button" :aria-label="`为${dimension.name}添加标签`" @click="openEditor(dimension)"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg></button></div>
            </header>
            <div v-if="loading" class="sdp-tag-dictionary__empty">正在加载标签...</div>
            <div v-else-if="dimension.tags.length === 0" class="sdp-tag-dictionary__empty">暂无标签值</div>
            <ul v-else>
              <li v-for="tag in dimension.tags" :key="tag.id">
                <span class="sdp-tag-dictionary__tag-dot" aria-hidden="true" />
                <strong>{{ tag.name }}</strong>
                <div v-if="canManage"><button type="button" :aria-label="`编辑标签 ${tag.name}`" @click="openEditor(dimension, tag)">编辑</button><button type="button" :aria-label="`删除标签 ${tag.name}`" @click="removeTag(tag)">删除</button></div>
              </li>
            </ul>
            <footer><span>{{ dimension.tags.length }} 个标签</span><button v-if="canManage" type="button" :aria-label="`添加${dimension.name}标签`" @click="openEditor(dimension)">添加标签</button></footer>
          </article>
          <button v-if="canManage" type="button" class="sdp-tag-dictionary__new-dimension" @click="openDimensionEditor()">+ 新增标签维度</button>
        </section>

        <section v-else class="sdp-tag-dictionary__feedback" aria-label="标签待确认队列"><header><div><p>Learning Feedback</p><h2>标签待确认队列</h2></div><div><SdpButton variant="secondary" size="sm" :disabled="!selectedFeedback.length" @click="reviewSelected('rejected')">批量驳回</SdpButton><SdpButton size="sm" :disabled="!selectedFeedback.length" @click="reviewSelected('reviewed')">批量确认</SdpButton></div></header><div v-if="feedbackLoading" class="sdp-tag-dictionary__empty">正在加载反馈...</div><div v-else-if="!feedbackItems.length" class="sdp-tag-dictionary__empty">暂无待确认反馈</div><ul v-else><li v-for="item in feedbackItems" :key="item.id"><label><input v-model="selectedFeedback" type="checkbox" :value="item.id"><span><strong>{{ item.document_title }}</strong><small>{{ item.tag_name || item.original_tag }} · {{ item.feedback === 'correct' ? '标记正确' : '标记错误' }}</small></span></label><time>{{ formatDate(item.created_at) }}</time></li></ul></section>
      </div>

      <div v-if="dimensionEditorOpen" class="sdp-tag-dictionary__backdrop" role="presentation" @mousedown.self="closeDimensionEditor"><section class="sdp-tag-dictionary__dialog" role="dialog" aria-modal="true" aria-labelledby="dimension-editor-title"><header><div><p>Dimension</p><h2 id="dimension-editor-title">{{ editingDimension ? '编辑标签维度' : '新增标签维度' }}</h2></div><button type="button" aria-label="关闭维度编辑" @click="closeDimensionEditor">关闭</button></header><form @submit.prevent="saveDimension"><label><span>维度名称</span><input v-model.trim="dimensionForm.name" required maxlength="128"></label><label><span>维度代码</span><input v-model.trim="dimensionForm.code" required maxlength="64" placeholder="business"></label><label><span>说明</span><input v-model.trim="dimensionForm.description" maxlength="500"></label><label><span>状态</span><select v-model="dimensionForm.enabled"><option :value="true">启用</option><option :value="false">停用</option></select></label><div class="sdp-tag-dictionary__dialog-actions"><SdpButton v-if="editingDimension" variant="secondary" @click="removeDimension">删除维度</SdpButton><SdpButton variant="secondary" @click="closeDimensionEditor">取消</SdpButton><SdpButton :loading="savingDimension" @click="saveDimension">保存维度</SdpButton></div></form></section></div>

      <div v-if="editorOpen" class="sdp-tag-dictionary__backdrop" role="presentation" @mousedown.self="closeEditor">
        <section ref="dialogRef" class="sdp-tag-dictionary__dialog" role="dialog" aria-modal="true" aria-labelledby="tag-editor-title" @keydown.esc="closeEditor" @keydown.tab="trapFocus">
          <header><div><p>{{ editingTag ? 'Edit Entry' : 'New Entry' }}</p><h2 id="tag-editor-title">{{ editingTag ? '编辑标签' : '添加标签' }}</h2></div><button type="button" aria-label="关闭标签编辑对话框" @click="closeEditor"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18" /></svg></button></header>
          <form @submit.prevent="saveTag">
            <label for="tag-dimension"><span>标签维度</span><select id="tag-dimension" v-model="form.dimensionId" aria-label="标签维度" required><option v-for="dimension in dimensions" :key="dimension.id" :value="dimension.id">{{ dimension.name }}</option></select></label>
            <label for="tag-name"><span>标签名称</span><input id="tag-name" ref="nameInputRef" v-model.trim="form.name" type="text" maxlength="128" required placeholder="输入标签名称" aria-label="标签名称"></label>
            <label for="tag-order"><span>排序值</span><input id="tag-order" v-model.number="form.sortOrder" type="number" min="0" step="1" aria-label="标签排序值"></label>
            <div class="sdp-tag-dictionary__dialog-actions"><SdpButton variant="secondary" aria-label="取消编辑标签" @click="closeEditor">取消</SdpButton><SdpButton :loading="saving" :disabled="!form.name || !form.dimensionId" aria-label="保存标签" @click="saveTag">保存标签</SdpButton></div>
          </form>
        </section>
      </div>
    </div>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { createTag, createTagDimension, deleteTag, deleteTagDimension, getTagDictionary, getTagDimensions, getTagFeedbackQueue, reviewTagFeedback, updateTag, updateTagDimension, type TagDimension, type TagEntry, type TagFeedbackItem } from '@/api/tags'
import { SdpButton } from '@/components/design'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'
import { getRoleFromToken } from '@/utils/jwt'

const fallbackDimensions = [
  ['doc_category', '文档类型', '按内容载体与用途分类'], ['biz_category', '业务分类', '按业务流程与领域分类'],
  ['industry', '行业', '标识内容适用的行业范围'], ['keyword', '关键词', '沉淀可检索的核心主题词'],
  ['security_level', '安全等级', '定义内容访问与传播边界'], ['timeliness', '时效性', '标记内容的有效时间特征'],
  ['dept_scope', '部门范围', '标识主要负责或适用部门'],
].map(([code, name, description], index) => ({ id: code, code, name, description, enabled: true, sort_order: index * 10 }))

const dimensions = ref<TagDimension[]>([])
const tags = ref<TagEntry[]>([])
const loading = ref(true)
const saving = ref(false)
const message = ref('')
const messageType = ref<'success' | 'error'>('success')
const accuracy = ref(92.6)
const editorOpen = ref(false)
const editingTag = ref<TagEntry | null>(null)
const form = ref({ dimensionId: '', name: '', sortOrder: 0 })
const dialogRef = ref<HTMLElement | null>(null)
const nameInputRef = ref<HTMLInputElement | null>(null)
const activeTab = ref<'dictionary' | 'feedback'>('dictionary')
const feedbackItems = ref<TagFeedbackItem[]>([])
const selectedFeedback = ref<string[]>([])
const feedbackLoading = ref(false)
const dimensionEditorOpen = ref(false)
const editingDimension = ref<TagDimension | null>(null)
const savingDimension = ref(false)
const dimensionForm = ref({ name: '', code: '', description: '', enabled: true, sort_order: 0 })
const role = getRoleFromToken()
const canManage = role === 'super_admin'
const canReview = ['super_admin', 'department_admin'].includes(role)

const dimensionCards = computed(() => dimensions.value.map(dimension => {
  return { ...dimension, tags: tags.value.filter(tag => tag.dimension_id === dimension.id).sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name, 'zh-CN')) }
}))

async function loadDictionary() {
  loading.value = true
  message.value = ''
  try {
    const [dictionaryResponse, dimensionResponse] = await Promise.all([getTagDictionary(), getTagDimensions()])
    dimensions.value = dimensionResponse.data.dimensions || dictionaryResponse.data.dimensions || []
    tags.value = dictionaryResponse.data.tags || []
  } catch (error: unknown) {
    dimensions.value = fallbackDimensions
    tags.value = []
    showMessage(errorMessage(error, '标签字典加载失败，请检查系统管理员权限。'), 'error')
  } finally { loading.value = false }
}

async function openEditor(dimension: TagDimension, tag?: TagEntry) {
  editingTag.value = tag || null
  const nextSortOrder = tags.value.filter(item => item.dimension_id === dimension.id).length * 10
  form.value = { dimensionId: tag?.dimension_id || dimension.id, name: tag?.name || '', sortOrder: tag?.sort_order ?? nextSortOrder }
  editorOpen.value = true
  await nextTick()
  nameInputRef.value?.focus()
}

function closeEditor() { if (!saving.value) editorOpen.value = false }
async function saveTag() {
  if (!form.value.name || !form.value.dimensionId || saving.value) return
  saving.value = true
  try {
    const payload = { dimension_id: form.value.dimensionId, name: form.value.name, color: '', sort_order: Number(form.value.sortOrder) || 0 }
    if (editingTag.value) await updateTag(editingTag.value.id, payload)
    else await createTag(payload)
    editorOpen.value = false
    showMessage(editingTag.value ? '标签已更新。' : '标签已添加。', 'success')
    await loadDictionary()
  } catch (error: unknown) { showMessage(errorMessage(error, '标签保存失败。'), 'error') } finally { saving.value = false }
}

async function removeTag(tag: TagEntry) {
  if (!window.confirm(`确定删除标签“${tag.name}”吗？`)) return
  try { await deleteTag(tag.id); showMessage('标签已删除。', 'success'); await loadDictionary() }
  catch (error: unknown) { showMessage(errorMessage(error, '标签删除失败。'), 'error') }
}

function openDimensionEditor(dimension?: TagDimension) { editingDimension.value = dimension || null; dimensionForm.value = { name: dimension?.name || '', code: dimension?.code || '', description: dimension?.description || '', enabled: dimension?.enabled ?? true, sort_order: dimension?.sort_order || 0 }; dimensionEditorOpen.value = true }
function closeDimensionEditor() { if (!savingDimension.value) dimensionEditorOpen.value = false }
async function saveDimension() { if (!dimensionForm.value.name || !dimensionForm.value.code || savingDimension.value) return; savingDimension.value = true; try { if (editingDimension.value) await updateTagDimension(editingDimension.value.id, dimensionForm.value); else await createTagDimension(dimensionForm.value); dimensionEditorOpen.value = false; showMessage('标签维度已保存。', 'success'); await loadDictionary() } catch (error: unknown) { showMessage(errorMessage(error, '标签维度保存失败。'), 'error') } finally { savingDimension.value = false } }
async function removeDimension() { if (!editingDimension.value || !window.confirm(`确定删除维度“${editingDimension.value.name}”吗？`)) return; try { await deleteTagDimension(editingDimension.value.id); dimensionEditorOpen.value = false; showMessage('标签维度已删除。', 'success'); await loadDictionary() } catch (error: unknown) { showMessage(errorMessage(error, '标签维度删除失败。'), 'error') } }
async function openFeedbackTab() { activeTab.value = 'feedback'; await loadFeedback() }
async function loadFeedback() { feedbackLoading.value = true; try { const response = await getTagFeedbackQueue(); feedbackItems.value = response.data.items || []; selectedFeedback.value = [] } catch (error: unknown) { showMessage(errorMessage(error, '反馈队列加载失败。'), 'error') } finally { feedbackLoading.value = false } }
async function reviewSelected(decision: 'reviewed' | 'rejected') { if (!selectedFeedback.value.length) return; try { await reviewTagFeedback(selectedFeedback.value, decision); showMessage(decision === 'reviewed' ? '反馈已确认。' : '反馈已驳回。', 'success'); await loadFeedback() } catch (error: unknown) { showMessage(errorMessage(error, '反馈审核失败。'), 'error') } }
function formatDate(value: string) { return value ? new Date(value).toLocaleString('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }) : '—' }

function showMessage(value: string, type: 'success' | 'error') { message.value = value; messageType.value = type }
function errorMessage(error: unknown, fallback: string) { const value = error as { message?: string; response?: { data?: { error?: string; message?: string } } }; return value?.response?.data?.error || value?.response?.data?.message || value?.message || fallback }
function trapFocus(event: KeyboardEvent) { const elements = Array.from(dialogRef.value?.querySelectorAll<HTMLElement>('button:not(:disabled), input:not(:disabled), select:not(:disabled)') || []); if (!elements.length) return; const first = elements[0]; const last = elements[elements.length - 1]; if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus() } else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus() } }

onMounted(loadDictionary)
</script>

<style scoped>
.sdp-tag-dictionary { min-height: 100dvh; color: var(--ink-900); background: var(--ink-100); font-family: var(--font-body); }
.sdp-tag-dictionary__shell { display: grid; gap: var(--space-6); width: min(100%, var(--content-max-width)); margin-inline: auto; padding: var(--space-8); }
.sdp-tag-dictionary__header, .sdp-tag-dictionary__accuracy, .sdp-tag-dictionary__card header, .sdp-tag-dictionary__card li, .sdp-tag-dictionary__card footer, .sdp-tag-dictionary__dialog header, .sdp-tag-dictionary__dialog-actions { display: flex; align-items: center; }
.sdp-tag-dictionary__header { justify-content: space-between; gap: var(--space-6); }
.sdp-tag-dictionary__header p, .sdp-tag-dictionary__accuracy p, .sdp-tag-dictionary__dialog header p { color: var(--brand-700); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); letter-spacing: .08em; text-transform: uppercase; }
.sdp-tag-dictionary__header h1 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-4xl); line-height: var(--leading-tight); }
.sdp-tag-dictionary__header > div > span { display: block; margin-top: var(--space-2); color: var(--ink-600); font-size: var(--text-base); }
.sdp-tag-dictionary__accuracy { position: relative; justify-content: space-between; gap: var(--space-8); overflow: hidden; padding: var(--space-6); border: 1px solid var(--brand-300); border-radius: var(--radius-lg); color: var(--ink-50); background: linear-gradient(135deg, var(--brand-900), var(--brand-700)); box-shadow: var(--shadow-md); }
.sdp-tag-dictionary__accuracy p { color: var(--brand-200); }
.sdp-tag-dictionary__accuracy h2 { margin-top: var(--space-1); font-family: var(--font-display); font-size: var(--text-xl); }
.sdp-tag-dictionary__accuracy > div:first-child > span { display: block; margin-top: var(--space-2); color: var(--brand-100); font-size: var(--text-sm); }
.sdp-tag-dictionary__score { flex: 0 0 auto; text-align: right; }
.sdp-tag-dictionary__score strong { display: block; font-family: var(--font-display); font-size: var(--text-4xl); line-height: var(--leading-tight); }
.sdp-tag-dictionary__score span { color: var(--brand-100); font-size: var(--text-xs); }
.sdp-tag-dictionary__meter { position: absolute; inset: auto var(--space-6) var(--space-4) var(--space-6); height: var(--space-1); overflow: hidden; border-radius: var(--radius-pill); background: var(--brand-900); }
.sdp-tag-dictionary__meter span { display: block; height: 100%; border-radius: inherit; background: var(--brand-300); }
.sdp-tag-dictionary__message { padding: var(--space-3) var(--space-4); border: 1px solid var(--brand-300); border-radius: var(--radius-sm); color: var(--brand-900); background: var(--brand-50); font-size: var(--text-sm); }
.sdp-tag-dictionary__message--error { border-color: var(--ink-400); color: var(--ink-900); background: var(--ink-200); }
.sdp-tag-dictionary__grid { display: grid; grid-template-columns: repeat(2, minmax(var(--space-0), 1fr)); gap: var(--space-4); align-items: start; }
.sdp-tag-dictionary__tabs { display: flex; gap: var(--space-2); border-bottom: 1px solid var(--ink-300); }
.sdp-tag-dictionary__tabs button { padding: var(--space-3) var(--space-4); border: 0; border-bottom: 2px solid transparent; color: var(--ink-600); background: transparent; cursor: pointer; }
.sdp-tag-dictionary__tabs button.active { border-bottom-color: var(--brand-700); color: var(--brand-900); font-weight: var(--font-weight-semibold); }
.sdp-tag-dictionary__tabs span { padding: 0 var(--space-2); border-radius: var(--radius-pill); color: var(--ink-50); background: var(--brand-700); font-size: var(--text-xs); }
.sdp-tag-dictionary__dimension-actions { display: flex; gap: var(--space-2); }
.sdp-tag-dictionary__new-dimension { min-height: calc(var(--space-24) * 2); border: 1px dashed var(--brand-500); border-radius: var(--radius-lg); color: var(--brand-800); background: var(--brand-50); font-weight: var(--font-weight-semibold); cursor: pointer; }
.sdp-tag-dictionary__feedback { overflow: hidden; border: 1px solid var(--ink-200); border-radius: var(--radius-lg); background: var(--ink-50); }
.sdp-tag-dictionary__feedback > header { display: flex; align-items: center; justify-content: space-between; gap: var(--space-4); padding: var(--space-5); border-bottom: 1px solid var(--ink-200); }
.sdp-tag-dictionary__feedback > header > div:last-child { display: flex; gap: var(--space-2); }
.sdp-tag-dictionary__feedback ul { list-style: none; }
.sdp-tag-dictionary__feedback li, .sdp-tag-dictionary__feedback label { display: flex; align-items: center; }
.sdp-tag-dictionary__feedback li { justify-content: space-between; gap: var(--space-4); padding: var(--space-4) var(--space-5); border-bottom: 1px solid var(--ink-200); }
.sdp-tag-dictionary__feedback label { gap: var(--space-3); }
.sdp-tag-dictionary__feedback strong, .sdp-tag-dictionary__feedback small { display: block; }
.sdp-tag-dictionary__feedback small, .sdp-tag-dictionary__feedback time { margin-top: var(--space-1); color: var(--ink-500); font-size: var(--text-xs); }
.sdp-tag-dictionary__card { overflow: hidden; border: 1px solid var(--ink-200); border-radius: var(--radius-lg); background: var(--ink-50); box-shadow: var(--shadow-xs); }
.sdp-tag-dictionary__card header { align-items: flex-start; justify-content: space-between; gap: var(--space-4); padding: var(--space-5); border-bottom: 1px solid var(--ink-200); background: var(--ink-100); }
.sdp-tag-dictionary__card header > div > span { color: var(--brand-700); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); }
.sdp-tag-dictionary__card h2 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-xl); }
.sdp-tag-dictionary__card header p { margin-top: var(--space-2); color: var(--ink-600); font-size: var(--text-xs); }
.sdp-tag-dictionary__card header button, .sdp-tag-dictionary__dialog header button { width: var(--space-10); height: var(--space-10); display: inline-flex; flex: 0 0 var(--space-10); align-items: center; justify-content: center; border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--brand-800); background: var(--ink-50); cursor: pointer; }
.sdp-tag-dictionary__card svg, .sdp-tag-dictionary__dialog svg { width: var(--space-5); height: var(--space-5); fill: none; stroke: currentColor; stroke-linecap: round; stroke-width: 2; }
.sdp-tag-dictionary__card ul { padding: var(--space-2) var(--space-5); list-style: none; }
.sdp-tag-dictionary__card li { gap: var(--space-3); padding: var(--space-3) var(--space-0); border-bottom: 1px solid var(--ink-200); }
.sdp-tag-dictionary__card li:last-child { border-bottom: 0; }
.sdp-tag-dictionary__tag-dot { width: var(--space-2); height: var(--space-2); flex: 0 0 var(--space-2); border-radius: var(--radius-pill); background: var(--brand-600); }
.sdp-tag-dictionary__card li strong { min-width: var(--space-0); flex: 1; overflow: hidden; color: var(--ink-900); font-size: var(--text-sm); text-overflow: ellipsis; white-space: nowrap; }
.sdp-tag-dictionary__card li div { display: flex; gap: var(--space-2); }
.sdp-tag-dictionary__card li button, .sdp-tag-dictionary__card footer button { min-width: var(--space-8); min-height: var(--space-8); padding: var(--space-2) var(--space-3); border: 0; border-radius: var(--radius-sm); color: var(--brand-800); background: transparent; font-family: var(--font-body); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); cursor: pointer; }
.sdp-tag-dictionary__card footer { justify-content: space-between; gap: var(--space-4); padding: var(--space-4) var(--space-5); border-top: 1px solid var(--ink-200); }
.sdp-tag-dictionary__card footer span, .sdp-tag-dictionary__empty { color: var(--ink-500); font-size: var(--text-xs); }
.sdp-tag-dictionary__empty { padding: var(--space-8); text-align: center; }
.sdp-tag-dictionary__backdrop { position: fixed; inset: var(--space-0); z-index: var(--z-modal); display: grid; place-items: center; padding: var(--space-6); background: color-mix(in oklch, var(--ink-950) 72%, transparent); }
.sdp-tag-dictionary__dialog { width: min(100%, calc(var(--space-24) * 5)); padding: var(--space-6); border: 1px solid var(--ink-200); border-radius: var(--radius-lg); background: var(--ink-50); box-shadow: var(--shadow-xl); }
.sdp-tag-dictionary__dialog header { align-items: flex-start; justify-content: space-between; gap: var(--space-4); }
.sdp-tag-dictionary__dialog h2 { margin-top: var(--space-1); color: var(--ink-950); font-family: var(--font-display); font-size: var(--text-2xl); }
.sdp-tag-dictionary__dialog form { display: grid; gap: var(--space-5); margin-top: var(--space-6); }
.sdp-tag-dictionary__dialog label { display: grid; gap: var(--space-2); }
.sdp-tag-dictionary__dialog label span { color: var(--ink-800); font-size: var(--text-sm); font-weight: var(--font-weight-semibold); }
.sdp-tag-dictionary__dialog input, .sdp-tag-dictionary__dialog select { width: 100%; min-height: var(--space-12); padding: var(--space-3); border: 1px solid var(--ink-300); border-radius: var(--radius-sm); color: var(--ink-900); background: var(--ink-50); font-family: var(--font-body); font-size: var(--text-sm); }
.sdp-tag-dictionary__dialog-actions { justify-content: flex-end; gap: var(--space-3); padding-top: var(--space-2); }
.sdp-tag-dictionary button:focus-visible, .sdp-tag-dictionary input:focus-visible, .sdp-tag-dictionary select:focus-visible { outline: 2px solid var(--brand-500); outline-offset: 2px; }
@media (max-width: 48rem) { .sdp-tag-dictionary__shell { padding: var(--space-6); } .sdp-tag-dictionary__grid { grid-template-columns: 1fr; } }
@media (max-width: 40rem) { .sdp-tag-dictionary__shell { padding: var(--space-4); } .sdp-tag-dictionary__header, .sdp-tag-dictionary__accuracy { align-items: flex-start; flex-direction: column; } .sdp-tag-dictionary__header h1 { font-size: var(--text-3xl); } .sdp-tag-dictionary__header :deep(.sdp-button) { width: 100%; } .sdp-tag-dictionary__score { text-align: left; } .sdp-tag-dictionary__backdrop { align-items: end; padding: var(--space-0); } .sdp-tag-dictionary__dialog { border-radius: var(--radius-lg) var(--radius-lg) var(--radius-none) var(--radius-none); } .sdp-tag-dictionary__dialog-actions { align-items: stretch; flex-direction: column-reverse; } .sdp-tag-dictionary__dialog-actions :deep(.sdp-button) { width: 100%; } }
</style>
