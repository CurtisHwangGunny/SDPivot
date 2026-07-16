<template>
  <div class="admin-shell">
    <section class="admin-hero">
      <div>
        <p class="admin-eyebrow">Admin workspace</p>
        <h1>管理后台</h1>
        <p class="admin-subtitle">统一查看空间、文档与成员信息。保留现有管理员接口，优先提升信息层级、概览效率与运营观感。</p>
      </div>
      <div class="admin-hero-card">
        <span>访问提醒</span>
        <strong>需要管理员权限</strong>
        <small>数据仍来自现有管理接口，不改变权限判断逻辑</small>
      </div>
    </section>

    <t-loading v-if="loading" text="正在加载管理数据..." class="admin-loading" />

    <template v-else>
      <section class="admin-metrics">
        <article class="admin-metric accent-card">
          <span>知识空间</span>
          <strong>{{ stats.space_count || 0 }}</strong>
        </article>
        <article class="admin-metric">
          <span>文档总数</span>
          <strong>{{ stats.document_count || 0 }}</strong>
        </article>
        <article class="admin-metric">
          <span>成员总数</span>
          <strong>{{ stats.member_count || 0 }}</strong>
        </article>
      </section>

      <section class="admin-panels">
        <Suspense>
          <AdminSpacesPanel :spaces="spaces" :space-columns="spaceColumns" :format-date="formatDate" />
          <template #fallback>
            <div class="admin-panel panel-loading">
              <t-loading size="small" text="正在加载空间管理面板..." />
            </div>
          </template>
        </Suspense>

        <Suspense>
          <AdminMembersPanel :members="members" :member-columns="memberColumns" />
          <template #fallback>
            <div class="admin-panel panel-loading">
              <t-loading size="small" text="正在加载成员管理面板..." />
            </div>
          </template>
        </Suspense>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent, onMounted, ref } from 'vue'
import { getAdminStats, getAdminMembers, getAdminSpaces } from '@/api/qa'
import { MessagePlugin } from 'tdesign-vue-next'

const AdminSpacesPanel = defineAsyncComponent(() => import('@/components/admin/AdminSpacesPanel.vue'))
const AdminMembersPanel = defineAsyncComponent(() => import('@/components/admin/AdminMembersPanel.vue'))

const loading = ref(true)
const stats = ref<any>({})
const spaces = ref<any[]>([])
const members = ref<any[]>([])

const spaceColumns = [
  { colKey: 'name', title: '空间名称', ellipsis: true },
  { colKey: 'visibility', title: '可见性', width: 110 },
  { colKey: 'created_at', title: '创建时间', width: 140 },
]
const memberColumns = [
  { colKey: 'user_id', title: '用户 ID', ellipsis: true },
  { colKey: 'role', title: '角色', width: 110 },
  { colKey: 'status', title: '状态', width: 90 },
]

function formatDate(t: string) {
  return t ? new Date(t).toLocaleDateString('zh-CN') : ''
}

async function load() {
  loading.value = true
  try {
    const [statsRes, spacesRes, membersRes] = await Promise.all([
      getAdminStats().catch(() => ({ data: {} })),
      getAdminSpaces().catch(() => ({ data: { spaces: [] } })),
      getAdminMembers().catch(() => ({ data: { members: [] } })),
    ])
    stats.value = statsRes.data || {}
    spaces.value = spacesRes.data.spaces || []
    members.value = membersRes.data.members || []
  } catch {
    MessagePlugin.warning('需要管理员权限')
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.admin-shell {
  display: grid;
  gap: 24px;
}
.admin-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.8fr) minmax(280px, 0.9fr);
  gap: 20px;
}
.admin-eyebrow,
.panel-kicker {
  margin: 0 0 8px;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--brand-primary);
}
.admin-hero h1,
.panel-head h2 {
  margin: 0;
  font-size: 30px;
  line-height: 1.1;
  color: var(--text-primary);
}
.admin-subtitle {
  margin: 14px 0 0;
  max-width: 700px;
  color: var(--text-secondary);
  line-height: 1.7;
}
.admin-hero-card,
.admin-metric,
.admin-panel {
  border-radius: 24px;
  border: 1px solid var(--border-soft);
  background: var(--surface-elevated);
  box-shadow: var(--shadow-soft);
}
.admin-hero-card {
  padding: 24px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 8px;
  background: linear-gradient(135deg, rgba(0, 185, 107, 0.12), rgba(31, 41, 55, 0.04));
}
.admin-hero-card span,
.admin-metric span {
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-muted);
}
.admin-hero-card strong,
.admin-metric strong {
  font-size: 28px;
  color: var(--text-primary);
}
.admin-hero-card small {
  color: var(--text-secondary);
}
.admin-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 18px;
}
.admin-metric {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.accent-card {
  background: linear-gradient(145deg, color-mix(in srgb, var(--brand-primary) 14%, transparent), color-mix(in srgb, var(--surface-elevated) 96%, transparent));
}
.admin-panels {
  display: grid;
  grid-template-columns: 1fr;
  gap: 20px;
}
.admin-panel {
  padding: 24px;
}
.panel-loading {
  min-height: 260px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
}
.admin-loading {
  min-height: 240px;
  display: flex;
  align-items: center;
  justify-content: center;
}
@media (max-width: 1080px) {
  .admin-hero,
  .admin-metrics {
    grid-template-columns: 1fr;
  }
}
</style>
