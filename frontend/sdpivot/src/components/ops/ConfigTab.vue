<template>
  <div>
    <t-card title="试用期配置">
      <div style="display:flex;gap:16px;align-items:center;margin-bottom:16px;flex-wrap:wrap">
        <span>试用期天数:</span>
        <t-input v-model="trialForm.trial_days" type="number" style="width:80px" />
        <span>认证后延长:</span>
        <t-input v-model="trialForm.extended_trial_days" type="number" style="width:80px" />
        <t-button theme="primary" @click="saveTrial">保存</t-button>
      </div>
    </t-card>
    <t-card title="全局配置" style="margin-top:12px">
      <t-table :data="configs" :columns="configColumns" row-key="id" hover stripe size="small">
        <template #operation="{ row }">
          <t-button size="small" theme="primary" variant="text" @click="editConfig(row)">编辑</t-button>
        </template>
      </t-table>
    </t-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import * as opsApi from '@/api/ops'

const configs = ref<any[]>([])
const trialForm = reactive({ trial_days: '30', extended_trial_days: '90' })
const configColumns = [
  { colKey: 'key', title: '配置项', width: 200 },
  { colKey: 'value', title: '值', ellipsis: true },
  { colKey: 'description', title: '说明', ellipsis: true },
  { colKey: 'operation', title: '操作', width: 80 },
]

async function loadConfigs() {
  try {
    const r = await opsApi.listConfigs()
    configs.value = (r.data as any).configs || []
  } catch {}
}

async function loadTrialConfig() {
  try {
    const r = await opsApi.getTrialConfig()
    const d = r.data as any
    trialForm.trial_days = d.trial_days || '30'
    trialForm.extended_trial_days = d.extended_trial_days || '90'
  } catch {}
}

async function saveTrial() {
  try {
    await opsApi.updateTrialConfig({ trial_days: parseInt(trialForm.trial_days), extended_trial_days: parseInt(trialForm.extended_trial_days) })
    MessagePlugin.success('保存成功')
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.error || '保存失败')
  }
}

function editConfig(row: any) {
  const val = window.prompt('修改配置值：', row.value)
  if (val !== null) {
    opsApi.updateConfig(row.key, val, row.description)
      .then(() => {
        MessagePlugin.success('更新成功')
        loadConfigs()
      })
      .catch(() => MessagePlugin.error('更新失败'))
  }
}

onMounted(async () => {
  await Promise.all([loadConfigs(), loadTrialConfig()])
})
</script>
