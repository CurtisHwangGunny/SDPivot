<template>
  <div class="admin-panel">
    <div class="panel-head">
      <div>
        <p class="panel-kicker">Member registry</p>
        <h2>成员管理</h2>
      </div>
      <t-tag variant="light">{{ members.length }} 位成员</t-tag>
    </div>
    <t-table v-if="members.length > 0" :data="members" :columns="memberColumns" row-key="id" hover size="small">
      <template #role="{ row }">
        <t-tag :theme="row.role === 'admin' ? 'success' : 'default'" variant="light" size="small">{{ row.role }}</t-tag>
      </template>
    </t-table>
    <div v-else class="empty-state">暂无成员数据</div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  members: any[]
  memberColumns: any[]
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
