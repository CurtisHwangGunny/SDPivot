<template>
  <div class="org-shell">
    <section class="org-hero">
      <div>
        <p class="org-eyebrow">Organization workspace</p>
        <h1>企业管理</h1>
        <p class="org-subtitle">管理你所属的企业组织、查看认证状态，并通过创建或加入企业接入更完整的协作能力。</p>
      </div>
      <div class="org-hero-actions">
        <t-button theme="primary" size="large" @click="showCreate = true">创建企业</t-button>
        <t-button variant="outline" size="large" @click="showJoin = true">加入企业</t-button>
      </div>
    </section>

    <section class="org-summary">
      <article class="summary-card accent-card">
        <span>已关联企业</span>
        <strong>{{ orgs.length }}</strong>
        <small>当前账号可见的组织数量</small>
      </article>
      <article class="summary-card">
        <span>认证状态</span>
        <strong>{{ activeStatus }}</strong>
        <small>以当前首个企业状态作为展示基准</small>
      </article>
      <article class="summary-card">
        <span>下一步</span>
        <strong>{{ orgs.length ? '完善资料' : '创建或加入' }}</strong>
        <small>继续补齐组织信息与协作流程</small>
      </article>
    </section>

    <t-loading v-if="loading" text="正在加载企业信息..." class="org-loading" />

    <template v-else-if="orgs.length === 0">
      <section class="org-empty">
        <div class="empty-illustration">企业</div>
        <h2>尚未关联企业</h2>
        <p>创建企业或通过企业 ID 加入已有企业，后续可承接团队协作、空间共享与组织级管理能力。</p>
        <div class="empty-actions">
          <t-button theme="primary" @click="showCreate = true">立即创建企业</t-button>
          <t-button variant="outline" @click="showJoin = true">通过 ID 加入</t-button>
        </div>
      </section>
    </template>

    <section v-else class="org-list">
      <article v-for="org in orgs" :key="org.id" class="org-card">
        <div class="org-card-head">
          <div>
            <p class="org-card-kicker">Organization</p>
            <h3>{{ org.name }}</h3>
          </div>
          <t-tag theme="success" variant="light">{{ org.auth_status || 'active' }}</t-tag>
        </div>
        <p class="org-desc">{{ org.description || '当前企业还没有补充描述信息。' }}</p>
        <div class="org-meta-grid">
          <div class="meta-item">
            <span>企业 ID</span>
            <strong>{{ org.id }}</strong>
          </div>
          <div class="meta-item">
            <span>状态</span>
            <strong>{{ org.auth_status || 'active' }}</strong>
          </div>
        </div>
      </article>
    </section>

    <Suspense>
      <OrgCreateDialog
        v-if="showCreate"
        v-model:visible="showCreate"
        :loading="creating"
        :form="createForm"
        @update:form="createForm = $event"
        @confirm="handleCreate"
      />
    </Suspense>

    <Suspense>
      <OrgJoinDialog
        v-if="showJoin"
        v-model:visible="showJoin"
        :loading="joining"
        :form="joinForm"
        @update:form="joinForm = $event"
        @confirm="handleJoin"
      />
    </Suspense>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, defineAsyncComponent } from 'vue'
import { listOrganizations, createOrganization, joinOrganization, type Organization } from '@/api/org'
import { MessagePlugin } from 'tdesign-vue-next'

const OrgCreateDialog = defineAsyncComponent(() => import('@/components/org/OrgCreateDialog.vue'))
const OrgJoinDialog = defineAsyncComponent(() => import('@/components/org/OrgJoinDialog.vue'))

const orgs = ref<Organization[]>([])
const loading = ref(true)
const showCreate = ref(false)
const showJoin = ref(false)
const creating = ref(false)
const joining = ref(false)
const createForm = ref({ name: '', description: '' })
const joinForm = ref({ org_id: '' })

const activeStatus = computed(() => orgs.value[0]?.auth_status || (orgs.value.length ? 'active' : '待创建'))

async function load() {
  loading.value = true
  try {
    const res = await listOrganizations()
    orgs.value = (res.data as any).organizations || []
  } catch {
    orgs.value = []
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!createForm.value.name) {
    MessagePlugin.warning('请输入企业名称')
    return
  }
  creating.value = true
  try {
    await createOrganization(createForm.value)
    MessagePlugin.success('企业创建成功')
    showCreate.value = false
    createForm.value = { name: '', description: '' }
    load()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '创建失败')
  } finally {
    creating.value = false
  }
}

async function handleJoin() {
  if (!joinForm.value.org_id) {
    MessagePlugin.warning('请输入企业ID')
    return
  }
  joining.value = true
  try {
    await joinOrganization(joinForm.value)
    MessagePlugin.success('申请已提交')
    showJoin.value = false
    joinForm.value = { org_id: '' }
    load()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '加入失败')
  } finally {
    joining.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.org-shell {
  display: grid;
  gap: 24px;
}
.org-hero {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
  flex-wrap: wrap;
}
.org-eyebrow,
.org-card-kicker {
  margin: 0 0 8px;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--brand-primary);
}
.org-hero h1 {
  margin: 0;
  font-size: 30px;
  line-height: 1.1;
  color: var(--text-primary);
}
.org-subtitle {
  max-width: 720px;
  margin: 14px 0 0;
  color: var(--text-secondary);
  line-height: 1.7;
}
.org-hero-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}
.org-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 18px;
}
.summary-card,
.org-card,
.org-empty {
  border-radius: 24px;
  border: 1px solid var(--border-soft);
  background: var(--surface-elevated);
  box-shadow: var(--shadow-soft);
}
.summary-card {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.summary-card span,
.meta-item span {
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-muted);
}
.summary-card strong,
.meta-item strong {
  font-size: 28px;
  color: var(--text-primary);
}
.summary-card small {
  color: var(--text-secondary);
}
.accent-card {
  background: linear-gradient(145deg, color-mix(in srgb, var(--brand-primary) 14%, transparent), color-mix(in srgb, var(--surface-elevated) 96%, transparent));
}
.org-empty {
  padding: 48px 24px;
  text-align: center;
}
.empty-illustration {
  width: 88px;
  height: 88px;
  margin: 0 auto 20px;
  border-radius: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, rgba(0, 185, 107, 0.16), rgba(31, 41, 55, 0.06));
  color: var(--brand-primary);
  font-weight: 700;
}
.org-empty h2 {
  margin: 0 0 10px;
  color: var(--text-primary);
}
.org-empty p {
  max-width: 640px;
  margin: 0 auto;
  color: var(--text-secondary);
  line-height: 1.7;
}
.empty-actions {
  display: flex;
  justify-content: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-top: 24px;
}
.org-list {
  display: grid;
  gap: 16px;
}
.org-card {
  padding: 24px;
}
.org-card-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 12px;
}
.org-card-head h3 {
  margin: 0;
  font-size: 22px;
  color: var(--text-primary);
}
.org-desc {
  margin: 0 0 18px;
  color: var(--text-secondary);
  line-height: 1.7;
}
.org-meta-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}
.meta-item {
  border-radius: 18px;
  padding: 16px 18px;
  background: color-mix(in srgb, var(--sk-surface-soft) 88%, transparent);
  border: 1px solid var(--border-soft);
}
.org-loading {
  min-height: 240px;
  display: flex;
  align-items: center;
  justify-content: center;
}
@media (max-width: 980px) {
  .org-summary,
  .org-meta-grid {
    grid-template-columns: 1fr;
  }
}
</style>
