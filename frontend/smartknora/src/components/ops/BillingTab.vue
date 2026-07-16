<template>
  <t-card>
    <div style="display:flex;gap:8px;margin-bottom:12px">
      <t-input v-model="planForm.name" placeholder="方案名称" style="width:120px" />
      <t-input v-model="planForm.price" placeholder="价格" type="number" style="width:100px" />
      <t-input v-model="planForm.token_quota" placeholder="Token额度" type="number" style="width:120px" />
      <t-input v-model="planForm.storage_quota" placeholder="存储(MB)" type="number" style="width:120px" />
      <t-button theme="primary" @click="addPlan" :disabled="!planForm.name">添加方案</t-button>
    </div>
    <t-table :data="plans" :columns="planColumns" row-key="id" hover stripe size="small">
      <template #operation="{ row }">
        <t-button size="small" theme="danger" variant="text" @click="delPlan(row)">删除</t-button>
      </template>
    </t-table>
  </t-card>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import * as opsApi from '@/api/ops'

const plans = ref<any[]>([])
const planForm = reactive({ name: '', price: '0', token_quota: '0', storage_quota: '0' })
const planColumns = [
  { colKey: 'name', title: '方案名称', width: 120 },
  { colKey: 'price', title: '价格', width: 80 },
  { colKey: 'token_quota', title: 'Token额度', width: 120 },
  { colKey: 'storage_quota', title: '存储(MB)', width: 100 },
  { colKey: 'status', title: '状态', width: 80 },
  { colKey: 'operation', title: '操作', width: 80 },
]

async function loadPlans() {
  try {
    const r = await opsApi.listBillingPlans()
    plans.value = (r.data as any).plans || []
  } catch {}
}

async function addPlan() {
  try {
    await opsApi.createBillingPlan({ name: planForm.name, price: parseFloat(planForm.price), token_quota: parseInt(planForm.token_quota), storage_quota: parseInt(planForm.storage_quota) * 1048576 })
    MessagePlugin.success('添加成功')
    planForm.name = ''
    planForm.price = '0'
    planForm.token_quota = '0'
    planForm.storage_quota = '0'
    loadPlans()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '添加失败')
  }
}

async function delPlan(row: any) {
  try {
    await opsApi.deleteBillingPlan(row.id)
    MessagePlugin.success('删除成功')
    loadPlans()
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '删除失败')
  }
}

onMounted(loadPlans)
</script>
