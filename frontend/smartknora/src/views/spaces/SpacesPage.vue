<template>
  <div class="spaces-page">
    <div class="page-header">
      <h1>知识空间</h1>
      <t-button theme="primary" @click="showCreate = true">
        <template #icon><t-icon name="add" /></template>
        新建空间
      </t-button>
    </div>

    <t-loading v-if="loading" />
    <div v-else-if="spaces.length === 0" class="empty">
      <t-icon name="folder-open" size="64px" style="color:#d0d0d0" />
      <p class="empty-title">还没有知识空间</p>
      <p class="empty-desc">创建您的第一个知识空间，开始管理企业知识</p>
      <t-button theme="primary" @click="showCreate = true">创建知识空间</t-button>
    </div>
    <div v-else class="space-grid">
      <t-card
        v-for="space in spaces"
        :key="space.id"
        class="space-card"
        :title="space.name"
        :description="space.description || '暂无描述'"
        hover-shadow
        @click="goDetail(space.id)"
      >
        <template #actions>
          <t-button variant="text" theme="primary" @click.stop="goDetail(space.id)">
            <t-icon name="chevron-right" />
          </t-button>
        </template>
      </t-card>
    </div>

    <t-dialog v-model:visible="showCreate" header="创建知识空间" @confirm="handleCreate" :confirm-btn="{ loading: creating }">
      <t-form>
        <t-form-item label="空间名称" name="name">
          <t-input v-model="form.name" placeholder="请输入空间名称" :maxlength="30" />
        </t-form-item>
        <t-form-item label="空间描述" name="description">
          <t-textarea v-model="form.description" placeholder="选填" :autosize="{ minRows: 2 }" />
        </t-form-item>
        <t-form-item label="可见性" name="visibility">
          <t-select v-model="form.visibility">
            <t-option value="private" label="私密" />
            <t-option value="team" label="团队可见" />
            <t-option value="enterprise" label="企业可见" />
          </t-select>
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listSpaces, createSpace, type Space } from '@/api/spaces'
import { MessagePlugin } from 'tdesign-vue-next'

const router = useRouter()
const spaces = ref<Space[]>([])
const loading = ref(true)
const showCreate = ref(false)
const creating = ref(false)
const form = ref({ name: '', description: '', visibility: 'private' })

async function load() {
  loading.value = true
  try {
    const res = await listSpaces()
    spaces.value = (res.data as any).spaces || []
  } catch { spaces.value = [] }
  finally { loading.value = false }
}

async function handleCreate() {
  if (!form.value.name) { MessagePlugin.warning('请输入空间名称'); return }
  creating.value = true
  try {
    await createSpace(form.value)
    MessagePlugin.success('创建成功')
    showCreate.value = false
    form.value = { name: '', description: '', visibility: 'private' }
    load()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '创建失败')
  } finally { creating.value = false }
}

function goDetail(id: string) {
  router.push(`/spaces/${id}`)
}

onMounted(load)
</script>

<style scoped>
.spaces-page { max-width: 1200px; margin: 0 auto; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px; }
.page-header h1 { font-size: 24px; font-weight: 600; }
.empty { text-align: center; padding: 80px 20px; }
.empty-title { font-size: 18px; font-weight: 500; margin: 16px 0 8px; }
.empty-desc { color: #999; margin-bottom: 24px; }
.space-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 16px; }
.space-card { cursor: pointer; }
</style>
