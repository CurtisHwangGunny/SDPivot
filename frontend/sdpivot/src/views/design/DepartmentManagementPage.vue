<template>
  <SdpSidebarLayout mode="admin">
    <div class="sdp-dept-list">
      <div class="sdp-dept-list__shell">
        <header class="sdp-dept-list__header">
          <div>
            <p>Organization Structure</p>
            <h1>部门管理</h1>
            <span>管理部门架构、层级关系与成员归属。</span>
          </div>
          <SdpButton aria-label="创建新部门" @click="openCreateDialog">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 5v14m-7-7h14" /></svg>
            创建部门
          </SdpButton>
        </header>

        <section class="sdp-dept-list__stats" aria-label="部门统计">
          <article><span>部门总数</span><strong>{{ loading ? '—' : stats.total }}</strong><small>含子部门</small></article>
          <article><span>顶级部门</span><strong>{{ loading ? '—' : stats.topLevel }}</strong><small>根节点</small></article>
          <article><span>最大层级</span><strong>{{ loading ? '—' : stats.maxDepth }}</strong><small>层</small></article>
        </section>

        <section class="sdp-dept-list__panel" aria-labelledby="dept-table-title">
          <header class="sdp-dept-list__panel-head">
            <div><p>Department Tree</p><h2 id="dept-table-title">部门列表</h2></div>
            <SdpButton variant="secondary" size="sm" aria-label="刷新部门列表" :loading="loading" @click="loadDepartments">刷新数据</SdpButton>
          </header>

          <div v-if="loadError" class="sdp-dept-list__error" role="alert">
            <strong>部门列表加载失败</strong>
            <p>{{ loadError }}</p>
            <button type="button" aria-label="重试加载部门列表" @click="loadDepartments">重试</button>
          </div>

          <div v-else class="sdp-dept-list__tree">
            <table>
              <caption class="sr-only">部门架构列表</caption>
              <thead><tr><th scope="col">部门名称</th><th scope="col">层级</th><th scope="col">操作</th></tr></thead>
              <tbody>
                <tr v-if="loading"><td colspan="3" class="sdp-dept-list__empty">正在加载部门数据...</td></tr>
                <tr v-else-if="flatDepartments.length === 0"><td colspan="3" class="sdp-dept-list__empty">暂无部门数据，请创建第一个部门</td></tr>
                <tr v-for="dept in flatDepartments" v-else :key="dept.id">
                  <td>
                    <div class="sdp-dept-list__name" :style="{ paddingLeft: (dept._depth ?? 0) * 1.5 + 'rem' }">
                      <span v-if="(dept._depth ?? 0) > 0" class="sdp-dept-list__tree-line" aria-hidden="true">└</span>
                      <strong>{{ dept.name }}</strong>
                      <small v-if="dept.description">{{ dept.description }}</small>
                    </div>
                  </td>
                  <td><span class="sdp-dept-list__level">L{{ (dept._depth ?? 0) + 1 }}</span></td>
                  <td>
                    <div class="sdp-dept-list__actions">
                      <button type="button" :aria-label="`编辑部门 ${dept.name}`" @click="openEditDialog(dept)">编辑</button>
                      <button type="button" class="sdp-dept-list__delete" :aria-label="`删除部门 ${dept.name}`" @click="confirmDelete(dept)">删除</button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>

      <div v-if="showDialog" class="sdp-dept-list__dialog-overlay" role="dialog" aria-modal="true" :aria-label="editingDept ? '编辑部门' : '创建部门'" @click.self="closeDialog">
        <div class="sdp-dept-list__dialog">
          <header><h3>{{ editingDept ? '编辑部门' : '创建部门' }}</h3><button type="button" aria-label="关闭对话框" @click="closeDialog">x</button></header>
          <form @submit.prevent="submitForm">
            <label class="sdp-dept-list__field">
              <span>部门名称<sup>*</sup></span>
              <input v-model="formData.name" type="text" required placeholder="请输入部门名称" aria-label="部门名称">
            </label>
            <label class="sdp-dept-list__field">
              <span>上级部门</span>
              <select v-model="formData.parent_id" aria-label="上级部门">
                <option value="">无（顶级部门）</option>
                <option v-for="dept in selectableParents" :key="dept.id" :value="dept.id">{{ dept.name }}</option>
              </select>
            </label>
            <label class="sdp-dept-list__field">
              <span>描述</span>
              <textarea v-model="formData.description" placeholder="可选，部门职能描述" aria-label="部门描述" rows="2"></textarea>
            </label>
            <div v-if="formError" class="sdp-dept-list__form-error" role="alert">{{ formError }}</div>
            <footer>
              <SdpButton variant="secondary" size="sm" type="button" aria-label="取消" @click="closeDialog">取消</SdpButton>
              <SdpButton size="sm" type="submit" :loading="submitting" aria-label="确认保存">{{ editingDept ? '保存' : '创建' }}</SdpButton>
            </footer>
          </form>
        </div>
      </div>
    </div>
  </SdpSidebarLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import client from '@/api/client'
import SdpSidebarLayout from '@/layouts/design/SdpSidebarLayout.vue'
import SdpButton from '@/components/design/SdpButton.vue'

interface Department {
  id: string
  name: string
  parent_id?: string
  description?: string
  children?: Department[]
  _depth?: number
}

const loading = ref(false)
const loadError = ref('')
const departments = ref<Department[]>([])
const showDialog = ref(false)
const editingDept = ref<Department | null>(null)
const submitting = ref(false)
const formError = ref('')
const formData = ref({ name: '', parent_id: '', description: '' })

const flatDepartments = computed<Department[]>(() => {
  const result: Department[] = []
  const walk = (list: Department[], depth: number) => {
    for (const dept of list) {
      result.push({ ...dept, _depth: depth })
      if (dept.children && dept.children.length) walk(dept.children, depth + 1)
    }
  }
  walk(departments.value, 0)
  return result
})

const stats = computed(() => {
  const flat = flatDepartments.value
  return {
    total: flat.length,
    topLevel: departments.value.length,
    maxDepth: flat.reduce((max, d) => Math.max(max, (d._depth ?? 0) + 1), 0),
  }
})

const selectableParents = computed(() => {
  if (!editingDept.value) return flatDepartments.value
  const exclude = new Set<string>()
  const collect = (id: string) => {
    exclude.add(id)
    flatDepartments.value.filter(d => d.parent_id === id).forEach(c => collect(c.id))
  }
  collect(editingDept.value.id)
  return flatDepartments.value.filter(d => !exclude.has(d.id))
})

async function loadDepartments() {
  loading.value = true
  loadError.value = ''
  try {
    const res = await client.get('/admin/departments')
    departments.value = res.data?.data ?? []
  } catch (e: any) {
    loadError.value = e?.response?.data?.error || e?.message || '未知错误'
    departments.value = []
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editingDept.value = null
  formData.value = { name: '', parent_id: '', description: '' }
  formError.value = ''
  showDialog.value = true
}

function openEditDialog(dept: Department) {
  editingDept.value = dept
  formData.value = { name: dept.name, parent_id: dept.parent_id ?? '', description: dept.description ?? '' }
  formError.value = ''
  showDialog.value = true
}

function closeDialog() {
  showDialog.value = false
  editingDept.value = null
  formError.value = ''
}

async function submitForm() {
  if (!formData.value.name.trim()) {
    formError.value = '部门名称不能为空'
    return
  }
  submitting.value = true
  formError.value = ''
  try {
    const payload = { name: formData.value.name.trim(), parent_id: formData.value.parent_id || undefined, description: formData.value.description.trim() || undefined }
    if (editingDept.value) {
      await client.put(`/admin/departments/${editingDept.value.id}`, payload)
    } else {
      await client.post('/admin/departments', payload)
    }
    closeDialog()
    await loadDepartments()
  } catch (e: any) {
    formError.value = e?.response?.data?.error || e?.message || '操作失败'
  } finally {
    submitting.value = false
  }
}

async function confirmDelete(dept: Department) {
  if (!confirm(`确定要删除部门「${dept.name}」吗？此操作不可撤销。`)) return
  try {
    await client.delete(`/admin/departments/${dept.id}`)
    await loadDepartments()
  } catch (e: any) {
    alert(e?.response?.data?.error || e?.message || '删除失败')
  }
}

onMounted(loadDepartments)
</script>

<style scoped>
.sdp-dept-list { min-height: 100vh; }
.sdp-dept-list__shell { max-width: 1100px; margin: 0 auto; padding: 2.5rem 1.5rem; }
.sdp-dept-list__header { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; margin-bottom: 2rem; flex-wrap: wrap; }
.sdp-dept-list__header p { font-size: 0.75rem; letter-spacing: 0.08em; text-transform: uppercase; color: var(--brand-700); margin: 0 0 0.25rem; font-weight: 600; }
.sdp-dept-list__header h1 { font-size: 1.75rem; font-weight: 700; color: var(--ink-900); margin: 0 0 0.375rem; }
.sdp-dept-list__header span { font-size: 0.875rem; color: var(--ink-500); }
.sdp-dept-list__stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 1rem; margin-bottom: 2rem; }
.sdp-dept-list__stats article { background: var(--ink-50); border: 1px solid var(--ink-100); border-radius: 0.75rem; padding: 1rem 1.25rem; }
.sdp-dept-list__stats span { display: block; font-size: 0.75rem; color: var(--ink-500); margin-bottom: 0.25rem; }
.sdp-dept-list__stats strong { font-size: 1.5rem; font-weight: 700; color: var(--ink-900); }
.sdp-dept-list__stats small { display: block; font-size: 0.6875rem; color: var(--ink-400); margin-top: 0.125rem; }
.sdp-dept-list__panel { background: var(--ink-0); border: 1px solid var(--ink-100); border-radius: 1rem; overflow: hidden; }
.sdp-dept-list__panel-head { display: flex; align-items: center; justify-content: space-between; padding: 1.25rem 1.5rem; border-bottom: 1px solid var(--ink-100); }
.sdp-dept-list__panel-head p { font-size: 0.6875rem; letter-spacing: 0.08em; text-transform: uppercase; color: var(--brand-700); margin: 0 0 0.125rem; font-weight: 600; }
.sdp-dept-list__panel-head h2 { font-size: 1.125rem; font-weight: 600; color: var(--ink-900); margin: 0; }
.sdp-dept-list__error { padding: 1.5rem; text-align: center; }
.sdp-dept-list__error strong { display: block; color: var(--danger-600, #dc2626); margin-bottom: 0.5rem; }
.sdp-dept-list__error p { color: var(--ink-500); font-size: 0.875rem; margin: 0 0 0.75rem; }
.sdp-dept-list__error button { font-size: 0.8125rem; color: var(--brand-700); background: none; border: none; cursor: pointer; text-decoration: underline; }
.sdp-dept-list__tree { overflow-x: auto; }
.sdp-dept-list__tree table { width: 100%; border-collapse: collapse; }
.sdp-dept-list__tree th { text-align: left; padding: 0.75rem 1rem; font-size: 0.75rem; font-weight: 600; color: var(--ink-500); border-bottom: 1px solid var(--ink-100); white-space: nowrap; }
.sdp-dept-list__tree td { padding: 0.875rem 1rem; border-bottom: 1px solid var(--ink-50); vertical-align: middle; }
.sdp-dept-list__tree tr:last-child td { border-bottom: none; }
.sdp-dept-list__tree tr:hover td { background: var(--ink-25, var(--ink-50)); }
.sdp-dept-list__empty { text-align: center; color: var(--ink-400); padding: 3rem 1rem !important; }
.sdp-dept-list__name { display: flex; align-items: center; gap: 0.5rem; }
.sdp-dept-list__name strong { font-weight: 600; color: var(--ink-900); font-size: 0.9375rem; }
.sdp-dept-list__name small { font-size: 0.75rem; color: var(--ink-400); }
.sdp-dept-list__tree-line { color: var(--ink-300); font-size: 0.875rem; }
.sdp-dept-list__level { display: inline-block; font-size: 0.6875rem; font-weight: 600; padding: 0.125rem 0.5rem; border-radius: 0.375rem; background: var(--brand-50); color: var(--brand-700); }
.sdp-dept-list__actions { display: flex; gap: 0.5rem; }
.sdp-dept-list__actions button { font-size: 0.8125rem; color: var(--brand-700); background: none; border: none; cursor: pointer; padding: 0; transition: color 0.15s; }
.sdp-dept-list__actions button:hover { color: var(--brand-800); text-decoration: underline; }
.sdp-dept-list__delete { color: var(--danger-600, #dc2626) !important; }
.sdp-dept-list__delete:hover { color: var(--danger-700, #b91c1c) !important; }
.sdp-dept-list__dialog-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.4); backdrop-filter: blur(4px); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 1rem; }
.sdp-dept-list__dialog { background: var(--ink-0); border-radius: 1rem; padding: 1.5rem; width: 100%; max-width: 420px; box-shadow: 0 20px 60px rgba(0,0,0,0.15); }
.sdp-dept-list__dialog header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 1.25rem; }
.sdp-dept-list__dialog header h3 { font-size: 1.125rem; font-weight: 600; color: var(--ink-900); margin: 0; }
.sdp-dept-list__dialog header button { font-size: 1.5rem; color: var(--ink-400); background: none; border: none; cursor: pointer; line-height: 1; padding: 0; }
.sdp-dept-list__dialog header button:hover { color: var(--ink-600); }
.sdp-dept-list__field { display: block; margin-bottom: 1rem; }
.sdp-dept-list__field span { display: block; font-size: 0.8125rem; font-weight: 500; color: var(--ink-700); margin-bottom: 0.375rem; }
.sdp-dept-list__field sup { color: var(--danger-600, #dc2626); }
.sdp-dept-list__field input, .sdp-dept-list__field select, .sdp-dept-list__field textarea { width: 100%; padding: 0.5rem 0.75rem; border: 1px solid var(--ink-200); border-radius: 0.5rem; font-size: 0.875rem; color: var(--ink-900); background: var(--ink-0); transition: border-color 0.15s, box-shadow 0.15s; box-sizing: border-box; }
.sdp-dept-list__field input:focus, .sdp-dept-list__field select:focus, .sdp-dept-list__field textarea:focus { outline: none; border-color: var(--brand-500); box-shadow: 0 0 0 3px oklch(from var(--brand-500) l c h / 0.12); }
.sdp-dept-list__field textarea { resize: vertical; font-family: inherit; }
.sdp-dept-list__form-error { font-size: 0.8125rem; color: var(--danger-600, #dc2626); margin-bottom: 0.75rem; }
.sdp-dept-list__dialog footer { display: flex; justify-content: flex-end; gap: 0.5rem; }
.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }
@media (max-width: 640px) {
  .sdp-dept-list__shell { padding: 1.5rem 1rem; }
  .sdp-dept-list__header { flex-direction: column; }
  .sdp-dept-list__panel-head { flex-direction: column; align-items: flex-start; gap: 0.5rem; }
}
</style>
