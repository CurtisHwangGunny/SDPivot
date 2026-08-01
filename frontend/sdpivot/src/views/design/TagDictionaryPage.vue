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

        <section class="sdp-tag-dictionary__grid" aria-label="七维标签字典">
          <article v-for="dimension in dimensionCards" :key="dimension.code" class="sdp-tag-dictionary__card">
            <header>
              <div><span>{{ dimension.code }}</span><h2>{{ dimension.name }}</h2><p>{{ dimension.description }}</p></div>
              <button type="button" :aria-label="`为${dimension.name}添加标签`" @click="openEditor(dimension)"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg></button>
            </header>
            <div v-if="loading" class="sdp-tag-dictionary__empty">正在加载标签...</div>
            <div v-else-if="dimension.tags.length === 0" class="sdp-tag-dictionary__empty">暂无标签值</div>
            <ul v-else>
              <li v-for="tag in dimension.tags" :key="tag.id">
                <span class="sdp-tag-dictionary__tag-dot" aria-hidden="true" />
                <strong>{{ tag.name }}</strong>
                <div><button type="button" :aria-label="`编辑标签 ${tag.name}`" @click="openEditor(dimension, tag)">编辑</button><button type="button" :aria-label="`删除标签 ${tag.name}`" @click="deleteTag(tag)">删除</button></div>
              </li>
            </ul>
            <footer><span>{{ dimension.tags.length }} 个标签</span><button type="button" :aria-label="`添加${dimension.name}标签`" @click="openEditor(dimension)">添加标签</button></footer>
          </article>
        </section>
      </div>

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
import axios from 'axios'
import { computed, nextTick, onMounted, ref } from 'vue'
import { STORAGE_KEYS } from '@/utils/storage'
import { SdpButton } from '@/components/design'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'

interface TagDimension { id: string; code: string; name: string; description: string; sort_order: number }
interface TagEntry { id: string; dimension_id: string; name: string; color?: string; sort_order: number }

const systemClient = axios.create({ baseURL: '/api/v1/system', timeout: 30000 })
systemClient.interceptors.request.use(config => { const token = localStorage.getItem(STORAGE_KEYS.accessToken); if (token) config.headers.Authorization = `Bearer ${token}`; return config })

const fallbackDimensions = [
  ['doc_category', '文档类型', '按内容载体与用途分类'], ['biz_category', '业务分类', '按业务流程与领域分类'],
  ['industry', '行业', '标识内容适用的行业范围'], ['keyword', '关键词', '沉淀可检索的核心主题词'],
  ['security_level', '安全等级', '定义内容访问与传播边界'], ['timeliness', '时效性', '标记内容的有效时间特征'],
  ['dept_scope', '部门范围', '标识主要负责或适用部门'],
].map(([code, name, description], index) => ({ id: code, code, name, description, sort_order: index * 10 }))

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

const dimensionCards = computed(() => fallbackDimensions.map(fallback => {
  const dimension = dimensions.value.find(item => item.code === fallback.code) || fallback
  return { ...dimension, tags: tags.value.filter(tag => tag.dimension_id === dimension.id).sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name, 'zh-CN')) }
}))

async function loadDictionary() {
  loading.value = true
  message.value = ''
  try {
    const [dimensionResponse, dictionaryResponse] = await Promise.all([systemClient.get<TagDimension[]>('/tag-dimensions'), systemClient.get<{ tags: TagEntry[] }>('/tag-dictionary')])
    dimensions.value = dimensionResponse.data || []
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
    if (editingTag.value) await systemClient.put(`/admin/tag-dictionary/${encodeURIComponent(editingTag.value.id)}`, payload)
    else await systemClient.post('/admin/tag-dictionary', payload)
    editorOpen.value = false
    showMessage(editingTag.value ? '标签已更新。' : '标签已添加。', 'success')
    await loadDictionary()
  } catch (error: unknown) { showMessage(errorMessage(error, '标签保存失败。'), 'error') } finally { saving.value = false }
}

async function deleteTag(tag: TagEntry) {
  if (!window.confirm(`确定删除标签“${tag.name}”吗？`)) return
  try { await systemClient.delete(`/admin/tag-dictionary/${encodeURIComponent(tag.id)}`); showMessage('标签已删除。', 'success'); await loadDictionary() }
  catch (error: unknown) { showMessage(errorMessage(error, '标签删除失败。'), 'error') }
}

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
.sdp-tag-dictionary__card li button, .sdp-tag-dictionary__card footer button { border: 0; color: var(--brand-800); background: transparent; font-family: var(--font-body); font-size: var(--text-xs); font-weight: var(--font-weight-semibold); cursor: pointer; }
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
