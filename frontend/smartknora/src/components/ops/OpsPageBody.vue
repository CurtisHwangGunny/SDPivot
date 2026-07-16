<template>
  <div>
    <t-tabs v-model="activeTab" style="margin-top:16px">
      <t-tab-panel value="enterprises" label="企业管理">
        <EnterpriseTab v-if="activeTab === 'enterprises'" />
      </t-tab-panel>
      <t-tab-panel value="users" label="用户管理">
        <UsersTab v-if="activeTab === 'users'" />
      </t-tab-panel>
      <t-tab-panel value="filters" label="敏感词管理">
        <FiltersTab v-if="activeTab === 'filters'" />
      </t-tab-panel>
      <t-tab-panel value="billing" label="计费管理">
        <BillingTab v-if="activeTab === 'billing'" />
      </t-tab-panel>
      <t-tab-panel value="config" label="系统配置">
        <ConfigTab v-if="activeTab === 'config'" />
      </t-tab-panel>
      <t-tab-panel value="models" label="模型管理">
        <ModelsTab v-if="activeTab === 'models'" />
      </t-tab-panel>
      <t-tab-panel value="audit" label="审计日志">
        <AuditTab v-if="activeTab === 'audit'" />
      </t-tab-panel>
      <t-tab-panel value="announcements" label="公告管理">
        <AnnouncementsTab v-if="activeTab === 'announcements'" />
      </t-tab-panel>
    </t-tabs>
  </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent, ref } from 'vue'

const activeTab = ref('enterprises')

const EnterpriseTab = defineAsyncComponent(() => import('./EnterpriseTab.vue'))
const UsersTab = defineAsyncComponent(() => import('./UsersTab.vue'))
const FiltersTab = defineAsyncComponent(() => import('./FiltersTab.vue'))
const BillingTab = defineAsyncComponent(() => import('./BillingTab.vue'))
const ConfigTab = defineAsyncComponent(() => import('./ConfigTab.vue'))
const ModelsTab = defineAsyncComponent(() => import('./ModelsTab.vue'))
const AuditTab = defineAsyncComponent(() => import('./AuditTab.vue'))
const AnnouncementsTab = defineAsyncComponent(() => import('./AnnouncementsTab.vue'))
</script>
