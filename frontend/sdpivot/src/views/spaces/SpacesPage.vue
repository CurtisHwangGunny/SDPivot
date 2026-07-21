<template>
  <div class="spaces-page page-shell">
    <section class="spaces-hero page-section-card">
      <div>
        <div class="hero-badge">Knowledge Spaces</div>
        <h1 class="sdp-page-title">把企业知识整理成可持续复用的空间。</h1>
        <p class="sdp-page-subtitle">
          每个知识空间都可以承载文档、导入任务、问答上下文与写作来源，先建结构，再让 AI 真正接得住。
        </p>
      </div>
      <div class="hero-actions">
        <t-button theme="primary" size="large" @click="showCreate = true">
          <template #icon><t-icon name="add" /></template>
          新建空间
        </t-button>
        <t-button variant="outline" size="large" @click="load">
          <template #icon><t-icon name="refresh" /></template>
          刷新列表
        </t-button>
      </div>
    </section>

    <section class="spaces-summary">
      <div class="summary-card page-section-card">
        <span class="summary-label">空间总数</span>
        <strong>{{ loading ? '--' : spaces.length }}</strong>
        <p>当前租户已建立的知识空间数量</p>
      </div>
      <div class="summary-card page-section-card">
        <span class="summary-label">默认推荐</span>
        <strong>先建结构</strong>
        <p>建议先按部门、业务场景或项目维度拆分空间</p>
      </div>
      <div class="summary-card page-section-card">
        <span class="summary-label">下一步动作</span>
        <strong>导入文档</strong>
        <p>空间创建后即可继续进入文档导入与问答配置</p>
      </div>
    </section>

    <section v-if="loading" class="space-list-skeleton page-section-card">
      <div v-for="item in 3" :key="item" class="skeleton-row"></div>
    </section>

    <section v-else-if="spaces.length === 0" class="empty-card page-section-card">
      <div class="empty-icon">
        <t-icon name="folder-open" size="48px" />
      </div>
      <h2>还没有知识空间</h2>
      <p>从一个清晰的空间开始，把问答、写作和文档协作都挂到同一条知识链路上。</p>
      <t-button theme="primary" size="large" @click="showCreate = true">创建第一个知识空间</t-button>
    </section>

    <section v-else class="space-list page-section-card">
      <div
        v-for="space in spaces"
        :key="space.id"
        class="space-row"
        @click="goDetail(space.id)"
      >
        <div class="space-row-main">
          <div class="space-icon">
            <t-icon name="folder" size="20px" />
          </div>
          <div class="space-copy">
            <div class="space-title-row">
              <h3>{{ space.name }}</h3>
              <t-tag size="small" theme="success" variant="light">{{ visibilityLabel(space.visibility) }}</t-tag>
            </div>
            <p>{{ space.description || '这个知识空间还没有补充描述，可进入详情页继续完善。' }}</p>
          </div>
        </div>
        <div class="space-row-side">
          <span class="space-id">ID · {{ space.id }}</span>
          <t-button variant="text" theme="primary" @click.stop="goDetail(space.id)">
            进入
            <template #suffix-icon><t-icon name="chevron-right" /></template>
          </t-button>
        </div>
      </div>
    </section>

    <Suspense>
      <SpaceCreateDialog
        v-if="showCreate"
        v-model:visible="showCreate"
        :loading="creating"
        :form="form"
        @update:form="form = $event"
        @confirm="handleCreate"
      />
    </Suspense>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, defineAsyncComponent } from 'vue'
import { useRouter } from 'vue-router'
import { listSpaces, createSpace, type Space } from '@/api/spaces'
import { MessagePlugin } from 'tdesign-vue-next'

const SpaceCreateDialog = defineAsyncComponent(() => import('@/components/spaces/SpaceCreateDialog.vue'))

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
  } catch {
    spaces.value = []
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!form.value.name) {
    MessagePlugin.warning('请输入空间名称')
    return
  }
  creating.value = true
  try {
    await createSpace(form.value)
    MessagePlugin.success('创建成功')
    showCreate.value = false
    form.value = { name: '', description: '', visibility: 'private' }
    load()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '创建失败')
  } finally {
    creating.value = false
  }
}

function goDetail(id: string) {
  router.push(`/spaces/${id}`)
}

function visibilityLabel(value?: string) {
  if (value === 'team') return '团队可见'
  if (value === 'enterprise') return '企业可见'
  return '私密'
}

onMounted(load)
</script>

<style scoped>
.spaces-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.spaces-hero {
  padding: 24px 28px;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
}

.hero-badge {
  display: inline-flex;
  align-items: center;
  padding: 8px 14px;
  border-radius: 999px;
  background: var(--sdp-brand-soft);
  color: var(--sdp-brand-deep);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.hero-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.spaces-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.summary-card {
  padding: 20px 22px;
}

.summary-label {
  display: block;
  font-size: 12px;
  font-weight: 700;
  color: var(--sdp-text-soft);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.summary-card strong {
  display: block;
  margin-top: 16px;
  font-size: 28px;
  line-height: 1.1;
  color: var(--sdp-text);
}

.summary-card p {
  margin: 10px 0 0;
  color: var(--sdp-text-soft);
  font-size: 14px;
}

.space-list-skeleton,
.space-list {
  padding: 14px;
}

.skeleton-row {
  height: 92px;
  border-radius: 12px;
  background: linear-gradient(90deg, color-mix(in srgb, var(--sdp-border) 32%, transparent), color-mix(in srgb, var(--sdp-border-strong) 55%, transparent), color-mix(in srgb, var(--sdp-border) 32%, transparent));
  margin-bottom: 12px;
}

.empty-card {
  padding: 56px 24px;
  text-align: center;
}

.empty-icon {
  width: 84px;
  height: 84px;
  margin: 0 auto 18px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  background: var(--sdp-brand-soft);
  color: var(--sdp-brand);
}

.empty-card h2 {
  margin: 0;
  font-size: 24px;
  color: var(--sdp-text);
}

.empty-card p {
  max-width: 520px;
  margin: 12px auto 24px;
  color: var(--sdp-text-soft);
}

.space-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 18px;
  border-radius: 12px;
  transition: transform 0.24s cubic-bezier(0.16, 1, 0.3, 1), background 0.24s ease;
  cursor: pointer;
}

.space-row:hover {
  background: var(--sdp-surface-soft);
  transform: translateY(-1px);
}

.space-row + .space-row {
  margin-top: 8px;
}

.space-row-main {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  min-width: 0;
}

.space-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  background: var(--sdp-brand-soft);
  color: var(--sdp-brand-deep);
  flex-shrink: 0;
}

.space-copy {
  min-width: 0;
}

.space-title-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.space-title-row h3 {
  margin: 0;
  font-size: 18px;
  color: var(--sdp-text);
}

.space-copy p {
  margin: 10px 0 0;
  color: var(--sdp-text-soft);
  font-size: 14px;
}

.space-row-side {
  min-width: 110px;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 10px;
}

.space-id {
  color: var(--sdp-text-muted);
  font-size: 12px;
}

@media (max-width: 1024px) {
  .spaces-summary {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .spaces-hero,
  .space-row {
    flex-direction: column;
    align-items: stretch;
  }

  .space-row-side {
    align-items: flex-start;
  }
}
</style>
