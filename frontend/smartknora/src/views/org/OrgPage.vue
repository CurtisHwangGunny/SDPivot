<template>
  <div class="org-page">
    <div class="page-header">
      <h1>企业管理</h1>
      <t-space>
        <t-button theme="primary" @click="showCreate = true">创建企业</t-button>
        <t-button variant="outline" @click="showJoin = true">加入企业</t-button>
      </t-space>
    </div>

    <t-loading v-if="loading" />
    <div v-else-if="orgs.length === 0" class="empty">
      <t-icon name="building" size="64px" style="color:#d0d0d0" />
      <p class="empty-title">尚未关联企业</p>
      <p class="empty-desc">创建企业或通过企业ID加入已有企业</p>
    </div>
    <div v-else class="org-list">
      <t-card v-for="org in orgs" :key="org.id" class="org-card">
        <div class="org-info">
          <h3>{{ org.name }}</h3>
          <p class="org-desc">{{ org.description || '暂无描述' }}</p>
          <t-tag theme="success" variant="light">{{ org.auth_status || 'active' }}</t-tag>
        </div>
      </t-card>
    </div>

    <t-dialog v-model:visible="showCreate" header="创建企业" @confirm="handleCreate" :confirm-btn="{ loading: creating }">
      <t-form>
        <t-form-item label="企业名称" name="name">
          <t-input v-model="createForm.name" placeholder="请输入企业名称" />
        </t-form-item>
        <t-form-item label="企业描述" name="description">
          <t-textarea v-model="createForm.description" placeholder="选填" :autosize="{ minRows: 2 }" />
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog v-model:visible="showJoin" header="加入企业" @confirm="handleJoin" :confirm-btn="{ loading: joining }">
      <t-form>
        <t-form-item label="企业ID" name="org_id">
          <t-input v-model="joinForm.org_id" placeholder="请输入企业ID" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listOrganizations, createOrganization, joinOrganization, type Organization } from '@/api/org'
import { MessagePlugin } from 'tdesign-vue-next'

const orgs = ref<Organization[]>([])
const loading = ref(true)
const showCreate = ref(false)
const showJoin = ref(false)
const creating = ref(false)
const joining = ref(false)
const createForm = ref({ name: '', description: '' })
const joinForm = ref({ org_id: '' })

async function load() {
  loading.value = true
  try {
    const res = await listOrganizations()
    orgs.value = (res.data as any).organizations || []
  } catch { orgs.value = [] }
  finally { loading.value = false }
}

async function handleCreate() {
  if (!createForm.value.name) { MessagePlugin.warning('请输入企业名称'); return }
  creating.value = true
  try {
    await createOrganization(createForm.value)
    MessagePlugin.success('企业创建成功')
    showCreate.value = false
    createForm.value = { name: '', description: '' }
    load()
  } catch (e: any) { MessagePlugin.error(e.response?.data?.error || '创建失败') }
  finally { creating.value = false }
}

async function handleJoin() {
  if (!joinForm.value.org_id) { MessagePlugin.warning('请输入企业ID'); return }
  joining.value = true
  try {
    await joinOrganization(joinForm.value)
    MessagePlugin.success('申请已提交')
    showJoin.value = false
    joinForm.value = { org_id: '' }
    load()
  } catch (e: any) { MessagePlugin.error(e.response?.data?.error || '加入失败') }
  finally { joining.value = false }
}

onMounted(load)
</script>

<style scoped>
.org-page { max-width: 900px; margin: 0 auto; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px; }
.page-header h1 { font-size: 24px; font-weight: 600; }
.empty { text-align: center; padding: 80px 20px; }
.empty-title { font-size: 18px; font-weight: 500; margin: 16px 0 8px; }
.empty-desc { color: #999; }
.org-list { display: flex; flex-direction: column; gap: 12px; }
.org-card { cursor: pointer; }
.org-info h3 { font-size: 16px; font-weight: 600; margin-bottom: 4px; }
.org-desc { font-size: 13px; color: #999; margin-bottom: 8px; }
</style>
