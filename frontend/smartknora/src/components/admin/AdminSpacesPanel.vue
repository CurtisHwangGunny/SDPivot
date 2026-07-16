<template>
  <div class="admin-panel">
    <div class="panel-head">
      <div>
        <p class="panel-kicker">Space registry</p>
        <h2>空间管理</h2>
      </div>
      <t-tag variant="light">{{ spaces.length }} 个空间</t-tag>
    </div>
    <t-table v-if="spaces.length > 0" :data="spaces" :columns="spaceColumns" row-key="id" hover size="small">
      <template #visibility="{ row }">
        <t-tag theme="success" variant="light" size="small">{{ row.visibility }}</t-tag>
      </template>
      <template #created_at="{ row }">{{ formatDate(row.created_at) }}</template>
    </t-table>
    <div v-else class="empty-state">暂无空间数据</div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  spaces: any[]
  spaceColumns: any[]
  formatDate: (value: string) => string
}>()
</script>

<style scoped>
.admin-panel {
  padding: 24px;
  border-radius: 24px;
  border: 1px solid var(--border-soft);
  background: var(--surface-elevated);
  box-shadow: var(--shadow-soft);
}
.panel-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 18px;
}
.panel-kicker {
  margin: 0 0 8px;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--brand-primary);
}
.panel-head h2 {
  margin: 0;
  font-size: 30px;
  line-height: 1.1;
  color: var(--text-primary);
}
.empty-state {
  padding: 32px 16px;
  text-align: center;
  color: var(--text-secondary);
}
</style>
